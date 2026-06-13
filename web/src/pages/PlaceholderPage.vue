<template>
  <el-empty :description="description" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()

const description = computed(() => {
  const map: Record<string, string> = {
    '/models/deployments': 'placeholder.modelsDeployments',
    '/models/instances': 'placeholder.modelsInstances',
    '/runtime/environments': 'placeholder.runtimeEnvironments',
    '/observability/grafana': 'placeholder.grafana',
  }
  return t(map[route.path] || 'common.noData')
})
</script>
