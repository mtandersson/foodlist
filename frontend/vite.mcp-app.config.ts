import { defineConfig } from 'vite'
import { viteSingleFile } from 'vite-plugin-singlefile'

export default defineConfig({
  root: 'mcp-app',
  plugins: [viteSingleFile()],
  build: {
    outDir: '../../backend/mcp_app_dist',
    emptyOutDir: process.env.MCP_APP !== 'recipes',
    rollupOptions: { input: `mcp-app/${process.env.MCP_APP === 'recipes' ? 'recipes' : 'shopping'}.html` },
  },
})
