import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    // Optimize bundle size for production
    rollupOptions: {
      output: {
        manualChunks: {
          // Split large vendor packages
          'react-vendor': ['react', 'react-dom'],
          'mermaid-vendor': ['mermaid'],
          'ui-vendor': ['lucide-react']
        }
      }
    },
    // Increase chunk size warning limit for large apps
    chunkSizeWarningLimit: 1000,
    // Source maps for debugging (optional in production)
    sourcemap: false,
  },
  server: {
    // Development proxy to backend
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  }
})
