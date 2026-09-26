import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// The Go backend serves the built frontend directly (see cmd/pos/main.go),
// so in production there is no separate origin and no proxy involved. This
// dev-server proxy only exists so `npm run dev` can run against a
// `go run ./cmd/pos` backend on 17831 without a CORS dance.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:17831',
      '/health': 'http://127.0.0.1:17831',
    },
  },
  build: {
    outDir: 'dist',
    target: 'es2020',
  },
})
