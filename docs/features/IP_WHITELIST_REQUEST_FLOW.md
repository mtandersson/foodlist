# Request Flow Diagram

This document provides visual representations of how requests are handled with the IP whitelist and secret path features.

## Request Flow Decision Tree

```
┌─────────────────────────────────┐
│   HTTP Request Arrives          │
│   (Any path, any IP)            │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  Extract Client IP              │
│  • Check X-Real-IP              │
│  • Check X-Forwarded-For        │
│  • Fallback to RemoteAddr       │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  Is path a PWA file?            │
│  (manifest.json, icon.svg, etc) │
└────┬────────────────────────┬───┘
     │ YES                    │ NO
     │                        │
     ▼                        ▼
┌─────────────┐    ┌──────────────────────────┐
│  Allow      │    │  Is path = /<SECRET>/* ? │
│  Access     │    └──────┬────────────────┬──┘
│  (200 OK)   │           │ YES            │ NO
└─────────────┘           │                │
                          ▼                ▼
              ┌─────────────────┐  ┌──────────────────────────┐
              │  Allow          │  │  Is IP in whitelist?     │
              │  Access         │  └──────┬────────────────┬──┘
              │  (200 OK)       │         │ YES            │ NO
              └─────────────────┘         │                │
                                          ▼                ▼
                              ┌─────────────────┐  ┌─────────────┐
                              │ Is path / or    │  │  Return     │
                              │ /ws ?           │  │  404        │
                              └─────┬──────┬────┘  └─────────────┘
                                    │ YES  │ NO
                                    │      │
                                    ▼      ▼
                         ┌──────────────┐  └──────────────────┐
                         │ 301 Redirect │     ┌───────────────┤
                         │ to /<SECRET> │     │  Return       │
                         │ path         │     │  404          │
                         └──────────────┘     └───────────────┘
```

## Example Scenarios

### Scenario 1: Non-whitelisted IP accessing root

```
Request:
  GET https://example.com/
  From: 1.2.3.4 (not in whitelist)

Flow:
  1. Extract IP: 1.2.3.4
  2. Path is "/" (not secret path)
  3. IP not in whitelist
  4. → Response: 404 Not Found

Result: Access denied (security through obscurity)
```

### Scenario 2: Whitelisted IP accessing root

```
Request:
  GET https://example.com/
  From: 192.168.1.50 (in whitelist 192.168.1.0/24)

Flow:
  1. Extract IP: 192.168.1.50
  2. Path is "/" (not secret path)
  3. IP IS in whitelist
  4. Path is "/" (root)
  5. → Response: 301 → https://example.com/my-secret/

Result: Permanent redirect to secret path
```

### Scenario 3: Any IP accessing secret path

```
Request:
  GET https://example.com/my-secret/
  From: 1.2.3.4 (not in whitelist)

Flow:
  1. Extract IP: 1.2.3.4
  2. Path starts with "/my-secret/"
  3. → Response: 200 OK (serve application)

Result: Access granted (secret path is public)
```

### Scenario 4: PWA file access (any IP)

```
Request:
  GET https://example.com/manifest.json
  From: 1.2.3.4 (not in whitelist)

Flow:
  1. Extract IP: 1.2.3.4
  2. Path is "/manifest.json" (PWA file)
  3. → Response: 200 OK (serve manifest)

Result: PWA files always accessible for mobile installation
```

### Scenario 5: Whitelisted IP accessing WebSocket

```
Request:
  GET https://example.com/ws
  From: 192.168.1.50 (in whitelist)
  Upgrade: websocket

Flow:
  1. Extract IP: 192.168.1.50
  2. Path is "/ws" (not secret path)
  3. IP IS in whitelist
  4. Path is "/ws"
  5. → Response: 301 → https://example.com/my-secret/ws

Result: Permanent redirect, browser reconnects to secret path
```

### Scenario 6: Behind a proxy

```
Request:
  GET https://example.com/
  From: 172.16.0.1 (proxy IP)
  X-Real-IP: 192.168.1.50
  X-Forwarded-For: 192.168.1.50, 172.16.0.1

Flow:
  1. Extract IP: 192.168.1.50 (from X-Real-IP)
  2. Path is "/" (not secret path)
  3. IP IS in whitelist (192.168.1.0/24)
  4. Path is "/"
  5. → Response: 301 → https://example.com/my-secret/

Result: Proxy headers correctly identify client
```

## Configuration Matrix

| Secret Set | Whitelist Set | Behavior                                                    |
| ---------- | ------------- | ----------------------------------------------------------- |
| ❌ No      | ❌ No         | Normal operation, all paths accessible                      |
| ✅ Yes     | ❌ No         | App only at /<SECRET>/, 404 elsewhere                       |
| ✅ Yes     | ✅ Yes        | Full mode: whitelist gets redirects, others use secret path |

## Network Topology Example

```
┌─────────────────────────────────────────────────┐
│  Internet (1.2.3.4)                             │
│                                                  │
│  ❌ GET /         → 404 Not Found               │
│  ❌ GET /ws       → 404 Not Found               │
│  ✅ GET /secret/  → 200 OK                      │
└──────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│  Home Network (192.168.1.0/24)                  │
│                                                  │
│  ✅ GET /         → 301 → /secret/              │
│  ✅ GET /ws       → 301 → /secret/ws            │
│  ✅ GET /secret/  → 200 OK                      │
└──────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│  Office Network (10.0.0.0/8)                    │
│                                                  │
│  ✅ GET /         → 301 → /secret/              │
│  ✅ GET /ws       → 301 → /secret/ws            │
│  ✅ GET /secret/  → 200 OK                      │
└──────────────────────────────────────────────────┘
```

## Path Table

| Request Path     | Whitelisted IP       | Non-Whitelisted IP |
| ---------------- | -------------------- | ------------------ |
| `/`              | 301 → `/<SECRET>`    | 404 Not Found      |
| `/ws`            | 301 → `/<SECRET>/ws` | 404 Not Found      |
| `/<SECRET>/`     | 200 OK               | 200 OK             |
| `/<SECRET>/ws`   | 200 OK               | 200 OK             |
| `/manifest.json` | 200 OK               | 200 OK (PWA)       |
| `/icon.svg`      | 200 OK               | 200 OK (PWA)       |
| `/favicon.ico`   | 200 OK               | 200 OK (PWA)       |
| `/icons/*`       | 200 OK               | 200 OK (PWA)       |
| `/api/anything`  | 404 Not Found        | 404 Not Found      |
| `/anything`      | 404 Not Found        | 404 Not Found      |

## Frontend WebSocket Connection

```
User navigates to:
  https://example.com/my-secret/

Frontend automatically constructs:
  wss://example.com/my-secret/ws

This works because:
  1. Frontend reads window.location.pathname
  2. Appends "ws" to the base path
  3. No hardcoded URLs needed
```

## CIDR Block Examples

```
Single IP:
  192.168.1.100/32
  → Exactly 192.168.1.100

Small subnet (256 IPs):
  192.168.1.0/24
  → 192.168.1.0 to 192.168.1.255

Medium subnet (4,096 IPs):
  192.168.0.0/20
  → 192.168.0.0 to 192.168.15.255

Large private network (16.7M IPs):
  10.0.0.0/8
  → 10.0.0.0 to 10.255.255.255

Multiple networks:
  192.168.1.0/24,10.0.0.0/8,172.16.0.0/12
```

## Security Visualization

```
┌────────────────────────────────────────┐
│  Public Internet                       │
│                                        │
│  • No knowledge of secret path         │
│  • All root requests → 404             │
│  • Can't discover the application      │
└────────────────────────────────────────┘
              ▲
              │
              │ Security through obscurity
              │
┌─────────────┴──────────────────────────┐
│  Secret Path (/<SHARED_SECRET>/)       │
│                                        │
│  • Known to authorized users           │
│  • Accessible from anywhere            │
│  • Shareable URL                       │
└────────────────────────────────────────┘
              ▲
              │
              │ Convenience redirects
              │
┌─────────────┴──────────────────────────┐
│  Trusted Networks (Whitelisted CIDRs)  │
│                                        │
│  • Can use short URLs (/, /ws)         │
│  • Automatically redirected            │
│  • Best UX for trusted environments    │
└────────────────────────────────────────┘
```

## Deployment Patterns

### Pattern 1: Personal Home Use

```yaml
SHARED_SECRET: my-grocery-list-a8f3k2m9
CIDR_WHITELIST: 192.168.1.0/24

Use case:
  - Home: Use https://home.example.com/
  - Mobile: Use https://home.example.com/my-grocery-list-a8f3k2m9/
  - Share with family: Send secret URL
```

### Pattern 2: Small Office

```yaml
SHARED_SECRET: office-tasks-x7k2m9p4
CIDR_WHITELIST: 10.0.0.0/8

Use case:
  - Office network: Use https://tasks.example.com/
  - Remote work: VPN or use secret URL
  - Partners: Share secret URL for specific lists
```

### Pattern 3: Public-ish Deployment

```yaml
SHARED_SECRET: shared-list-k3m8x2n9
CIDR_WHITELIST:

Use case:
  - Everyone uses: https://lists.example.com/shared-list-k3m8x2n9/
  - Share secret URL via messaging/email
  - No network restrictions
  - Security through unguessable URL
```
