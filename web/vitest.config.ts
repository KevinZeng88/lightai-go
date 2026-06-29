import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    globals: false,
    setupFiles: ['./tests/setup/vitest.setup.ts'],
    include: [
      'src/components/**/*.render.test.ts',
      'src/components/**/__tests__/*.test.ts',
      'src/pages/**/*.integration.test.ts',
      'src/composables/__tests__/*.test.ts',
      'src/stores/__tests__/*.test.ts',
      'src/pages/__tests__/*.test.ts',
      'src/utils/__tests__/*.test.ts',
    ],
    exclude: [
      'node_modules',
      'dist',
      'tests/e2e',
    ],
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
})
