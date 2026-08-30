import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  // Removed the base property completely so it runs from the root of the subdomain
  plugins: [react(), tailwindcss()],
  server: {
    port: 4992,
    host: true, // Listen to local network
    allowedHosts: ['lofp.sanburnlab.com'], // Trusted subdomain
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