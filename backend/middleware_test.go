package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPWhitelistMiddleware_SecretPathAccessibleToAll(t *testing.T) {
	// Handler that should only be reached if middleware allows
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	whitelist := []string{"10.0.0.0/8"}
	sharedSecret := "test-secret"
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 0, handler)

	// Test non-whitelisted IP accessing secret path
	req := httptest.NewRequest("GET", "/test-secret/ws", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for secret path access, got %d", w.Code)
	}
}

func TestIPWhitelistMiddleware_SecretPathWithoutTrailingSlashRedirects(t *testing.T) {
	// Handler that should only be reached if middleware allows
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	whitelist := []string{"10.0.0.0/8"}
	sharedSecret := "test-secret"
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 0, handler)

	// Test non-whitelisted IP accessing /<secret> without trailing slash
	req := httptest.NewRequest("GET", "/test-secret", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusPermanentRedirect {
		t.Errorf("Expected status 308 for canonical secret redirect, got %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/test-secret/" {
		t.Errorf("Expected Location /test-secret/, got %q", got)
	}
}

func TestIPWhitelistMiddleware_NonWhitelistedIPReturns404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	whitelist := []string{"10.0.0.0/8"}
	sharedSecret := "test-secret"
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 0, handler)

	// Test non-whitelisted IP accessing root path
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-whitelisted IP, got %d", w.Code)
	}
}

func TestIPWhitelistMiddleware_WhitelistedIPRedirects(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	whitelist := []string{"10.0.0.0/8", "192.168.0.0/16"}
	sharedSecret := "test-secret"
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 0, handler)

	tests := []struct {
		name       string
		path       string
		ip         string
		headers    map[string]string
		wantStatus int
		wantPath   string
	}{
		{
			name:       "root path redirects",
			path:       "/",
			ip:         "10.0.0.5:12345",
			wantStatus: http.StatusMovedPermanently,
			wantPath:   "/test-secret/",
		},
		{
			name:       "ws path non-websocket redirects",
			path:       "/ws",
			ip:         "192.168.1.10:54321",
			wantStatus: http.StatusMovedPermanently,
			wantPath:   "/test-secret/ws",
		},
		{
			name:       "ws path with websocket upgrade allowed",
			path:       "/ws",
			ip:         "192.168.1.10:54321",
			headers:    map[string]string{"Upgrade": "websocket"},
			wantStatus: http.StatusOK,
			wantPath:   "",
		},
		{
			name:       "other path returns 404",
			path:       "/api/test",
			ip:         "10.0.1.5:12345",
			wantStatus: http.StatusNotFound,
			wantPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			req.RemoteAddr = tt.ip
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantPath != "" {
				location := w.Header().Get("Location")
				if location != tt.wantPath {
					t.Errorf("Expected redirect to %s, got %s", tt.wantPath, location)
				}
			}
		})
	}
}

func TestIPWhitelistMiddleware_XForwardedForHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	whitelist := []string{"10.0.0.0/8"}
	sharedSecret := "test-secret"
	// Trust 1 proxy to allow X-Forwarded-For header
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 1, handler)

	// Test with X-Forwarded-For header (common in proxy scenarios)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"          // Proxy IP
	req.Header.Set("X-Forwarded-For", "10.0.0.5") // Real client IP (whitelisted)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected redirect for whitelisted X-Forwarded-For IP, got status %d", w.Code)
	}
}

func TestIPWhitelistMiddleware_XRealIPHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	whitelist := []string{"172.16.0.0/12"}
	sharedSecret := "test-secret"
	// Trust 1 proxy to allow X-Real-IP header
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 1, handler)

	// Test with X-Real-IP header (common in nginx proxy scenarios)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"       // Proxy IP
	req.Header.Set("X-Real-IP", "172.16.5.10") // Real client IP (whitelisted)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected redirect for whitelisted X-Real-IP, got status %d", w.Code)
	}
}

func TestIPWhitelistMiddleware_InvalidCIDR(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Include invalid CIDR that should be ignored
	whitelist := []string{"10.0.0.0/8", "invalid-cidr", "192.168.0.0/16"}
	sharedSecret := "test-secret"
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 0, handler)

	// Test that valid CIDRs still work
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected redirect despite invalid CIDR in list, got status %d", w.Code)
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name            string
		remoteAddr      string
		xForwardedFor   string
		xRealIP         string
		proxyTrustCount int
		expectedIP      string
	}{
		{
			name:            "RemoteAddr only (proxyTrustCount=0)",
			remoteAddr:      "192.168.1.100:12345",
			proxyTrustCount: 0,
			expectedIP:      "192.168.1.100",
		},
		{
			name:            "Proxy headers ignored when proxyTrustCount=0",
			remoteAddr:      "192.168.1.100:12345",
			xRealIP:         "10.0.0.5",
			xForwardedFor:   "172.16.0.1",
			proxyTrustCount: 0,
			expectedIP:      "192.168.1.100", // Should use RemoteAddr, not proxy headers
		},
		{
			name:            "X-Real-IP takes precedence (proxyTrustCount=1)",
			remoteAddr:      "192.168.1.100:12345",
			xRealIP:         "10.0.0.5",
			xForwardedFor:   "172.16.0.1",
			proxyTrustCount: 1,
			expectedIP:      "10.0.0.5",
		},
		{
			name:            "X-Forwarded-For when no X-Real-IP (proxyTrustCount=1)",
			remoteAddr:      "192.168.1.100:12345",
			xForwardedFor:   "172.16.0.1, 192.168.1.1",
			proxyTrustCount: 1,
			expectedIP:      "172.16.0.1",
		},
		{
			name:            "X-Forwarded-For with multiple proxies (proxyTrustCount=2)",
			remoteAddr:      "192.168.1.100:12345",
			xForwardedFor:   "10.0.0.5, 172.16.0.1, 192.168.1.1",
			proxyTrustCount: 2,
			expectedIP:      "10.0.0.5", // Should take client IP (first one)
		},
		{
			name:            "X-Forwarded-For with fewer proxies than trusted",
			remoteAddr:      "192.168.1.100:12345",
			xForwardedFor:   "10.0.0.5",
			proxyTrustCount: 2,          // Trust 2 proxies but only 1 in header
			expectedIP:      "10.0.0.5", // Should take leftmost (client)
		},
		{
			name:            "IPv6 address",
			remoteAddr:      "[2001:db8::1]:12345",
			proxyTrustCount: 0,
			expectedIP:      "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := extractClientIP(req, tt.proxyTrustCount)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestIPWhitelistMiddleware_PWAFilesAccessible(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	whitelist := []string{"10.0.0.0/8"}
	sharedSecret := "test-secret"
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 0, handler)

	pwaFiles := []string{
		"/manifest.json",
		"/manifest.webmanifest",
		"/icon.svg",
		"/icon.png",
		"/apple-touch-icon.png",
		"/favicon.ico",
		"/robots.txt",
		"/site.webmanifest",
		"/icons/icon-192.png",
		"/assets/icons/favicon.ico",
	}

	for _, path := range pwaFiles {
		t.Run(path, func(t *testing.T) {
			// Test non-whitelisted IP accessing PWA file (should succeed)
			req := httptest.NewRequest("GET", path, nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for PWA file %s from non-whitelisted IP, got %d", path, w.Code)
			}

			if w.Body.String() != "success" {
				t.Errorf("Expected body 'success', got '%s'", w.Body.String())
			}
		})
	}
}

func TestIPWhitelistMiddleware_NonPWAFilesRestricted(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	whitelist := []string{"10.0.0.0/8"}
	sharedSecret := "test-secret"
	middleware := IPWhitelistMiddleware(whitelist, sharedSecret, 0, handler)

	restrictedPaths := []string{
		"/index.html",
		"/assets/index.js",
		"/api/data",
		"/something.json",
	}

	for _, path := range restrictedPaths {
		t.Run(path, func(t *testing.T) {
			// Test non-whitelisted IP accessing non-PWA file (should fail)
			req := httptest.NewRequest("GET", path, nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("Expected status 404 for non-PWA file %s from non-whitelisted IP, got %d", path, w.Code)
			}
		})
	}
}
