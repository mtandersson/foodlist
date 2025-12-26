/// <reference types="vitest" />
import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  // For production builds we need relative asset URLs so the app can be served
  // from any prefix like /<SHARED_SECRET>/ (e.g. /dev/).
  // Keep dev as absolute root for best HMR behavior.
  base: mode === 'production' ? './' : '/',
  plugins: [svelte()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.ts'],
    include: ['src/**/*.{test,spec}.{js,ts}'],
  },
  server: {
    host: '0.0.0.0', // Allow external connections
    proxy: {
      '/ws': {
        target: process.env.VITE_BACKEND_URL || 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
}))
