<template>
  <el-button size="small" text @click="doCopy">
    <el-icon><DocumentCopy /></el-icon>
    {{ copied ? t('common.copied') : '' }}
  </el-button>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'

const props = defineProps<{ text: string }>()
const { t } = useI18n()
const copied = ref(false)

async function doCopy() {
  try {
    await navigator.clipboard.writeText(props.text)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
    ElMessage.success(t('common.copied'))
  } catch {
    ElMessage.error(t('common.copyFailed'))
  }
}
</script>
