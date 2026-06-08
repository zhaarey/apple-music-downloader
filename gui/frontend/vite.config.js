import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// When running `wails dev`, proxy splice-media to the Wails asset server (default port 34115).
const wailsDevServer = process.env.WAILS_DEVSERVERURL || 'http://127.0.0.1:34115'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/splice-media': {
        target: wailsDevServer,
        changeOrigin: true,
      },
    },
  },
})
