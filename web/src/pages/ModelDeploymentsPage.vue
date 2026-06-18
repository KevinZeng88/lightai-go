<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('deployments.title') }}</h2>
      <div>
        <el-button type="primary" @click="startWizard">{{ $t('startWizard.title') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column prop="name" :label="$t('deployments.name')" width="150" />
      <el-table-column prop="status" :label="$t('deployments.status')" width="100" />
      <el-table-column prop="model_artifact_id" :label="$t('deployments.artifact')" width="200" />
      <el-table-column prop="backend_runtime_id" :label="$t('deployments.runtime')" width="200" />
      <el-table-column :label="$t('common.actions')" width="320">
        <template #default="{ row }">
          <el-button size="small" @click="doDryRun(row)">{{ $t('deployments.dryRun') }}</el-button>
          <el-button size="small" type="success" @click="doStart(row)">{{ $t('deployments.start') }}</el-button>
          <el-button size="small" type="warning" @click="doStop(row)">{{ $t('deployments.stop') }}</el-button>
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
      </el-form>
      <template #footer><el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doCreate" :loading="saving">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="dryRunVisible" title="Dry Run Result" width="700px">
      <pre v-if="dryRunResult" style="white-space:pre-wrap;max-height:400px;overflow:auto">{{ JSON.stringify(dryRunResult, null, 2) }}</pre>
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
        <el-select v-model="wizardModelId" :placeholder="$t('startWizard.selectModel')" style="width:100%" filterable>
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
        <el-select v-model="wizardVersionId" :placeholder="$t('startWizard.selectVersion')" style="width:100%" filterable @change="wizardRuntimeId=''">
          <el-option v-for="v in versions" :key="v.id" :label="v.display_name || v.version" :value="v.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardVersionId" @click="wizardStep=3">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 3">
        <el-select v-model="wizardRuntimeId" :placeholder="$t('startWizard.selectRuntime')" style="width:100%" filterable>
          <el-option v-for="r in filteredRuntimes" :key="r.id" :label="`${r.name} (${r.vendor})`" :value="r.id" />
        </el-select>
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
          <el-alert v-for="e in preflightResult.errors" :key="e" type="error" :title="e" show-icon :closable="false" />
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
        </el-form>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=4">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" @click="doWizardStart" :loading="wizardStarting">{{ $t('startWizard.start') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="runPlanVisible" title="RunPlan / Docker Preview" width="700px">
      <pre v-if="runPlanData" style="white-space:pre-wrap;max-height:400px;overflow:auto">{{ runPlanData }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'

const loading = ref(false); const saving = ref(false)
const items = ref<any[]>([]); const models = ref<any[]>([]); const runtimes = ref<any[]>([]); const backends = ref<any[]>([]); const versions = ref<any[]>([])
const createVisible = ref(false); const dryRunVisible = ref(false); const runPlanVisible = ref(false)
const dryRunResult = ref<any>(null); const runPlanData = ref('')
const createForm = ref({ name: '', model_artifact_id: '', backend_runtime_id: '', node_id: '', gpu_ids: '[]', host_port: 8000, placement_json: '{}', service_json: '{}', parameters_json: '{}', env_overrides_json: '{}' })

// Wizard state
const wizardVisible = ref(false); const wizardStep = ref(0)
const wizardModelId = ref(''); const wizardBackendId = ref(''); const wizardVersionId = ref(''); const wizardRuntimeId = ref('')
const wizardStartNode = ref(''); const wizardHostPort = ref(8004)
const preflightLoading = ref(false); const preflightResult = ref<any>(null)
const wizardStarting = ref(false)

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
    createForm.value.service_json = JSON.stringify({ host_port: createForm.value.host_port })
    await apiClient.post('/deployments', createForm.value)
    ElMessage.success('Created'); createVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  saving.value = false
}

async function doDryRun(row: any) {
  try { dryRunResult.value = await apiClient.post(`/deployments/${row.id}/dry-run`, {}) } catch (e: any) {}
  dryRunVisible.value = true
}
async function doStart(row: any) {
  try { const res = await apiClient.post(`/deployments/${row.id}/start`, {}); runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2); runPlanVisible.value = true; ElMessage.success('Started'); await refresh() } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
}
async function doStop(row: any) {
  try { await apiClient.post(`/deployments/${row.id}/stop`, {}); ElMessage.success('Stopped'); await refresh() } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
}
async function handleDelete(row: any) {
  try { await ElMessageBox.confirm(`Delete ${row.name}?`, 'Confirm', { type: 'warning' }); await apiClient.delete(`/deployments/${row.id}`); ElMessage.success('Deleted'); await refresh() } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || 'Failed') }
}

// ---- Start Wizard ----
const filteredRuntimes = computed(() => runtimes.value.filter((r) => !wizardVersionId.value || r.backend_version_id === wizardVersionId.value))

async function onBackendSelected() {
  wizardVersionId.value = ''
  wizardRuntimeId.value = ''
  try { versions.value = await apiClient.get(`/backends/${wizardBackendId.value}/versions`) } catch { versions.value = [] }
}

function startWizard() { wizardVisible.value = true; wizardStep.value = 0; wizardModelId.value = ''; wizardBackendId.value = ''; wizardVersionId.value = ''; wizardRuntimeId.value = ''; versions.value = []; preflightResult.value = null; wizardStartNode.value = ''; loadRefs() }
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
    // Create deployment then start
    const name = `wizard-${Date.now()}`
    const deploy = await apiClient.post('/deployments', {
      name, model_artifact_id: wizardModelId.value, backend_runtime_id: wizardRuntimeId.value,
      placement_json: { node_id: wizardStartNode.value, gpu_ids: [] },
      service_json: { host_port: wizardHostPort.value },
      parameters_json: {}, env_overrides_json: {},
    })
    const res = await apiClient.post(`/deployments/${deploy.id}/start`, {})
    runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2)
    runPlanVisible.value = true; wizardVisible.value = false
    ElMessage.success('Started'); await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  wizardStarting.value = false
}
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
</style>
