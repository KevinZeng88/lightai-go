// Vitest setup for Vue + Element Plus + i18n component tests.
import { config } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { createI18n } from 'vue-i18n'

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  fallbackLocale: 'en-US',
  messages: {},
})

// Register Element Plus and vue-i18n globally so components that use
// useI18n() or el-* components resolve without additional setup.
config.global.plugins = [ElementPlus, i18n]

// Stub ResizeObserver (used by Element Plus components).
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.ResizeObserver = ResizeObserverStub as any

// Stub matchMedia for Element Plus responsive components.
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})

// Stub IntersectionObserver.
class IntersectionObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.IntersectionObserver = IntersectionObserverStub as any

// Suppress Vue warnings about missing translations in tests.
// Tests should still fail on actual errors.
const originalWarn = console.warn
console.warn = (...args: any[]) => {
  const msg = String(args[0])
  // Ignore expected i18n fallback warnings during test
  if (msg.includes('Fallback to') && msg.includes('locale')) return
  originalWarn(...args)
}
