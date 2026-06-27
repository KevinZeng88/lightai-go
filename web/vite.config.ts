import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

const backendTarget = 'http://127.0.0.1:18080'

const backendProxy = {
  target: backendTarget,
  changeOrigin: true,
  configure: (proxy: any) => {
    proxy.on('proxyReq', (proxyReq: any) => {
      proxyReq.removeHeader('origin')
      proxyReq.setHeader('Origin', backendTarget)
    })
  },
}

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 15173,
    strictPort: true,
    proxy: {
      '/api': backendProxy,
      '/metrics': backendProxy,
      '/healthz': backendProxy,
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
  },
})
