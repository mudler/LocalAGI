import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Define backend URL with port from environment variable or default to 8080
const backendUrl = `http://${process.env.BACKEND_HOST || 'localhost'}:${process.env.BACKEND_PORT || '3000'}`

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/app',  // Set the base path for production builds
  server: {
    proxy: {
      // Proxy API requests to your Go backend
      '/api': backendUrl,
      // Proxy SSE endpoints
      '/sse': backendUrl,
      // Add other endpoints as needed
      '/settings': backendUrl,
      '/agents': backendUrl,
      '/create': backendUrl,
      '/delete': backendUrl,
      '/pause': backendUrl,
      '/start': backendUrl,
      '/talk': backendUrl,
      '/notify': backendUrl,
      '/chat': backendUrl,
      '/status': backendUrl,
      '/action': backendUrl,
      '/actions': backendUrl,
    }
  }
});