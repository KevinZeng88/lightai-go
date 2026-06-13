import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCN from 'element-plus/es/locale/lang/zh-cn'
import enUS from 'element-plus/es/locale/lang/en'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import i18n from './locales'

const app = createApp(App)

// Register Element Plus icons globally.
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// Set Element Plus locale based on saved language.
const savedLocale = localStorage.getItem('lightai-locale') || 'zh-CN'
app.use(ElementPlus, {
  locale: savedLocale === 'en-US' ? enUS : zhCN,
})

app.use(createPinia())
app.use(router)
app.use(i18n)
app.mount('#app')
