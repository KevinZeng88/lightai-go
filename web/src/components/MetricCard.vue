<template>
  <el-card shadow="hover" class="metric-card">
    <div class="metric-title">{{ title }}</div>
    <div class="metric-value">
      {{ value }}<span v-if="unit" class="metric-unit">{{ unit }}</span>
    </div>
    <div v-if="description" class="metric-desc">{{ description }}</div>
  </el-card>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  value: string | number
  unit?: string
  description?: string
}>()
</script>

<style scoped>
.metric-card {
  text-align: center;
}
.metric-title {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}
.metric-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.metric-unit {
  font-size: 14px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.metric-desc {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  margin-top: 4px;
}
</style>
