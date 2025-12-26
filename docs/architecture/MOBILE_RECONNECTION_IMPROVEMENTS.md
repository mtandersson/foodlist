# Mobile Reconnection Improvements

## Problem

The WebSocket reconnection on mobile devices (especially in dev mode) was not working reliably. The issues identified were:

1. **Visibility Change Detection Insufficient**: The app only checked if the internal state said "connected", but didn't verify the actual WebSocket readyState
2. **WebSocket State Mismatch**: On mobile, when the app is backgrounded, the WebSocket connection often dies silently - the browser closes it, but the app might not detect this immediately
3. **Slow Stale Detection**: The heartbeat check ran every 10 seconds and only triggered after 60 seconds of no messages
4. **Incomplete Reconnection Cleanup**: The `reconnect()` method didn't fully clean up the old WebSocket before creating a new one
5. **Limited Logging**: Not enough diagnostic information to debug mobile-specific issues

## Changes Made

### 1. Enhanced Visibility Change Handler (`websocket.ts` lines 54-91)

**Before:**
- Only checked internal `connectionState`
- Didn't verify actual WebSocket `readyState`

**After:**
```typescript
const wsActuallyConnected = this.ws && this.ws.readyState === WebSocket.OPEN;

if (!wsActuallyConnected) {
  // WebSocket is not actually open, force reconnect
  console.log('WebSocket not actually open (readyState:', this.ws?.readyState, '), forcing reconnect...');
  this.reconnect();
}
```

**Benefits:**
- Checks both internal state AND actual WebSocket readyState
- Immediately reconnects when returning to foreground if WebSocket is not actually open
- Handles all states: CONNECTING, RECONNECTING, CONNECTED
- Adds detailed logging for debugging

### 2. Improved `reconnect()` Method (lines 158-188)

**Before:**
- Simply reset attempts counter and called `connect()`
- Didn't explicitly clean up old WebSocket

**After:**
- Completely cleans up old WebSocket:
  - Removes all event listeners
  - Explicitly closes the connection
  - Sets ws to null
- Resets all reconnection state
- Sets state to CONNECTING before attempting connection
- Adds error handling for cleanup failures

**Benefits:**
- No zombie connections
- Clean state transitions
- Better error handling

### 3. More Aggressive Heartbeat (lines 143-156)

**Before:**
- Checked every 10 seconds
- Triggered reconnect after 60 seconds of no messages

**After:**
- Checks every 5 seconds (2x more frequent)
- Triggers reconnect after 30 seconds of no messages (2x more sensitive)

**Rationale:**
- Mobile networks are less reliable
- Backend sends frequent updates, so 30s without messages indicates a problem
- Faster detection = faster recovery

### 4. Enhanced Logging (multiple locations)

Added detailed logging for debugging:

```typescript
console.log('WebSocket closed:', 
  'code:', event.code, 
  'reason:', event.reason || '(none)',
  'wasClean:', event.wasClean,
  'readyState:', this.ws?.readyState
);
```

**Benefits:**
- Can diagnose issues from console logs
- Shows exact WebSocket state during transitions
- Helps understand mobile-specific behaviors

## Testing

### Desktop Testing

1. Start dev server: `make dev-network`
2. Open browser console
3. Background the browser tab (switch tabs)
4. Return to the tab after 10+ seconds
5. Should see: "Page became visible" → checks connection → reconnects if needed

### Mobile Testing (Dev Mode)

1. Start server: `make dev-network`
2. Note the IP address displayed (e.g., `http://192.168.1.100:5173`)
3. Open the URL on your phone
4. Test scenarios:

   **Scenario A: App Backgrounding**
   - Background the app (home button)
   - Wait 5-10 seconds
   - Return to app
   - Expected: "Page became visible" → immediate reconnection check → reconnects if needed

   **Scenario B: Network Toggle**
   - Turn on airplane mode
   - Wait 5 seconds
   - Turn off airplane mode
   - Expected: "Network came online" → immediate reconnection

   **Scenario C: Stale Connection**
   - Leave app open but inactive for 35+ seconds
   - Expected: Heartbeat detects stale connection → reconnects automatically

### Expected Console Logs

```
# On page visibility change (coming to foreground):
Page became visible, checking connection...
WebSocket not actually open (readyState: 3), forcing reconnect...
Forcing reconnection...
WebSocket connected

# On network online:
Network came online, reconnecting...
Forcing reconnection...
WebSocket connected

# On stale connection:
Connection appears stale (no messages for 31234 ms), reconnecting...
Forcing reconnection...
WebSocket connected

# On unexpected disconnect:
WebSocket closed: code: 1006 reason: (none) wasClean: false readyState: 3
Unexpected disconnect, will attempt to reconnect
Scheduling reconnect attempt 1 in 1000ms
Reconnecting (attempt 1)...
WebSocket connected
```

## Key Improvements Summary

| Aspect | Before | After | Impact |
|--------|--------|-------|--------|
| Visibility check | Internal state only | Internal state + actual readyState | More reliable detection |
| Stale timeout | 60 seconds | 30 seconds | 2x faster recovery |
| Heartbeat frequency | Every 10s | Every 5s | 2x more responsive |
| Reconnect cleanup | Basic | Complete cleanup | No zombie connections |
| Logging | Minimal | Detailed | Better debugging |
| Mobile handling | Generic | Mobile-optimized | Better UX on phones |

## Configuration

The behavior can be customized when creating the WebSocket:

```typescript
const ws = new TodoWebSocket(url, {
  reconnectDelay: 1000,        // Initial delay (default: 1000ms)
  maxReconnectAttempts: Infinity, // Max attempts (default: Infinity)
  enableHeartbeat: true,       // Enable heartbeat (default: true)
});
```

## Related Files

- `frontend/src/lib/websocket.ts` - WebSocket client implementation
- `frontend/src/lib/websocket.test.ts` - Test suite (all tests passing)
- `docs/architecture/WEBSOCKET_RECONNECTION_FIX.md` - Original reconnection documentation
- `docs/guides/NETWORK_DEV_MODE.md` - Network development mode guide

## Known Limitations

1. **Relies on Browser Events**: The visibility change API may not fire in all mobile browsers
2. **Network-Dependent**: On very poor networks, reconnection may still be slow
3. **Dev Mode Specific**: This primarily addresses dev mode; production mode may have different characteristics

## Future Enhancements

Potential improvements:

- Add ping/pong WebSocket messages for true keepalive
- Implement exponential backoff with jitter
- Add connection quality indicators
- Store offline actions and replay on reconnect
- Add service worker for background reconnection

