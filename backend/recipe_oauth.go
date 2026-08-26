package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	codexOAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexOAuthTokenURL = "https://auth.openai.com/oauth/token"
	codexRefreshSkew   = 2 * time.Minute
	codexTokenMaxBytes = 64 * 1024
)

type CodexCredentials struct {
	AccessToken string
	AccountID   string
}

// CodexOAuthManager owns one dedicated Codex CLI credential file. A process
// lock prevents multiple FoodList instances from rotating the same refresh
// token. It cannot coordinate with Codex/OpenClaw, which is why the imported
// OAuth grant must be dedicated to FoodList.
type CodexOAuthManager struct {
	path     string
	tokenURL string
	http     *http.Client
	mu       sync.Mutex
	lockFile *os.File
}

func NewCodexOAuthManager(path, dataDir string) (*CodexOAuthManager, error) {
	return newCodexOAuthManager(path, dataDir, noRedirectHTTPClient(20*time.Second))
}

func newCodexOAuthManager(path, dataDir string, client *http.Client) (*CodexOAuthManager, error) {
	resolved, err := validateCodexAuthPath(path, dataDir)
	if err != nil {
		return nil, fmt.Errorf("%w: oauth credential file", ErrLLMConfigInvalid)
	}
	if client == nil {
		client = noRedirectHTTPClient(20 * time.Second)
	}

	lockPath := resolved + ".lock"
	if info, err := os.Lstat(lockPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: oauth lock is symlink", ErrLLMConfigInvalid)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: oauth lock stat", ErrLLMConfigInvalid)
	}
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: oauth lock open", ErrLLMConfigInvalid)
	}
	if err := lockFile.Chmod(0o600); err != nil {
		_ = lockFile.Close()
		return nil, fmt.Errorf("%w: oauth lock permissions", ErrLLMConfigInvalid)
	}
	if err := unix.Flock(int(lockFile.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lockFile.Close()
		return nil, fmt.Errorf("%w: oauth credentials already in use", ErrLLMConfigInvalid)
	}

	m := &CodexOAuthManager{path: resolved, tokenURL: codexOAuthTokenURL, http: client, lockFile: lockFile}
	if _, _, err := m.load(); err != nil {
		_ = m.Close()
		return nil, fmt.Errorf("%w: oauth credential contents", ErrLLMConfigInvalid)
	}
	return m, nil
}

func (m *CodexOAuthManager) Close() error {
	if m == nil || m.lockFile == nil {
		return nil
	}
	_ = unix.Flock(int(m.lockFile.Fd()), unix.LOCK_UN)
	err := m.lockFile.Close()
	m.lockFile = nil
	return err
}

func validateCodexAuthPath(path, dataDir string) (string, error) {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(dataDir) == "" {
		return "", ErrLLMConfigInvalid
	}
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absData, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrLLMConfigInvalid
	}
	realData, err := filepath.EvalSymlinks(absData)
	if err != nil {
		return "", ErrLLMConfigInvalid
	}
	realParent, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		return "", ErrLLMConfigInvalid
	}
	realRel, err := filepath.Rel(realData, realParent)
	if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
		return "", ErrLLMConfigInvalid
	}
	info, err := os.Lstat(absPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return "", ErrLLMConfigInvalid
	}
	dirInfo, err := os.Stat(filepath.Dir(absPath))
	if err != nil || !dirInfo.IsDir() || dirInfo.Mode().Perm()&0o077 != 0 {
		return "", ErrLLMConfigInvalid
	}
	return absPath, nil
}

func noRedirectHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (m *CodexOAuthManager) Credentials(ctx context.Context, forceRefresh bool) (CodexCredentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	doc, tokens, err := m.load()
	if err != nil {
		return CodexCredentials{}, newRecipeLLMError(LLMErrorAuth, err)
	}
	access := stringValue(tokens["access_token"])
	account := stringValue(tokens["account_id"])
	claimAccount, expiresAt, claimErr := codexAccessTokenMetadata(access)
	if account == "" {
		account = claimAccount
	}
	if claimAccount != "" && account != claimAccount {
		return CodexCredentials{}, newRecipeLLMError(LLMErrorAuth, errors.New("oauth account mismatch"))
	}
	if access == "" || account == "" {
		return CodexCredentials{}, newRecipeLLMError(LLMErrorAuth, errors.New("oauth credentials incomplete"))
	}

	shouldRefresh := forceRefresh || claimErr != nil || time.Until(expiresAt) <= codexRefreshSkew
	if shouldRefresh {
		access, account, err = m.refresh(ctx, doc, tokens, account)
		if err != nil {
			return CodexCredentials{}, err
		}
	}
	return CodexCredentials{AccessToken: access, AccountID: account}, nil
}

func (m *CodexOAuthManager) load() (map[string]any, map[string]any, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil, nil, err
	}
	if len(data) > codexTokenMaxBytes {
		return nil, nil, errors.New("oauth credential file too large")
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, err
	}
	if stringValue(doc["auth_mode"]) != "chatgpt" {
		return nil, nil, errors.New("oauth auth_mode must be chatgpt")
	}
	tokens, ok := doc["tokens"].(map[string]any)
	if !ok {
		return nil, nil, errors.New("oauth tokens missing")
	}
	if stringValue(tokens["access_token"]) == "" || stringValue(tokens["refresh_token"]) == "" {
		return nil, nil, errors.New("oauth token values missing")
	}
	return doc, tokens, nil
}

func (m *CodexOAuthManager) refresh(ctx context.Context, doc, tokens map[string]any, expectedAccount string) (string, string, error) {
	refreshToken := stringValue(tokens["refresh_token"])
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {codexOAuthClientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", newRecipeLLMError(LLMErrorUpstream, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "foodlist/oauth-refresh")
	resp, err := m.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", "", newRecipeLLMError(LLMErrorTimeout, err)
		}
		return "", "", newRecipeLLMError(LLMErrorUpstream, err)
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, codexTokenMaxBytes+1)
	body, readErr := io.ReadAll(limited)
	if readErr != nil || len(body) > codexTokenMaxBytes {
		return "", "", newRecipeLLMError(LLMErrorUpstream, errors.New("oauth refresh response invalid"))
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", "", newRecipeLLMQuotaError(resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", "", newRecipeLLMError(LLMErrorAuth, errors.New("oauth refresh rejected"))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", newRecipeLLMError(LLMErrorUpstream, errors.New("oauth refresh failed"))
	}

	var refreshed map[string]any
	if err := json.Unmarshal(body, &refreshed); err != nil {
		return "", "", newRecipeLLMError(LLMErrorUpstream, err)
	}
	newAccess := stringValue(refreshed["access_token"])
	if newAccess == "" {
		return "", "", newRecipeLLMError(LLMErrorAuth, errors.New("oauth refresh omitted access token"))
	}
	claimAccount, _, err := codexAccessTokenMetadata(newAccess)
	if err != nil || (claimAccount != "" && expectedAccount != "" && claimAccount != expectedAccount) {
		return "", "", newRecipeLLMError(LLMErrorAuth, errors.New("oauth refresh account mismatch"))
	}
	if claimAccount == "" {
		claimAccount = expectedAccount
	}

	tokens["access_token"] = newAccess
	if rotated := stringValue(refreshed["refresh_token"]); rotated != "" {
		tokens["refresh_token"] = rotated
	}
	if idToken := stringValue(refreshed["id_token"]); idToken != "" {
		tokens["id_token"] = idToken
	}
	tokens["account_id"] = claimAccount
	doc["last_refresh"] = time.Now().UTC().Format(time.RFC3339Nano)
	doc["tokens"] = tokens
	if err := m.save(doc); err != nil {
		// The provider may already have rotated the remote refresh token. The
		// safe recovery is a new dedicated login; never retry the old token.
		return "", "", newRecipeLLMError(LLMErrorAuth, errors.New("oauth refresh could not be persisted"))
	}
	return newAccess, claimAccount, nil
}

func (m *CodexOAuthManager) save(doc map[string]any) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(m.path)
	tmp, err := os.CreateTemp(dir, ".recipe-openai-auth-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, m.path); err != nil {
		return err
	}
	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func codexAccessTokenMetadata(token string) (string, time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", time.Time{}, errors.New("oauth access token is not a jwt")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", time.Time{}, err
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", time.Time{}, err
	}
	expFloat, ok := claims["exp"].(float64)
	if !ok || expFloat <= 0 {
		return "", time.Time{}, errors.New("oauth access token expiry missing")
	}
	account := ""
	if authClaims, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		account = stringValue(authClaims["chatgpt_account_id"])
	}
	return account, time.Unix(int64(expFloat), 0), nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}
