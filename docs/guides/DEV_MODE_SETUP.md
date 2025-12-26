# Development Mode Setup Guide

## Quick Start Options

You now have multiple development modes to choose from:

### Option 1: Simple Development (No Security)
```bash
make dev           # Localhost only, no IP restrictions
make dev-network   # Network accessible, no IP restrictions
```
Use this for **simple development** when you don't need to test security features.
- Access at: `http://localhost:5173/`
- Phone access (network mode): `http://192.168.0.46:5173/`

### Option 2: Secure Development (With IP Whitelist)
```bash
make dev-secure        # With security, localhost only
make dev-secure-net    # With security + network access
```
Use this to **test IP whitelist features** with the `/dev/` secret path.
- Localhost: `http://localhost:5173/` → redirects to `/dev/`
- Or direct: `http://localhost:5173/dev/`
- Phone (network mode): `http://192.168.0.46:5173/dev/` (must use secret path!)

## Secure Mode Configuration

When using `make dev-secure` or `make dev-secure-net`, security is enabled via `backend/.env`:

```bash
SHARED_SECRET=dev
CIDR_WHITELIST=127.0.0.0/8,::1/128
```

**What this means:**
- **Secret Path**: `/dev/` 
- **Whitelisted**: Localhost only (`127.0.0.1`, `::1`)
- **Others**: Must use `/dev/` path (no root access)

## Comparison Table

| Feature | make dev | make dev-network | make dev-secure | make dev-secure-net |
|---------|----------|------------------|-----------------|---------------------|
| Security | ❌ No | ❌ No | ✅ Yes | ✅ Yes |
| Network Access | ❌ Localhost only | ✅ Yes | ❌ Localhost only | ✅ Yes |
| Localhost URL | `/` | `/` | `/` → `/dev/` | `/` → `/dev/` |
| Phone URL | N/A | `/` | N/A | `/dev/` only |
| Use Case | Quick dev | Phone testing | Test security | Test security + phone |

## Detailed Usage

### make dev (Simple, Localhost)

**Best for**: Daily development, no security testing needed

```bash
make dev
# Opens: http://localhost:5173/
```

**Access:**
- ✅ `http://localhost:5173/`
- ✅ All paths work normally
- ❌ Not accessible from network

---

### make dev-network (Simple, Network)

**Best for**: Testing on your phone without security

```bash
make dev-network
# Opens: http://localhost:5173/
# Phone: http://192.168.0.46:5173/
```

**Access:**
- ✅ `http://localhost:5173/`
- ✅ `http://192.168.0.46:5173/` (from phone)
- ✅ All paths work normally
- ❌ No security restrictions

---

### make dev-secure (Secure, Localhost)

**Best for**: Testing IP whitelist with redirects locally

```bash
make dev-secure
# Opens: http://localhost:5173/dev/
```

**Access:**
- ✅ `http://localhost:5173/` → redirects to `/dev/`
- ✅ `http://localhost:5173/dev/` (direct)
- ✅ `http://localhost:5173/manifest.json` (PWA files always work)
- ❌ Not accessible from network

**What you'll see in logs:**
```
INFO redirecting whitelisted IP to secret path 
  client_ip=127.0.0.1 from=/ to=/dev/
```

---

### make dev-secure-net (Secure, Network)

**Best for**: Testing security features on your phone

```bash
make dev-secure-net
# Opens: http://localhost:5173/dev/
# Phone must use: http://192.168.0.46:5173/dev/
```

**Access from localhost:**
- ✅ `http://localhost:5173/` → redirects to `/dev/`
- ✅ `http://localhost:5173/dev/` (direct)

**Access from phone** (`192.168.0.x`):
- ❌ `http://192.168.0.46:5173/` → 404 Not Found
- ✅ `http://192.168.0.46:5173/dev/` → Works!
- ✅ `http://192.168.0.46:5173/manifest.json` → Works (PWA)

**What you'll see in logs:**
```
# From localhost:
INFO redirecting whitelisted IP to secret path 
  client_ip=127.0.0.1 from=/ to=/dev/

# From phone:
WARN unauthorized access attempt 
  client_ip=192.168.0.50 path=/ method=GET

INFO request to secret path 
  client_ip=192.168.0.50 path=/dev/ method=GET
```

## Configuration File

The secure modes use: `backend/.env`

```bash
# Server Configuration
BIND_ADDR=0.0.0.0
PORT=8080
STATIC_DIR=../frontend/dist
DATA_DIR=.
LOG_FORMAT=logfmt

# Security Configuration
SHARED_SECRET=dev
CIDR_WHITELIST=127.0.0.0/8,::1/128
```

## Customizing Security

### Allow your phone's network too

Edit `backend/.env`:

```bash
CIDR_WHITELIST=127.0.0.0/8,::1/128,192.168.0.0/24
```

Now phones on your network can also use short URLs!

### Change the secret path

```bash
SHARED_SECRET=my-custom-dev
```

Access at: `http://localhost:5173/my-custom-dev/`

### Disable security temporarily

Just use `make dev` or `make dev-network` - they explicitly disable security features.

## Testing Scenarios

### Test WebSocket Redirect (Secure Mode)

The WebSocket will get redirected from `/ws` to `/dev/ws`:

```bash
make dev-secure

# In logs, you'll see:
# INFO redirecting whitelisted IP to secret path 
#   client_ip=::1 from=/ws to=/dev/ws
```

The frontend automatically follows the redirect.

### Test 404 on Non-Whitelisted IP

```bash
make dev-secure-net

# From your phone, try root path:
curl http://192.168.0.46:5173/
# → 404 Not Found

# Try secret path:
curl http://192.168.0.46:5173/dev/
# → 200 OK (works!)
```

### Test PWA Files Always Accessible

```bash
make dev-secure-net

# From your phone:
curl http://192.168.0.46:5173/manifest.json
# → 200 OK (PWA files bypass security)
```

## Troubleshooting

### Getting 404 in secure mode

**Problem**: Can't access `http://localhost:5173/`

**Check**: Are you using `make dev-secure`? Try the secret path:
```bash
http://localhost:5173/dev/
```

Or switch to regular dev mode:
```bash
make dev
```

### WebSocket connection failures

**Problem**: Frontend can't connect to WebSocket in secure mode

**Reason**: Frontend is trying `/ws` which redirects to `/dev/ws`

**Solution**: This should work automatically. If you see repeated redirects in logs:
1. Stop the server (Ctrl+C)
2. Clear browser cache
3. Restart with `make dev-secure`
4. Hard refresh browser (Cmd+Shift+R / Ctrl+Shift+R)

### Phone can't access in dev-secure-net

**Problem**: Phone gets 404

**Solution**: You MUST use the secret path on phone:
```
❌ http://192.168.0.46:5173/
✅ http://192.168.0.46:5173/dev/
```

To allow phone to use root path, add to `backend/.env`:
```bash
CIDR_WHITELIST=127.0.0.0/8,::1/128,192.168.0.0/24
```

## Summary

**For regular development:**
- `make dev` - Simple, fast, no security overhead

**For phone testing:**
- `make dev-network` - No security, just network access

**For security feature testing:**
- `make dev-secure` - Test locally with redirects
- `make dev-secure-net` - Test on phone with secret path

**Files created:**
- `backend/.env` - Security configuration (git-ignored)
- Logs show all access patterns and redirects
