package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// IPWhitelistMiddleware checks if the client IP is in the CIDR whitelist
// If not in whitelist, returns 404 for security (doesn't reveal the endpoint exists)
// If in whitelist and accessing root paths, redirects to secret path
// PWA files (manifest.json, icons) are always accessible for mobile app installation
// proxyTrustCount: number of proxies to trust (0 = don't trust any proxy headers, use RemoteAddr only)
func IPWhitelistMiddleware(whitelistCIDRs []string, sharedSecret string, proxyTrustCount int, next http.Handler) http.Handler {
	// Parse CIDR blocks
	var allowedNets []*net.IPNet
	for _, cidr := range whitelistCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Warn("invalid CIDR in whitelist", "cidr", cidr, "error", err)
			continue
		}
		allowedNets = append(allowedNets, ipNet)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP (respecting proxy trust count)
		clientIP := extractClientIP(r, proxyTrustCount)

		// Parse client IP
		ip := net.ParseIP(clientIP)
		if ip == nil {
			slog.Warn("failed to parse client IP", "remote_addr", r.RemoteAddr, "parsed_ip", clientIP)
			http.NotFound(w, r)
			return
		}

		// Check if accessing PWA files (always accessible for mobile app installation)
		if isPWAFile(r.URL.Path) {
			slog.Info("request to PWA file",
				"client_ip", clientIP,
				"path", r.URL.Path,
				"method", r.Method,
			)
			next.ServeHTTP(w, r)
			return
		}

		// Check if accessing secret path
		secretPrefix := "/" + sharedSecret + "/"
		secretRoot := strings.TrimSuffix(secretPrefix, "/") // e.g. "/dev"
		isSecretPath := strings.HasPrefix(r.URL.Path, secretPrefix)

		// Ensure /<secret> (no trailing slash) canonicalizes to /<secret>/.
		// This also fixes cases where browsers may have cached an older redirect target.
		if r.URL.Path == secretRoot {
			slog.Info("redirecting to canonical secret path",
				"client_ip", clientIP,
				"from", r.URL.Path,
				"to", secretPrefix,
			)
			http.Redirect(w, r, secretPrefix, http.StatusPermanentRedirect)
			return
		}

		// Secret path is accessible to everyone
		if isSecretPath {
			slog.Info("request to secret path",
				"client_ip", clientIP,
				"path", r.URL.Path,
				"method", r.Method,
			)
			// Wrap response writer to log when response is written and ensure flushing
			wrapped := &responseWriter{ResponseWriter: w, statusCode: 0, hijacked: false}
			next.ServeHTTP(wrapped, r)
			// Ensure response is flushed (but not if connection was hijacked for WebSocket)
			if !wrapped.hijacked && wrapped.ResponseWriter != nil {
				if flusher, ok := wrapped.ResponseWriter.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			if wrapped.statusCode == 0 {
				slog.Warn("handler did not write response",
					"client_ip", clientIP,
					"path", r.URL.Path,
				)
			} else {
				slog.Info("response sent",
					"client_ip", clientIP,
					"path", r.URL.Path,
					"status", wrapped.statusCode,
				)
			}
			return
		}

		// Check if IP is in whitelist
		allowed := false
		for _, ipNet := range allowedNets {
			if ipNet.Contains(ip) {
				allowed = true
				break
			}
		}

		if !allowed {
			// IP not in whitelist and not accessing secret path - return 404
			slog.Warn("unauthorized access attempt",
				"client_ip", clientIP,
				"path", r.URL.Path,
				"method", r.Method,
			)
			http.NotFound(w, r)
			return
		}

		// IP is in whitelist - handle root paths
		if r.URL.Path == "/" {
			// Redirect homepage to secret path (with trailing slash for directory)
			targetPath := secretPrefix // Already has trailing slash

			slog.Info("redirecting whitelisted IP to secret path",
				"client_ip", clientIP,
				"from", r.URL.Path,
				"to", targetPath,
			)

			http.Redirect(w, r, targetPath, http.StatusMovedPermanently)
			return
		}

		if r.URL.Path == "/ws" {
			// WebSocket connections can't follow redirects!
			// Check if this is a WebSocket upgrade request
			isWebSocket := strings.ToLower(r.Header.Get("Upgrade")) == "websocket"

			if isWebSocket {
				// Allow WebSocket connection through for whitelisted IPs
				// The frontend should connect to /dev/ws, but we'll be flexible
				slog.Info("allowing whitelisted IP WebSocket at root path",
					"client_ip", clientIP,
					"path", r.URL.Path,
				)
				next.ServeHTTP(w, r)
				return
			}

			// Non-WebSocket request to /ws - redirect
			targetPath := secretPrefix + "ws"
			slog.Info("redirecting whitelisted IP to secret path",
				"client_ip", clientIP,
				"from", r.URL.Path,
				"to", targetPath,
			)
			http.Redirect(w, r, targetPath, http.StatusMovedPermanently)
			return
		}

		// Other paths for whitelisted IPs - return 404 for security
		slog.Warn("whitelisted IP accessing non-root path without secret",
			"client_ip", clientIP,
			"path", r.URL.Path,
		)
		http.NotFound(w, r)
	})
}

// extractClientIP extracts the real client IP from the request
// proxyTrustCount: number of proxies to trust (0 = don't trust any proxy headers)
// If proxyTrustCount is 0, only RemoteAddr is used (most secure)
// If proxyTrustCount > 0, X-Real-IP and X-Forwarded-For headers are trusted
func extractClientIP(r *http.Request, proxyTrustCount int) string {
	// If we don't trust any proxies, only use RemoteAddr
	if proxyTrustCount == 0 {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// If SplitHostPort fails, might be IPv6 or just IP without port
			return r.RemoteAddr
		}
		return host
	}

	// Trust proxy headers - try X-Real-IP first (most reliable if set by trusted proxy)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Try X-Forwarded-For (may contain multiple IPs)
	// Format: "client, proxy1, proxy2, ..."
	// We trust proxyTrustCount proxies, so we take the IP at position (len - proxyTrustCount - 1)
	// If there are fewer proxies than we trust, we take the leftmost (original client)
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		// Trim spaces from all IPs
		for i := range ips {
			ips[i] = strings.TrimSpace(ips[i])
		}

		if len(ips) > 0 {
			// Calculate which IP to trust
			// If we trust N proxies and there are M IPs:
			// - If M > N: take IP at position (M - N - 1), which is the client IP
			// - If M <= N: take the leftmost IP (original client)
			trustedIndex := len(ips) - proxyTrustCount - 1
			if trustedIndex < 0 {
				trustedIndex = 0
			}
			if trustedIndex < len(ips) {
				return ips[trustedIndex]
			}
		}
	}

	// Fall back to RemoteAddr if proxy headers are not present or invalid
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, might be IPv6 or just IP without port
		return r.RemoteAddr
	}
	return host
}

// isPWAFile checks if the requested path is a PWA-related file that should be publicly accessible
// These files are needed for iOS/Android home screen installation to work properly
func isPWAFile(path string) bool {
	// List of PWA files that must be publicly accessible
	pwaFiles := []string{
		"/manifest.json",
		"/manifest.webmanifest",
		"/icon.svg",
		"/icon.png",
		"/apple-touch-icon.png",
		"/favicon.ico",
		"/robots.txt",
		"/site.webmanifest",
	}

	for _, file := range pwaFiles {
		if path == file {
			return true
		}
	}

	// Also allow common icon paths
	if strings.HasPrefix(path, "/icons/") || strings.HasPrefix(path, "/assets/icons/") {
		return true
	}

	return false
}

// responseWriter wraps http.ResponseWriter to track status code
// It implements all necessary interfaces including http.Hijacker for WebSocket upgrades
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	hijacked   bool
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = 200
	}
	return rw.ResponseWriter.Write(b)
}

// Implement http.Hijacker for WebSocket upgrades
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		rw.hijacked = true
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// Implement http.Flusher
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// HTTPLoggingMiddleware logs all HTTP requests with comprehensive details
// including IP addresses, headers, response time, and standard HTTP fields
func HTTPLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract client IP information
		clientIP := extractClientIPForLogging(r)
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		xRealIP := r.Header.Get("X-Real-IP")
		userAgent := r.Header.Get("User-Agent")
		referer := r.Header.Get("Referer")

		// Get request size
		requestSize := int64(0)
		if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
			if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
				requestSize = size
			}
		}

		// Wrap response writer to track response details
		loggingWriter := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     0,
			responseSize:   0,
			hijacked:       false,
		}

		// Process request
		next.ServeHTTP(loggingWriter, r)

		// Calculate response time
		duration := time.Since(start)

		// Build log attributes
		logAttrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"client_ip", clientIP,
			"status", loggingWriter.statusCode,
			"response_time_ms", duration.Milliseconds(),
			"response_time_us", duration.Microseconds(),
			"response_size", loggingWriter.responseSize,
		}

		// Add request size if available
		if requestSize > 0 {
			logAttrs = append(logAttrs, "request_size", requestSize)
		}

		// Add proxy headers if present
		if xForwardedFor != "" {
			logAttrs = append(logAttrs, "x_forwarded_for", xForwardedFor)
		}
		if xRealIP != "" {
			logAttrs = append(logAttrs, "x_real_ip", xRealIP)
		}

		// Add user agent if present
		if userAgent != "" {
			logAttrs = append(logAttrs, "user_agent", userAgent)
		}

		// Add referer if present
		if referer != "" {
			logAttrs = append(logAttrs, "referer", referer)
		}

		// Add protocol
		if r.TLS != nil {
			logAttrs = append(logAttrs, "protocol", "HTTPS")
		} else {
			logAttrs = append(logAttrs, "protocol", "HTTP")
		}

		// Add remote address (original connection info)
		logAttrs = append(logAttrs, "remote_addr", r.RemoteAddr)

		// Log based on status code
		switch {
		case loggingWriter.hijacked:
			// WebSocket connections are hijacked, log differently
			slog.Info("http request (websocket)", logAttrs...)
		case loggingWriter.statusCode == 0:
			// Handler didn't write response
			slog.Warn("http request (no response)", logAttrs...)
		case loggingWriter.statusCode >= 500:
			slog.Error("http request", logAttrs...)
		case loggingWriter.statusCode >= 400:
			slog.Warn("http request", logAttrs...)
		default:
			slog.Info("http request", logAttrs...)
		}
	})
}

// extractClientIPForLogging extracts the client IP for logging purposes
// This is similar to extractClientIP but always extracts all available information
// for logging, regardless of proxy trust settings
func extractClientIPForLogging(r *http.Request) string {
	// Try X-Real-IP first (most reliable if set by trusted proxy)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Try X-Forwarded-For (may contain multiple IPs)
	// For logging, we'll take the leftmost IP (original client)
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// loggingResponseWriter wraps http.ResponseWriter to track response details
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
	hijacked     bool
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	n, err := lrw.ResponseWriter.Write(b)
	lrw.responseSize += int64(n)
	return n, err
}

// Implement http.Hijacker for WebSocket upgrades
func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		lrw.hijacked = true
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// Implement http.Flusher
func (lrw *loggingResponseWriter) Flush() {
	if flusher, ok := lrw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
