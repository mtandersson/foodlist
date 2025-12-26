# Network Development Mode

This document describes the network-accessible development mode feature for testing FoodList on mobile devices.

## Overview

The `dev-network` make target allows you to run the FoodList development environment with network access enabled, making it possible to test the application on phones, tablets, or other devices on your local network.

## Usage

```bash
make dev-network
```

This command will:

1. Start the Vite dev server bound to `0.0.0.0:5173` (accessible from network)
2. Start the Go backend with Air (live reload) bound to `0.0.0.0:8080`
3. Display your local network IP address for easy access
4. Open the application in your default browser

Example output:

```text
🌐 Network Access Enabled:
   Frontend: http://192.168.1.100:5173
   Backend:  http://192.168.1.100:8080

📱 Use the frontend URL above to access from your phone
```

## Testing on Mobile

1. **Ensure same Wi-Fi network**: Your development machine and mobile device must be on the same Wi-Fi network
2. **Run dev-network mode**: `make dev-network`
3. **Note the IP address**: The command output will show your network IP
4. **Open on phone**: Navigate to `http://YOUR_IP:5173` on your phone's browser
5. **Test**: The app will function normally with WebSocket connection proxied through Vite

## Architecture

### Frontend (Vite)

The Vite dev server is configured to:
- Listen on `0.0.0.0` (all network interfaces)
- Proxy WebSocket connections to the backend on port 8080
- Serve the Svelte application on port 5173

The frontend automatically detects the current hostname and connects to the WebSocket through Vite's proxy:

```typescript
// In TodoList.svelte
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsUrl = import.meta.env.DEV 
  ? `${wsProtocol}//${window.location.hostname}:5173/ws`
  : `${wsProtocol}//${window.location.host}/ws`;
```

This means:
- On your computer: connects to `ws://localhost:5173/ws`
- On your phone: connects to `ws://192.168.1.100:5173/ws` (or whatever your IP is)
- Vite then proxies both to the backend at `localhost:8080`

Configuration in `frontend/vite.config.ts`:

```typescript
server: {
  host: '0.0.0.0', // Allow external connections
  proxy: {
    '/ws': {
      target: process.env.VITE_BACKEND_URL || 'ws://localhost:8080',
      ws: true,
      changeOrigin: true,
    },
  },
}
```

### Backend (Go)

The backend server is configured to:
- Accept the `BIND_ADDR` environment variable
- Default to `localhost` for normal dev mode
- Use `0.0.0.0` for network dev mode

Configuration in `backend/main.go`:

```go
bindAddr := os.Getenv("BIND_ADDR")
if bindAddr == "" {
    bindAddr = "localhost"
}
```

## Environment Variables

| Variable | Description | Default | Network Mode |
|----------|-------------|---------|--------------|
| `BIND_ADDR` | Backend bind address | `localhost` | `0.0.0.0` |
| `PORT` | Backend port | `8080` | `8080` |
| `VITE_BACKEND_URL` | Backend WebSocket URL for Vite proxy | `ws://localhost:8080` | (automatic) |

## Security Considerations

**Important**: The network dev mode binds to `0.0.0.0`, making your development server accessible to any device on your local network.

- **Local network only**: Only use on trusted networks (home/office Wi-Fi)
- **Firewall**: Your system firewall may need to allow connections on ports 5173 and 8080
- **Production**: Never use `0.0.0.0` binding in production without proper security measures

## Comparison with Other Dev Modes

| Mode | Command | Bind Address | Use Case |
|------|---------|--------------|----------|
| Local Dev | `make dev` | `localhost` | Standard local development |
| JSON Logging | `make dev-json` | `localhost` | Local dev with JSON logs |
| Network Dev | `make dev-network` | `0.0.0.0` | Mobile/network testing |

## Troubleshooting

### Can't connect from phone

1. **Check Wi-Fi**: Ensure both devices are on the same network
2. **Check firewall**: Temporarily disable firewall or allow ports 5173 and 8080
3. **Verify IP**: Run `ifconfig` (macOS/Linux) or `ipconfig` (Windows) to confirm your IP
4. **Test locally**: Try accessing `http://localhost:5173` first

### WebSocket connection fails

The WebSocket connection is proxied through Vite on port 5173:

1. **Check both services are running**: `make dev-network` should start both frontend (5173) and backend (8080)
2. **Verify backend is accessible**: Check backend logs for "starting server" message
3. **Check the proxy configuration** in `vite.config.ts`
4. **Browser console**: Open browser developer tools and check for WebSocket errors

The connection flow is:
- Phone/Browser → `ws://YOUR_IP:5173/ws` (Vite)
- Vite → `ws://localhost:8080/ws` (Backend)

Since the frontend connects to Vite on port 5173, and Vite proxies to the backend, you only need to ensure:
- Port 5173 is accessible from your phone (frontend + WebSocket proxy)
- Backend is running on 8080 (accessed by Vite locally)

### Port already in use

If ports 5173 or 8080 are already in use:

```bash
# Find what's using the port (macOS/Linux)
lsof -i :5173
lsof -i :8080

# Kill the process
kill -9 <PID>
```

## Implementation Files

- `Makefile`: Added `dev-network` target
- `backend/main.go`: Added `BIND_ADDR` environment variable support
- `frontend/vite.config.ts`: Configured for network access with `host: '0.0.0.0'`
- `README.md`: Updated documentation with network dev mode instructions

## Future Enhancements

Potential improvements:

- QR code generation for easy mobile access
- HTTPS support for testing secure contexts (camera, geolocation, etc.)
- Automatic IP detection and display without running ifconfig
- Network discovery for finding other devices running FoodList

