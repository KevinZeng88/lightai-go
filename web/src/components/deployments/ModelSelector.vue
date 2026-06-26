<template>
  <div>
    <el-input v-model="search" :placeholder="$t('common.search')" clearable style="margin-bottom:12px" />
    <el-table :data="filteredArtifacts" highlight-current-row @current-change="selected = $event" max-height="400">
      <el-table-column :label="$t('deployments.artifact')" min-width="200">
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="format" :label="$t('artifacts.format')" width="100" />
      <el-table-column prop="task_type" :label="$t('artifacts.taskType')" width="120" />
      <el-table-column prop="quantization" :label="$t('artifacts.quantization')" width="100" />
      <el-table-column :label="$t('artifacts.path')" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.path }}</template>
      </el-table-column>
    </el-table>
    <div v-if="selected" style="margin-top:12px">
      <el-tag size="small" type="success">{{ $t('deployments.selectedArtifact') }}: {{ selected.display_name || selected.name }}</el-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  artifacts: any[]
  modelValue: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const search = ref('')
const selected = ref<any>(null)

const filteredArtifacts = computed(() => {
  if (!search.value) return props.artifacts
  const q = search.value.toLowerCase()
  return props.artifacts.filter((a: any) =>
    (a.name || '').toLowerCase().includes(q) ||
    (a.display_name || '').toLowerCase().includes(q) ||
    (a.path || '').toLowerCase().includes(q)
  )
})

watch(selected, (val) => {
  if (val) emit('update:modelValue', val.id)
})
</script>
