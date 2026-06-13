<template>
  <el-tag :type="tagType" size="small" :effect="effect">
    {{ displayText }}
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { getStatusType } from '@/utils/status'

const props = withDefaults(defineProps<{
  status: string
  effect?: 'light' | 'dark' | 'plain'
}>(), {
  effect: 'light',
})

const { t } = useI18n()

const tagType = computed(() => {
  const type = getStatusType(props.status)
  return type || 'info'
})

const displayText = computed(() => {
  const key = `status.${props.status}`
  const translated = t(key)
  return translated === key ? props.status : translated
})
</script>
