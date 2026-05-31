/// <reference types="vitest" />
import {defineConfig} from "vitest/config"
import {svelte} from "@sveltejs/vite-plugin-svelte"
import {VitePWA} from "vite-plugin-pwa"
import {readFileSync, existsSync} from "fs"
import {join} from "path"

// Read version from environment variable, VERSION file, or default to "dev"
function getVersion(): string {
  // CI builds: use RELEASE_VERSION if set (indicates CI build)
  if (process.env.RELEASE_VERSION) {
    return process.env.RELEASE_VERSION
  }

  // Local builds: try to read from VERSION file
  try {
    const versionPath = join(__dirname, "..", "VERSION")
    const version = readFileSync(versionPath, "utf-8").trim()
    if (version) {
      return version
    }
  } catch {
    // VERSION file doesn't exist or can't be read
  }

  // Default fallback
  return "dev"
}

// Append "-dev" suffix if not in CI
// CI is detected by: RELEASE_VERSION env var (most reliable), CI=true, or GITHUB_ACTIONS=true
function getVersionWithSuffix(): string {
  const version = getVersion()

  // If RELEASE_VERSION is set, it's a CI build - don't append -dev
  const isCI =
    !!process.env.RELEASE_VERSION ||
    process.env.CI === "true" ||
    process.env.GITHUB_ACTIONS === "true"

  if (!isCI && version !== "dev") {
    return `${version}-dev`
  }

  return version
}

const appVersion = getVersionWithSuffix()

/**
 * Pull SHARED_SECRET out of a backend `.env` file body. Pure string
 * function (no fs) so it can be unit-tested in isolation. Honors basic
 * shell-style quoting and ignores comments.
 *
 * Exported because the dev proxy needs to add the same secret-path
 * prefix the backend mounts every route under (see backend/main.go's
 * `pathPrefix`). Without it, Vite forwards unprefixed URLs and the
 * backend's IPWhitelistMiddleware 404s them - the exact symptom
 * behind "GET /api/v1/recipes/parse 404".
 */
export function extractSharedSecretFromEnv(text: string): string {
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line || line.startsWith("#")) continue
    const m = line.match(/^SHARED_SECRET\s*=\s*(.*)$/)
    if (!m) continue
    let value = m[1].trim()
    // Strip a single layer of matching surrounding quotes.
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1)
    }
    return value
  }
  return ""
}

/**
 * Compute the path prefix the dev proxy should prepend to every
 * request before forwarding to the Go backend. Resolution order:
 *
 *   1. VITE_BACKEND_PATH_PREFIX env var (explicit override; empty
 *      string disables the rewrite).
 *   2. SHARED_SECRET from backend/.env (auto-detected so the common
 *      `SHARED_SECRET=dev` setup just works).
 *   3. Empty string (no rewrite, suitable when the backend has no
 *      shared secret configured).
 *
 * Returns either "" or "/<prefix>" - never with a trailing slash.
 */
function getBackendDevPathPrefix(): string {
  const explicit = process.env.VITE_BACKEND_PATH_PREFIX
  if (explicit !== undefined) {
    const trimmed = explicit.replace(/\/$/, "")
    if (trimmed === "") return ""
    return trimmed.startsWith("/") ? trimmed : `/${trimmed}`
  }
  try {
    const envPath = join(__dirname, "..", "backend", ".env")
    if (!existsSync(envPath)) return ""
    const text = readFileSync(envPath, "utf-8")
    const secret = extractSharedSecretFromEnv(text)
    if (secret) return `/${secret}`
  } catch {
    // backend/.env unreadable; fall through to no rewrite.
  }
  return ""
}

const backendPathPrefix = getBackendDevPathPrefix()

// https://vite.dev/config/
export default defineConfig(({mode}) => ({
  // For production builds we need relative asset URLs so the app can be served
  // from any prefix like /<SHARED_SECRET>/ (e.g. /dev/).
  // Keep dev as absolute root for best HMR behavior.
  base: mode === "production" ? "./" : "/",
  // Ensure Svelte resolves to the browser runtime during tests (jsdom) so
  // component mounting works (Svelte 5 exports also include a server runtime).
  resolve: {
    conditions: ["browser", "module", "import", "default"],
  },
  define: {
    "import.meta.env.VITE_APP_VERSION": JSON.stringify(appVersion),
  },
  plugins: [
    svelte(),
    VitePWA({
      registerType: "autoUpdate",
      // Use relative paths to work with secret paths (e.g., /dev/)
      // The service worker will be registered relative to the current page
      strategies: "generateSW",
      includeAssets: ["favicon.ico", "icon.svg"],
      // Ensure service worker works with relative base paths
      injectRegister: "auto",
      manifest: {
        name: "FoodList",
        short_name: "FoodList",
        description: "A beautiful, real-time shopping list app",
        theme_color: "#6366f1",
        background_color: "#6366f1",
        display: "standalone",
        start_url: "./",
        icons: [
          {
            src: "./icon.svg",
            sizes: "any",
            type: "image/svg+xml",
            purpose: "any maskable",
          },
        ],
      },
      workbox: {
        // Aggressive update strategy: check for updates frequently
        skipWaiting: true,
        clientsClaim: true,
        // Use NetworkFirst for fast updates - always try network first
        // This ensures users get the latest version quickly
        runtimeCaching: [
          {
            // Static assets (JS, CSS, HTML, JSON) - NetworkFirst for fast updates
            urlPattern: /\.(js|css|html|json)$/,
            handler: "NetworkFirst",
            options: {
              cacheName: "static-resources",
              expiration: {
                maxEntries: 50,
                maxAgeSeconds: 60 * 60 * 24, // 24 hours
              },
              cacheableResponse: {
                statuses: [0, 200],
              },
              networkTimeoutSeconds: 3, // Fall back to cache quickly if network is slow
            },
          },
          {
            // Images - CacheFirst for performance
            urlPattern: /\.(png|jpg|jpeg|svg|gif|webp|ico)$/,
            handler: "CacheFirst",
            options: {
              cacheName: "images",
              expiration: {
                maxEntries: 100,
                maxAgeSeconds: 60 * 60 * 24 * 7, // 7 days
              },
            },
          },
          {
            // WebSocket connections - don't cache, always use network
            urlPattern: /^wss?:\/\/.*$/,
            handler: "NetworkOnly",
          },
        ],
        // Enable navigation preload for faster page loads
        navigationPreload: true,
        // Clean up old caches on update
        cleanupOutdatedCaches: true,
      },
      // Enable in dev mode for testing
      devOptions: {
        enabled: true,
        type: "module",
      },
    }),
  ],
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./src/setupTests.ts"],
    include: ["src/**/*.{test,spec}.{js,ts}"],
    testTimeout: 5000,
  },
  server: {
    host: "0.0.0.0", // Allow external connections
    proxy: {
      "/ws": {
        target: process.env.VITE_BACKEND_URL || "ws://localhost:8080",
        ws: true,
        // Prepend the backend's secret-path prefix so /ws hits
        // /<secret>/ws on the backend. Without this, requests bypass
        // the secret path and rely on whitelisted-IP fallthrough,
        // which is fragile.
        rewrite: (path) =>
          backendPathPrefix ? `${backendPathPrefix}${path}` : path,
      },
      // MCP streamable HTTP (same origin as dev UI; backend serves /mcp).
      // /mcp is publicly accessible on the backend, so no rewrite needed.
      "/mcp": {
        target: process.env.VITE_BACKEND_HTTP_URL || "http://localhost:8080",
        changeOrigin: true,
      },
      // Recipe REST API (and any future /api/v1/* HTTP endpoints).
      // Without this, Vite's HTML fallback serves index.html for these
      // paths, causing the frontend's `resp.json()` to fail with
      // "JSON.parse: unexpected character at line 1 column 1".
      // The rewrite prepends the backend's secret-path prefix so
      // requests reach /<secret>/api/v1/... instead of being 404'd
      // by IPWhitelistMiddleware.
      "/api": {
        target: process.env.VITE_BACKEND_HTTP_URL || "http://localhost:8080",
        changeOrigin: true,
        rewrite: (path) =>
          backendPathPrefix ? `${backendPathPrefix}${path}` : path,
      },
    },
  },
}))
