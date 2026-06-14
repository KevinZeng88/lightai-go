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

        <!-- P1-004: Host system resources -->
        <h4 style="margin-top: 16px">{{ t('nodes.hostResources') }}</h4>
        <div v-if="sysLoading" style="text-align: center; padding: 16px">
          <el-icon class="is-loading"><Loading /></el-icon>
        </div>
        <div v-else-if="nodeSystem">
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item :label="t('nodes.cpuCores')">{{ nodeSystem.cpu_cores }}</el-descriptions-item>
            <el-descriptions-item :label="t('nodes.cpuUsage')">{{ nodeSystem.cpu_utilization_percent }}%</el-descriptions-item>
            <el-descriptions-item :label="t('nodes.memory')">
              {{ formatBytes(nodeSystem.memory_used_bytes) }} / {{ formatBytes(nodeSystem.memory_total_bytes) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('nodes.loadAvg')">
              {{ nodeSystem.load1 }} / {{ nodeSystem.load5 }} / {{ nodeSystem.load15 }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('nodes.uptime')">
              {{ formatUptime(parseInt(nodeSystem.uptime_seconds) || 0) }}
            </el-descriptions-item>
          </el-descriptions>
          <h5 style="margin-top: 12px">{{ t('nodes.filesystems') }}</h5>
          <el-table :data="nodeSystem.filesystems || []" size="small" v-if="nodeSystem.filesystems?.length">
            <el-table-column prop="mount_point" :label="t('nodes.mountPoint')" width="140" />
            <el-table-column :label="t('nodes.diskUsage')" width="180">
              <template #default="{ row }">{{ formatBytes(row.used_bytes) }} / {{ formatBytes(row.total_bytes) }}</template>
            </el-table-column>
            <el-table-column :label="t('nodes.diskFree')" width="120">
              <template #default="{ row }">{{ formatBytes(row.free_bytes) }}</template>
            </el-table-column>
          </el-table>
          <h5 style="margin-top: 12px">{{ t('nodes.network') }}</h5>
          <el-table :data="nodeSystem.networks?.filter((n: any) => n.up) || []" size="small" v-if="nodeSystem.networks?.length">
            <el-table-column prop="name" :label="t('nodes.interface')" width="120" />
            <el-table-column :label="t('nodes.rxBytes')" width="140">
              <template #default="{ row }">{{ formatBytes(row.bytes_recv) }}</template>
            </el-table-column>
            <el-table-column :label="t('nodes.txBytes')" width="140">
              <template #default="{ row }">{{ formatBytes(row.bytes_sent) }}</template>
            </el-table-column>
          </el-table>
        </div>
        <div v-else style="color: var(--el-text-color-secondary); font-size: 13px; padding: 8px">
          {{ t('nodes.noSystemData') }}
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchNodes, fetchNodeSystem, type Node, type NodeSystemInfo } from '@/api/nodes'
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
const sysLoading = ref(false)
const nodeSystem = ref<NodeSystemInfo | null>(null)

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
  sysLoading.value = true
  nodeSystem.value = null
  try { nodeGpus.value = await fetchGPUs({ node_id: row.id }) } catch { /* */ }
  gpuLoading.value = false
  try { nodeSystem.value = await fetchNodeSystem(row.id) } catch { /* */ }
  sysLoading.value = false
}

function formatUptime(seconds: number): string {
  if (!seconds || seconds <= 0) return '0s'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return d + 'd ' + h + 'h'
  if (h > 0) return h + 'h ' + m + 'm'
  return m + 'm'
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
