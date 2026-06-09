import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 4992,
    host: 'localhost',
    proxy: {
      '/api': 'http://localhost:4993',
      '/healthz': 'http://localhost:4993',
      '/ws': {
        target: 'http://localhost:4993',
        ws: true,
      },
    },
  },
})
