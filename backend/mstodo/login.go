package mstodo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/browser"
)

// Login performs the OAuth2 flow to get a refresh token
func Login(clientID string, port string) error {
	redirectURI := fmt.Sprintf("http://localhost:%s/callback", port)
	scope := "Tasks.Read openid profile offline_access"

	// Create a channel to signal when we have the code
	codeCh := make(chan string)
	errorCh := make(chan error)

	// Start a local server to handle the callback
	server := &http.Server{Addr: ":" + port}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			codeCh <- code
			fmt.Fprint(w, "You can close this window now.")
		} else {
			errorCh <- fmt.Errorf("no code received")
			fmt.Fprint(w, "Error: No code received.")
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorCh <- err
		}
	}()

	// Construct the authorization URL
	authURL := fmt.Sprintf("https://login.microsoftonline.com/common/oauth2/v2.0/authorize?client_id=%s&response_type=code&redirect_uri=%s&response_mode=query&scope=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scope),
	)

	fmt.Println("Opening browser for login...")
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Printf("Please open this URL manually:\n%s\n", authURL)
	}

	// Wait for the code
	var code string
	select {
	case code = <-codeCh:
	case err := <-errorCh:
		return err
	case <-time.After(2 * time.Minute):
		return fmt.Errorf("timeout waiting for login")
	}

	// Shutdown the server
	go server.Shutdown(context.Background())

	// Exchange code for token
	fmt.Println("Exchanging code for token...")
	return exchangeCodeForToken(clientID, code, redirectURI)
}

func exchangeCodeForToken(clientID, code, redirectURI string) error {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", "Tasks.Read openid profile offline_access")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(defaultTokenEndpoint, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed: %s", resp.Status)
	}

	// Reuse existing struct for response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	fmt.Printf("\nSUCCESS!\n")
	fmt.Printf("Refresh Token: %s\n", tokenResp.RefreshToken)
	fmt.Printf("Access Token: %s\n", tokenResp.AccessToken)

	return nil
}
