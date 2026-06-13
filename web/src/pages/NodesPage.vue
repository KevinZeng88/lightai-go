<template>
  <div class="nodes-page">
    <div class="page-header">
      <h2>{{ t('nodes.title') }}</h2>
      <el-button @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
    </div>

    <div class="toolbar">
      <el-select v-model="statusFilter" :placeholder="t('nodes.status')" clearable size="small" style="width: 140px">
        <el-option :label="t('status.online')" value="online" />
        <el-option :label="t('status.offline')" value="offline" />
      </el-select>
      <el-input v-model="search" :placeholder="t('common.search')" clearable size="small" style="width: 240px; margin-left: 8px" />
    </div>

    <el-table :data="filteredNodes" v-loading="loading" size="small" @row-click="openDetail">
      <el-table-column prop="status" :label="t('nodes.status')" width="100">
        <template #default="{ row }"><StatusTag :status="row.status" /></template>
      </el-table-column>
      <el-table-column prop="hostname" :label="t('nodes.hostname')" />
      <el-table-column :label="t('nodes.nodeId')" width="120" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="mono">{{ row.id?.substring(0, 8) }}...</span>
          <CopyButton :text="row.id" />
        </template>
      </el-table-column>
      <el-table-column prop="advertised_address" :label="t('nodes.address')" width="160" />
      <el-table-column :label="t('nodes.lastHeartbeat')" width="160">
        <template #default="{ row }">{{ formatDateTime(row.last_heartbeat_at) }}</template>
      </el-table-column>
      <el-table-column :label="t('nodes.createdAt')" width="160">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <template #empty>{{ t('nodes.noNodes') }}</template>
    </el-table>

    <el-drawer v-model="drawerVisible" :title="t('nodes.detail')" size="500px">
      <template v-if="selectedNode">
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item :label="t('nodes.nodeId')">
            <span class="mono">{{ selectedNode.id }}</span>
            <CopyButton :text="selectedNode.id" />
          </el-descriptions-item>
          <el-descriptions-item :label="t('nodes.hostname')">{{ selectedNode.hostname }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.status')"><StatusTag :status="selectedNode.status" /></el-descriptions-item>
          <el-descriptions-item :label="t('nodes.address')">{{ selectedNode.advertised_address }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.lastHeartbeat')">{{ formatDateTime(selectedNode.last_heartbeat_at) }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.createdAt')">{{ formatDateTime(selectedNode.created_at) }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.updatedAt')">{{ formatDateTime(selectedNode.updated_at) }}</el-descriptions-item>
        </el-descriptions>
        <h4 style="margin-top: 16px">{{ t('nodes.gpusOnNode') }}</h4>
        <el-table :data="nodeGpus" size="small" v-loading="gpuLoading">
          <el-table-column prop="vendor" :label="t('gpus.vendor')" width="80" />
          <el-table-column prop="name" :label="t('gpus.name')" show-overflow-tooltip />
          <el-table-column :label="t('gpus.memoryUsed')" width="130">
            <template #default="{ row }">{{ formatBytes(row.memory_used_bytes) }} / {{ formatBytes(row.memory_total_bytes) }}</template>
          </el-table-column>
          <el-table-column :label="t('gpus.gpuUtilization')" width="100">
            <template #default="{ row }">{{ formatPercent(row.gpu_utilization_percent) }}</template>
          </el-table-column>
          <template #empty>{{ t('gpus.noGpus') }}</template>
        </el-table>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchNodes, type Node } from '@/api/nodes'
import { fetchGPUs, type GPU } from '@/api/gpus'
import StatusTag from '@/components/StatusTag.vue'
import CopyButton from '@/components/CopyButton.vue'
import { formatBytes, formatDateTime, formatPercent } from '@/utils/format'

const { t } = useI18n()

const nodes = ref<Node[]>([])
const loading = ref(false)
const search = ref('')
const statusFilter = ref('')
const drawerVisible = ref(false)
const selectedNode = ref<Node | null>(null)
const nodeGpus = ref<GPU[]>([])
const gpuLoading = ref(false)

const filteredNodes = computed(() => {
  let result = nodes.value
  if (statusFilter.value) result = result.filter(n => n.status === statusFilter.value)
  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter(n =>
      n.hostname.toLowerCase().includes(q) ||
      n.id.toLowerCase().includes(q)
    )
  }
  return result
})

async function refresh() {
  loading.value = true
  try { nodes.value = await fetchNodes() } catch { /* */ }
  loading.value = false
}

async function openDetail(row: Node) {
  selectedNode.value = row
  drawerVisible.value = true
  gpuLoading.value = true
  try { nodeGpus.value = await fetchGPUs({ node_id: row.id }) } catch { /* */ }
  gpuLoading.value = false
}

onMounted(refresh)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.page-header h2 { margin: 0; }
.toolbar { margin-bottom: 12px; }
.mono { font-family: monospace; font-size: 12px; }
</style>
