<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('deployments.title') }}</h2>
      <div>
        <el-button type="primary" @click="startWizard">{{ $t('startWizard.title') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column :label="$t('deployments.name')" width="180">
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('deployments.status')" width="120">
        <template #default="{ row }">
          <el-tag :type="deploymentStatusType(row.status)" size="small">{{ deploymentStatusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="model_artifact_id" :label="$t('deployments.artifact')" width="200" />
      <el-table-column prop="backend_runtime_id" :label="$t('deployments.runtime')" width="200" />
      <el-table-column :label="$t('common.actions')" width="320">
        <template #default="{ row }">
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" @click="doDryRun(row)">{{ $t('deployments.viewRunPlan') }}</el-button>
          <el-button size="small" type="success" :disabled="isRunBlocked(row.status)" @click="doStart(row)">{{ $t('deployments.runExisting') }}</el-button>
          <el-button size="small" type="warning" @click="doStop(row)">{{ $t('deployments.stop') }}</el-button>
          <el-button size="small" @click="doRestart(row)">{{ $t('deployments.restart') }}</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create Dialog -->
    <el-dialog v-model="createVisible" :title="$t('common.create')" width="500px">
      <el-form :model="createForm" label-width="140px">
        <el-form-item :label="$t('deployments.name')"><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item :label="$t('deployments.artifact')"><el-input v-model="createForm.model_artifact_id" /></el-form-item>
        <el-form-item :label="$t('deployments.runtime')"><el-input v-model="createForm.backend_runtime_id" /></el-form-item>
        <el-form-item :label="$t('deployments.nodeId')"><el-input v-model="createForm.node_id" /></el-form-item>
        <el-form-item :label="$t('deployments.hostPort')"><el-input v-model.number="createForm.host_port" /></el-form-item>
        <el-form-item :label="$t('deployments.containerPort')"><el-input v-model.number="createForm.container_port" /></el-form-item>
        <el-form-item :label="$t('deployments.appPort')"><el-input v-model.number="createForm.app_port" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doCreate" :loading="saving">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="dryRunVisible" :title="$t('common.dryRunTitle')" width="700px">
      <pre v-if="dryRunResult" style="white-space:pre-wrap;max-height:400px;overflow:auto">{{ JSON.stringify(dryRunResult, null, 2) }}</pre>
    </el-dialog>

    <el-dialog v-model="editVisible" :title="$t('deployments.editDeployment')" width="600px">
      <el-form :model="editForm" label-width="160px">
        <el-form-item :label="$t('deployments.name')">
          <span>{{ editForm.original_name }}</span>
          <el-tag size="small" type="info" style="margin-left:8px">{{ $t('common.readonly') }}</el-tag>
        </el-form-item>
        <el-form-item :label="$t('deployments.displayName')">
          <el-input v-model="editForm.display_name" />
        </el-form-item>
        <el-form-item :label="$t('deployments.artifact')">
          <el-select v-model="editForm.model_artifact_id" filterable style="width:100%">
            <el-option v-for="m in models" :key="m.id" :label="`${m.display_name || m.name}`" :value="m.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.runtime')">
          <el-select v-model="editForm.backend_runtime_id" filterable style="width:100%">
            <el-option v-for="r in runtimes" :key="r.id" :label="`${r.display_name || r.name} (${r.vendor})`" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.hostPort')">
          <el-input v-model.number="editForm.host_port" />
        </el-form-item>
        <el-form-item :label="$t('deployments.containerPort')">
          <el-input v-model.number="editForm.container_port" />
        </el-form-item>
        <el-form-item :label="$t('deployments.appPort')">
          <el-input v-model.number="editForm.app_port" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display:flex;justify-content:space-between;width:100%">
          <div>
            <el-button v-if="editForm.source_template_name" @click="doTemplateSyncPreview" size="small">{{ $t('deployments.previewTemplateDiff') }}</el-button>
            <el-button v-if="editForm.source_template_name" @click="doTemplateSyncApply" size="small" type="warning">{{ $t('deployments.applyTemplateChanges') }}</el-button>
          </div>
          <div>
            <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="doEdit" :loading="saving">{{ $t('common.save') }}</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="syncPreviewVisible" :title="$t('deployments.templateSyncPreview')" width="700px">
      <div v-if="syncPreviewData">
        <el-alert v-if="syncPreviewData.template_changed" type="warning" :closable="false" style="margin-bottom:12px">
          {{ $t('deployments.templateChanged') }}: {{ syncPreviewData.source_template_name }}
        </el-alert>
        <el-alert v-else type="success" :closable="false" style="margin-bottom:12px">
          {{ $t('deployments.templateUnchanged') }}
        </el-alert>
        <el-table v-if="syncPreviewData.diffs?.length" :data="syncPreviewData.diffs" stripe size="small">
          <el-table-column prop="field" :label="$t('deployments.syncField')" width="200" />
          <el-table-column :label="$t('deployments.syncDeployValue')">
            <template #default="{ row }"><code>{{ JSON.stringify(row.deploy_value) }}</code></template>
          </el-table-column>
          <el-table-column :label="$t('deployments.syncTemplateValue')">
            <template #default="{ row }"><code>{{ JSON.stringify(row.template_value) }}</code></template>
          </el-table-column>
        </el-table>
        <div v-if="!syncPreviewData.diffs?.length" style="padding:12px;color:var(--el-text-color-secondary)">
          {{ $t('deployments.noDiffs') }}
        </div>
      </div>
      <template #footer>
        <el-button @click="syncPreviewVisible = false">{{ $t('common.cancel') }}</el-button>
      </template>
    </el-dialog>

    <!-- Start Wizard -->
    <el-dialog v-model="wizardVisible" :title="$t('startWizard.title')" width="800px" :close-on-click-modal="false">
      <el-steps :active="wizardStep" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('startWizard.selectModel')" />
        <el-step :title="$t('startWizard.selectBackend')" />
        <el-step :title="$t('startWizard.selectVersion')" />
        <el-step :title="$t('startWizard.selectRuntime')" />
        <el-step :title="$t('startWizard.preflight')" />
        <el-step :title="$t('startWizard.start')" />
      </el-steps>

      <div v-if="wizardStep === 0">
        <el-select v-model="wizardModelId" :placeholder="$t('startWizard.selectModel')" style="width:100%" filterable @change="onWizAutoNext">
          <el-option v-for="m in models" :key="m.id" :label="`${m.name} (${m.format})`" :value="m.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizardModelId" @click="wizardStep=1">{{ $t('common.next') }}</el-button></div>
      </div>

      <div v-if="wizardStep === 1">
        <el-select v-model="wizardBackendId" :placeholder="$t('startWizard.selectBackend')" style="width:100%" filterable @change="onBackendSelected">
          <el-option v-for="b in backends" :key="b.id" :label="b.display_name || b.name" :value="b.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardBackendId" @click="wizardStep=2">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 2">
        <el-select v-model="wizardVersionId" :placeholder="$t('startWizard.selectVersion')" style="width:100%" filterable @change="wizardRuntimeId=''; if($event) { onVersionSelected($event) }">
          <el-option v-for="v in versions" :key="v.id" :label="v.display_name || v.version" :value="v.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardVersionId" @click="wizardStep=3">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 3">
        <el-select v-model="wizardRuntimeId" :placeholder="$t('startWizard.selectRuntime')" style="width:100%" filterable @change="$event && doPreflight()">
          <el-option v-for="r in filteredRuntimes" :key="r.id" :label="`${r.display_name || r.name} (${r.vendor})${r.is_editable ? '' : ' [' + $t('startWizard.systemBuiltin') + ']'}`" :value="r.id" />
        </el-select>
        <el-alert v-if="wizardVersionId && filteredRuntimes.length === 0" type="info" :closable="false" style="margin-top:8px">
          {{ $t('startWizard.noRuntimeForVersion') }}
        </el-alert>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=2">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardRuntimeId" @click="doPreflight">{{ $t('startWizard.preflight') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 4" v-loading="preflightLoading">
        <el-alert v-if="preflightResult" :type="preflightResult.can_run ? 'success' : 'warning'" :closable="false">
          {{ preflightResult.can_run ? $t('preflight.canRun') : $t('preflight.noNodes') }}
        </el-alert>
        <el-table v-if="preflightResult?.candidate_nodes?.length" :data="preflightResult.candidate_nodes" stripe size="small" style="margin-top:8px" @row-click="onPreflightNodeClick" highlight-current-row>
          <el-table-column :label="$t('modelLocations.node')">
            <template #default="{ row }">{{ nodeLabel(row.node_id) }}</template>
          </el-table-column>
          <el-table-column prop="status" :label="$t('preflight.canRun')" width="80" />
        </el-table>
        <div v-if="preflightResult?.errors?.length" style="margin-top:8px">
          <el-alert v-for="(e, idx) in preflightResult.errors" :key="idx" type="error" :closable="false">
            <template #title>
              {{ preflightErrorText(e) }}
            </template>
            <template v-if="e.context" #default>
              <div style="font-size:12px;color:var(--el-color-info);margin-top:4px">
                <span v-if="e.context.node_id">node: {{ e.context.node_id }}</span>
                <span v-if="e.context.artifact_id"> | artifact: {{ e.context.artifact_id }}</span>
                <span v-if="e.context.runtime_id"> | runtime: {{ e.context.runtime_id }}</span>
              </div>
            </template>
          </el-alert>
        </div>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=3">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!preflightResult?.can_run" @click="wizardStep=5">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 5">
        <el-form label-width="120px">
          <el-form-item :label="$t('modelLocations.node')"><el-input v-model="wizardStartNode" disabled /></el-form-item>
          <el-form-item :label="$t('deployments.hostPort')"><el-input v-model.number="wizardHostPort" /></el-form-item>
          <el-form-item :label="$t('deployments.containerPort')"><el-input v-model.number="wizardContainerPort" /></el-form-item>
          <el-form-item :label="$t('deployments.appPort')"><el-input v-model.number="wizardAppPort" /></el-form-item>
        </el-form>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=4">{{ $t('common.prev') }}</el-button>
          <el-button @click="doWizardPreview" :loading="wizardStarting">{{ $t('deployments.previewRunPlan') }}</el-button>
          <el-button @click="doWizardSave" :loading="wizardStarting">{{ $t('deployments.saveConfig') }}</el-button>
          <el-button type="primary" @click="doWizardStart" :loading="wizardStarting">{{ $t('deployments.saveAndRun') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="runPlanVisible" :title="$t('common.runPlanTitle')" width="700px">
      <pre v-if="runPlanData" style="white-space:pre-wrap;max-height:400px;overflow:auto">{{ runPlanData }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import { useWizardAutoAdvance } from '@/composables/useWizardAutoAdvance'

const { t } = useI18n()

const loading = ref(false); const saving = ref(false)
const items = ref<any[]>([]); const models = ref<any[]>([]); const runtimes = ref<any[]>([]); const backends = ref<any[]>([]); const versions = ref<any[]>([])
const createVisible = ref(false); const dryRunVisible = ref(false); const runPlanVisible = ref(false)
const dryRunResult = ref<any>(null); const runPlanData = ref('')
const editVisible = ref(false); const selectedEditRow = ref<any>(null)
const editForm = ref({ display_name: '', model_artifact_id: '', backend_runtime_id: '', host_port: 8000, container_port: 0, app_port: 0, original_name: '', source_template_name: '', source_backend_runtime_id: '', copied_at: '' })
const createForm = ref({ name: '', model_artifact_id: '', backend_runtime_id: '', node_id: '', gpu_ids: '[]', host_port: 8000, container_port: 0, app_port: 0, placement_json: '{}', service_json: '{}', parameters_json: '{}', env_overrides_json: '{}' })

// Wizard state
const wizardVisible = ref(false); const wizardStep = ref(0)
const wizardModelId = ref(''); const wizardBackendId = ref(''); const wizardVersionId = ref(''); const wizardRuntimeId = ref('')
const wizardStartNode = ref(''); const wizardHostPort = ref(8004); const wizardContainerPort = ref(0); const wizardAppPort = ref(0); const wizardDeploymentId = ref('')
const preflightLoading = ref(false); const preflightResult = ref<any>(null)
const wizardStarting = ref(false)

const { onSelectAutoNext: onWizAutoNext } = useWizardAutoAdvance(wizardStep, () => { wizardStep.value++ })

onMounted(async () => { await refresh(); await loadRefs() })
async function refresh() { loading.value = true; try { items.value = await apiClient.get('/deployments') } catch (e: any) {} loading.value = false }
const { loadNodes, nodeLabel } = useNodeLabels()

async function loadRefs() {
  try { models.value = await apiClient.get('/model-artifacts') } catch { models.value = [] }
  try { runtimes.value = await apiClient.get('/backend-runtimes') } catch { runtimes.value = [] }
  try { backends.value = await apiClient.get('/backends') } catch { backends.value = [] }
  loadNodes()
}

function showCreate() { createVisible.value = true }
async function doCreate() {
  saving.value = true
  try {
    createForm.value.placement_json = JSON.stringify({ node_id: createForm.value.node_id, gpu_ids: JSON.parse(createForm.value.gpu_ids || '[]') })
    createForm.value.service_json = JSON.stringify(servicePayload(createForm.value.host_port, createForm.value.container_port, createForm.value.app_port))
    await apiClient.post('/deployments', createForm.value)
    ElMessage.success(t('deployments.created')); createVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function doDryRun(row: any) {
  try { dryRunResult.value = await apiClient.post(`/deployments/${row.id}/dry-run`, {}) } catch (e: any) {}
  dryRunVisible.value = true
}
async function doStart(row: any) {
  try { const res = await apiClient.post(`/deployments/${row.id}/start`, {}); runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2); runPlanVisible.value = true; ElMessage.success(t('deployments.started')); await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function doStop(row: any) {
  try { await apiClient.post(`/deployments/${row.id}/stop`, {}); ElMessage.success(t('deployments.stopped')); await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function doRestart(row: any) {
  try { await apiClient.post(`/deployments/${row.id}/stop`, {}); const res = await apiClient.post(`/deployments/${row.id}/start`, {}); runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2); runPlanVisible.value = true; ElMessage.success(t('deployments.restarted')); await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function handleDelete(row: any) {
  try { await ElMessageBox.confirm(t('deployments.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' }); await apiClient.delete(`/deployments/${row.id}`); ElMessage.success(t('deployments.deleted')); await refresh() } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}

// ---- Deployment Edit ----
function showEdit(row: any) {
  selectedEditRow.value = row
  editForm.value.original_name = row.name || row.display_name || ''
  editForm.value.display_name = row.display_name || ''
  editForm.value.model_artifact_id = row.model_artifact_id || ''
  editForm.value.backend_runtime_id = row.backend_runtime_id || ''
  editForm.value.source_template_name = row.source_template_name || ''
  editForm.value.source_backend_runtime_id = row.source_backend_runtime_id || ''
  editForm.value.copied_at = row.copied_at || ''
  try {
    const svc = typeof row.service_json === 'string' ? JSON.parse(row.service_json) : (row.service_json || {})
    editForm.value.host_port = svc.host_port || 8000
    editForm.value.container_port = svc.container_port || 0
    editForm.value.app_port = svc.app_port || 0
  } catch {
    editForm.value.host_port = 8000
    editForm.value.container_port = 0
    editForm.value.app_port = 0
  }
  editVisible.value = true
}

async function doEdit() {
  if (!selectedEditRow.value) return
  saving.value = true
  try {
    const payload: any = {
      display_name: editForm.value.display_name,
      model_artifact_id: editForm.value.model_artifact_id,
      backend_runtime_id: editForm.value.backend_runtime_id,
      service_json: servicePayload(editForm.value.host_port, editForm.value.container_port, editForm.value.app_port),
    }
    await apiClient.patch(`/deployments/${selectedEditRow.value.id}`, payload)
    ElMessage.success(t('deployments.saved'))
    editVisible.value = false
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

// ---- Start Wizard ----
const filteredRuntimes = computed(() => runtimes.value.filter((r) => !wizardVersionId.value || r.backend_version_id === wizardVersionId.value))

// Map preflight error code to i18n-keyed user-facing text.
function preflightErrorText(e: any): string {
  if (!e || typeof e !== 'object') return String(e)
  // Structured error with code
  if (e.code) {
    const codeMap: Record<string, string> = {
      model_location_missing: 'preflight.reason.modelLocationMissing',
      node_backend_runtime_not_ready: 'preflight.reason.nbrNotReady',
      node_offline: 'preflight.reason.nodeOffline',
      backend_version_mismatch: 'preflight.reason.backendVersionMismatch',
      docker_image_missing: 'preflight.reason.dockerImageMissing',
      runtime_disabled: 'preflight.reason.runtimeDisabled',
    }
    const i18nKey = codeMap[e.code]
    if (i18nKey) return t(i18nKey)
    // Fallback: show the message but prefixed with code
    return `[${e.code}] ${e.message || ''}`
  }
  // Legacy string error (backward compat)
  return typeof e === 'string' ? e : (e.message || JSON.stringify(e))
}

async function onBackendSelected() {
  wizardVersionId.value = ''
  wizardRuntimeId.value = ''
  try { versions.value = await apiClient.get(`/backends/${wizardBackendId.value}/versions`) } catch { versions.value = [] }
  if (wizardBackendId.value) wizardStep.value = 2
}

async function onVersionSelected(versionId: string) {
  if (!versionId) return
  try {
    const v = versions.value.find((ver: any) => ver.id === versionId)
    if (v?.default_container_port && v.default_container_port > 0) {
      wizardContainerPort.value = v.default_container_port
      if (wizardAppPort.value === 0) wizardAppPort.value = v.default_container_port
    }
  } catch { /* keep defaults */ }
  wizardStep.value = 3
}

function startWizard() { wizardVisible.value = true; wizardStep.value = 0; wizardModelId.value = ''; wizardBackendId.value = ''; wizardVersionId.value = ''; wizardRuntimeId.value = ''; versions.value = []; preflightResult.value = null; wizardStartNode.value = ''; wizardDeploymentId.value = ''; wizardContainerPort.value = 0; wizardAppPort.value = 0; loadRefs() }
async function doPreflight() {
  preflightLoading.value = true; wizardStep.value = 4
  try {
    preflightResult.value = await apiClient.post('/deployments/preflight', { model_artifact_id: wizardModelId.value, backend_runtime_id: wizardRuntimeId.value, host_port: wizardHostPort.value })
    if (preflightResult.value?.candidate_nodes?.length) wizardStartNode.value = preflightResult.value.candidate_nodes[0].node_id
  } catch (e: any) { preflightResult.value = { can_run: false, errors: [e?.message || 'preflight failed'], candidate_nodes: [] } }
  preflightLoading.value = false
}
function onPreflightNodeClick(row: any) { wizardStartNode.value = row.node_id }
async function doWizardStart() {
  wizardStarting.value = true
  try {
    const deploy = await ensureWizardDeployment()
    const res = await apiClient.post(`/deployments/${deploy.id}/start`, {})
    runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2)
    runPlanVisible.value = true; wizardVisible.value = false
    ElMessage.success(t('deployments.started')); await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardStarting.value = false
}
async function doWizardSave() {
  wizardStarting.value = true
  try { await ensureWizardDeployment(); ElMessage.success(t('deployments.saved')); wizardVisible.value = false; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardStarting.value = false
}
async function doWizardPreview() {
  wizardStarting.value = true
  try { const deploy = await ensureWizardDeployment(); dryRunResult.value = await apiClient.post(`/deployments/${deploy.id}/dry-run`, {}); dryRunVisible.value = true; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardStarting.value = false
}
async function ensureWizardDeployment() {
  if (wizardDeploymentId.value) return { id: wizardDeploymentId.value }
  const name = `wizard-${Date.now()}`
  const deploy = await apiClient.post('/deployments', {
    name, display_name: name, model_artifact_id: wizardModelId.value, backend_runtime_id: wizardRuntimeId.value,
    placement_json: { node_id: wizardStartNode.value, gpu_ids: [] },
    service_json: servicePayload(wizardHostPort.value, wizardContainerPort.value, wizardAppPort.value),
    parameters_json: {}, env_overrides_json: {},
  })
  wizardDeploymentId.value = deploy.id
  return deploy
}
function servicePayload(hostPort: number, containerPort?: number, appPort?: number) {
  const payload: any = { host_port: hostPort }
  if (containerPort && containerPort > 0) payload.container_port = containerPort
  if (appPort && appPort > 0) payload.app_port = appPort
  payload.health_port = hostPort
  payload.api_test_port = hostPort
  return payload
}
// ---- Template Sync ----
const syncPreviewVisible = ref(false)
const syncPreviewData = ref<any>(null)

async function doTemplateSyncPreview() {
  if (!selectedEditRow.value) return
  try {
    syncPreviewData.value = await apiClient.post(`/deployments/${selectedEditRow.value.id}/template-sync/preview`, {})
    syncPreviewVisible.value = true
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}

async function doTemplateSyncApply() {
  if (!selectedEditRow.value) return
  try {
    await ElMessageBox.confirm(
      t('deployments.syncConfirm'),
      t('deployments.applyTemplateChanges'),
      { type: 'warning', confirmButtonText: t('common.yes'), cancelButtonText: t('common.no') }
    )
    const res = await apiClient.post(`/deployments/${selectedEditRow.value.id}/template-sync/apply`, { strategy: 'preserve_overrides' })
    ElMessage.success(t('deployments.synced'))
    editVisible.value = false
    await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}

function isRunBlocked(status: string) { return ['starting', 'pending', 'provisioning', 'running', 'healthy', 'stopping'].includes(status) }
function deploymentStatusType(status: string) { if (['running', 'healthy'].includes(status)) return 'success'; if (['failed'].includes(status)) return 'danger'; if (['starting', 'pending', 'provisioning', 'stopping'].includes(status)) return 'warning'; return 'info' }
function deploymentStatusText(status: string) { return t(`deployments.status_${status || 'unknown'}`) }
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
</style>
