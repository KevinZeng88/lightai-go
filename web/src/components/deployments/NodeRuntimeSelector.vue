<template>
  <div>
    <el-table :data="props.nodeRuntimes" highlight-current-row @current-change="onSelect" max-height="400" :row-class-name="rowClass">
      <el-table-column :label="$t('deployments.runtime')" min-width="200">
        <template #default="{ row }">{{ row.display_name || row.backend_runtime?.display_name || row.backend_runtime?.name || row.id }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.node')" width="140">
        <template #default="{ row }">{{ row.node_id }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.backend')" width="120">
        <template #default="{ row }">{{ row.backend_runtime?.name || '' }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.vendor')" width="100">
        <template #default="{ row }">{{ row.backend_runtime?.vendor || '' }}</template>
      </el-table-column>
      <el-table-column prop="image_ref" :label="$t('runtimes.image')" min-width="200" show-overflow-tooltip />
      <el-table-column :label="$t('common.status')" width="160">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row)" size="small">
            {{ row.status }}
          </el-tag>
          <div v-if="row.disabled_reason" style="font-size:11px;color:var(--el-color-warning);margin-top:2px">
            {{ row.disabled_reason }}
          </div>
          <div v-if="row.warnings?.length && !row.disabled_reason" style="font-size:11px;color:var(--el-color-warning);margin-top:2px">
            {{ row.warnings[0] }}
          </div>
          <div v-if="!isNBRDeployable(row)" style="font-size:11px;color:var(--el-color-danger);margin-top:2px">
            {{ $t('runnerConfigs.needsCheckFirst') || 'Run check-request first' }}
          </div>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!props.nodeRuntimes.length" :description="$t('common.noData') || 'No node runtime configs available'" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  nodeRuntimes: any[]
  modelValue: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

function isNBRDeployable(row: any): boolean {
  if (row.deployable === true) return true
  if (row.status === 'ready' || row.status === 'ready_with_warnings') return true
  return false
}

function statusTagType(row: any): string {
  if (row.status === 'ready') return 'success'
  if (row.status === 'ready_with_warnings') return 'warning'
  if (row.status === 'needs_check') return 'info'
  if (row.status === 'missing_image' || row.status === 'error') return 'danger'
  return 'info'
}

function rowClass({ row }: { row: any }): string {
  if (row && !isNBRDeployable(row)) return 'nbr-row--disabled'
  return ''
}

function onSelect(row: any) {
  if (!row) return
  if (!isNBRDeployable(row)) {
    // Silently reject non-deployable rows — do not emit id.
    return
  }
  emit('update:modelValue', row.id)
}
</script>

<style scoped>
:deep(.nbr-row--disabled) {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
