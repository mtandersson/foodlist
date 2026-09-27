import { defineConfig } from 'vite'
import { viteSingleFile } from 'vite-plugin-singlefile'

export default defineConfig({
  root: 'mcp-app',
  plugins: [viteSingleFile()],
  build: {
    outDir: '../../backend/mcp_app_dist',
    emptyOutDir: true,
    rollupOptions: { input: 'mcp-app/shopping.html' },
  },
})
