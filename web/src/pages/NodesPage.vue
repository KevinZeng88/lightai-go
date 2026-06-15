<template>
  <div class="nodes-page">
    <div class="page-header">
      <h2>{{ t('nodes.title') }} ({{ filteredNodes.length }})</h2>
      <div class="header-actions">
        <el-button size="small" @click="resetWidths">{{ t('common.reset') }}</el-button>
        <el-button @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <div class="toolbar">
      <el-select v-model="statusFilter" :placeholder="t('nodes.status')" clearable size="small" style="width: 140px">
        <el-option :label="t('status.online')" value="online" />
        <el-option :label="t('status.offline')" value="offline" />
      </el-select>
      <el-input v-model="search" :placeholder="t('common.search')" clearable size="small" style="width: 240px; margin-left: 8px" />
    </div>

    <div class="table-wrap">
    <el-table :data="filteredNodes" v-loading="loading" size="small" @row-click="openDetail" style="width: 100%">
      <el-table-column :label="t('nodes.status')" :width="colWidth('status')">
        <template #header>
          <span>{{ t('nodes.status') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('status', $event)"></span>
        </template>
        <template #default="{ row }"><StatusTag :status="row.status" /></template>
      </el-table-column>
      <el-table-column prop="hostname" :label="t('nodes.hostname')" :width="colWidth('hostname')" show-overflow-tooltip>
        <template #header>
          <span>{{ t('nodes.hostname') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('hostname', $event)"></span>
        </template>
        <template #default="{ row }">{{ row.hostname || '-' }}</template>
      </el-table-column>
      <el-table-column :label="t('nodes.primaryIp')" :width="colWidth('primaryIp')" show-overflow-tooltip>
        <template #header>
          <span>{{ t('nodes.primaryIp') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('primaryIp', $event)"></span>
        </template>
        <template #default="{ row }">{{ row.primary_ip || '-' }}</template>
      </el-table-column>
      <el-table-column :label="t('nodes.gpuCount')" :width="colWidth('gpuCount')" align="center">
        <template #header>
          <span>{{ t('nodes.gpuCount') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('gpuCount', $event)"></span>
        </template>
        <template #default="{ row }">
          <span>{{ healthyGpuCount(row.id) }} / {{ (gpusByNodeId.get(row.id) || []).length }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('nodes.gpuMemory')" :width="colWidth('gpuMemory')">
        <template #header>
          <span>{{ t('nodes.gpuMemory') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('gpuMemory', $event)"></span>
        </template>
        <template #default="{ row }">{{ gpuMemorySummary(row.id) }}</template>
      </el-table-column>
      <el-table-column :label="t('nodes.agentVersion')" :width="colWidth('agentVersion')" show-overflow-tooltip>
        <template #header>
          <span>{{ t('nodes.agentVersion') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('agentVersion', $event)"></span>
        </template>
        <template #default="{ row }">{{ row.agent_version || '-' }}</template>
      </el-table-column>
      <el-table-column :label="t('nodes.lastHeartbeat')" :width="colWidth('lastHeartbeat')">
        <template #header>
          <span>{{ t('nodes.lastHeartbeat') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('lastHeartbeat', $event)"></span>
        </template>
        <template #default="{ row }">{{ formatRelativeTime(row.last_heartbeat_at, locale) }}</template>
      </el-table-column>
      <el-table-column :label="t('nodes.createdAt')" :width="colWidth('createdAt')">
        <template #header>
          <span>{{ t('nodes.createdAt') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('createdAt', $event)"></span>
        </template>
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <template #empty>{{ t('nodes.noNodes') }}</template>
    </el-table>
    </div>

    <el-drawer v-model="drawerVisible" :title="t('nodes.detail')" size="550px">
      <template v-if="selectedNode">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="t('nodes.nodeId')" :span="2">
            <span class="mono">{{ selectedNode.id }}</span>
            <CopyButton :text="selectedNode.id" />
          </el-descriptions-item>
          <el-descriptions-item :label="t('nodes.hostname')">{{ selectedNode.hostname || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.status')"><StatusTag :status="selectedNode.status" /></el-descriptions-item>
          <el-descriptions-item :label="t('nodes.primaryIp')">{{ selectedNode.primary_ip || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.advertiseAddr')">{{ selectedNode.advertised_address || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.agentVersion')">{{ selectedNode.agent_version || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.os')">{{ osInfo(selectedNode) }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.lastHeartbeat')">{{ formatDateTime(selectedNode.last_heartbeat_at) }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.createdAt')">{{ formatDateTime(selectedNode.created_at) }}</el-descriptions-item>
          <el-descriptions-item :label="t('nodes.gpuCount')" :span="2">
            {{ t('nodes.gpuCount') }}: {{ (gpusByNodeId.get(selectedNode.id) || []).length }}
            ({{ t('status.healthy') }}: {{ healthyGpuCount(selectedNode.id) }})
          </el-descriptions-item>
          <el-descriptions-item :label="t('nodes.gpuMemory')" :span="2">
            {{ gpuMemorySummary(selectedNode.id) }}
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 16px">{{ t('nodes.gpusOnNode') }} ({{ nodeGpus.length }})</h4>
        <el-table :data="nodeGpus" size="small" v-loading="gpuLoading">
          <el-table-column prop="index" :label="t('gpus.index')" width="50" />
          <el-table-column prop="vendor" :label="t('gpus.vendor')" width="80" />
          <el-table-column prop="name" :label="t('gpus.name')" show-overflow-tooltip />
          <el-table-column :label="t('gpus.memory')" width="200">
            <template #default="{ row }">
              {{ formatGB(row.memory_used_bytes) }} / {{ formatGB(row.memory_total_bytes) }}
              <span style="color: var(--el-text-color-secondary); font-size: 11px; display: block">
                {{ t('gpus.free') }}: {{ formatGB(row.memory_free_bytes) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column :label="t('gpus.gpuUtilization')" width="100">
            <template #default="{ row }">{{ formatPercent(row.gpu_utilization_percent) }}</template>
          </el-table-column>
          <el-table-column :label="t('gpus.health')" width="80">
            <template #default="{ row }"><StatusTag :status="row.health" /></template>
          </el-table-column>
          <template #empty>{{ t('gpus.noGpus') }}</template>
        </el-table>

        <h4 style="margin-top: 16px">{{ t('nodes.hostResources') }}</h4>
        <div v-if="sysLoading" style="text-align: center; padding: 16px"><el-icon class="is-loading"><Loading /></el-icon></div>
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
            <el-descriptions-item :label="t('nodes.uptime')">{{ formatUptime(parseInt(nodeSystem.uptime_seconds) || 0) }}</el-descriptions-item>
          </el-descriptions>
        </div>
        <div v-else style="color: var(--el-text-color-secondary); font-size: 13px; padding: 8px">{{ t('nodes.noSystemData') }}</div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchNodes, fetchNodeSystem, type Node, type NodeSystemInfo } from '@/api/nodes'
import { fetchGPUs, type GPU } from '@/api/gpus'
import StatusTag from '@/components/StatusTag.vue'
import CopyButton from '@/components/CopyButton.vue'
import { formatBytes, formatDateTime, formatPercent, formatRelativeTime, formatGB } from '@/utils/format'
import { useResizableColumns, groupGpusByNodeId } from '@/composables/useResizableColumns'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t, locale } = useI18n()

const NODE_COLUMN_DEFAULTS = {
  status: 80, hostname: 140, primaryIp: 140, gpuCount: 90, gpuMemory: 160, agentVersion: 120, lastHeartbeat: 100, createdAt: 160,
}
const { colWidth, startResize, resetWidths } = useResizableColumns('nodes', NODE_COLUMN_DEFAULTS, 50)

const nodes = ref<Node[]>([])
const gpus = ref<GPU[]>([])
const search = ref('')
const statusFilter = ref('')
const drawerVisible = ref(false)
const selectedNode = ref<Node | null>(null)
const nodeGpus = ref<GPU[]>([])
const gpuLoading = ref(false)
const sysLoading = ref(false)
const nodeSystem = ref<NodeSystemInfo | null>(null)

const { loading, refresh } = useAutoRefresh(async () => {
  const [n, g] = await Promise.all([fetchNodes(), fetchGPUs()])
  nodes.value = Array.isArray(n) ? n : []
  gpus.value = Array.isArray(g) ? g : []
})

const gpusByNodeId = computed(() => groupGpusByNodeId(gpus.value))

function healthyGpuCount(nodeId: string): number {
  return (gpusByNodeId.value.get(nodeId) || []).filter(g => g.health === 'healthy').length
}

function gpuMemorySummary(nodeId: string): string {
  const gpuList = gpusByNodeId.value.get(nodeId) || []
  if (gpuList.length === 0) return '-'
  let total = 0, used = 0, free = 0
  for (const g of gpuList) {
    total += g.memory_total_bytes || 0
    used += g.memory_used_bytes || 0
    free += g.memory_free_bytes || 0
  }
  return `${formatGB(used)} / ${formatGB(total)}`
}

function osInfo(node: Node): string {
  const parts = [node.os, node.arch, node.kernel].filter(Boolean)
  return parts.length > 0 ? parts.join(' / ') : '-'
}

const filteredNodes = computed(() => {
  let result = nodes.value
  if (statusFilter.value) result = result.filter(n => n.status === statusFilter.value)
  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter(n => n.hostname.toLowerCase().includes(q) || n.primary_ip.toLowerCase().includes(q) || n.id.toLowerCase().includes(q))
  }
  return result
})

async function openDetail(row: Node) {
  selectedNode.value = row
  drawerVisible.value = true
  gpuLoading.value = true
  sysLoading.value = true
  nodeSystem.value = null
  try { nodeGpus.value = (await fetchGPUs({ node_id: row.id })) || [] } catch { nodeGpus.value = [] }
  gpuLoading.value = false
  try { nodeSystem.value = await fetchNodeSystem(row.id) } catch { /* */ }
  sysLoading.value = false
}

function formatUptime(seconds: number): string {
  if (!seconds || seconds <= 0) return '0s'
  const d = Math.floor(seconds / 86400), h = Math.floor((seconds % 86400) / 3600), m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return d + 'd ' + h + 'h'
  if (h > 0) return h + 'h ' + m + 'm'
  return m + 'm'
}

</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
.header-actions { display: flex; gap: 8px; }
.toolbar { margin-bottom: 12px; }
.table-wrap { overflow-x: auto; }
.mono { font-family: monospace; font-size: 12px; }
.resize-handle { display: inline-block; width: 6px; height: 100%; cursor: col-resize; position: absolute; right: 0; top: 0; bottom: 0; }
.resize-handle:hover { background: var(--el-color-primary-light-5); }
:deep(.el-table th) { position: relative; }
</style>
