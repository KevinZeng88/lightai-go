<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runtimes.title') }}</h2>
      <div>
        <el-button type="primary" @click="showCreate">{{ $t('runtimes.createFromTemplate') }}</el-button>
        <el-button type="primary" @click="startWizard">{{ $t('runtimeWizard.title') }}</el-button>
      </div>
    </div>

    <el-table :data="runtimes" v-loading="loading" stripe>
      <el-table-column prop="name" :label="$t('runtimes.name')" min-width="180" />
      <el-table-column prop="vendor" :label="$t('runtimes.vendor')" width="100" />
      <el-table-column prop="image_name" :label="$t('runtimes.image')" min-width="220" />
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
          <el-button v-if="!row.is_editable" size="small" type="warning" @click="doClone(row)">{{ $t('runtimes.clone') }}</el-button>
          <el-button size="small" type="danger" :disabled="!row.is_editable" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create-from-template dialog (preserved) -->
    <el-dialog v-model="createVisible" :title="$t('runtimes.createFromTemplate')" width="560px">
      <el-form :model="createForm" label-width="150px">
        <el-form-item :label="$t('runtimes.templateName')">
          <el-select v-model="createForm.template_name" :placeholder="$t('runtimes.selectTemplate')">
            <el-option v-for="t in templates" :key="t.name" :label="t.name" :value="t.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('runtimes.name')"><el-input v-model="createForm.name" /></el-form-item>
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
        <h3>{{ $t('runtimes.highRiskOptions') }}</h3>
        <div class="option-grid">
          <RuntimeOption v-for="opt in scalarOptions" :key="opt.key" v-model:enabled="opt.enabled" v-model:value="opt.value" :label="$t(opt.label)" :warning="$t(opt.warning)" />
        </div>
        <h3>{{ $t('runtimes.listOptions') }}</h3>
        <div class="textarea-grid">
          <RuntimeTextarea v-for="opt in listOptions" :key="opt.key" v-model:enabled="opt.enabled" v-model:value="opt.value" :label="$t(opt.label)" />
        </div>
        <h3>{{ $t('runtimes.customOptions') }}</h3>
        <div class="textarea-grid">
          <RuntimeTextarea v-model:enabled="customArgs.enabled" v-model:value="customArgs.value" :label="$t('runtimes.customArgs')" />
          <RuntimeTextarea v-model:enabled="customEnv.enabled" v-model:value="customEnv.value" :label="$t('runtimes.customEnv')" />
          <RuntimeTextarea v-model:enabled="customDocker.enabled" v-model:value="customDocker.value" :label="$t('runtimes.customDockerOptions')" />
        </div>
      </el-form>
      <el-divider /><h3>{{ $t('runtimes.commandPreview') }}</h3><pre class="preview">{{ commandPreview }}</pre>
      <template #footer><el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doEdit" :loading="editing" :disabled="selected && !selected.is_editable">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <!-- Detail drawer with node management -->
    <el-drawer v-model="detailVisible" :title="$t('common.detail')" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runtimes.name')">{{ selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.displayName')">{{ selected.display_name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.vendor')">{{ selected.vendor }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_name }}</el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top:16px">{{ $t('nodeRuntime.title') }}</h4>
        <el-button size="small" type="primary" @click="showAddNode" style="margin-bottom:8px">{{ $t('nodeRuntime.addNode') }}</el-button>
        <el-table :data="nodeRuntimes" stripe size="small" v-loading="nrLoading">
          <el-table-column prop="node_id" :label="$t('modelLocations.node')" width="220" show-overflow-tooltip />
          <el-table-column :label="$t('nodeRuntime.imageRef')" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.image_ref || '-' }}</template>
          </el-table-column>
          <el-table-column :label="$t('nodeRuntime.imagePresent')" width="90">
            <template #default="{ row }"><el-tag :type="row.image_present ? 'success' : 'danger'" size="small">{{ row.image_present ? 'Yes' : 'No' }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="status" :label="$t('nodeRuntime.status')" width="100">
            <template #default="{ row }"><el-tag :type="row.status==='ready'?'success':row.status==='missing_image'?'warning':'info'" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column :label="$t('common.actions')" width="150">
            <template #default="{ row: nr }">
              <el-button size="small" @click="doCheckNode(nr)">{{ $t('nodeRuntime.recheck') }}</el-button>
              <el-button size="small" type="danger" @click="doDeleteNode(nr)">{{ $t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>

    <!-- Wizard dialog -->
    <el-dialog v-model="wizardVisible" :title="$t('runtimeWizard.title')" width="800px" :close-on-click-modal="false">
      <el-steps :active="wizStep" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('runtimeWizard.selectBackend')" />
        <el-step :title="$t('runtimeWizard.selectNode')" />
        <el-step :title="$t('runtimeWizard.browseImage')" />
        <el-step :title="$t('runtimeWizard.save')" />
      </el-steps>

      <div v-if="wizStep===0">
        <el-select v-model="wizBackendId" :placeholder="$t('runtimeWizard.selectBackend')" style="width:100%" filterable @change="onBackendChange">
          <el-option v-for="b in backends" :key="b.id" :label="`${b.name} (${b.display_name})`" :value="b.id" />
        </el-select>
        <el-select v-if="wizBackendId" v-model="wizVersionId" :placeholder="$t('runtimeWizard.selectVersion')" style="width:100%;margin-top:8px" filterable>
          <el-option v-for="v in wizVersions" :key="v.id" :label="v.display_name || v.version" :value="v.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizBackendId||!wizVersionId" @click="wizStep=1">{{ $t('common.next') }}</el-button></div>
      </div>

      <div v-if="wizStep===1">
        <el-select v-model="wizNodeId" :placeholder="$t('runtimeWizard.selectNode')" style="width:100%" filterable>
          <el-option v-for="n in nodes" :key="n.id" :label="n.id" :value="n.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizStep=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizNodeId" @click="wizStep=2">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizStep===2">
        <DockerImagePicker v-if="wizNodeId" :node-id="wizNodeId" @select="onImageSelect" />
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizStep=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizImageRef" @click="wizStep=3">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizStep===3">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runtimeWizard.selectBackend')">{{ wizBackendId }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimeWizard.selectVersion')">{{ wizVersionId }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimeWizard.selectNode')">{{ wizNodeId }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimeWizard.imageRef')">{{ wizImageRef }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="wizCheckResult" style="margin-top:8px">
          <el-alert :type="wizCheckResult.status==='ready'?'success':'warning'" :title="`Runtime status: ${wizCheckResult.status}`" :description="wizCheckResult.status_reason" show-icon :closable="false" />
        </div>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizStep=2">{{ $t('common.prev') }}</el-button>
          <el-button @click="doCheckRuntime" :loading="wizChecking">{{ $t('runtimeWizard.checkRuntime') }}</el-button>
          <el-button type="primary" :disabled="!wizCheckResult || wizCheckResult.status==='template_only'" @click="doWizardSave" :loading="wizSaving">{{ $t('runtimeWizard.save') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Add node dialog -->
    <el-dialog v-model="addNodeVisible" :title="$t('nodeRuntime.addNode')" width="700px">
      <el-select v-model="addNodeId" :placeholder="$t('runtimeWizard.selectNode')" style="width:100%;margin-bottom:8px" filterable>
        <el-option v-for="n in nodes" :key="n.id" :label="n.id" :value="n.id" />
      </el-select>
      <DockerImagePicker v-if="addNodeId" :node-id="addNodeId" @select="(img:any) => addNodeImage = img.image_ref || img.image_ref" />
      <template #footer>
        <el-button @click="addNodeVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!addNodeId||!addNodeImage" @click="doAddNode" :loading="addNodeSaving">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { ElCheckbox, ElInput, ElMessage, ElMessageBox } from 'element-plus'
import { listRuntimes, createRuntimeFromTemplate, patchRuntime, deleteRuntime, type BackendRuntime } from '@/api/runtimes'
import { listRuntimeTemplates, listBackends, listBackendVersions, type BackendRuntimeTemplate } from '@/api/backends'
import { apiClient } from '@/api/client'
import DockerImagePicker from '@/components/DockerImagePicker.vue'
import { useI18n } from 'vue-i18n'

const RuntimeOption = defineComponent({
  props: { enabled: Boolean, value: String, label: String, warning: String },
  emits: ['update:enabled', 'update:value'],
  setup(props, { emit }) {
    return () => h('div', { class: 'option-block' }, [
      h(ElCheckbox, { modelValue: props.enabled, 'onUpdate:modelValue': (v: unknown) => emit('update:enabled', v === true) }, () => props.label),
      h(ElInput, { modelValue: props.value, disabled: !props.enabled, 'onUpdate:modelValue': (v: string) => emit('update:value', v) }),
      props.enabled ? h('div', { class: 'risk-text' }, props.warning) : null,
    ])
  },
})
const RuntimeTextarea = defineComponent({
  props: { enabled: Boolean, value: String, label: String },
  emits: ['update:enabled', 'update:value'],
  setup(props, { emit }) {
    return () => h('div', { class: 'textarea-block' }, [
      h(ElCheckbox, { modelValue: props.enabled, 'onUpdate:modelValue': (v: unknown) => emit('update:enabled', v === true) }, () => props.label),
      h(ElInput, { modelValue: props.value, type: 'textarea', rows: 4, disabled: !props.enabled, 'onUpdate:modelValue': (v: string) => emit('update:value', v) }),
    ])
  },
})

const { t } = useI18n()
const loading = ref(false); const creating = ref(false); const editing = ref(false)
const runtimes = ref<BackendRuntime[]>([]); const templates = ref<BackendRuntimeTemplate[]>([]); const nodes = ref<any[]>([]); const backends = ref<any[]>([])
const selected = ref<BackendRuntime | null>(null)
const createVisible = ref(false); const editVisible = ref(false); const detailVisible = ref(false)
const createForm = ref({ template_name: 'vllm-nvidia-docker', name: '', vendor: 'nvidia', image_name: '', backend_name: 'vllm', backend_version: 'openai-latest', display_name: '' })
const editForm = reactive({ display_name: '', image_name: '', vendor: '' }); let editingId = ''

const scalarOptions = reactive([
  { key: 'privileged', label: 'runtimes.privileged', warning: 'runtimes.privilegedRisk', enabled: false, value: 'true' },
  { key: 'ipc_mode', label: 'runtimes.ipcMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'uts_mode', label: 'runtimes.utsMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'network_mode', label: 'runtimes.networkMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'pid_mode', label: 'runtimes.pidMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'shm_size', label: 'runtimes.shmSize', warning: 'runtimes.resourceRisk', enabled: false, value: '16gb' },
])
const listOptions = reactive([
  { key: 'devices', label: 'runtimes.devices', enabled: false, value: '' },
  { key: 'optional_devices', label: 'runtimes.optionalDevices', enabled: false, value: '' },
  { key: 'group_add', label: 'runtimes.groupAdd', enabled: false, value: '' },
  { key: 'security_options', label: 'runtimes.securityOpt', enabled: false, value: '' },
  { key: 'cap_add', label: 'runtimes.capAdd', enabled: false, value: '' },
  { key: 'device_cgroup_rules', label: 'runtimes.deviceCgroupRules', enabled: false, value: '' },
  { key: 'extra_hosts', label: 'runtimes.extraHosts', enabled: false, value: '' },
  { key: 'ulimits', label: 'runtimes.ulimits', enabled: false, value: '' },
  { key: 'env', label: 'runtimes.env', enabled: false, value: '' },
  { key: 'extra_mounts', label: 'runtimes.extraMounts', enabled: false, value: '' },
])
const customArgs = reactive({ enabled: false, value: '' }); const customEnv = reactive({ enabled: false, value: '' }); const customDocker = reactive({ enabled: false, value: '' })

// Node runtime management state
const nodeRuntimes = ref<any[]>([]); const nrLoading = ref(false)
const addNodeVisible = ref(false); const addNodeId = ref(''); const addNodeImage = ref(''); const addNodeSaving = ref(false)

// Wizard state
const wizardVisible = ref(false); const wizStep = ref(0)
const wizBackendId = ref(''); const wizVersionId = ref(''); const wizVersions = ref<any[]>([])
const wizNodeId = ref(''); const wizImageRef = ref('')
const wizChecking = ref(false); const wizSaving = ref(false); const wizCheckResult = ref<any>(null)

onMounted(async () => { await refresh(); await loadRefs() })
async function refresh() { loading.value = true; try { runtimes.value = await listRuntimes() } finally { loading.value = false } }
async function loadRefs() {
  try { templates.value = await listRuntimeTemplates(); backends.value = await listBackends(); nodes.value = await apiClient.get('/nodes') } catch { nodes.value = [] }
}

function showCreate() { createVisible.value = true }
async function doCreate() { creating.value = true; try { await createRuntimeFromTemplate(createForm.value); ElMessage.success(t('runtimes.created')); createVisible.value = false; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.requestFailed')) } creating.value = false }
function showEdit(row: BackendRuntime) { selected.value = row; editingId = row.id; editForm.display_name = row.display_name; editForm.image_name = row.image_name; editForm.vendor = row.vendor; loadDockerJson(row); editVisible.value = true }
async function doEdit() { editing.value = true; try { await patchRuntime(editingId, buildPayload()); ElMessage.success(t('runtimes.saved')); editVisible.value = false; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.requestFailed')) } editing.value = false }
async function handleDelete(row: BackendRuntime) { try { await ElMessageBox.confirm(t('runtimes.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' }); await deleteRuntime(row.id); ElMessage.success(t('runtimes.deleted')); await refresh() } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.requestFailed')) } }

// Clone
async function doClone(row: BackendRuntime) {
  try {
    await apiClient.post(`/backend-runtimes/${row.id}/clone`)
    ElMessage.success('Cloned'); await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
}

// Detail + node management
async function showDetail(row: BackendRuntime) { selected.value = row; await loadNodeRuntimes(row.id); detailVisible.value = true }
async function loadNodeRuntimes(runtimeID: string) { nrLoading.value = true; try { nodeRuntimes.value = await apiClient.get(`/nodes/${nodes.value[0]?.id || ''}/backend-runtimes`) } catch { nodeRuntimes.value = [] }; nrLoading.value = false }

function showAddNode() { addNodeVisible.value = true; addNodeId.value = ''; addNodeImage.value = '' }
async function doAddNode() {
  if (!selected.value || !addNodeId.value || !addNodeImage.value) return; addNodeSaving.value = true
  try {
    await apiClient.post(`/nodes/${addNodeId.value}/backend-runtimes/enable`, { backend_runtime_id: selected.value.id, image_ref: addNodeImage.value, image_present: true, docker_available: true })
    ElMessage.success('Node added'); addNodeVisible.value = false; await loadNodeRuntimes(selected.value.id)
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  addNodeSaving.value = false
}
async function doCheckNode(nr: any) {
  try {
    await apiClient.post(`/nodes/${nr.node_id}/backend-runtimes/check`, { backend_runtime_id: nr.backend_runtime_id })
    ElMessage.success('Checked'); await loadNodeRuntimes(selected.value!.id)
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
}
async function doDeleteNode(nr: any) {
  try {
    await ElMessageBox.confirm('Delete node runtime?', 'Confirm', { type: 'warning' })
    await apiClient.delete(`/nodes/${nr.node_id}/backend-runtimes/${nr.id}`)
    ElMessage.success('Deleted'); await loadNodeRuntimes(selected.value!.id)
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || 'Failed') }
}

// Wizard
function startWizard() { wizardVisible.value = true; wizStep.value = 0; wizBackendId.value = ''; wizVersionId.value = ''; wizNodeId.value = ''; wizImageRef.value = ''; wizCheckResult.value = null; loadRefs() }
async function onBackendChange(id: string) {
  try { wizVersions.value = await apiClient.get(`/backends/${id}/versions`) } catch { wizVersions.value = [] }
  wizVersionId.value = ''
}
function onImageSelect(img: any) { wizImageRef.value = img.image_ref || img.image || '' }
async function doCheckRuntime() {
  wizChecking.value = true
  try {
    wizCheckResult.value = await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/check`, { backend_runtime_id: 'runtime.vllm.nvidia-docker', image_ref: wizImageRef.value, image_present: !!wizImageRef.value, docker_available: true })
  } catch (e: any) { wizCheckResult.value = { status: 'unknown', status_reason: e?.message || 'check failed' } }
  wizChecking.value = false
}
async function doWizardSave() {
  wizSaving.value = true
  try {
    await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/enable`, { backend_runtime_id: 'runtime.vllm.nvidia-docker', image_ref: wizImageRef.value, image_present: true, docker_available: true })
    ElMessage.success('Runtime configured'); wizardVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  wizSaving.value = false
}

// Ported from original (loadDockerJson, buildPayload, commandPreview, parseLines, etc.)
function loadDockerJson(row: BackendRuntime) {
  const docker = row.docker_json || {}
  for (const opt of scalarOptions) { const v = docker[opt.key]; opt.enabled = v !== undefined && v !== '' && v !== false; opt.value = typeof v === 'boolean' ? String(v) : (v || opt.value) }
  for (const opt of listOptions) { const v = docker[opt.key]; opt.enabled = Array.isArray(v) ? v.length > 0 : !!v; opt.value = Array.isArray(v) ? v.map(formatListValue).join('\n') : '' }
  customArgs.enabled = Array.isArray(row.args_override_json) && row.args_override_json.length > 0; customArgs.value = Array.isArray(row.args_override_json) ? row.args_override_json.join('\n') : ''
}
function buildPayload() {
  const docker: Record<string, any> = {}
  for (const opt of scalarOptions) { if (!opt.enabled) continue; docker[opt.key] = opt.key === 'privileged' ? opt.value === 'true' : opt.value }
  for (const opt of listOptions) { if (!opt.enabled) continue; const lines = parseLines(opt.value); if (opt.key === 'devices' || opt.key === 'optional_devices' || opt.key === 'extra_mounts') docker[opt.key] = lines.map(parseMapping); else if (opt.key === 'ulimits') docker[opt.key] = Object.fromEntries(lines.map(parseKeyValue)); else if (opt.key === 'env') (docker as any).default_env = Object.fromEntries(lines.map(parseKeyValue)); else docker[opt.key] = lines }
  return { display_name: editForm.display_name, image_name: editForm.image_name, vendor: editForm.vendor, docker_json: docker, args_override_json: customArgs.enabled ? parseLines(customArgs.value) : [] }
}
const commandPreview = computed(() => {
  const payload = buildPayload(); const docker = payload.docker_json; const parts = ['docker', 'run', '-d']
  if (docker.privileged) parts.push('--privileged')
  for (const key of ['ipc_mode', 'uts_mode', 'network_mode', 'pid_mode', 'shm_size']) { if (docker[key]) parts.push(`--${key.replace('_mode','').replace('_','-')}`, String(docker[key])) }
  for (const d of docker.devices || []) parts.push('--device', `${d.host_path}:${d.container_path}`)
  for (const g of docker.group_add || []) parts.push('--group-add', g)
  for (const s of docker.security_options || []) parts.push('--security-opt', s)
  for (const [k, v] of Object.entries(docker.ulimits || {})) parts.push('--ulimit', `${k}=${v}`)
  for (const [k, v] of Object.entries((docker as any).default_env || {})) parts.push('-e', `${k}=${v}`)
  parts.push(payload.image_name || '<image>'); parts.push(...payload.args_override_json)
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
.risk-text { color: var(--el-color-danger); font-size: 12px; line-height: 1.4; margin-top: 4px; }
.preview { background: var(--el-fill-color-light); border: 1px solid var(--el-border-color); border-radius: 6px; padding: 12px; white-space: pre-wrap; word-break: break-all; }
@media (max-width: 900px) { .form-grid, .option-grid, .textarea-grid { grid-template-columns: 1fr; } }
</style>
