declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

declare module '*.js' {
  const value: any
  export default value
  export const inferModelCapabilities: any
  export const recommendedTestMode: any
  export const capabilityLabel: any
  export const testModeLabel: any
  export const formatTestFailure: any
}
