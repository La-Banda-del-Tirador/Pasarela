import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Cambiar el puerto si el backend Go corre en otro puerto
const BACKEND = 'http://localhost:9090'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api':     { target: BACKEND, changeOrigin: true },
      '/webhook': { target: BACKEND, changeOrigin: true },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
