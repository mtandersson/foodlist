# WebSocket Redirect Fix

## The Problem

When implementing IP whitelist security with redirects, we encountered an issue:

**WebSocket connections cannot follow HTTP 301/302 redirects!**

### What Was Happening

1. Frontend tries to connect WebSocket at `/ws`
2. Backend sends `301 Moved Permanently` to `/dev/ws`
3. WebSocket upgrade fails (WebSocket protocol doesn't support redirects)
4. Frontend keeps retrying, creating a loop
5. Logs showed: `redirecting whitelisted IP to secret path client_ip=::1 from=/ws to=/dev/ws`
6. Vite showed errors: `ws proxy error: Error: write ECONNRESET`

## The Solution

For **whitelisted IPs only**, we now detect WebSocket upgrade requests and **allow them through at `/ws`** instead of redirecting:

### Changes Made

#### 1. Middleware (`backend/middleware.go`)

```go
if r.URL.Path == "/ws" {
    // Check if this is a WebSocket upgrade request
    isWebSocket := strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
    
    if isWebSocket {
        // Allow WebSocket connection through for whitelisted IPs
        slog.Info("allowing whitelisted IP WebSocket at root path",
            "client_ip", clientIP,
            "path", r.URL.Path,
        )
        next.ServeHTTP(w, r)
        return
    }
    
    // Non-WebSocket request to /ws - redirect  
    http.Redirect(w, r, secretPrefix+"ws", http.StatusMovedPermanently)
    return
}
```

#### 2. Main Router (`backend/main.go`)

Register WebSocket handler at **both** paths:

```go
// WebSocket endpoint at secret path
wsPath := pathPrefix + "ws"  // /dev/ws
mux.HandleFunc(wsPath, server.HandleWebSocket)

// Also register at /ws for whitelisted IPs
if cfg.SharedSecret != "" && len(cfg.CIDRWhitelist) > 0 {
    mux.HandleFunc("/ws", server.HandleWebSocket)
}
```

## How It Works Now

### For Whitelisted IPs (localhost)

| Request Type | Path | Result |
|--------------|------|--------|
| Browser (HTTP) | `/` | 301 → `/dev/` |
| Browser (HTTP) | `/ws` | 301 → `/dev/ws` |
| **WebSocket** | **`/ws`** | **✅ Connects directly** |
| WebSocket | `/dev/ws` | ✅ Connects |

### For Non-Whitelisted IPs

| Request Type | Path | Result |
|--------------|------|--------|
| Browser (HTTP) | `/` | 404 Not Found |
| Browser (HTTP) | `/ws` | 404 Not Found |
| WebSocket | `/ws` | 404 Not Found |
| WebSocket | `/dev/ws` | ✅ Connects |

### PWA Files (All IPs)

| Request Type | Path | Result |
|--------------|------|--------|
| Any | `/manifest.json` | ✅ Serves file |
| Any | `/icon.svg` | ✅ Serves file |

## Logs

### Successful WebSocket Connection (Whitelisted IP)

Before (broken):
```
INFO redirecting whitelisted IP to secret path client_ip=::1 from=/ws to=/dev/ws
ERROR ws proxy error: write ECONNRESET
```

After (working):
```
INFO allowing whitelisted IP WebSocket at root path client_ip=::1 path=/ws
INFO new websocket connection remote_addr=[::1]:55459
INFO client connected total_clients=1
```

### HTTP Redirect Still Works

```
INFO redirecting whitelisted IP to secret path client_ip=::1 from=/ to=/dev
```

## Why This Approach

### Alternative Approaches Considered

1. **Frontend always connects to `/dev/ws`**
   - ❌ Breaks development workflow
   - ❌ Requires environment-specific configuration
   - ❌ Complicates frontend code

2. **Use JavaScript redirect on HTTP endpoint**
   - ❌ Adds unnecessary complexity
   - ❌ Slower (extra round trip)
   - ❌ Still requires special handling

3. **Disable security in development**
   - ✅ We offer this with `make dev`
   - ❌ Can't test security features

4. **Current solution: Smart detection**
   - ✅ Works transparently
   - ✅ No frontend changes needed
   - ✅ Can test security features
   - ✅ Whitelisted IPs get convenience
   - ✅ Security maintained for non-whitelisted IPs

## Testing

### Test WebSocket Upgrade Detection

```bash
make dev-secure

# Should see in logs:
# INFO allowing whitelisted IP WebSocket at root path
# INFO new websocket connection
# INFO client connected
```

### Test HTTP Redirect Still Works

```bash
# In browser, navigate to: http://localhost:5173/
# Should redirect to: http://localhost:5173/dev/
```

### Test Non-WebSocket /ws Request

```bash
curl -I http://localhost:8080/ws
# Should return: 301 Moved Permanently
# Location: /dev/ws
```

### Test WebSocket from Non-Whitelisted IP

```bash
# Simulate external IP
curl -H "X-Real-IP: 1.2.3.4" -H "Upgrade: websocket" http://localhost:8080/ws
# Should return: 404 Not Found
```

## Implementation Notes

### Why Check the Upgrade Header?

The WebSocket protocol uses HTTP upgrade mechanism:

```http
GET /ws HTTP/1.1
Host: localhost:8080
Upgrade: websocket
Connection: Upgrade
```

By checking `Upgrade: websocket`, we can distinguish between:
- Regular HTTP requests to `/ws` → Redirect
- WebSocket upgrade requests to `/ws` → Allow through

### Security Implications

**Q: Does this weaken security?**
**A: No!** Non-whitelisted IPs still can't access `/ws`:

```go
if !allowed {
    // IP not in whitelist and not accessing secret path - return 404
    http.NotFound(w, r)
    return
}
```

The WebSocket-specific handling only applies AFTER the whitelist check passes.

### Backend Handles Both Paths

The WebSocket handler is registered at both paths:
- `/ws` - For whitelisted IPs (backwards compatibility)
- `/dev/ws` - For everyone with the secret

This provides flexibility without security compromise.

## Summary

- ✅ WebSocket connections work without redirect loops
- ✅ HTTP redirects still work for convenience
- ✅ Security maintained (non-whitelisted IPs blocked)
- ✅ No frontend code changes needed
- ✅ All tests pass
- ✅ Development workflow smooth

The fix elegantly handles the WebSocket redirect limitation while maintaining all security properties.

