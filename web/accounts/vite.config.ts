import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import tsconfigPaths from 'vite-tsconfig-paths'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'

export default defineConfig({
  plugins: [
    TanStackRouterVite({
      routesDirectory: './src/routes',
      generatedRouteTree: './src/routeTree.gen.ts',
    }),
    react(),
    tailwindcss(),
    tsconfigPaths({ projects: ['./tsconfig.json'] }),
  ],
  server: {
    port: 3001,
    proxy: {
      '/v1': 'http://localhost:8000',
      '/oauth': 'http://localhost:8000',
      '/login': 'http://localhost:8000',
      '/authorize': 'http://localhost:8000',
      '/.well-known': 'http://localhost:8000',
    },
  },
})
