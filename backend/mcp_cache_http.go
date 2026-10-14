package main

import (
	"net/http"
)

const mcpHTTPCacheControl = "private, no-store, no-transform"

// mcpHTTPNoStore overrides the Go MCP SDK's default Cache-Control header.
// The SDK sets "no-cache, no-transform" immediately before writing the
// response, so setting the header only before ServeHTTP is insufficient.
// Preserve "no-transform" for SSE while preventing HTTP cache storage.
func mcpHTTPNoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", mcpHTTPCacheControl)
		next.ServeHTTP(&mcpNoStoreWriter{ResponseWriter: w}, r)
	})
}

type mcpNoStoreWriter struct {
	http.ResponseWriter
}

func (w *mcpNoStoreWriter) enforceCacheControl() {
	w.Header().Set("Cache-Control", mcpHTTPCacheControl)
}

func (w *mcpNoStoreWriter) WriteHeader(statusCode int) {
	w.enforceCacheControl()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *mcpNoStoreWriter) Write(b []byte) (int, error) {
	w.enforceCacheControl()
	return w.ResponseWriter.Write(b)
}

// FlushError preserves the ability to stream SSE through ResponseController.
// The SDK's streams may flush headers before writing the first body chunk.
func (w *mcpNoStoreWriter) FlushError() error {
	w.enforceCacheControl()
	return http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap preserves ResponseController support (deadlines, hijacking, etc.).
func (w *mcpNoStoreWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
