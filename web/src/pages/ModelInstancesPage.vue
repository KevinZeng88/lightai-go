<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('instances.title') }}</h2>
      <el-button :icon="RefreshRight" :loading="loading" @click="refresh">{{ t('common.refresh') }}</el-button>
    </div>

    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="200" />
      <el-table-column prop="deployment_id" :label="t('instances.deployment')" width="200" />
      <el-table-column prop="actual_state" :label="t('instances.state')" width="120">
        <template #default="{ row }">
          <StatusTag :status="row.actual_state || 'unknown'" />
        </template>
      </el-table-column>
      <el-table-column prop="node_id" :label="t('instances.node')" width="160" />
      <el-table-column prop="container_id" :label="t('instances.container')" width="180" show-overflow-tooltip />
      <el-table-column prop="host_port" :label="t('instances.port')" width="90" />
      <el-table-column prop="endpoint_url" :label="t('instances.endpoint')" min-width="200" show-overflow-tooltip />
      <el-table-column :label="t('common.actions')" width="320" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ t('common.detail') }}</el-button>
          <el-button
            size="small"
            type="primary"
            :disabled="row.actual_state !== 'running'"
            :loading="testing && testRow?.id === row.id"
            @click="doTest(row)"
          >
            {{ t('instances.test') }}
          </el-button>
          <el-button
            size="small"
            :icon="Document"
            :disabled="!row.current_run_plan_id"
            @click="openLogs(row)"
          >
            {{ t('dockerLogs.title') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="testVisible" :title="t('instances.testResult')" width="560px">
      <div v-if="testResult" class="test-result">
        <el-alert
          :type="testResult.ok ? 'success' : 'error'"
          :title="testResult.ok ? t('instances.testPassed') : t('instances.testFailed')"
          :description="testResult.ok ? '' : testErrorMessage"
          show-icon
          :closable="false"
        />
        <el-descriptions v-if="testResult.ok" :column="1" border size="small" style="margin-top:12px">
          <el-descriptions-item :label="t('instances.testMode')">{{ testResult.mode === 'completion' ? 'Completion' : 'Chat' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testEndpoint')">{{ testResult.endpoint }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testModel')">{{ testResult.model }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testResolveMethod')">{{ testResult.model_resolution_method || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testLatency')">{{ testResult.latency_ms }} ms</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testPreview')">{{ testResult.response_preview || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testCheckedAt')">{{ testResult.checked_at || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions v-else :column="1" border size="small" style="margin-top:12px">
          <el-descriptions-item v-if="testResult.model_resolution_method" :label="t('instances.testResolveMethod')">{{ testResult.model_resolution_method }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testEndpoint')">{{ testResult.endpoint || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testModel')">{{ testResult.model || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testLatency')">{{ testResult.latency_ms != null ? testResult.latency_ms + ' ms' : '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testPreview')">{{ testResult.response_preview || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testCheckedAt')">{{ testResult.checked_at || '-' }}</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>

    <el-dialog v-model="detailVisible" :title="t('instances.detail')" width="640px">
      <div v-if="selected" class="detail-grid">
        <div v-for="(v, k) in selected" :key="k" class="detail-row">
          <strong>{{ k }}:</strong>
          <span>{{ typeof v === 'object' ? JSON.stringify(v) : v }}</span>
        </div>
      </div>
    </el-dialog>

    <el-drawer v-model="logsVisible" :title="t('dockerLogs.title')" size="60%">
      <div class="logs-toolbar">
        <el-input-number v-model="logsTail" :min="1" :max="5000" :step="100" size="small" />
        <el-button :icon="RefreshRight" :loading="logsLoading" @click="loadLogs">{{ t('dockerLogs.refresh') }}</el-button>
        <el-button :icon="CopyDocument" :disabled="!logsText" @click="copyLogs">{{ t('dockerLogs.copy') }}</el-button>
      </div>

      <el-alert
        v-if="logsError"
        type="error"
        :title="t('dockerLogs.loadFailed')"
        :description="logsError"
        show-icon
        :closable="false"
      />

      <el-descriptions v-if="logsMeta" :column="2" border size="small" class="logs-meta">
        <el-descriptions-item :label="t('dockerLogs.taskId')">{{ logsMeta.task_id || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('instances.container')">{{ logsMeta.container_id || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('instances.node')">{{ logsMeta.node_id || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dockerLogs.runtimeState')">{{ logsMeta.runtime_state || '-' }}</el-descriptions-item>
      </el-descriptions>

      <pre class="docker-log-output">{{ logsText || t('dockerLogs.empty') }}</pre>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { CopyDocument, Document, RefreshRight } from '@element-plus/icons-vue'
import { apiClient } from '@/api/client'
import StatusTag from '@/components/StatusTag.vue'

const { t } = useI18n()

const loading = ref(false)
const items = ref<any[]>([])
const selected = ref<any>(null)
const detailVisible = ref(false)
const logsVisible = ref(false)
const logsLoading = ref(false)
const logsTail = ref(200)
const logsText = ref('')
const logsError = ref('')
const logsMeta = ref<any>(null)
const logsRow = ref<any>(null)
const autoOpenedFailedLogs = ref(false)

// Model smoke test
const testing = ref(false)
const testVisible = ref(false)
const testRow = ref<any>(null)
const testResult = ref<any>(null)

// Reason code → i18n key mapping for test failures
const testReasonI18n: Record<string, string> = {
  instance_not_running: 'instances.testReasonNotRunning',
  no_endpoint: 'instances.testReasonNoEndpoint',
  network_error: 'instances.testReasonNetworkError',
  model_id_not_resolved: 'instances.testReasonModelNotResolved',
  models_endpoint_failed: 'instances.testReasonModelsEndpointFailed',
  chat_endpoint_failed: 'instances.testReasonChatFailed',
  completion_endpoint_failed: 'instances.testReasonCompletionFailed',
  empty_model_response: 'instances.testReasonEmptyResponse',
  inference_endpoint_not_supported: 'instances.testReasonInferenceNotSupported',
  model_test_failed: 'instances.testReasonTestFailed',
}
for (let i = 400; i < 600; i++) {
  testReasonI18n[`http_${i}`] = 'instances.testReasonHttpError'
}

const testErrorMessage = computed(() => {
  if (!testResult.value || testResult.value.ok) return ''
  const code = testResult.value.reason_code || ''
  const key = testReasonI18n[code]
  if (key) return t(key, { code, message: testResult.value.message || '' })
  // Check prefix match for http_xxx
  if (code.startsWith('http_')) return t('instances.testReasonHttpError', { code, message: testResult.value.message || '' })
  return testResult.value.message || code
})

onMounted(async () => {
  await refresh()
})

async function refresh() {
  loading.value = true
  try {
    items.value = await apiClient.get('/model-instances')
    if (!autoOpenedFailedLogs.value) {
      const failed = items.value.find((it) => it.actual_state === 'failed' && it.current_run_plan_id)
      if (failed) {
        autoOpenedFailedLogs.value = true
        await nextTick()
        openLogs(failed)
      }
    }
  } catch (e: any) {
    ElMessage.error(e?.message || t('common.requestFailed'))
  } finally {
    loading.value = false
  }
}

function showDetail(row: any) {
  selected.value = row
  detailVisible.value = true
}

async function openLogs(row: any) {
  logsRow.value = row
  logsVisible.value = true
  await loadLogs()
}

async function loadLogs() {
  if (!logsRow.value?.current_run_plan_id) {
    logsError.value = t('dockerLogs.noRunPlan')
    return
  }
  logsLoading.value = true
  logsError.value = ''
  try {
    const resp = await apiClient.get(`/node-run-plans/${logsRow.value.current_run_plan_id}/logs?tail=${logsTail.value}`)
    logsMeta.value = resp
    logsText.value = resp?.logs || ''
  } catch (e: any) {
    logsMeta.value = null
    logsText.value = ''
    logsError.value = e?.message || t('dockerLogs.loadFailed')
  } finally {
    logsLoading.value = false
  }
}

async function copyLogs() {
  try {
    await navigator.clipboard.writeText(logsText.value || '')
    ElMessage.success(t('common.copied'))
  } catch {
    ElMessage.error(t('common.copyFailed'))
  }
}

async function doTest(row: any) {
  testRow.value = row
  testing.value = true
  testVisible.value = true
  testResult.value = null
  try {
    testResult.value = await apiClient.post(`/model-instances/${row.id}/test`, { mode: 'chat', prompt: 'ping', timeout_seconds: 30 })
  } catch (e: any) {
    testResult.value = { ok: false, reason_code: 'network_error', message: e?.message || t('common.requestFailed') }
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.detail-grid {
  display: grid;
  gap: 8px;
}

.detail-row {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 12px;
  word-break: break-all;
}

.logs-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.logs-meta {
  margin: 12px 0;
}

.docker-log-output {
  min-height: 360px;
  max-height: calc(100vh - 280px);
  overflow: auto;
  margin: 0;
  padding: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background: #111827;
  color: #e5e7eb;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
