import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  // Load environment variables
  const env = loadEnv(mode, process.cwd(), '')

  // Define backend URL with port from environment variable or default to 8080
  const backendUrl = `http://${env.BACKEND_HOST || 'localhost'}:${env.BACKEND_PORT || '3000'}`

  return {
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
        '/avatars': backendUrl
      }
    }
  }
});

