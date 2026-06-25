<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runtimes.title') }}</h2>
      <div>
        <el-button type="primary" @click="showCreate">{{ $t('runtimes.createFromTemplate') }}</el-button>
      </div>
    </div>

    <el-table :data="runtimes" v-loading="loading" stripe>
      <el-table-column :label="$t('runtimes.name')" min-width="200">
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="backend_id" :label="$t('runtimes.backend')" min-width="140" show-overflow-tooltip />
      <el-table-column prop="backend_version_id" :label="$t('runtimes.backendVersion')" min-width="180" show-overflow-tooltip />
      <el-table-column prop="vendor" :label="$t('runtimes.vendor')" width="100" />
      <el-table-column prop="image_name" :label="$t('runtimes.image')" min-width="220" />
      <el-table-column prop="node_count" :label="$t('runtimes.nodeCount')" width="90" />
      <el-table-column prop="ready_count" :label="$t('runtimes.readyCount')" width="90" />
      <el-table-column :label="$t('runtimes.managedBy')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.is_editable ? 'success' : 'info'">
            {{ row.is_editable ? $t('runtimes.userManaged') : $t('runtimes.systemManaged') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.actions')" width="400">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" :disabled="!row.is_editable" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button v-if="!row.is_editable" size="small" type="warning" @click="showClone(row)">{{ $t('runtimes.clone') }}</el-button>
          <el-button size="small" type="danger" :disabled="!row.is_editable" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create-from-template dialog (preserved) -->
    <el-dialog v-model="createVisible" :title="$t('runtimes.createFromTemplate')" width="560px">
      <el-form :model="createForm" label-width="150px">
        <el-form-item :label="$t('runtimes.backend')">
          <el-select v-model="createForm.backend_id" :placeholder="$t('backends.title')" style="width:100%" @change="onCreateBackendSelected">
            <el-option v-for="b in backends" :key="b.id" :label="b.display_name || b.name" :value="b.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('runtimes.backendVersion')">
          <el-select v-model="createForm.backend_version_id" :placeholder="$t('backends.versions')" style="width:100%" @change="onCreateVersionSelected">
            <el-option v-for="v in createVersions" :key="v.id" :label="v.display_name || v.version" :value="v.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('runtimes.templateName')">
          <el-select v-model="createForm.template_name" :placeholder="$t('runtimes.selectTemplate')" @change="onCreateTemplateSelected">
            <el-option v-for="t in templates" :key="t.name" :label="t.name" :value="t.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('runtimes.name')"><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item :label="$t('runtimes.displayName')"><el-input v-model="createForm.display_name" /></el-form-item>
        <el-form-item :label="$t('runtimes.vendor')"><el-input v-model="createForm.vendor" /></el-form-item>
        <el-form-item :label="$t('runtimes.image')"><el-input v-model="createForm.image_name" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doCreate" :loading="creating">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <!-- Edit dialog (preserved) -->
    <el-dialog v-model="editVisible" :title="$t('runtimes.editRuntime')" width="920px" class="runtime-dialog">
      <el-alert v-if="selected && !selected.is_editable" :title="$t('runtimes.systemReadonly')" type="info" show-icon />
      <el-form :model="editForm" label-position="top" class="runtime-form">
        <div class="form-grid">
          <el-form-item :label="$t('runtimes.displayName')"><el-input v-model="editForm.display_name" /></el-form-item>
          <el-form-item :label="$t('runtimes.image')"><el-input v-model="editForm.image_name" /></el-form-item>
          <el-form-item :label="$t('runtimes.vendor')"><el-input v-model="editForm.vendor" /></el-form-item>
        </div>
      </el-form>
      <el-divider />
      <h3>{{ $t('runtimes.structuredParameters') }}</h3>
      <RuntimeParameterEditor v-model="parameterEditorModel" :backend-schema="backendSchema" :readonly="!!(selected && !selected.is_editable)" :help-backend="helpBackend" :help-version="helpVersion" />
      <el-divider /><h3>{{ $t('runtimes.commandPreview') }}</h3><pre class="preview">{{ commandPreview }}</pre>
      <template #footer><el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doEdit" :loading="editing" :disabled="selected && !selected.is_editable">{{ $t('common.save') }}</el-button></template>
    </el-dialog>


    <!-- Clone-to-user dialog (pre-edit before save) -->
    <el-dialog v-model="cloneVisible" :title="$t('runtimes.cloneToUserConfig')" width="720px">
      <el-alert v-if="cloneSource" :title="$t('runtimes.cloneSourceTemplate', { name: cloneSource.display_name || cloneSource.name })" type="info" show-icon />
      <el-form :model="cloneForm" label-position="top" class="runtime-form" style="margin-top:12px">
        <el-form-item :label="$t('runtimes.displayName')">
          <el-input v-model="cloneForm.display_name" @input="onCloneDisplayNameChange" />
        </el-form-item>
        <el-form-item v-if="cloneForm.name" :label="$t('runtimes.internalName')">
          <el-input :model-value="cloneForm.name" disabled />
        </el-form-item>
        <el-form-item :label="$t('runtimes.image')">
          <el-input v-model="cloneForm.image_name" />
        </el-form-item>
        <el-form-item :label="$t('runtimes.vendor')">
          <el-input v-model="cloneForm.vendor" />
        </el-form-item>
      </el-form>
      <el-divider />
      <h3>{{ $t('runtimes.structuredParameters') }}</h3>
      <RuntimeParameterEditor v-model="cloneParameterEditorModel" :backend-schema="backendSchema" :help-backend="helpBackend" :help-version="helpVersion" />
      <el-divider /><h3>{{ $t('runtimes.commandPreview') }}</h3><pre class="preview">{{ cloneCommandPreview }}</pre>
      <template #footer><el-button @click="cloneVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doCloneSave" :loading="cloneSaving">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <!-- Detail drawer with read-only usage references -->
    <el-drawer v-model="detailVisible" :title="$t('common.detail')" size="65%">
      <template v-if="selected">
        <el-alert v-if="!selected.is_editable" :title="$t('runtimes.systemReadonly')" type="info" show-icon :closable="false" style="margin-bottom:12px" />
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runtimes.name')">{{ selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.displayName')">{{ selected.display_name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.sourceTemplate')">{{ selected.source_template_name || selected.backend_version_id || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.vendor')">{{ selected.vendor }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.backend')">{{ selected.backend_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.backendVersion')">{{ selected.backend_version_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.managedBy')">
            <el-tag :type="selected.is_editable ? 'success' : 'info'" size="small">{{ selected.is_editable ? $t('runtimes.userManaged') : $t('runtimes.systemBuiltin') }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top:16px">{{ $t('runtimes.dockerConfig') }}</h4>
        <el-table :data="detailDockerRows" stripe size="small" v-if="detailDockerRows.length">
          <el-table-column prop="key" label="Key" width="200" />
          <el-table-column prop="value" label="Value" show-overflow-tooltip />
        </el-table>

        <h4 style="margin-top:16px">{{ $t('runtimes.appArgs') }}</h4>
        <pre class="preview" style="max-height:200px;overflow-y:auto">{{ detailArgs }}</pre>

        <h4 style="margin-top:16px">{{ $t('runtimes.detailEnv') }}</h4>
        <el-table :data="detailEnvRows" stripe size="small" v-if="detailEnvRows.length">
          <el-table-column prop="key" label="Key" width="260" />
          <el-table-column prop="value" label="Value" show-overflow-tooltip />
        </el-table>

        <h4 style="margin-top:16px">{{ $t('runtimes.rawJSON') }}</h4>
        <el-collapse>
          <el-collapse-item :title="$t('runtimes.viewRawJSON')">
            <pre class="preview" style="max-height:300px;overflow-y:auto">{{ detailRawJSON }}</pre>
          </el-collapse-item>
        </el-collapse>

        <h4 style="margin-top:16px">{{ $t('runtimes.usageRefs') }}</h4>
        <el-alert :title="$t('runtimes.usageRefsReadonly')" type="info" show-icon :closable="false" style="margin-bottom:8px" />
        <el-table :data="nodeRuntimes" stripe size="small" v-loading="nrLoading">
          <el-table-column :label="$t('modelLocations.node')" width="240" show-overflow-tooltip><template #default="{ row }">{{ nodeLabel(row.node_id) }}</template></el-table-column>
          <el-table-column :label="$t('nodeRuntime.imageRef')" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.image_ref || '-' }}</template>
          </el-table-column>
          <el-table-column :label="$t('nodeRuntime.imagePresent')" width="90">
            <template #default="{ row }"><el-tag :type="row.image_present ? 'success' : 'danger'" size="small">{{ row.image_present ? $t('common.yes') : $t('common.no') }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="status" :label="$t('nodeRuntime.status')" width="100">
            <template #default="{ row }"><el-tag :type="row.status==='ready'?'success':row.status==='missing_image'?'warning':'info'" size="small">{{ translateStatus(row.status, t) }}</el-tag></template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listRuntimes, createRuntimeFromTemplate, patchRuntime, deleteRuntime, type BackendRuntime } from '@/api/runtimes'
import { listBackends, listBackendVersions, listRuntimeTemplates, type BackendRuntimeTemplate } from '@/api/backends'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import { translateStatus } from '@/utils/status'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'
const { loadNodes, nodes: nodeItems, nodeLabel } = useNodeLabels()
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false); const creating = ref(false); const editing = ref(false)
const runtimes = ref<BackendRuntime[]>([]); const templates = ref<BackendRuntimeTemplate[]>([]); const nodes = ref<any[]>([]); const backends = ref<any[]>([])
const createVersions = ref<any[]>([])
const selected = ref<BackendRuntime | null>(null)
const createVisible = ref(false); const editVisible = ref(false); const detailVisible = ref(false)
const createForm = ref({ template_name: 'vllm-nvidia-docker', name: '', vendor: 'nvidia', image_name: '', backend_id: '', backend_version_id: '', display_name: '' })
const editForm = reactive({ display_name: '', image_name: '', vendor: '' }); let editingId = ''

// Unified parameter model — single source of truth for edit dialog
// RuntimeParameterEditor manages all Docker args, backend serve args, env, and custom args
const parameterEditorModel = ref<any>({ docker_json: {}, args_override_json: [], default_env_json: {}, parameter_values_json: [] })
// Clone dialog has its own parameter model
const cloneParameterEditorModel = ref<any>({ docker_json: {}, args_override_json: [], default_env_json: {}, parameter_values_json: [] })
const parameterValues = ref<any[]>([]); const parameterSchema = ref<any[]>([])
const backendSchema = ref<any[]>([])
const helpBackend = ref('')
const helpVersion = ref('')

// Clone-to-user state
const cloneVisible = ref(false); const cloneSaving = ref(false); const cloneSource = ref<BackendRuntime | null>(null)
const cloneForm = reactive({ name: '', display_name: '', image_name: '', vendor: '' })

// Node runtime management state
const nodeRuntimes = ref<any[]>([]); const nrLoading = ref(false)

// Wizard state
onMounted(async () => { await refresh(); await loadRefs() })
async function refresh() { loading.value = true; try { runtimes.value = await listRuntimes() } finally { loading.value = false } }
async function loadRefs() {
  try { templates.value = await listRuntimeTemplates() } catch { templates.value = [] }
  try { backends.value = await listBackends() } catch { backends.value = [] }
  loadNodes()
}

function showCreate() { createForm.value.template_name = 'vllm-nvidia-docker'; createForm.value.name = ''; createForm.value.backend_id = ''; createForm.value.backend_version_id = ''; createVisible.value = true; loadRefs() }
async function onCreateBackendSelected(backendId: string) {
  createForm.value.backend_version_id = ''
  createVersions.value = await listBackendVersions(backendId)
}
function onCreateVersionSelected(versionId: string) {
  const version = createVersions.value.find((v: any) => v.id === versionId)
  if (!version) return
  const candidates = Array.isArray(version.image_candidates_json) ? version.image_candidates_json : []
  const image = candidates[0] || version.default_images_json?.default || ''
  if (image) createForm.value.image_name = image
  if (!createForm.value.name) createForm.value.name = `${version.version}-${createForm.value.vendor}-template`
}
function onCreateTemplateSelected(templateName: string) {
  const template = templates.value.find((t: any) => t.name === templateName)
  if (!template) return
  const suffix = t('runtimes.customSuffix')
  const baseName = `${template.name}${suffix}`
  const existingNames = new Set(runtimes.value.map((r: any) => r.name))
  let candidate = baseName
  let counter = 2
  while (existingNames.has(candidate)) {
    candidate = `${baseName} ${counter}`
    counter++
  }
  createForm.value.name = candidate
  createForm.value.display_name = candidate
}
async function doCreate() { creating.value = true; try { if (!createForm.value.display_name) createForm.value.display_name = createForm.value.name; await createRuntimeFromTemplate(createForm.value); ElMessage.success(t('runtimes.created')); createVisible.value = false; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.requestFailed')) } creating.value = false }
function showEdit(row: BackendRuntime) {
  selected.value = row; editingId = row.id
  editForm.display_name = row.display_name; editForm.image_name = row.image_name; editForm.vendor = row.vendor
  // Load all parameters into the unified model — single source of truth
  parameterEditorModel.value = {
    docker_json: row.docker_json || {},
    args_override_json: Array.isArray(row.args_override_json) ? row.args_override_json : [],
    default_env_json: typeof row.default_env_json === 'object' && row.default_env_json !== null ? row.default_env_json : {},
    parameter_values_json: Array.isArray(row.parameter_values_json) ? row.parameter_values_json : [],
  }
  parameterValues.value = Array.isArray(row.parameter_values_json) ? [...row.parameter_values_json] : []
  parameterSchema.value = Array.isArray(row.parameter_schema_json) ? [...row.parameter_schema_json] : []
  editVisible.value = true
  loadBackendSchema(row.backend_version_id, row.backend_id)
}
async function doEdit() { editing.value = true; try { await patchRuntime(editingId, buildPayload()); ElMessage.success(t('runtimes.saved')); editVisible.value = false; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.requestFailed')) } editing.value = false }

async function loadBackendSchema(versionId: string, backendId: string) {
  backendSchema.value = []
  helpBackend.value = ''
  helpVersion.value = ''
  if (!backendId) return
  try {
    const versions = await listBackendVersions(backendId)
    const version = versions.find((v: any) => v.id === versionId)
    if (version?.default_args_schema_json) {
      backendSchema.value = Array.isArray(version.default_args_schema_json) ? version.default_args_schema_json : []
    }
    if (version) {
      helpBackend.value = backendId
      helpVersion.value = (version as any).name || version.version || version.id || ''
    }
  } catch { backendSchema.value = [] }
}
async function handleDelete(row: BackendRuntime) { try { await ElMessageBox.confirm(t('runtimes.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' }); await deleteRuntime(row.id); ElMessage.success(t('runtimes.deleted')); await refresh() } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.requestFailed')) } }

// Clone with pre-edit dialog
function showClone(row: BackendRuntime) {
  cloneSource.value = row
  const suffix = t('runtimes.customSuffix')
  cloneForm.name = `${row.name}${suffix}`
  cloneForm.display_name = `${row.display_name || row.name}${suffix}`
  cloneForm.image_name = row.image_name
  cloneForm.vendor = row.vendor
  // Load ALL parameters into clone model — preserves enabled/value for copy
  cloneParameterEditorModel.value = {
    docker_json: row.docker_json || {},
    args_override_json: Array.isArray(row.args_override_json) ? row.args_override_json : [],
    default_env_json: typeof row.default_env_json === 'object' && row.default_env_json !== null ? row.default_env_json : {},
    parameter_values_json: Array.isArray(row.parameter_values_json) ? row.parameter_values_json : [],
  }
  cloneVisible.value = true
  loadBackendSchema(row.backend_version_id, row.backend_id)
}
function buildClonePayload() {
  const m = cloneParameterEditorModel.value
  return {
    name: cloneForm.name, display_name: cloneForm.display_name, image_name: cloneForm.image_name, vendor: cloneForm.vendor,
    docker_json: m.docker_json || {},
    args_override_json: m.args_override_json || [],
    default_env_json: m.default_env_json || {},
    parameter_values_json: m.parameter_values_json || [],
    entrypoint_override_json: cloneSource.value?.entrypoint_override_json,
  }
}
const cloneCommandPreview = computed(() => {
  if (!cloneSource.value) return ''
  const payload = buildClonePayload(); const docker: Record<string, any> = payload.docker_json || {}; const parts = ['docker', 'run', '-d']
  if (docker.privileged) parts.push('--privileged')
  for (const key of ['ipc_mode', 'uts_mode', 'network_mode', 'pid_mode', 'shm_size']) { if (docker[key]) parts.push(`--${key.replace('_mode','').replace('_','-')}`, String(docker[key])) }
  for (const d of (Array.isArray(docker.devices) ? docker.devices : [])) parts.push('--device', typeof d === 'string' ? d : `${d.host_path}:${d.container_path || d.host_path}`)
  for (const g of (Array.isArray(docker.group_add) ? docker.group_add : [])) parts.push('--group-add', g)
  for (const s of (Array.isArray(docker.security_options) ? docker.security_options : [])) parts.push('--security-opt', s)
  for (const [k, v] of Object.entries(docker.ulimits || {})) parts.push('--ulimit', `${k}=${v}`)
  for (const [k, v] of Object.entries(docker.default_env || {})) parts.push('-e', `${k}=${v}`)
  parts.push(payload.image_name || '<image>'); parts.push(...(payload.args_override_json || []))
  return parts.join(' ')
})
async function doCloneSave() {
  cloneSaving.value = true; try {
    if (!cloneSource.value) return
    // Send full payload including user-modified docker_json, args, env, entrypoint.
    // buildClonePayload() includes all user overrides from the clone dialog.
    const payload = buildClonePayload()
    if (!payload.name) payload.name = sanitizeName(payload.display_name || cloneSource.value.display_name || cloneSource.value.name)
    await apiClient.post(`/backend-runtimes/${cloneSource.value.id}/clone`, payload)
    ElMessage.success(t('runtimes.cloned')); cloneVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  cloneSaving.value = false
}
function sanitizeName(s: string): string { return s.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '') }
function onCloneDisplayNameChange(val: string) {
  cloneForm.name = sanitizeName(val)
}
// Legacy quick-clone (without dialog) — replaced by showClone
async function doClone(row: BackendRuntime) {
  try {
    await apiClient.post(`/backend-runtimes/${row.id}/clone`)
    ElMessage.success(t('runtimes.cloned')); await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}

// Detail computed
const detailDockerRows = computed(() => {
  if (!selected.value) return []
  const docker = selected.value.docker_json || {}
  const rows: {key: string, value: string}[] = []
  for (const [k, v] of Object.entries(docker)) {
    if (k === 'devices' || k === 'ulimits' || k === 'security_options' || k === 'group_add') continue
    rows.push({ key: k, value: typeof v === 'object' ? JSON.stringify(v) : String(v) })
  }
  if (docker.devices) rows.push({ key: 'devices', value: (docker.devices as any[]).map((d: any) => d.host_path || d).join(', ') })
  if (docker.ulimits) rows.push({ key: 'ulimits', value: JSON.stringify(docker.ulimits) })
  if (docker.security_options) rows.push({ key: 'security_options', value: (docker.security_options as string[]).join(', ') })
  if (docker.group_add) rows.push({ key: 'group_add', value: (docker.group_add as string[]).join(', ') })
  return rows
})
const detailArgs = computed(() => {
  if (!selected.value) return '-'
  const args = selected.value.args_override_json
  if (Array.isArray(args) && args.length) return args.join(' ')
  return '-'
})
const detailEnvRows = computed(() => {
  if (!selected.value) return []
  const env = selected.value.default_env_json || {}
  if (typeof env !== 'object') return []
  return Object.entries(env as Record<string, string>).map(([k, v]) => ({ key: k, value: String(v) }))
})
const detailRawJSON = computed(() => {
  if (!selected.value) return '{}'
  return JSON.stringify({ id: selected.value.id, name: selected.value.name, display_name: selected.value.display_name, source_template_name: selected.value.source_template_name, backend_id: selected.value.backend_id, backend_version_id: selected.value.backend_version_id, vendor: selected.value.vendor, image_name: selected.value.image_name, docker_json: selected.value.docker_json, args_override_json: selected.value.args_override_json, default_env_json: selected.value.default_env_json, entrypoint_override_json: selected.value.entrypoint_override_json, model_mount_json: selected.value.model_mount_json, health_check_override_json: selected.value.health_check_override_json, is_builtin: selected.value.is_builtin, is_editable: selected.value.is_editable }, null, 2)
})

// Detail + node management
async function showDetail(row: BackendRuntime) { selected.value = row; await loadNodeRuntimes(row.id); detailVisible.value = true }
async function loadNodeRuntimes(runtimeID: string) { nrLoading.value = true; try { const all: any[] = []; for (const n of nodeItems.value) { try { const nrs = await apiClient.get(`/nodes/${n.id}/backend-runtimes`); if (Array.isArray(nrs)) for (const nr of nrs) { if (nr.backend_runtime_id === runtimeID) all.push(nr) } } catch {} }; nodeRuntimes.value = all } catch { nodeRuntimes.value = [] }; nrLoading.value = false }

function buildPayload() {
  const m = parameterEditorModel.value
  return {
    display_name: editForm.display_name,
    image_name: editForm.image_name,
    vendor: editForm.vendor,
    docker_json: m.docker_json || {},
    args_override_json: m.args_override_json || [],
    default_env_json: m.default_env_json || {},
    parameter_values_json: m.parameter_values_json || parameterValues.value,
    parameter_schema_json: parameterSchema.value,
  }
}
const commandPreview = computed(() => {
  const payload = buildPayload(); const docker: Record<string, any> = payload.docker_json || {}; const parts = ['docker', 'run', '-d']
  if (docker.privileged) parts.push('--privileged')
  for (const key of ['ipc_mode', 'uts_mode', 'network_mode', 'pid_mode', 'shm_size']) { if (docker[key]) parts.push(`--${key.replace('_mode','').replace('_','-')}`, String(docker[key])) }
  for (const d of (Array.isArray(docker.devices) ? docker.devices : [])) parts.push('--device', typeof d === 'string' ? d : `${d.host_path}:${d.container_path || d.host_path}`)
  for (const g of (Array.isArray(docker.group_add) ? docker.group_add : [])) parts.push('--group-add', g)
  for (const s of (Array.isArray(docker.security_options) ? docker.security_options : [])) parts.push('--security-opt', s)
  for (const [k, v] of Object.entries(docker.ulimits || {})) parts.push('--ulimit', `${k}=${v}`)
  for (const [k, v] of Object.entries(docker.default_env || {})) parts.push('-e', `${k}=${v}`)
  parts.push(payload.image_name || '<image>'); parts.push(...(payload.args_override_json || []))
  return parts.join(' ')
})
function parseLines(value: string) { return Array.from(new Set(value.split('\n').map(v => v.trim()).filter(Boolean))) }
function parseKeyValue(value: string): [string, string] { const idx = value.indexOf('='); return idx > 0 ? [value.slice(0, idx), value.slice(idx + 1)] : [value, ''] }
function parseMapping(value: string) { const [host, container = host, permissions = 'rwm'] = value.split(':'); return { host_path: host, container_path: container, permissions } }
function formatListValue(value: any) { if (typeof value === 'string') return value; if (value.host_path) return `${value.host_path}:${value.container_path || value.host_path}`; return JSON.stringify(value) }
const JSON = globalThis.JSON
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
.form-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.option-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.textarea-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.option-block, .textarea-block { border: 1px solid var(--el-border-color); border-radius: 6px; padding: 10px; }
.param-input { margin-top: 4px; }
.risk-text { color: var(--el-color-danger); font-size: 12px; line-height: 1.4; margin-top: 4px; }
.preview { background: var(--el-fill-color-light); border: 1px solid var(--el-border-color); border-radius: 6px; padding: 12px; white-space: pre-wrap; word-break: break-all; }
@media (max-width: 900px) { .form-grid, .option-grid, .textarea-grid { grid-template-columns: 1fr; } }
</style>
