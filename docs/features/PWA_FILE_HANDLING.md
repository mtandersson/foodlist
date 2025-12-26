# PWA File Handling for IP Whitelist Security

## Overview

This document explains how Progressive Web App (PWA) files are handled in the context of IP-based access control and secret path routing.

## Problem Statement

When implementing IP whitelisting and secret path routing, a critical requirement emerged:

**Mobile devices (iOS/Android) need to access PWA manifest and icon files to enable "Add to Home Screen" functionality, regardless of IP restrictions.**

If these files are blocked, users cannot install the app to their home screen, severely limiting the PWA experience.

## Solution

PWA-related files are **always publicly accessible**, bypassing both:
1. IP whitelist checks
2. Secret path requirements

This ensures mobile app installation works from any device, anywhere.

## Files Always Accessible

The middleware automatically allows access to these paths from any IP:

### Core PWA Files
```
/manifest.json          - PWA manifest (primary)
/manifest.webmanifest   - PWA manifest (alternative name)
/site.webmanifest       - PWA manifest (alternative name)
```

### Icon Files
```
/icon.svg               - Vector icon
/icon.png               - Raster icon
/apple-touch-icon.png   - iOS-specific home screen icon
/favicon.ico            - Browser favicon
```

### Additional Files
```
/robots.txt             - Search engine directives
/icons/*                - Any file in /icons/ directory
/assets/icons/*         - Any file in /assets/icons/ directory
```

## Implementation

### Middleware Check Order

The middleware checks requests in this order:

1. **PWA File?** → Allow immediately
2. **Secret Path?** → Allow immediately  
3. **Whitelisted IP?** → Check for redirect or 404
4. **Not Whitelisted?** → Return 404

### Code Example

```go
// In middleware.go
func isPWAFile(path string) bool {
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

    // Also allow icon directories
    if strings.HasPrefix(path, "/icons/") || 
       strings.HasPrefix(path, "/assets/icons/") {
        return true
    }

    return false
}
```

## Mobile Installation Flow

### iOS (Safari)

1. User navigates to `https://example.com/my-secret/`
2. User taps Share → Add to Home Screen
3. iOS requests `/manifest.json` → **Always succeeds** (bypasses security)
4. iOS requests `/icon.svg` → **Always succeeds**
5. iOS creates home screen icon
6. When tapped, app opens to `https://example.com/my-secret/` (start_url is relative)

### Android (Chrome)

1. User navigates to `https://example.com/my-secret/`
2. Chrome shows "Add to Home Screen" prompt (or user accesses via menu)
3. Chrome requests `/manifest.json` → **Always succeeds**
4. Chrome requests icons → **Always succeeds**
5. App installed with proper icon
6. When launched, opens to the current path

## Manifest Configuration

The manifest uses **relative URLs** to work with secret paths:

```json
{
  "name": "FoodList",
  "short_name": "FoodList",
  "start_url": "./",          ← Relative to current page
  "icons": [
    {
      "src": "./icon.svg",    ← Relative to manifest location
      "sizes": "any",
      "type": "image/svg+xml"
    }
  ]
}
```

### Why Relative URLs?

- If user is at `/my-secret/`, start_url resolves to `/my-secret/`
- If user is at `/other-secret/`, start_url resolves to `/other-secret/`
- Icon paths resolve relative to manifest location (always at root)
- No hardcoded paths needed

## Security Implications

### What's Exposed

These files are publicly accessible:
- ✅ `/manifest.json` - Contains app metadata only
- ✅ `/icon.svg` - Visual icon, no sensitive data
- ✅ `/favicon.ico` - Browser icon

### What's Protected

The actual application remains protected:
- 🔒 `/` - Returns 404 or redirects (depending on IP)
- 🔒 `/index.html` - Returns 404 for non-whitelisted IPs
- 🔒 `/assets/index.js` - Returns 404 for non-whitelisted IPs
- 🔒 All application code and data

### Risk Assessment

**Risk Level: Minimal**

- Manifest reveals app name and theme colors (already public via UI)
- Icon is visible anyway when app is used
- No sensitive data, authentication tokens, or business logic exposed
- Application itself remains fully protected

**Benefit: Essential**

- Without this, PWA installation completely breaks
- Mobile users cannot add app to home screen
- Defeats the purpose of having a PWA

## Testing

### Test PWA File Access

```bash
# From non-whitelisted IP, these should work:
curl https://example.com/manifest.json
# → 200 OK (returns manifest)

curl https://example.com/icon.svg
# → 200 OK (returns icon)

# But application files should not:
curl https://example.com/
# → 404 Not Found

curl https://example.com/index.html
# → 404 Not Found
```

### Test Mobile Installation

1. **Setup**: Deploy with secret path and IP whitelist
2. **Use mobile device** on non-whitelisted network (cellular data)
3. Navigate to secret path URL
4. Add to home screen
5. **Expected**: Installation succeeds
6. Open app from home screen
7. **Expected**: Opens to secret path, app works

## Logs

When PWA files are accessed, you'll see log entries like:

```
INFO request to PWA file client_ip=1.2.3.4 path=/manifest.json method=GET
INFO request to PWA file client_ip=1.2.3.4 path=/icon.svg method=GET
```

These indicate the PWA file bypass is working correctly.

## Common Issues

### Issue: Can't install app on iPhone

**Cause**: User trying to install from root path without navigating to secret path first

**Solution**: 
1. Navigate to full secret URL first: `https://example.com/my-secret/`
2. Then add to home screen
3. App will open to secret path

### Issue: App icon not showing

**Cause**: Icon file path incorrect in manifest

**Solution**:
- Ensure icon file exists at root: `/icon.svg`
- Check manifest references it correctly: `"src": "./icon.svg"`
- Verify icon file is actually accessible: `curl https://example.com/icon.svg`

### Issue: Manifest not loading

**Cause**: Manifest path incorrect in HTML

**Solution**:
- Check `index.html` has: `<link rel="manifest" href="/manifest.json">`
- Verify manifest is at root: `/manifest.json`
- Test direct access: `curl https://example.com/manifest.json`

## Files Modified

To implement PWA file handling:

1. **backend/middleware.go**
   - Added `isPWAFile()` helper function
   - Added PWA check at top of middleware flow
   - Logs PWA file access attempts

2. **backend/middleware_test.go**
   - Added `TestIPWhitelistMiddleware_PWAFilesAccessible`
   - Added `TestIPWhitelistMiddleware_NonPWAFilesRestricted`
   - Tests 10 different PWA file paths

3. **frontend/public/manifest.json**
   - Changed `start_url` from `/` to `./` (relative)
   - Changed icon `src` from `/icon.svg` to `./icon.svg`

## Future Considerations

### Additional PWA Files

If you add more PWA-related files, add them to `isPWAFile()`:

```go
pwaFiles := []string{
    "/manifest.json",
    "/your-new-file.json",  // Add here
    // ...
}
```

### Service Worker

If you implement a service worker (`/service-worker.js`), it should also be publicly accessible:

```go
if path == "/service-worker.js" {
    return true
}
```

### App Screenshots

If you add app screenshots to the manifest (for install prompt), ensure they're in an allowed directory:

```json
{
  "screenshots": [
    {
      "src": "./icons/screenshot1.png"  // ✅ /icons/ is allowed
    }
  ]
}
```

## Summary

PWA files are a **security exception by necessity**:

- ✅ Essential for mobile app installation
- ✅ Contains only public metadata
- ✅ No sensitive data exposed  
- ✅ Application code remains protected
- ✅ Tested and working on iOS/Android

This design prioritizes **user experience** while maintaining **security** for the actual application.

