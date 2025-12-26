# IP Whitelist and Secret Path Security

This document describes the IP-based access control and secret path routing features implemented in FoodList.

## Overview

FoodList now supports two security features:

1. **Secret Path Routing**: Access the application through a secret URL path (e.g., `/my-secret-123/`)
2. **CIDR IP Whitelisting**: Restrict access to specific IP address ranges

These features work together to provide flexible access control while keeping the application publicly accessible when needed.

**Special Handling for PWA Files**: Mobile app installation files (manifest.json, icons) are always publicly accessible to ensure iOS/Android "Add to Home Screen" functionality works properly.

## Configuration

### Environment Variables

Add these to your `backend/.env` file:

```bash
# The secret path component - application will be accessible at /<SHARED_SECRET>/
SHARED_SECRET=my-secret-path-123

# Comma-separated list of CIDR blocks for IP whitelisting
CIDR_WHITELIST=192.168.1.0/24,10.0.0.0/8,172.16.0.0/12
```

### Configuration Modes

The system operates in different modes depending on configuration:

| SHARED_SECRET | CIDR_WHITELIST | Behavior                                                                       |
| ------------- | -------------- | ------------------------------------------------------------------------------ |
| Empty         | Empty          | Normal operation, all paths accessible to everyone                             |
| Set           | Empty          | Application only accessible via secret path                                    |
| Set           | Set            | Full security: whitelisted IPs can use short URLs, others must use secret path |

## How It Works

### 1. PWA Files (Always Public)

The following files are **always accessible** from any IP to support mobile app installation:

```
/manifest.json          → PWA manifest
/manifest.webmanifest   → Alternative manifest name
/icon.svg               → App icon
/icon.png               → App icon (PNG)
/apple-touch-icon.png   → iOS home screen icon
/favicon.ico            → Browser favicon
/robots.txt             → Search engine directives
/site.webmanifest       → Alternative manifest name
/icons/*                → Icon directory
/assets/icons/*         → Assets icon directory
```

This ensures iOS and Android can install the app to the home screen regardless of IP restrictions.

### 2. Secret Path Access (All IPs)

The secret path `/<SHARED_SECRET>/` is accessible from **any IP address**:

```
https://yourdomain.com/my-secret-123/          → Application
https://yourdomain.com/my-secret-123/ws        → WebSocket
```

This is the primary access method for sharing the application.

### 3. Whitelisted IP Behavior

IPs in the CIDR whitelist can access root paths, which **redirect permanently** to the secret path:

```
Request from 192.168.1.50:
  GET https://yourdomain.com/
  → 301 Redirect to https://yourdomain.com/my-secret-123/

  GET https://yourdomain.com/ws
  → 301 Redirect to https://yourdomain.com/my-secret-123/ws
```

This provides convenience for trusted networks (home, office) while still using secure paths.

### 4. Non-Whitelisted IP Behavior

IPs **not** in the whitelist get 404 responses on non-secret paths:

```
Request from 1.2.3.4:
  GET https://yourdomain.com/          → 404 Not Found
  GET https://yourdomain.com/ws        → 404 Not Found
  GET https://yourdomain.com/api/test  → 404 Not Found

  GET https://yourdomain.com/my-secret-123/  → 200 OK (Success!)
```

This provides **security through obscurity** - attackers don't know the secret path exists.

## Mobile/PWA Considerations

### iOS Home Screen Installation

When you add FoodList to your iPhone home screen:

1. iOS needs to access `/manifest.json` and `/icon.svg`
2. These files are **always public** (bypass IP restrictions)
3. The `start_url` in manifest is relative (`./`), so it opens at the current path
4. If you're on `/my-secret/`, the app opens to `/my-secret/`

**Important**: When sharing with iPhone users, make sure they:

- Navigate to the full secret URL first
- Then add to home screen from there
- The app will remember the secret path

### Android Installation

Same behavior as iOS - manifest and icons are always accessible.

### Testing PWA Installation

```bash
# These should always work, even with IP restrictions:
curl https://yourdomain.com/manifest.json  # → 200 OK
curl https://yourdomain.com/icon.svg       # → 200 OK

# But the app itself requires secret path:
curl https://yourdomain.com/               # → 404 Not Found (if not whitelisted)
curl https://yourdomain.com/my-secret/     # → 200 OK (always works)
```

## CIDR Notation Examples

CIDR notation defines IP ranges. Here are common examples:

```bash
# Single IP address
CIDR_WHITELIST=192.168.1.100/32

# Home network (typical router subnet)
CIDR_WHITELIST=192.168.1.0/24

# Multiple networks
CIDR_WHITELIST=192.168.1.0/24,10.0.0.0/8,172.16.0.0/12

# IPv6 support
CIDR_WHITELIST=2001:db8::/32
```

### Private IP Ranges

Common private IP ranges you might want to whitelist:

- `10.0.0.0/8` - Large private network (10.0.0.0 to 10.255.255.255)
- `172.16.0.0/12` - Medium private network (172.16.0.0 to 172.31.255.255)
- `192.168.0.0/16` - Common home networks (192.168.0.0 to 192.168.255.255)
- `192.168.1.0/24` - Specific home subnet (192.168.1.0 to 192.168.1.255)

## Proxy Support

The middleware correctly handles reverse proxy scenarios, but **by default does not trust proxy headers** for security.

### Proxy Trust Count Configuration

**Default: `PROXY_TRUST_COUNT=0`** (most secure - only uses `RemoteAddr`)

When `PROXY_TRUST_COUNT=0`:

- Proxy headers (`X-Real-IP`, `X-Forwarded-For`) are **ignored**
- Only `RemoteAddr` is used to extract client IP
- This prevents IP spoofing attacks via proxy headers

When `PROXY_TRUST_COUNT=1` (behind a single reverse proxy):

- Trusts `X-Real-IP` header (if present)
- Trusts first IP in `X-Forwarded-For` header
- Falls back to `RemoteAddr` if headers not present

When `PROXY_TRUST_COUNT=2` (behind two proxies, e.g., load balancer + reverse proxy):

- Calculates client IP from `X-Forwarded-For` based on number of trusted proxies
- Example: `X-Forwarded-For: client, proxy1, proxy2` → extracts `client` when `PROXY_TRUST_COUNT=2`

### Configuration Examples

**Direct connection (no proxy):**

```bash
PROXY_TRUST_COUNT=0  # Default - most secure
```

**Behind nginx reverse proxy:**

```bash
PROXY_TRUST_COUNT=1
```

**Behind load balancer + reverse proxy:**

```bash
PROXY_TRUST_COUNT=2
```

### Nginx Configuration

#### X-Real-IP Header (Recommended)

```nginx
location / {
    proxy_pass http://backend:8080;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

Set `PROXY_TRUST_COUNT=1` in your `.env` file.

#### X-Forwarded-For Header (General)

```nginx
location / {
    proxy_pass http://backend:8080;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

Set `PROXY_TRUST_COUNT=1` in your `.env` file.

### IP Extraction Order

When `PROXY_TRUST_COUNT > 0`, headers are checked in this order:

1. `X-Real-IP` (highest priority, if present)
2. `X-Forwarded-For` (first IP in list, adjusted for proxy count)
3. `RemoteAddr` (fallback)

**Security Warning**: Only set `PROXY_TRUST_COUNT > 0` if you trust your proxy infrastructure. Setting it too high allows attackers to spoof IPs via the `X-Forwarded-For` header.

## Frontend Integration

The frontend automatically detects the current path and connects to WebSocket at the same base path:

```typescript
// If page is at https://example.com/my-secret/
// WebSocket connects to wss://example.com/my-secret/ws
```

This works seamlessly with secret paths - no frontend configuration needed.

## Security Considerations

### Secret Path Strength

- Use a long, random secret (e.g., `my-secret-a8f3k2m9x7q1`)
- Don't use guessable paths (e.g., `admin`, `secret`, `private`)
- Consider rotating the secret periodically

### CIDR Whitelist

- Be specific with CIDR ranges (prefer `/24` over `/8` when possible)
- Only whitelist networks you control
- Review and update the whitelist as your network changes

### Defense in Depth

This is **security through obscurity** combined with network-level access control:

- ✅ Prevents casual scanning and bot attacks
- ✅ Provides convenient access for trusted networks
- ❌ Should not be the only security layer for sensitive data
- ❌ Not a replacement for authentication for multi-user scenarios

## Example Configurations

### Home User (Simple Secret Path)

```bash
SHARED_SECRET=grocery-list-k9x2m4n7
CIDR_WHITELIST=
```

Share `https://yourdomain.com/grocery-list-k9x2m4n7/` with family.

### Home User with Local Network Convenience

```bash
SHARED_SECRET=grocery-list-k9x2m4n7
CIDR_WHITELIST=192.168.1.0/24
```

- At home: Use `https://yourdomain.com/` (auto-redirects)
- Away: Use `https://yourdomain.com/grocery-list-k9x2m4n7/`

### Office Deployment

```bash
SHARED_SECRET=company-todo-x8k3m9p2
CIDR_WHITELIST=10.0.0.0/8,172.16.0.0/12
```

- On company network: Use short URL (auto-redirects)
- Remote/mobile: Use secret path URL

## Testing

The implementation includes comprehensive tests:

```bash
cd backend
go test -v -run TestIPWhitelist ./...
```

Test coverage includes:

- Secret path access from any IP
- Whitelisted IP redirects
- Non-whitelisted IP 404 responses
- Proxy header handling (X-Real-IP, X-Forwarded-For)
- IPv6 support
- Invalid CIDR handling

## Troubleshooting

### Getting 404 on secret path

- Check `SHARED_SECRET` matches exactly (case-sensitive)
- Ensure path includes trailing slash: `/my-secret/` not `/my-secret`

### Not getting redirected on local network

- Verify your IP is in the CIDR range
- Check if you're behind a proxy (IP might be different)
- Look at server logs to see what IP is being detected

### Reverse proxy issues

- Ensure proxy sets `X-Real-IP` or `X-Forwarded-For`
- Check that the proxy isn't modifying the URL path
- Verify proxy passes WebSocket upgrade headers

## Implementation Details

### Files Modified/Created

- `backend/middleware.go` - New IP whitelist middleware with proxy trust count support
- `backend/middleware_test.go` - Comprehensive middleware tests including proxy scenarios
- `backend/main.go` - Updated routing and configuration
- `frontend/src/lib/TodoList.svelte` - Dynamic WebSocket path
- `backend/env.example` - Documentation for new variables including `PROXY_TRUST_COUNT`

### Architecture

```
Request Flow:
1. HTTP Request arrives
2. Middleware extracts client IP (checking proxy headers)
3. Check if path starts with /<SHARED_SECRET>/
   - YES: Allow (accessible to everyone)
   - NO: Continue to step 4
4. Check if IP is in CIDR whitelist
   - NO: Return 404
   - YES: Continue to step 5
5. Check if path is / or /ws
   - YES: Redirect to /<SHARED_SECRET>/ or /<SHARED_SECRET>/ws
   - NO: Return 404
```

## Future Enhancements

Possible improvements for future versions:

- [ ] Rate limiting per IP
- [ ] Automatic secret rotation
- [ ] Multiple secret paths with different permissions
- [ ] Time-based access (e.g., only allow during business hours)
- [ ] Audit logging of access attempts
- [ ] Integration with authentication systems
