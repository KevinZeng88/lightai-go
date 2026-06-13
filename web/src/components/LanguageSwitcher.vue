<template>
  <el-dropdown @command="switchLang" trigger="click">
    <span class="lang-switch">
      <el-icon><Switch /></el-icon>
      {{ locale === 'zh-CN' ? t('common.zhCN') : t('common.enUS') }}
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="zh-CN">{{ t('common.zhCN') }}</el-dropdown-item>
        <el-dropdown-item command="en-US">{{ t('common.enUS') }}</el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { locale, t } = useI18n()
const router = useRouter()

function switchLang(lang: string) {
  locale.value = lang
  localStorage.setItem('lightai-locale', lang)
  location.reload()
}
</script>

<style scoped>
.lang-switch {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
  color: var(--el-text-color-regular);
}
</style>
