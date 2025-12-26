# Development Setup: How Ports Work

## Understanding the Development Flow

When running `make dev-secure` or `make dev-secure-net`, you have **two separate servers**:

### Port 5173 - Vite Dev Server (Frontend)
- **Purpose**: Serves frontend files with hot reload
- **No middleware**: Vite doesn't know about IP whitelist
- **Proxies WebSocket**: Vite forwards `/ws` requests to backend:8080

### Port 8080 - Go Backend
- **Purpose**: Handles WebSocket connections and API
- **Has middleware**: IP whitelist and redirect logic here
- **Serves static files**: Can serve the built frontend (not used in dev)

## In Development Mode

```
Browser → http://localhost:5173/ (Vite)
  ↓
Vite serves index.html from frontend/src/
  ↓
Browser loads JavaScript
  ↓
JavaScript connects WebSocket to /ws
  ↓
Vite proxies → http://localhost:8080/ws (Backend)
  ↓
Backend middleware checks IP whitelist
  ↓
WebSocket connection established ✅
```

## How to Test Properly

### Testing the Frontend (Port 5173)
```bash
# Open in browser:
http://localhost:5173/

# This loads from Vite, which serves the frontend files
# WebSocket connections are proxied to backend:8080
```

### Testing Backend Middleware (Port 8080)
```bash
# Test redirect
curl -I http://localhost:8080/
# → 301 Moved Permanently, Location: /dev

# Test WebSocket (with upgrade header)
curl -I -H "Upgrade: websocket" http://localhost:8080/ws
# → 400 Bad Request (expected - needs full WebSocket handshake)

# Test secret path
curl http://localhost:8080/dev/
# → 200 OK (serves index.html)
```

## In Production

In production (or when testing with `go run` or built binary):

```
Browser → http://localhost:8080/ (Backend only)
  ↓
Backend middleware checks IP whitelist
  ↓
Redirects to http://localhost:8080/dev/
  ↓
Backend serves static files from frontend/dist/
  ↓
Browser loads app
  ↓
JavaScript connects WebSocket to /dev/ws
  ↓
Backend handles WebSocket at /dev/ws
```

## Why Your Test Didn't Show Redirect

You tested:
```bash
curl http://localhost:5173/
```

This hits **Vite** (port 5173), which:
- Has no middleware
- Just serves your frontend files
- Doesn't redirect anything

To test redirects in development, use:
```bash
curl http://localhost:8080/
```

This hits the **backend** (port 8080), which:
- Has IP whitelist middleware ✅
- Redirects / → /dev ✅
- Handles WebSocket security ✅

## Vite Proxy Configuration

Vite is configured to proxy WebSocket requests to the backend:

**frontend/vite.config.ts** (default Vite behavior):
```typescript
server: {
  proxy: {
    '/ws': {
      target: 'http://localhost:8080',
      ws: true
    }
  }
}
```

This means:
- Frontend connects to `/ws` (on port 5173)
- Vite forwards it to backend (port 8080)
- Backend middleware handles security
- Connection established through proxy

## Testing the Full Flow

### 1. Check Backend is Running
```bash
curl http://localhost:8080/dev/
# Should return: 200 OK (HTML content)
```

### 2. Check Redirect Works
```bash
curl -I http://localhost:8080/
# Should return: 301 Moved Permanently
#                Location: /dev
```

### 3. Open Browser
```bash
# Go to: http://localhost:5173/
# Or: http://localhost:5173/dev/
```

Browser will:
- Load frontend from Vite (5173)
- Connect WebSocket through Vite proxy → Backend (8080)
- Backend middleware allows it (whitelisted IP)

### 4. Check Logs
You should see:
```
INFO allowing whitelisted IP WebSocket at root path 
  client_ip=::1 path=/ws
INFO new websocket connection remote_addr=[::1]:57399
INFO client connected total_clients=1
```

## Production Testing

To test as it would work in production:

### 1. Build frontend
```bash
cd frontend && npm run build
```

### 2. Run backend only
```bash
cd backend && go run .
```

### 3. Test on port 8080
```bash
# Test redirect
curl -I http://localhost:8080/
# → 301 Moved Permanently, Location: /dev

# Open browser to backend
http://localhost:8080/
# → Redirects to http://localhost:8080/dev/
# → Serves app from backend
```

## Summary

**Development (2 servers)**:
- Port 5173 (Vite): Frontend files + hot reload
- Port 8080 (Backend): Middleware + WebSocket

**Production (1 server)**:
- Port 8080 (Backend): Everything (static files + WebSocket + middleware)

**To test redirects**:
- ❌ Don't use: `curl http://localhost:5173/` (Vite, no middleware)
- ✅ Use: `curl http://localhost:8080/` (Backend, has middleware)

**In the browser**:
- Go to `http://localhost:5173/` (Vite serves frontend)
- WebSocket connects through Vite proxy to backend
- Everything works transparently ✅

