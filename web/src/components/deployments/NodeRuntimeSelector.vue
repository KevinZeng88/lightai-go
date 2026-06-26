<template>
  <div>
    <el-table :data="props.nodeRuntimes" highlight-current-row @current-change="onSelect" max-height="400">
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
      <el-table-column :label="$t('common.status')" width="140">
        <template #default="{ row }">
          <el-tag :type="row.status === 'ready' ? 'success' : row.status === 'ready_with_warnings' ? 'warning' : 'info'">
            {{ row.status }}
          </el-tag>
          <div v-if="row.warnings?.length" style="font-size:11px;color:var(--el-color-warning);margin-top:2px">
            {{ row.warnings[0] }}
          </div>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!props.nodeRuntimes.length" :description="$t('common.noData')" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  nodeRuntimes: any[]
  modelValue: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

function onSelect(row: any) {
  if (row) emit('update:modelValue', row.id)
}
</script>
