package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Upload size cap for recipe-bearing endpoints. 10 MB is enough for
// modern phone photos after client-side resize but small enough that an
// abuser cannot exhaust memory with a single request.
const recipeUploadMaxBytes = 10 << 20

// recipeMetadataMaxBytes caps the JSON metadata part on POST/PATCH so a
// caller cannot attach a multi-megabyte string to overload memory or
// disk before bounds validation runs.
const recipeMetadataMaxBytes = 256 * 1024

// recipeAPIErrorBody is the JSON shape of every error response.
type recipeAPIErrorBody struct {
	Error string `json:"error"`
}

// recipeListResponse is the JSON returned by GET /recipes.
type recipeListResponse struct {
	Recipes []recipeListItem `json:"recipes"`
}

type recipeListItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	ImageURL  string    `json:"imageUrl"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type recipeDetailResponse struct {
	Recipe   Recipe `json:"recipe"`
	ImageURL string `json:"imageUrl"`
}

type recipeParseResponse struct {
	Parsed Recipe `json:"parsed"`
}

// RecipeAPI bundles the dependencies needed to serve recipe HTTP routes.
// Routes are only mounted when auth is configured (see main.go).
type RecipeAPI struct {
	store     *RecipeStore
	llm       *RecipeLLMClient
	server    *Server
	pathBase  string
	parseRate *rateLimiter
}

// NewRecipeAPI constructs the handler. The server is required so that
// successful Save/Update/Delete (which the store now hooks) and PATCH
// step-pruning can broadcast WS messages.
func NewRecipeAPI(store *RecipeStore, llm *RecipeLLMClient, server *Server, pathBase string, parseRPM int) *RecipeAPI {
	if parseRPM <= 0 {
		parseRPM = 10
	}
	return &RecipeAPI{
		store:     store,
		llm:       llm,
		server:    server,
		pathBase:  strings.TrimRight(pathBase, "/"),
		parseRate: newRateLimiter(parseRPM, time.Minute),
	}
}

// Register attaches the API to the given mux. The provided wrap function
// is responsible for authorization (e.g. apiBearerAuth or pathPrefix
// middleware) - both reads and writes share the same wrapper to keep the
// auth posture consistent.
func (a *RecipeAPI) Register(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	prefix := a.pathBase + "/api/v1/recipes"
	mux.Handle(prefix, wrap(http.HandlerFunc(a.handleCollection)))
	mux.Handle(prefix+"/", wrap(http.HandlerFunc(a.handleItem)))
}

func (a *RecipeAPI) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.list(w, r)
	case http.MethodPost:
		a.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleItem dispatches /api/v1/recipes/{id}, /api/v1/recipes/{id}/image,
// and the special /api/v1/recipes/parse endpoint.
func (a *RecipeAPI) handleItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, a.pathBase+"/api/v1/recipes/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if path == "parse" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		a.parse(w, r)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	// UUID-validate every {id} segment before any filesystem access. We
	// never join the raw value into a path; the store also re-validates
	// and prefix-checks, but failing fast here keeps error messages
	// identical for missing recipes and probing attempts.
	if _, err := uuid.Parse(id); err != nil {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 2 && parts[1] == "image" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		a.image(w, r, id)
		return
	}
	if len(parts) > 1 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.get(w, r, id)
	case http.MethodPatch:
		a.update(w, r, id)
	case http.MethodDelete:
		a.delete(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *RecipeAPI) list(w http.ResponseWriter, r *http.Request) {
	metas, err := a.store.List()
	if err != nil {
		// Generic message; the underlying error may include FS paths.
		writeAPIError(w, http.StatusInternalServerError, "failed to list recipes")
		slog.Error("list recipes failed", "error_class", "store_list")
		return
	}
	items := make([]recipeListItem, 0, len(metas))
	for _, m := range metas {
		items = append(items, recipeListItem{
			ID:        m.ID,
			Title:     m.Title,
			ImageURL:  a.imageURL(r, m.ID),
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, recipeListResponse{Recipes: items})
}

func (a *RecipeAPI) get(w http.ResponseWriter, r *http.Request, id string) {
	recipe, err := a.store.Get(id)
	if err != nil {
		if errors.Is(err, ErrRecipeNotFound) {
			http.NotFound(w, r)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to read recipe")
		return
	}
	writeJSON(w, http.StatusOK, recipeDetailResponse{
		Recipe:   recipe,
		ImageURL: a.imageURL(r, id),
	})
}

func (a *RecipeAPI) image(w http.ResponseWriter, r *http.Request, id string) {
	data, mime, err := a.store.ReadImage(id)
	if err != nil {
		if errors.Is(err, ErrRecipeNotFound) {
			http.NotFound(w, r)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to read image")
		return
	}
	// The MIME comes from server-side sniffing at save time, never from
	// client headers. Combined with X-Content-Type-Options: nosniff the
	// browser cannot misinterpret the bytes as HTML/JS.
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (a *RecipeAPI) parse(w http.ResponseWriter, r *http.Request) {
	if a.llm == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "recipe parsing not configured")
		return
	}
	if !a.parseRate.allow() {
		writeAPIError(w, http.StatusTooManyRequests, "too many parse requests")
		return
	}

	imgBytes, mime, err := readImageMultipart(w, r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	// HEIC uploads (the iPhone default) are transcoded to JPEG here so
	// the bounds check, the LLM call, and any future storage step all
	// see a browser-renderable mime. The helper is a passthrough for
	// already-storable mimes.
	imgBytes, mime, err = transcodeForStorage(imgBytes, mime)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "unsupported image type")
		return
	}
	if _, err := a.store.CheckImageBounds(imgBytes); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid image")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), recipeLLMTimeout+5*time.Second)
	defer cancel()
	parsed, err := a.llm.ParseImage(ctx, imgBytes, mime)
	if err != nil {
		// The LLM client already logged the error class without leaking
		// upstream content; emit a generic message to the user.
		writeAPIError(w, http.StatusUnprocessableEntity, "kunde inte tolka receptet")
		return
	}
	writeJSON(w, http.StatusOK, recipeParseResponse{Parsed: parsed})
}

func (a *RecipeAPI) create(w http.ResponseWriter, r *http.Request) {
	imgBytes, mime, metadataBytes, err := readRecipeMultipart(w, r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	// See parse(): HEIC uploads are transcoded to JPEG before any
	// downstream step touches the bytes, so the sidecar saved on disk
	// is always one of the browser-renderable storable mimes.
	imgBytes, mime, err = transcodeForStorage(imgBytes, mime)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "unsupported image type")
		return
	}
	if _, err := a.store.CheckImageBounds(imgBytes); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid image")
		return
	}

	var meta Recipe
	if len(metadataBytes) == 0 {
		writeAPIError(w, http.StatusBadRequest, "missing metadata")
		return
	}
	if err := json.Unmarshal(metadataBytes, &meta); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid metadata")
		return
	}
	// Server controls id/timestamps/image filename; never trust the client.
	meta.ID = uuid.NewString()
	meta.CreatedAt = time.Time{}
	meta.UpdatedAt = time.Time{}
	meta.ImageFilename = ""
	meta.ImageMIME = ""

	saved, err := a.store.Save(meta, imgBytes, mime)
	if err != nil {
		if errors.Is(err, ErrRecipeInvalid) {
			writeAPIError(w, http.StatusBadRequest, "recipe invalid")
			return
		}
		if errors.Is(err, ErrUnsupportedImage) {
			writeAPIError(w, http.StatusBadRequest, "unsupported image type")
			return
		}
		slog.Error("save recipe failed", "error_class", "store_save")
		writeAPIError(w, http.StatusInternalServerError, "failed to save recipe")
		return
	}
	writeJSON(w, http.StatusCreated, recipeDetailResponse{
		Recipe:   saved,
		ImageURL: a.imageURL(r, saved.ID),
	})
}

type recipePatchBody struct {
	Title        *string       `json:"title,omitempty"`
	Ingredients  *[]Ingredient `json:"ingredients,omitempty"`
	Instructions *[]string     `json:"instructions,omitempty"`
}

func (a *RecipeAPI) update(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, recipeMetadataMaxBytes)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "metadata too large")
		return
	}
	var patch recipePatchBody
	if err := json.Unmarshal(body, &patch); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid metadata")
		return
	}

	updated, err := a.store.Update(id, func(curr Recipe) (Recipe, error) {
		if patch.Title != nil {
			curr.Title = *patch.Title
		}
		if patch.Ingredients != nil {
			curr.Ingredients = *patch.Ingredients
		}
		if patch.Instructions != nil {
			curr.Instructions = *patch.Instructions
		}
		return curr, nil
	})
	if err != nil {
		if errors.Is(err, ErrRecipeNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrRecipeInvalid) {
			writeAPIError(w, http.StatusBadRequest, "recipe invalid")
			return
		}
		slog.Error("update recipe failed", "error_class", "store_update")
		writeAPIError(w, http.StatusInternalServerError, "failed to update recipe")
		return
	}

	// If the new step list shrank, prune any out-of-range cook session
	// indices. PruneAbove fires the cook broadcast hook itself when
	// the step list changes, so the explicit enqueueBroadcast that
	// used to live here would have produced a duplicate message.
	if a.server != nil {
		if sessions := a.server.CookSessions(); sessions != nil {
			sessions.PruneAbove(id, len(updated.Instructions))
		}
	}

	writeJSON(w, http.StatusOK, recipeDetailResponse{
		Recipe:   updated,
		ImageURL: a.imageURL(r, id),
	})
}

func (a *RecipeAPI) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := a.store.Delete(id); err != nil {
		if errors.Is(err, ErrRecipeNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.Error("delete recipe failed", "error_class", "store_delete")
		writeAPIError(w, http.StatusInternalServerError, "failed to delete recipe")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// imageURL returns a same-origin URL the client can GET to fetch the
// recipe's image sidecar. We honor the request's host so the browser is
// not redirected away from the secret prefix when one is configured.
func (a *RecipeAPI) imageURL(_ *http.Request, id string) string {
	return a.pathBase + "/api/v1/recipes/" + id + "/image"
}

// readImageMultipart reads a single "image" multipart field within the
// upload size limit. Returns the bytes and the sniffed MIME.
//
// Passing the ResponseWriter to MaxBytesReader lets net/http surface
// the 413 (Request Entity Too Large) on its own when the limit is
// exceeded; with nil we'd only see a generic read error.
func readImageMultipart(w http.ResponseWriter, r *http.Request) ([]byte, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, recipeUploadMaxBytes)
	if err := r.ParseMultipartForm(recipeUploadMaxBytes); err != nil {
		return nil, "", fmt.Errorf("invalid multipart body")
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, "", fmt.Errorf("missing image part")
	}
	defer file.Close()
	imgBytes, err := io.ReadAll(io.LimitReader(file, recipeUploadMaxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("invalid image part")
	}
	if int64(len(imgBytes)) > recipeUploadMaxBytes {
		return nil, "", fmt.Errorf("image too large")
	}
	mime, err := SniffImageMIME(imgBytes)
	if err != nil {
		return nil, "", fmt.Errorf("unsupported image type")
	}
	return imgBytes, mime, nil
}

// readRecipeMultipart reads both the "image" and the "metadata" parts.
// Metadata is also size-capped to prevent DoS via giant strings.
//
// Passing the ResponseWriter is what lets net/http return a real 413
// when the limit is exceeded; nil here would degrade to a generic
// read error and a 400.
func readRecipeMultipart(w http.ResponseWriter, r *http.Request) ([]byte, string, []byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, recipeUploadMaxBytes+recipeMetadataMaxBytes)
	if err := r.ParseMultipartForm(recipeUploadMaxBytes); err != nil {
		return nil, "", nil, fmt.Errorf("invalid multipart body")
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, "", nil, fmt.Errorf("missing image part")
	}
	defer file.Close()
	imgBytes, err := io.ReadAll(io.LimitReader(file, recipeUploadMaxBytes+1))
	if err != nil {
		return nil, "", nil, fmt.Errorf("invalid image part")
	}
	if int64(len(imgBytes)) > recipeUploadMaxBytes {
		return nil, "", nil, fmt.Errorf("image too large")
	}
	mime, err := SniffImageMIME(imgBytes)
	if err != nil {
		return nil, "", nil, fmt.Errorf("unsupported image type")
	}

	metadataStr := r.FormValue("metadata")
	if int64(len(metadataStr)) > recipeMetadataMaxBytes {
		return nil, "", nil, fmt.Errorf("metadata too large")
	}
	return imgBytes, mime, []byte(metadataStr), nil
}

// writeJSON serializes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json failed", "error_class", "encode")
	}
}

func writeAPIError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(recipeAPIErrorBody{Error: msg})
}

// rateLimiter is a tiny fixed-window per-process rate limiter used to
// cap how often the LLM endpoint can be hit. A token bucket would be
// nicer but the fixed window keeps the implementation tiny and the
// failure mode (occasional 429) is well-understood.
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	count   int
	resetAt time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, resetAt: time.Now().Add(window)}
}

func (l *rateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.After(l.resetAt) {
		l.count = 0
		l.resetAt = now.Add(l.window)
	}
	if l.count >= l.limit {
		return false
	}
	l.count++
	return true
}
