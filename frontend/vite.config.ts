/// <reference types="vitest" />
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vite.dev/config/
export default defineConfig({
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
})
