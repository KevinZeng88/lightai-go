<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runnerConfigs.title') }}</h2>
      <el-button type="primary" @click="startWizard">{{ $t('runnerConfigs.newConfig') }}</el-button>
    </div>

    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column prop="name" :label="$t('runnerConfigs.name')" min-width="160" />
      <el-table-column :label="$t('modelLocations.node')" width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.node_label || row.node_id }}</template>
      </el-table-column>
      <el-table-column :label="$t('runnerConfigs.runnerType')" width="100">
        <template #default="{ row }">{{ row.runner_type === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : (row.runner_type || '-') }}</template>
      </el-table-column>
      <el-table-column :label="$t('nodeRuntime.status')" width="100">
        <template #default="{ row }"><el-tag :type="row.status==='ready'?'success':'warning'" size="small">{{ translateStatus(row.status, t) }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="image_ref" :label="$t('nodeRuntime.imageRef')" min-width="220" show-overflow-tooltip />
      <el-table-column prop="last_checked_at" :label="$t('nodeRuntime.lastChecked')" width="180" show-overflow-tooltip />
      <el-table-column :label="$t('common.actions')" width="310">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" type="warning" @click="checkRow(row)">{{ $t('runnerConfigs.check') }}</el-button>
          <el-button size="small" type="danger" @click="doDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Wizard dialog -->
    <el-dialog v-model="wizardVisible" :title="$t('runnerConfigs.wizardTitle')" width="800px" :close-on-click-modal="false">
      <el-steps :active="step" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('runnerConfigs.selectRunnerType')" />
        <el-step :title="$t('runnerConfigs.selectTemplate')" />
        <el-step :title="$t('runnerConfigs.selectNode')" />
        <el-step :title="$t('runnerConfigs.selectImage')" />
        <el-step :title="$t('runnerConfigs.create')" />
      </el-steps>

      <div v-if="step===0">
        <el-form label-width="140px" style="margin-bottom:12px">
          <el-form-item :label="$t('runnerConfigs.configName')"><el-input v-model="wizConfigName" /></el-form-item>
        </el-form>
        <el-select v-model="wizRunnerType" :placeholder="$t('runnerConfigs.selectRunnerType')" style="width:100%" @change="onWizAutoNext">
          <el-option label="Docker" value="docker" />
        </el-select>
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizRunnerType" @click="step=1">{{ $t('common.next') }}</el-button></div>
      </div>

      <div v-if="step===1">
        <el-select v-model="wizTemplateId" :placeholder="$t('runnerConfigs.selectTemplate')" style="width:100%" filterable @change="onWizTemplateSelected">
          <el-option v-for="t in templates" :key="t.id" :label="`${t.name} (${t.vendor})`" :value="t.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizTemplateId" @click="step=2">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===2">
        <el-select v-model="wizNodeId" :placeholder="$t('runnerConfigs.selectNode')" style="width:100%" filterable @change="onWizAutoNext">
          <el-option v-for="n in nodeItems" :key="n.id" :label="n.label" :value="n.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizNodeId" @click="step=wizRunnerType==='docker'?3:4">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===3 && wizRunnerType==='docker'">
        <DockerImagePicker v-if="wizNodeId" :node-id="wizNodeId" @select="onWizardImageSelected" />
        <el-form label-width="130px" style="margin-top:12px">
          <el-form-item :label="$t('dockerImages.selectedImage')"><el-input v-model="wizImageRef" @input="wizImagePresent = false" /></el-form-item>
        </el-form>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=2">{{ $t('common.prev') }}</el-button>
          <span v-if="wizImageRef" class="next-summary">{{ wizImageRef }}</span>
          <el-button type="primary" :disabled="!wizImageRef" @click="step=4">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===4">
        <el-form label-width="120px">
          <el-form-item :label="$t('runnerConfigs.template')"><span>{{ wizTemplateId }}</span></el-form-item>
          <el-form-item :label="$t('runnerConfigs.configName')"><el-input v-model="wizConfigName" /></el-form-item>
          <el-form-item :label="$t('runnerConfigs.runnerType')"><span>{{ wizRunnerType === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : wizRunnerType }}</span></el-form-item>
          <el-form-item :label="$t('modelLocations.node')"><span>{{ wizNodeId }}</span></el-form-item>
          <el-form-item v-if="wizImageRef" :label="$t('runnerConfigs.selectImage')"><span>{{ wizImageRef }}</span></el-form-item>
        </el-form>
        <div v-if="wizCheckResult" style="margin-top:8px">
          <el-alert :type="wizCheckResult.status==='ready'||wizCheckResult.status==='ready_with_warnings'?'success':wizCheckResult.status==='missing_image'||wizCheckResult.status==='agent_unreachable'||wizCheckResult.status==='inspect_failed'||wizCheckResult.status==='runtime_image_mismatch'||wizCheckResult.status==='docker_error'?'error':'warning'" :title="translateStatus(wizCheckResult.status, t)" :description="translateStatusReason(wizCheckResult.status_reason, t)" show-icon :closable="false" />
          <div v-if="wizCheckResult.probe_results" style="margin-top:8px">
            <el-collapse>
              <el-collapse-item title="Image Metadata" name="level2" v-if="wizCheckResult.probe_results.level2?.inspect_success">
                <el-descriptions :column="2" border size="small">
                  <el-descriptions-item label="Image ID">{{ (wizCheckResult.probe_results.level2?.image_id || '').slice(7,19) || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="Architecture">{{ wizCheckResult.probe_results.level2?.architecture || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="OS">{{ wizCheckResult.probe_results.level2?.os || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="Created">{{ (wizCheckResult.probe_results.level2?.created || '').slice(0,19) || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="Size">{{ formatBytes(wizCheckResult.probe_results.level2?.size_bytes) }}</el-descriptions-item>
                  <el-descriptions-item label="Entrypoint">{{ (wizCheckResult.probe_results.level2?.entrypoint || []).join(', ') || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="CMD">{{ (wizCheckResult.probe_results.level2?.cmd || []).join(', ') || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="Exposed Ports">{{ Object.keys(wizCheckResult.probe_results.level2?.exposed_ports || {}).join(', ') || '-' }}</el-descriptions-item>
                  <el-descriptions-item :span="2" label="RepoTags">{{ (wizCheckResult.probe_results.level2?.repotags || []).join(', ') || '-' }}</el-descriptions-item>
                </el-descriptions>
              </el-collapse-item>
              <el-collapse-item title="Backend Match" name="level3" v-if="wizCheckResult.probe_results.level3">
                <p>{{ wizCheckResult.probe_results.level3.match_detail || 'Not checked' }}</p>
              </el-collapse-item>
              <el-collapse-item title="Version Probe" name="level4" v-if="wizCheckResult.probe_results.level4">
                <p v-if="wizCheckResult.probe_results.level4.version_probed">Version: {{ wizCheckResult.probe_results.level4.version_string }}</p>
                <p v-else>Not probed: {{ wizCheckResult.probe_results.level4.probe_error || 'no probe data' }}</p>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=wizRunnerType==='docker'?3:2">{{ $t('common.prev') }}</el-button>
          <el-button @click="doCheck" :loading="checking">{{ $t('runnerConfigs.check') }}</el-button>
          <el-button type="primary" :disabled="!wizCheckResult || (wizCheckResult.status !== 'ready' && wizCheckResult.status !== 'ready_with_warnings')" @click="doCreateConfig" :loading="saving">{{ $t('runnerConfigs.create') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Detail drawer -->
    <el-drawer v-model="detailVisible" :title="$t('common.detail')" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runnerConfigs.name')">{{ selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelLocations.node')">{{ selected.node_label || selected.node_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.runnerType')">{{ selected.runner_type === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : (selected.runner_type || '-') }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.status')">
            <el-tag :type="selected.status==='ready'?'success':'warning'" size="small">{{ translateStatus(selected.status, t) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.imageRef')">{{ selected.image_ref || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.template')">{{ selected.template_name || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.statusReason')" :span="2">{{ translateStatusReason(selected.status_reason, t) }}</el-descriptions-item>
        </el-descriptions>
        <el-collapse v-if="selected?.probe_results_json && typeof selected.probe_results_json === 'object' && Object.keys(selected.probe_results_json).length > 0" style="margin-top:12px">
          <el-collapse-item title="Image Metadata" name="level2" v-if="selected.probe_results_json.level2?.inspect_success">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="Image ID">{{ (selected.probe_results_json.level2?.image_id || '').slice(7,19) || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Architecture">{{ selected.probe_results_json.level2?.architecture || '-' }}</el-descriptions-item>
              <el-descriptions-item label="OS">{{ selected.probe_results_json.level2?.os || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Created">{{ (selected.probe_results_json.level2?.created || '').slice(0,19) || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Size">{{ formatBytes(selected.probe_results_json.level2?.size_bytes) }}</el-descriptions-item>
              <el-descriptions-item label="Entrypoint">{{ (selected.probe_results_json.level2?.entrypoint || []).join(', ') || '-' }}</el-descriptions-item>
              <el-descriptions-item label="CMD">{{ (selected.probe_results_json.level2?.cmd || []).join(', ') || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Exposed Ports">{{ Object.keys(selected.probe_results_json.level2?.exposed_ports || {}).join(', ') || '-' }}</el-descriptions-item>
              <el-descriptions-item :span="2" label="RepoTags">{{ (selected.probe_results_json.level2?.repotags || []).join(', ') || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>
          <el-collapse-item title="Backend Match" name="level3" v-if="selected.probe_results_json.level3">
            <p>{{ selected.probe_results_json.level3.match_detail || 'Not checked' }}</p>
          </el-collapse-item>
          <el-collapse-item title="Version Probe" name="level4" v-if="selected.probe_results_json.level4">
            <p v-if="selected.probe_results_json.level4.version_probed">Version: {{ selected.probe_results_json.level4.version_string }}</p>
            <p v-else>Not probed: {{ selected.probe_results_json.level4.probe_error || 'no probe data' }}</p>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>

    <el-dialog v-model="editVisible" :title="$t('runnerConfigs.editConfig')" width="760px">
      <el-alert :title="$t('runnerConfigs.editAffectsNextStart')" type="warning" show-icon :closable="false" style="margin-bottom:12px" />
      <el-form label-position="top">
        <el-form-item :label="$t('runnerConfigs.configName')"><el-input v-model="editConfigName" /></el-form-item>
        <el-form-item :label="$t('nodeRuntime.imageRef')"><el-input v-model="editImageRef" /></el-form-item>
        <el-form-item :label="$t('runnerConfigs.snapshotJson')"><el-input v-model="editSnapshotText" type="textarea" :rows="10" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doEdit" :loading="saving">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import { listRuntimes } from '@/api/runtimes'
import DockerImagePicker from '@/components/DockerImagePicker.vue'
import { translateStatus, translateStatusReason } from '@/utils/status'
import { useWizardAutoAdvance } from '@/composables/useWizardAutoAdvance'
const { loadNodes, nodes: nodeItems, nodeLabel } = useNodeLabels()
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

const loading = ref(false); const saving = ref(false); const checking = ref(false)
const items = ref<any[]>([]); const templates = ref<any[]>([]); const selected = ref<any>(null); const detailVisible = ref(false)
const editVisible = ref(false); const editConfigName = ref(''); const editImageRef = ref(''); const editSnapshotText = ref('{}')

// Wizard
const wizardVisible = ref(false); const step = ref(0)
const wizTemplateId = ref(''); const wizRunnerType = ref('docker')
const wizNodeId = ref(''); const wizImageRef = ref(''); const wizImagePresent = ref(false)
const wizConfigName = ref(''); const wizCheckResult = ref<any>(null)

const { onSelectAutoNext: onWizAutoNext } = useWizardAutoAdvance(step, () => { step.value++ })

function formatBytes(bytes: any): string {
  if (bytes == null || bytes === 0) return '-'
  const n = Number(bytes)
  if (isNaN(n)) return '-'
  if (n < 1024) return n + ' B'
  if (n < 1048576) return (n / 1024).toFixed(1) + ' KB'
  if (n < 1073741824) return (n / 1048576).toFixed(1) + ' MB'
  return (n / 1073741824).toFixed(2) + ' GB'
}

onMounted(async () => { await loadRefs(); await refresh() })

async function refresh() {
  loading.value = true
  try {
    // Collect NodeBackendRuntime records from all nodes
    const nbrList: any[] = []
    for (const n of nodeItems.value) {
      try {
        const nbrs = await apiClient.get(`/nodes/${n.id}/backend-runtimes`)
        if (Array.isArray(nbrs)) {
          for (const nbr of nbrs) {
            nbrList.push({ ...nbr, _node_label: n.label, _node_id: n.id })
          }
        }
      } catch {}
    }
    items.value = nbrList.map((nbr: any) => ({
      id: nbr.id,
      name: nbr.display_name || nbr.name || nbr.backend_runtime?.display_name || nbr.backend_runtime?.name || nbr.backend_runtime_id,
      template_name: nbr.backend_runtime?.name || '-',
      runner_type: nbr.runner_type || 'docker',
      node_count: 1,
      ready_count: nbr.status === 'ready' ? 1 : 0,
      status: nbr.status,
      node_id: nbr._node_id,
        node_label: nbr._node_label,
        image_ref: nbr.image_ref,
        image_present: nbr.image_present,
        last_checked_at: nbr.last_checked_at,
        status_reason: nbr.status_reason,
        config_snapshot_json: nbr.config_snapshot_json || {},
        backend_runtime_id: nbr.backend_runtime_id,
    }))
  } catch {}
  loading.value = false
}

async function loadRefs() {
  try { templates.value = await listRuntimes() } catch { templates.value = [] }
  loadNodes()
}

function startWizard() { wizardVisible.value = true; step.value = 0; wizTemplateId.value = ''; wizRunnerType.value = 'docker'; wizNodeId.value = ''; wizImageRef.value = ''; wizImagePresent.value = false; wizConfigName.value = ''; wizCheckResult.value = null; loadRefs() }

function onWizTemplateSelected(templateId: string) {
  const template = templates.value.find((t: any) => t.id === templateId)
  if (!template) return
  // Only auto-generate name if user hasn't entered a custom one
  if (!wizConfigName.value || wizConfigName.value.trim() === '') {
    const suffix = t('runnerConfigs.customSuffix')
    const baseName = `${template.name}${suffix}`
    // Auto-append number if name conflicts with existing configs
    const existingNames = new Set(items.value.map((c: any) => c.name))
    let candidate = baseName
    let counter = 2
    while (existingNames.has(candidate)) {
      candidate = `${baseName} ${counter}`
      counter++
    }
    wizConfigName.value = candidate
  }
  // Auto-advance: this step has only one select control
  step.value = 2
}

async function doCheck() {
  checking.value = true
  try {
    // First enable (create/update NBR), then trigger server-side check.
    const nbr: any = await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/enable`, { backend_runtime_id: wizTemplateId.value, display_name: wizConfigName.value, image_ref: wizImageRef.value || '' })
    const nbrId = nbr?.id
    if (!nbrId) { wizCheckResult.value = { status: 'unknown', status_reason: 'failed to create node runtime config' }; checking.value = false; return }
    wizCheckResult.value = await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/${nbrId}/check-request`, {})
  } catch (e: any) { wizCheckResult.value = { status: 'unknown', status_reason: e?.message || 'check failed' } }
  checking.value = false
}

function onWizardImageSelected(img: any) {
  wizImageRef.value = img.image_ref || ''
  wizCheckResult.value = null
}

async function doCreateConfig() {
  saving.value = true
  try {
    // Enable the selected template on the selected node (creates NodeBackendRuntime only, no BackendRuntime clone)
    await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/enable`, { backend_runtime_id: wizTemplateId.value, display_name: wizConfigName.value, image_ref: wizImageRef.value })
    ElMessage.success(t('runnerConfigs.created')); wizardVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function showDetail(row: any) {
  selected.value = row
  detailVisible.value = true
}

function showEdit(row: any) {
  selected.value = row
  editConfigName.value = row.name || ''
  editImageRef.value = row.image_ref || ''
  editSnapshotText.value = JSON.stringify(row.config_snapshot_json || {}, null, 2)
  editVisible.value = true
}

async function doEdit() {
  if (!selected.value) return
  saving.value = true
  try {
    let snapshot: any = {}
    try { snapshot = JSON.parse(editSnapshotText.value || '{}') } catch { ElMessage.error(t('runnerConfigs.invalidJson')); saving.value = false; return }
    await apiClient.patch(`/nodes/${selected.value.node_id}/backend-runtimes/${selected.value.id}`, { display_name: editConfigName.value, image_ref: editImageRef.value, config_snapshot_json: snapshot })
    ElMessage.success(t('runnerConfigs.savedNeedsCheck'))
    editVisible.value = false
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function checkRow(row: any) {
  checking.value = true
  try {
    const result = await apiClient.post(`/nodes/${row.node_id}/backend-runtimes/${row.id}/check-request`, {})
    ElMessage.success(`${translateStatus(result.status, t)}: ${translateStatusReason(result.status_reason, t)}`)
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  checking.value = false
}

async function doDelete(row: any) {
  try {
    await ElMessageBox.confirm(t('runnerConfigs.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' })
    // Delete the NodeBackendRuntime record (node-level config only; template is preserved)
    await apiClient.delete(`/nodes/${row.node_id}/backend-runtimes/${row.id}`)
    ElMessage.success(t('runnerConfigs.deleted')); await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
.next-summary { color: var(--el-text-color-secondary); margin-right: 12px; font-size: 12px; }
</style>
