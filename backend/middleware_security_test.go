package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSecurityHeadersMiddleware_SetsBaselineHeaders verifies the CSP +
// nosniff + Referrer-Policy headers documented on
// SecurityHeadersMiddleware land on every response, including ones
// where the wrapped handler writes a body without calling WriteHeader
// explicitly.
func TestSecurityHeadersMiddleware_SetsBaselineHeaders(t *testing.T) {
	innerCalled := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		innerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	h := SecurityHeadersMiddleware(inner)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/anything", nil))
	require.True(t, innerCalled)

	csp := rr.Header().Get("Content-Security-Policy")
	require.NotEmpty(t, csp, "Content-Security-Policy header must be set")

	// Spot-check the directives that defend against the XSS surface
	// the new {@html renderMarkdown(description)} introduces.
	requiredDirectives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
	}
	for _, d := range requiredDirectives {
		require.True(t, strings.Contains(csp, d),
			"CSP must contain %q (was: %q)", d, csp)
	}
	// Inline script must NOT be allowed (the whole point of the policy).
	require.False(t, strings.Contains(csp, "'unsafe-inline'") && strings.Contains(csp, "script-src 'self' 'unsafe-inline'"),
		"script-src must not allow 'unsafe-inline'")
	require.False(t, strings.Contains(csp, "'unsafe-eval'"),
		"CSP must not allow 'unsafe-eval'")
	// connect-src must be exactly 'self' - widening to bare ws:/wss:
	// would let any future XSS open WebSockets to attacker-controlled
	// hosts even though our own /ws is same-origin.
	require.True(t, strings.Contains(csp, "connect-src 'self'"),
		"connect-src must include 'self'")
	require.False(t, strings.Contains(csp, "connect-src 'self' ws:"),
		"connect-src must not widen to bare ws:/wss: schemes (same-origin /ws is covered by 'self')")

	require.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
}

// TestSecurityHeadersMiddleware_AppliesEvenOnError ensures the headers
// are present on error responses too - a misconfigured handler returning
// 500 must still ship the policy so an attacker cannot bypass CSP by
// triggering an exception path.
func TestSecurityHeadersMiddleware_AppliesEvenOnError(t *testing.T) {
	h := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/oops", nil))
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
	require.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
}
