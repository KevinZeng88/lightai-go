<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('instances.title') }}</h2>
      <div class="header-actions">
        <el-checkbox v-model="showStopped">{{ t('instances.showStopped') }}</el-checkbox>
        <el-button :icon="RefreshRight" :loading="loading" @click="refresh">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <el-table :data="visibleItems" v-loading="loading" stripe>
      <el-table-column prop="id" :label="t('instances.instance')" width="200" show-overflow-tooltip />
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
      <el-table-column :label="t('common.actions')" width="420" fixed="right">
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
          <el-button
            size="small"
            type="danger"
            :disabled="row.actual_state !== 'running' && row.actual_state !== 'starting'"
            :loading="stopping && stoppingId === row.id"
            @click="doStop(row)"
          >
            {{ t('instances.stop') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="testVisible" :title="t('instances.testResult')" width="560px">
      <el-form label-position="top" style="margin-bottom:12px">
        <el-form-item :label="t('instances.testMode')">
          <el-select v-model="testMode" style="width:100%">
            <el-option :label="testModeText('auto')" value="auto" />
            <el-option :label="testModeText('chat')" value="chat" />
            <el-option :label="testModeText('completion')" value="completion" />
            <el-option :label="testModeText('embedding')" value="embedding" />
            <el-option :label="testModeText('rerank')" value="rerank" />
          </el-select>
        </el-form-item>
        <el-button type="primary" :loading="testing" :disabled="!testRow" @click="runSelectedTest">{{ t('instances.runTest') }}</el-button>
      </el-form>
      <div v-if="testResult" class="test-result">
        <el-alert
          :type="testResult.ok ? 'success' : 'error'"
          :title="testResult.ok ? t('instances.testPassed') : t('instances.testFailed')"
          :description="testResult.ok ? '' : testErrorMessage"
          show-icon
          :closable="false"
        />
        <el-descriptions v-if="testResult.ok" :column="1" border size="small" style="margin-top:12px">
          <el-descriptions-item :label="t('instances.testMode')">{{ testModeText(testResult.mode || testMode) }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testEndpoint')">{{ testResult.endpoint }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testModel')">{{ testResult.model }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testResolveMethod')">{{ testResult.model_resolution_method || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testLatency')">{{ testResult.latency_ms }} ms</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testPreview')">{{ testResult.response_preview || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testCheckedAt')">{{ testResult.checked_at || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions v-else :column="1" border size="small" style="margin-top:12px">
          <el-descriptions-item v-if="testResult.reason_code" :label="t('instances.testReasonCode')">{{ testResult.reason_code }}</el-descriptions-item>
          <el-descriptions-item v-if="testResult.http_status" :label="t('instances.testHttpStatus')">{{ testResult.http_status }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testEndpoint')">{{ testResult.endpoint || '-' }}</el-descriptions-item>
          <el-descriptions-item v-if="testResult.requested_model" :label="t('instances.testRequestedModel')">{{ testResult.requested_model }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testModel')">{{ testResult.model || '-' }}</el-descriptions-item>
          <el-descriptions-item v-if="testResult.available_models && testResult.available_models.length" :label="t('instances.testAvailableModels')">{{ testResult.available_models.join(', ') }}</el-descriptions-item>
          <el-descriptions-item v-if="testResult.model_resolution_method" :label="t('instances.testResolveMethod')">{{ testResult.model_resolution_method }}</el-descriptions-item>
          <el-descriptions-item v-if="testResult.hint" :label="t('instances.testHint')">{{ testResult.hint }}</el-descriptions-item>
          <el-descriptions-item v-if="testResult.error_body" :label="t('instances.testBackendError')"><span class="text-danger">{{ testResult.error_body }}</span></el-descriptions-item>
          <el-descriptions-item :label="t('instances.testLatency')">{{ testResult.latency_ms != null ? testResult.latency_ms + ' ms' : '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testPreview')">{{ testResult.response_preview || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.testCheckedAt')">{{ testResult.checked_at || '-' }}</el-descriptions-item>
        </el-descriptions>
        <h4 style="margin-top:12px">{{ t('instances.rawResponse') }}</h4>
        <el-collapse>
          <el-collapse-item :title="t('runnerConfigs.advancedJson')">
            <pre class="raw-response">{{ testResult.raw_response || testResult.response_preview || '-' }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-dialog>

    <el-dialog v-model="detailVisible" :title="t('instances.detail')" width="760px">
      <template v-if="selected">
        <h4>{{ t('instances.sectionBasic') }}</h4>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="t('instances.instance')">{{ selected.id }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.state')"><StatusTag :status="selected.actual_state || 'unknown'" /></el-descriptions-item>
          <el-descriptions-item :label="t('instances.deployment')">{{ selected.deployment_id || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.node')">{{ selected.node_id || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('common.createdAt')">{{ selected.created_at || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.startedAt')">{{ selected.started_at || '-' }}</el-descriptions-item>
        </el-descriptions>

        <h4>{{ t('instances.sectionRuntime') }}</h4>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="t('instances.container')">{{ selected.container_id || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.endpoint')">{{ selected.endpoint_url || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('instances.port')">{{ selected.host_port || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('deployments.containerPort')">{{ selected.container_port || '-' }}</el-descriptions-item>
        </el-descriptions>

        <h4>{{ t('instances.sectionDiagnostics') }}</h4>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item :label="t('dockerLogs.taskId')">{{ selected.current_run_plan_id || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('deployments.recentError')">{{ selected.last_error || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="t('runnerConfigs.advancedJson')">
            <pre class="raw-response">{{ JSON.stringify(selected, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-dialog>

    <el-drawer v-model="logsVisible" :title="t('dockerLogs.title')" size="60%" @closed="stopLogsTimer">
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
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { CopyDocument, Document, RefreshRight } from '@element-plus/icons-vue'
import { apiClient } from '@/api/client'
import StatusTag from '@/components/StatusTag.vue'
import { formatTestFailure, recommendedTestMode, testModeLabel } from '@/utils/modelCapabilities.js'

const { t, locale } = useI18n()

const loading = ref(false)
const items = ref<any[]>([])
const deployments = ref<any[]>([])
const models = ref<any[]>([])
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
let logsTimer: ReturnType<typeof setInterval> | null = null
const LOGS_REFRESH_INTERVAL_MS = 3000

// Model smoke test + instance actions
const testing = ref(false)
const testVisible = ref(false)
const testRow = ref<any>(null)
const testResult = ref<any>(null)
const stopping = ref(false)
const stoppingId = ref('')
const showStopped = ref(false)
const testMode = ref('auto')

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
  const formatted = formatTestFailure(testResult.value)
  if (formatted) return formatted
  const code = testResult.value.reason_code || ''
  const key = testReasonI18n[code]
  if (key) return t(key, { code, message: testResult.value.message || '' })
  // Check prefix match for http_xxx
  if (code.startsWith('http_')) return t('instances.testReasonHttpError', { code, message: testResult.value.message || '' })
  return testResult.value.message || code
})

const visibleItems = computed(() => items.value.filter((it) => showStopped.value || it.actual_state !== 'stopped'))

onMounted(async () => {
  await refresh()
})

onUnmounted(() => {
  stopLogsTimer()
})

async function refresh() {
  loading.value = true
  try {
    items.value = await apiClient.get('/model-instances')
    try { deployments.value = await apiClient.get('/deployments') } catch { deployments.value = [] }
    try { models.value = await apiClient.get('/model-artifacts') } catch { models.value = [] }
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

function modelForInstance(row: any): any {
  const dep = deployments.value.find((d: any) => d.id === row.deployment_id)
  const model = models.value.find((m: any) => m.id === dep?.model_artifact_id)
  return model || { name: row.model || row.deployment_id || '' }
}

function testModeText(mode: string): string {
  return testModeLabel(mode || 'auto', locale.value)
}

function stopLogsTimer() {
  if (logsTimer !== null) { clearInterval(logsTimer); logsTimer = null }
}

function startLogsTimer() {
  stopLogsTimer()
  // Only auto-refresh for running/starting/waiting instances.
  const state = logsRow.value?.actual_state
  if (state === 'stopped' || state === 'failed') return
  logsTimer = setInterval(() => {
    // Guard against concurrent requests.
    if (logsLoading.value) return
    loadLogs()
  }, LOGS_REFRESH_INTERVAL_MS)
}

async function openLogs(row: any) {
  stopLogsTimer()
  logsRow.value = row
  logsVisible.value = true
  await loadLogs()
  startLogsTimer()
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
    // Don't overwrite existing logs on transient refresh errors.
    if (!logsText.value) {
      logsMeta.value = null
      logsText.value = ''
    }
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
  testMode.value = recommendedTestMode(modelForInstance(row))
  testVisible.value = true
  await runSelectedTest()
}

async function runSelectedTest() {
  if (!testRow.value) return
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await apiClient.post(`/model-instances/${testRow.value.id}/test`, { mode: testMode.value, prompt: 'Reply with exactly one word: pong', timeout_seconds: 30 })
  } catch (e: any) {
    testResult.value = { ok: false, mode: testMode.value, reason_code: 'network_error', message: e?.message || t('common.requestFailed') }
  } finally {
    testing.value = false
  }
}

	async function doStop(row: any) {
	  stopping.value = true
	  stoppingId.value = row.id
	  try {
	    // Stop via deployment endpoint (no dedicated instance stop route).
	    await apiClient.post(`/deployments/${row.deployment_id}/stop`)
	    ElMessage.success(t('instances.stopped'))
	    await refresh()
	  } catch (e: any) {
	    ElMessage.error(e?.message || t('common.requestFailed'))
	  } finally {
	    stopping.value = false
	    stoppingId.value = ''
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
