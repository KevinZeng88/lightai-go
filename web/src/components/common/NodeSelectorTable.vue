<template>
  <div>
    <div v-if="loading" style="text-align:center;padding:40px">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
      <p>{{ $t('common.loading') }}</p>
    </div>
    <div v-else-if="error" style="text-align:center;padding:40px">
      <el-result icon="error" :title="$t('common.error')" :sub-title="error">
        <template #extra><el-button @click="$emit('refresh')">{{ $t('common.refresh') }}</el-button></template>
      </el-result>
    </div>
    <div v-else-if="!nodes.length" style="text-align:center;padding:40px">
      <el-empty :description="$t('nodes.noNodes')">
        <el-button @click="$emit('refresh')">{{ $t('common.refresh') }}</el-button>
      </el-empty>
    </div>
    <el-table v-else :data="nodes" highlight-current-row @current-change="onSelect" max-height="400">
      <el-table-column :label="$t('nodes.hostname')" min-width="160">
        <template #default="{ row }">{{ row.name || row.hostname || row.id }}</template>
      </el-table-column>
      <el-table-column prop="id" :label="$t('nodes.nodeId')" width="200" show-overflow-tooltip />
      <el-table-column :label="$t('common.status')" width="100">
        <template #default="{ row }">
          <StatusTag :status="row.status || 'unknown'" />
        </template>
      </el-table-column>
      <el-table-column v-if="showGpuInfo" :label="$t('nodes.gpuCount')" width="100">
        <template #default="{ row }">{{ row.gpu_count ?? '-' }}</template>
      </el-table-column>
    </el-table>
    <div v-if="nodes.length && !hideRefresh" style="margin-top:12px; text-align:right">
      <el-button size="small" @click="$emit('refresh')">{{ $t('common.refresh') }}</el-button>
    </div>
    <div v-if="selected && !hideSelectedTag" style="margin-top:12px">
      <el-tag type="success">{{ label }}: {{ selected.name || selected.hostname || selected.id }}</el-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import StatusTag from '@/components/StatusTag.vue'

const props = withDefaults(defineProps<{
  nodes: any[]
  label?: string
  loading?: boolean
  error?: string
  showGpuInfo?: boolean
  hideRefresh?: boolean
  hideSelectedTag?: boolean
}>(), {
  label: '',
  loading: false,
  error: '',
  showGpuInfo: true,
  hideRefresh: false,
  hideSelectedTag: false,
})

const emit = defineEmits<{
  select: [node: any]
  refresh: []
}>()

const selected = ref<any>(null)

function onSelect(row: any) {
  if (!row) return
  selected.value = row
  emit('select', row)
}

defineExpose({ selected, clear() { selected.value = null } })
</script>
