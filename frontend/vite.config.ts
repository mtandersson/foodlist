/// <reference types="vitest" />
import { defineConfig } from "vitest/config";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { VitePWA } from "vite-plugin-pwa";

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  // For production builds we need relative asset URLs so the app can be served
  // from any prefix like /<SHARED_SECRET>/ (e.g. /dev/).
  // Keep dev as absolute root for best HMR behavior.
  base: mode === "production" ? "./" : "/",
  // Ensure Svelte resolves to the browser runtime during tests (jsdom) so
  // component mounting works (Svelte 5 exports also include a server runtime).
  resolve: {
    conditions: ["browser", "module", "import", "default"],
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
  },
  server: {
    host: "0.0.0.0", // Allow external connections
    proxy: {
      "/ws": {
        target: process.env.VITE_BACKEND_URL || "ws://localhost:8080",
        ws: true,
        changeOrigin: true,
      },
    },
  },
}));
