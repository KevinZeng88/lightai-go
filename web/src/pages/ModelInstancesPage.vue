<template>
  <div class="mi-page">
    <div class="page-header">
      <h2>{{ t('modelInstances.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" size="small" @row-click="openDetail" highlight-current-row>
      <el-table-column :label="t('modelInstances.instanceId')" width="200" show-overflow-tooltip>
        <template #default="{ row }">{{ row.id }}</template>
      </el-table-column>
      <el-table-column prop="deployment_id" :label="t('modelInstances.deploymentId')" width="200" show-overflow-tooltip />
      <el-table-column :label="t('modelInstances.actualState')" width="100">
        <template #default="{ row }">
          <StatusTag :status="row.actual_state" />
        </template>
      </el-table-column>
      <el-table-column prop="node_id" :label="t('modelInstances.nodeId')" width="120" show-overflow-tooltip />
      <el-table-column prop="container_id" :label="t('modelInstances.containerId')" width="160" show-overflow-tooltip />
      <el-table-column prop="host_port" :label="t('modelInstances.hostPort')" width="90" />
      <el-table-column :label="t('modelInstances.startedAt')" width="160">
        <template #default="{ row }">{{ row.started_at ? formatDateTime(row.started_at) : '-' }}</template>
      </el-table-column>
      <el-table-column :label="t('modelInstances.lastError')" min-width="180" show-overflow-tooltip prop="last_error" />
      <el-table-column :label="t('common.actions')" width="100" fixed="right">
        <template #default="{ row }">
          <el-button size="small" text @click.stop="openLogs(row)">{{ t('modelInstances.logs') }}</el-button>
        </template>
      </el-table-column>
      <template #empty><el-empty :description="t('modelInstances.noData')" /></template>
    </el-table>

    <!-- Detail Drawer -->
    <el-drawer v-model="drawerVisible" :title="t('modelInstances.detail')" size="500px">
      <el-descriptions v-if="selected" :column="1" border size="small">
        <el-descriptions-item :label="t('modelInstances.instanceId')">{{ selected.id }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.deploymentId')">{{ selected.deployment_id }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.nodeId')">{{ selected.node_id }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.actualState')"><StatusTag :status="selected.actual_state" /></el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.containerId')">{{ selected.container_id || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.endpointUrl')">{{ selected.endpoint_url || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.hostPort')">{{ selected.host_port || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.startedAt')">{{ selected.started_at ? formatDateTime(selected.started_at) : '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.stoppedAt')">{{ selected.stopped_at ? formatDateTime(selected.stopped_at) : '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.restartCount')">{{ selected.restart_count }}</el-descriptions-item>
        <el-descriptions-item :label="t('modelInstances.lastError')">{{ selected.last_error || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-drawer>

    <!-- Logs Dialog -->
    <el-dialog v-model="logsVisible" :title="t('modelInstances.logsTitle')" width="700px">
      <div v-if="logsLoading" style="text-align:center;padding:40px">
        <el-icon class="is-loading"><Loading /></el-icon>
        <p>{{ t('modelInstances.logsPending') }}</p>
      </div>
      <div v-else-if="logsContent">
        <el-input :model-value="logsContent" type="textarea" :rows="20" readonly style="font-family:monospace;font-size:12px;white-space:pre" />
      </div>
      <el-empty v-else :description="t('modelInstances.logsEmpty')" />
      <template #footer>
        <el-button @click="fetchLogsInternal(true)">{{ t('modelInstances.refreshLogs') }}</el-button>
        <el-button @click="logsVisible = false">{{ t('common.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { RefreshRight, Loading } from '@element-plus/icons-vue'
import { fetchModelInstances, fetchInstanceLogs, type ModelInstance } from '@/api/modelInstances'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { formatDateTime } from '@/utils/format'
import StatusTag from '@/components/StatusTag.vue'

const { t } = useI18n()
const route = useRoute()
const deploymentId = (route.query.deployment_id as string) || ''

const items = ref<ModelInstance[]>([])
const { loading, refresh } = useAutoRefresh(async () => {
  items.value = await fetchModelInstances(deploymentId || undefined)
}, { intervalMs: 5000 })

const drawerVisible = ref(false)
const selected = ref<ModelInstance | null>(null)
const logsVisible = ref(false)
const logsContent = ref('')
const logsLoading = ref(false)
const currentLogsInstanceId = ref('')
let logsPollTimer: ReturnType<typeof setInterval> | null = null

function openDetail(row: ModelInstance) {
  selected.value = row
  drawerVisible.value = true
}

function openLogs(row: ModelInstance) {
  currentLogsInstanceId.value = row.id
  logsVisible.value = true
  logsContent.value = ''
  fetchLogsInternal(false)
}

async function fetchLogsInternal(manual: boolean) {
  if (!currentLogsInstanceId.value) return
  if (manual) {
    logsContent.value = ''
  }
  logsLoading.value = true
  try {
    const resp = await fetchInstanceLogs(currentLogsInstanceId.value)
    if (resp.logs) {
      logsContent.value = resp.logs
      logsLoading.value = false
      stopPolling()
    } else if (resp.status === 'pending') {
      logsLoading.value = true
      startPolling()
    } else {
      logsContent.value = resp.message || ''
      logsLoading.value = false
    }
  } catch (e: any) {
    logsContent.value = ''
    logsLoading.value = false
  }
}

function startPolling() {
  stopPolling()
  logsPollTimer = setInterval(() => fetchLogsInternal(false), 3000)
}

function stopPolling() {
  if (logsPollTimer) { clearInterval(logsPollTimer); logsPollTimer = null }
}
</script>
