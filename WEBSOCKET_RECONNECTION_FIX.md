# WebSocket Reconnection Fix

## Problem
The mobile app was frequently getting stuck in a reconnecting state and not actually reconnecting. The issues were:

1. **Wrong WebSocket URL in dev mode** - Using port 5173 (Vite) instead of 8080 (backend)
2. **No cleanup of old WebSocket connections** - Multiple connections could exist simultaneously
3. **Limited reconnection attempts** - Max of 10 attempts before giving up
4. **No mobile-specific handling** - App backgrounding and network changes weren't handled
5. **No stale connection detection** - Connections could appear connected but be dead

## Changes Made

### 1. Fixed WebSocket URL (`frontend/src/lib/TodoList.svelte`)
Changed from:
```typescript
const wsUrl = import.meta.env.DEV 
  ? `${wsProtocol}//${window.location.hostname}:5173/ws`
  : `${wsProtocol}//${window.location.host}/ws`;
```

To:
```typescript
const wsUrl = import.meta.env.DEV 
  ? `${wsProtocol}//${window.location.hostname}:8080/ws`
  : `${wsProtocol}//${window.location.host}/ws`;
```

### 2. Improved Connection Management (`frontend/src/lib/websocket.ts`)

#### Proper Cleanup
- Remove all event listeners from old WebSocket before creating new one
- Close old connections properly
- Clear pending reconnect timeouts

#### Infinite Reconnection Attempts
- Changed from max 10 attempts to infinite attempts
- Uses exponential backoff (1s, 1.5s, 2.25s, ...) up to 30s max
- Resets attempt counter on successful connection

#### Mobile-Specific Features

**Visibility Change Handling:**
- Detects when app comes back to foreground
- Checks if connection is stale (no messages for >5s)
- Immediately reconnects if in reconnecting state

**Online/Offline Events:**
- Listens for network status changes
- Immediately reconnects when network comes back online
- Cancels pending reconnection timers for immediate retry

**Heartbeat Detection:**
- Tracks timestamp of last received message
- Every 10 seconds checks if we've received anything in the last 60s
- Automatically reconnects if connection appears stale

#### Better Reconnection Logic
- `reconnect()` method for immediate reconnection (resets backoff)
- Proper cleanup in `close()` method
- Logs for better debugging

## Testing

### Desktop Testing
1. Start the dev server: `make dev-network`
2. Open browser console to see connection logs
3. Stop/restart backend to test reconnection
4. Should see: "WebSocket closed" → "Reconnecting" → "WebSocket connected"

### Mobile Testing
1. Start server with network access: `make dev-network`
2. Access from phone using displayed URL
3. Test scenarios:
   - Background app (home button) → Return to app → Should reconnect immediately
   - Toggle airplane mode → Turn back on → Should reconnect immediately
   - Let sit idle for 60s → Should detect stale connection and reconnect

### Debug Logs
The console will now show:
- `WebSocket connected` - Successfully connected
- `WebSocket closed: <code> <reason>` - Connection closed
- `Scheduling reconnect attempt N in Xms` - Planning reconnection
- `Reconnecting (attempt N)...` - Attempting to reconnect
- `Page became visible, checking connection...` - App came to foreground
- `Network came online, reconnecting...` - Network restored
- `Connection appears stale, reconnecting...` - Heartbeat timeout

## Benefits

1. **Reliable Reconnection** - Will keep trying indefinitely instead of giving up
2. **Mobile-Friendly** - Handles backgrounding and network changes gracefully
3. **Fast Recovery** - Immediate reconnection on network restore or app foreground
4. **Stale Connection Detection** - Won't get stuck thinking it's connected when it's not
5. **Clean Code** - Proper cleanup prevents memory leaks and zombie connections

## Configuration

You can customize reconnection behavior when creating the WebSocket:

```typescript
const ws = new TodoWebSocket(url, {
  reconnectDelay: 1000,        // Initial delay (default: 1000ms)
  maxReconnectAttempts: 10,    // Max attempts (default: Infinity)
});
```

