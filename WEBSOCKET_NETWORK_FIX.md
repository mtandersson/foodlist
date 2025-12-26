# WebSocket Network Connection Fix

## Problem

When accessing the FoodList app from a phone on the same network, the frontend would load but fail to connect to the backend via WebSocket. The issue was that the WebSocket URL was hardcoded to `ws://localhost:8080/ws` in development mode.

## Root Cause

In `frontend/src/lib/TodoList.svelte`, the WebSocket URL was hardcoded:

```typescript
// OLD CODE (broken for network access)
const wsUrl = import.meta.env.DEV 
  ? 'ws://localhost:8080/ws'        // ❌ Always connects to localhost
  : `${wsProtocol}//${window.location.host}/ws`;
```

When accessing from a phone at `http://192.168.1.100:5173`, the app would still try to connect to `ws://localhost:8080/ws`, which refers to the phone itself, not the development machine.

## Solution

Changed the WebSocket URL to dynamically use the current hostname:

```typescript
// NEW CODE (works for both localhost and network access)
const wsUrl = import.meta.env.DEV 
  ? `${wsProtocol}//${window.location.hostname}:5173/ws`  // ✅ Uses current host
  : `${wsProtocol}//${window.location.host}/ws`;
```

Now the WebSocket connection adapts to the accessing device:
- From computer: `ws://localhost:5173/ws`
- From phone: `ws://192.168.1.100:5173/ws`

## Architecture

The connection flow is:

```
Phone/Browser → ws://YOUR_IP:5173/ws → [Vite Proxy] → ws://localhost:8080 → Backend
```

1. **Frontend** serves on port 5173, accessible from network
2. **WebSocket** connects to Vite dev server on port 5173
3. **Vite proxy** (configured in `vite.config.ts`) forwards to backend on port 8080
4. **Backend** runs on localhost:8080, accessible only to Vite

This architecture means:
- Only port 5173 needs to be accessible from the network
- Backend stays on localhost (more secure)
- Vite handles the proxying transparently

## Files Changed

1. **frontend/src/lib/TodoList.svelte** (lines 13-17)
   - Changed WebSocket URL to use `window.location.hostname` in dev mode
   - Now dynamically adapts to the accessing hostname

2. **NETWORK_DEV_MODE.md**
   - Updated documentation to explain the WebSocket connection flow
   - Added troubleshooting information

## Testing

To test the fix:

1. Run `make dev-network`
2. Note your network IP address (e.g., 192.168.1.100)
3. On your phone, navigate to `http://192.168.1.100:5173`
4. The app should load AND connect (check connection indicator)
5. Try creating/completing todos - changes should sync in real-time

## Why This Works

The key insight is that **Vite's dev server acts as both a web server and a WebSocket proxy**:

- **Without this fix**: Phone tries to connect to `ws://localhost:8080` (phone's localhost, not computer's)
- **With this fix**: Phone connects to `ws://YOUR_IP:5173`, which Vite proxies to the backend

The proxy configuration in `vite.config.ts` makes this possible:

```typescript
server: {
  host: '0.0.0.0',  // Allow external connections
  proxy: {
    '/ws': {
      target: 'ws://localhost:8080',  // Proxy to backend
      ws: true,
      changeOrigin: true,
    },
  },
}
```

## Production Behavior

In production, the code still works correctly:
- Built app is served by the Go backend on port 8080
- WebSocket URL becomes `ws://domain.com:8080/ws` (same host, same port)
- No proxy needed since everything is on the same server

## Additional Notes

- The backend's `BIND_ADDR=0.0.0.0` setting is NOT strictly required for this to work, since Vite proxies on localhost
- However, it's useful if you want to test backend-only features from the network
- The fix is backward compatible - localhost development still works exactly as before

