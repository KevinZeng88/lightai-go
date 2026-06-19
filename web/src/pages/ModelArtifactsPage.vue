<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('artifacts.title') }}</h2>
      <div>
        <el-button type="primary" @click="startWizard">{{ $t('modelWizard.title') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column :label="$t('artifacts.displayName')" min-width="180">
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="name" :label="$t('artifacts.name')" min-width="150" />
      <el-table-column prop="format" :label="$t('artifacts.format')" width="100" />
      <el-table-column prop="size_label" :label="$t('artifacts.size')" width="80" />
      <el-table-column prop="path" :label="$t('artifacts.path')" min-width="200" show-overflow-tooltip />
      <el-table-column :label="$t('common.actions')" width="280">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Simple Create Dialog -->
    <el-dialog v-model="dialogVisible" :title="editingId ? $t('common.edit') : $t('common.create')" width="500px">
      <el-form :model="form" label-width="140px">
        <el-form-item :label="$t('artifacts.name')"><el-input v-model="form.name" disabled /><el-tag size="small" type="info" style="margin-left:8px">{{ $t('common.readonly') }}</el-tag></el-form-item>
        <el-form-item :label="$t('artifacts.displayName')"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item :label="$t('artifacts.path')"><el-input v-model="form.path" /></el-form-item>
        <el-form-item :label="$t('artifacts.format')"><el-select v-model="form.format" filterable allow-create style="width:100%"><el-option v-for="o in formatOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
        <el-form-item :label="$t('artifacts.quantization')"><el-select v-model="form.quantization" filterable allow-create style="width:100%"><el-option v-for="o in quantOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doSave" :loading="saving">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <!-- Detail Dialog with Locations -->
    <el-drawer v-model="detailVisible" :title="$t('artifacts.title')" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('artifacts.name')">{{ selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.displayName')">{{ selected.display_name || selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.format')">{{ selected.format }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.path')">{{ selected.path }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.size')">{{ selected.size_label || '-' }}</el-descriptions-item>
        </el-descriptions>
        <h4 style="margin-top:16px">{{ $t('modelLocations.title') }}</h4>
        <el-button size="small" type="primary" @click="showAddLocation" style="margin-bottom:8px">{{ $t('modelLocations.addLocation') }}</el-button>
        <el-table :data="locations" stripe size="small">
          <el-table-column :label="$t('modelLocations.node')" width="240" show-overflow-tooltip><template #default="{ row }">{{ nodeLabel(row.node_id) }}</template></el-table-column>
          <el-table-column prop="absolute_path" :label="$t('modelLocations.path')" min-width="200" show-overflow-tooltip />
          <el-table-column prop="verification_status" :label="$t('modelLocations.status')" width="100" />
          <el-table-column prop="match_status" :label="$t('modelLocations.matchStatus')" width="110" />
          <el-table-column :label="$t('common.actions')" width="180">
            <template #default="{ row: loc }">
              <el-button size="small" @click="doRescan(loc)">{{ $t('modelLocations.rescan') }}</el-button>
              <el-button size="small" type="danger" @click="doDeleteLocation(loc)">{{ $t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>

    <!-- Wizard Dialog -->
    <el-dialog v-model="wizardVisible" :title="$t('modelWizard.title')" width="800px" :close-on-click-modal="false">
      <el-steps :active="wizardStep" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('modelWizard.selectNode')" />
        <el-step :title="$t('modelWizard.browseDir')" />
        <el-step :title="$t('modelWizard.scanModel')" />
      </el-steps>
      <!-- Step 1: Select node -->
      <div v-if="wizardStep === 0">
        <el-select v-model="wizardNodeId" :placeholder="$t('modelWizard.selectNode')" style="width:100%" filterable @change="onWizAutoNext">
          <el-option v-for="n in nodeItems" :key="n.id" :label="n.label" :value="n.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizardNodeId" @click="wizardStep=1">{{ $t('common.next') }}</el-button></div>
      </div>
      <!-- Step 2: File browser -->
      <div v-if="wizardStep === 1">
        <RemoteFileBrowser :node-id="wizardNodeId" @select="onFileSelect" />
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardSelectedEntry" @click="doScan">{{ $t('modelWizard.scanModel') }}</el-button>
        </div>
      </div>
      <!-- Step 3: Scan results & save -->
      <div v-if="wizardStep === 2" v-loading="wizardScanning">
        <el-alert v-if="scanResult?.error" type="error" :title="scanResult.error" show-icon />
        <el-descriptions v-if="scanResult && !scanResult.error" :column="2" border size="small">
          <el-descriptions-item :label="$t('modelWizard.modelName')">
            <span>{{ wizardModelName }}</span>
            <el-tag size="small" type="info" style="margin-left:8px">{{ $t('modelWizard.nameHint') }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.displayName')">
            <el-input v-model="wizardDisplayName" size="small" />
          </el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.modelFormat')">{{ scanResult.format || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.architecture')">{{ (typeof scanResult.architecture === 'string') ? scanResult.architecture : JSON.stringify(scanResult.architecture) }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.size')">{{ scanResult.size_label || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.path')">{{ scanResult.absolute_path || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.type')">{{ scanResult.path_type || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!scanResult || !!scanResult.error" @click="doWizardSave" :loading="wizardSaving">{{ $t('modelWizard.createAndSave') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Add Location Dialog -->
    <el-dialog v-model="addLocVisible" :title="$t('modelLocations.addLocation')" width="600px">
      <el-select v-model="addLocNodeId" :placeholder="$t('modelWizard.selectNode')" style="width:100%;margin-bottom:8px" filterable>
        <el-option v-for="n in nodeItems" :key="n.id" :label="n.label" :value="n.id" />
      </el-select>
      <RemoteFileBrowser v-if="addLocNodeId" :node-id="addLocNodeId" @select="(e:any) => { addLocPath = e.relative_path; addLocSelected = e }" />
      <template #footer>
        <el-button @click="addLocVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!addLocPath" @click="doAddLocation" :loading="addLocSaving">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import RemoteFileBrowser from '@/components/RemoteFileBrowser.vue'
import { useWizardAutoAdvance } from '@/composables/useWizardAutoAdvance'
const { loadNodes, nodes: nodeItems, nodeLabel } = useNodeLabels()
const { t } = useI18n()

const loading = ref(false); const saving = ref(false)
const items = ref<any[]>([]); const dialogVisible = ref(false); const detailVisible = ref(false); const selected = ref<any>(null); const locations = ref<any[]>([])
const form = ref({ name: '', path: '', format: 'custom', task_type: 'chat', architecture: 'custom', size_label: '', quantization: 'unknown', source_type: 'local_path', display_name: '' })
let editingId = ''

// Wizard state
const wizardVisible = ref(false); const wizardStep = ref(0)
const wizardNodeId = ref(''); const wizardSelectedEntry = ref<any>(null)
const wizardScanning = ref(false); const wizardSaving = ref(false)
const scanResult = ref<any>(null); const wizardModelName = ref(''); const wizardDisplayName = ref('')

const { onSelectAutoNext: onWizAutoNext } = useWizardAutoAdvance(wizardStep, () => { wizardStep.value++ })

// Add location state
const addLocVisible = ref(false); const addLocNodeId = ref(''); const addLocPath = ref(''); const addLocSelected = ref<any>(null); const addLocSaving = ref(false)

const formatOptions = ['gguf', 'safetensors', 'huggingface', 'pt', 'onnx', 'other']
const quantOptions = ['Q4_K_M', 'Q5_K_M', 'Q8_0', 'FP16', 'BF16', 'FP8', 'INT8', 'INT4', 'none', 'other']

onMounted(async () => { await refresh(); loadNodes() })

async function refresh() {
  loading.value = true
  try { items.value = await apiClient.get('/api/v1/model-artifacts') } catch (e: any) { console.error(e) }
  loading.value = false
}
async function loadNodesLocal() { loadNodes() }

function showCreate() { editingId = ''; form.value = { name: '', path: '', format: 'custom', task_type: 'chat', architecture: 'custom', size_label: '', quantization: 'unknown', source_type: 'local_path', display_name: '' }; dialogVisible.value = true }
function showEdit(row: any) { editingId = row.id; Object.assign(form.value, row); dialogVisible.value = true }

async function doSave() {
  saving.value = true
  try {
    if (!form.value.display_name) form.value.display_name = form.value.name
    if (editingId) await apiClient.patch(`/api/v1/model-artifacts/${editingId}`, form.value)
    else await apiClient.post('/api/v1/model-artifacts', form.value)
    ElMessage.success(t('artifacts.saved')); dialogVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(t('artifacts.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' })
    await apiClient.delete(`/api/v1/model-artifacts/${row.id}`)
    ElMessage.success(t('artifacts.deleted')); await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}

async function showDetail(row: any) {
  selected.value = row
  try { locations.value = await apiClient.get(`/api/v1/model-artifacts/${row.id}`).then((r: any) => r.locations || []) } catch { locations.value = [] }
  detailVisible.value = true
}

// ---- Wizard ----
function startWizard() { wizardVisible.value = true; wizardStep.value = 0; wizardNodeId.value = ''; wizardSelectedEntry.value = null; scanResult.value = null; wizardModelName.value = ''; wizardDisplayName.value = '' }
function onFileSelect(entry: any) {
  wizardSelectedEntry.value = entry
  wizardModelName.value = entry.name
  wizardDisplayName.value = entry.name
}

async function doScan() {
  if (!wizardSelectedEntry.value || !wizardNodeId.value) return
  wizardScanning.value = true; wizardStep.value = 2
  try {
    const entry = wizardSelectedEntry.value
    const root = entry.root || ''
    const relPath = entry.relative_path || entry.name
    const resp = await apiClient.post(`/nodes/${wizardNodeId.value}/model-paths/scan`, { root_id: entry.root_id, root, relative_path: relPath, path_type: entry.path_type || (entry.is_dir ? 'directory' : 'file') })
    scanResult.value = resp
    if (resp.discovered_name) { wizardModelName.value = resp.discovered_name; wizardDisplayName.value = resp.discovered_name }
  } catch (e: any) { scanResult.value = { error: e?.message || t('modelWizard.scanFailed') } }
  wizardScanning.value = false
}

async function doWizardSave() {
  wizardSaving.value = true
  try {
    const artifact = await apiClient.post('/api/v1/model-artifacts', {
      name: wizardModelName.value, path: scanResult.value.absolute_path,
      format: scanResult.value.format || 'huggingface', task_type: 'chat',
      size_label: scanResult.value.size_label || '', source_type: 'local_path',
      display_name: wizardDisplayName.value || wizardModelName.value,
    })
    await apiClient.post(`/model-artifacts/${artifact.id}/locations`, {
      node_id: wizardNodeId.value, root_id: scanResult.value.root_id || wizardSelectedEntry.value?.root_id,
      model_root: scanResult.value.model_root || scanResult.value.root || wizardSelectedEntry.value?.root,
      relative_path: scanResult.value.relative_path || wizardSelectedEntry.value?.relative_path,
      path_type: scanResult.value.path_type || 'directory',
      verification_status: 'verified', match_status: 'exact_match',
    })
    ElMessage.success(t('artifacts.created')); wizardVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardSaving.value = false
}

// ---- Location management ----
function showAddLocation() { addLocVisible.value = true; addLocNodeId.value = ''; addLocPath.value = '' }
async function doAddLocation() {
  if (!selected.value || !addLocNodeId.value || !addLocPath.value) return
  addLocSaving.value = true
  try {
    await apiClient.post(`/model-artifacts/${selected.value.id}/locations`, {
      node_id: addLocNodeId.value, root_id: addLocSelected.value.root_id,
      model_root: addLocSelected.value.root,
      relative_path: addLocSelected.value.relative_path,
      path_type: addLocSelected.value.path_type || 'directory',
      verification_status: 'verified', match_status: 'exact_match',
    })
    ElMessage.success(t('modelLocations.added')); addLocVisible.value = false
    await showDetail(selected.value)
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  addLocSaving.value = false
}
async function doRescan(loc: any) {
  try {
    await apiClient.post(`/model-artifacts/${selected.value.id}/locations/${loc.id}/rescan`)
    ElMessage.success(t('modelLocations.rescanned')); await showDetail(selected.value)
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function doDeleteLocation(loc: any) {
  try {
    await ElMessageBox.confirm(t('modelLocations.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await apiClient.delete(`/model-artifacts/${selected.value.id}/locations/${loc.id}`)
    ElMessage.success(t('modelLocations.deleted')); await showDetail(selected.value)
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
</style>
