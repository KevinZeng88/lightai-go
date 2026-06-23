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
      <el-divider />
      <h3>{{ $t('runtimes.structuredParameters') }}</h3>
      <RuntimeParameterEditor v-model="parameterEditorModel" :readonly="!!(selected && !selected.is_editable)" />
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
        <h3>{{ $t('runtimes.highRiskOptions') }}</h3>
        <div class="option-grid">
          <RuntimeOption v-for="opt in cloneScalarOptions" :key="opt.key" v-model:enabled="opt.enabled" v-model:value="opt.value" :label="$t(opt.label)" :warning="$t(opt.warning)" />
        </div>
        <h3>{{ $t('runtimes.listOptions') }}</h3>
        <div class="textarea-grid">
          <RuntimeTextarea v-for="opt in cloneListOptions" :key="opt.key" v-model:enabled="opt.enabled" v-model:value="opt.value" :label="$t(opt.label)" />
        </div>
        <h3>{{ $t('runtimes.customOptions') }}</h3>
        <div class="textarea-grid">
          <RuntimeTextarea v-model:enabled="cloneCustomArgs.enabled" v-model:value="cloneCustomArgs.value" :label="$t('runtimes.customArgs')" />
          <RuntimeTextarea v-model:enabled="cloneCustomEnv.enabled" v-model:value="cloneCustomEnv.value" :label="$t('runtimes.customEnv')" />
          <RuntimeTextarea v-model:enabled="cloneCustomDocker.enabled" v-model:value="cloneCustomDocker.value" :label="$t('runtimes.customDockerOptions')" />
        </div>
      </el-form>
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
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { ElCheckbox, ElInput, ElMessage, ElMessageBox } from 'element-plus'
import { listRuntimes, createRuntimeFromTemplate, patchRuntime, deleteRuntime, type BackendRuntime } from '@/api/runtimes'
import { listBackends, listBackendVersions, listRuntimeTemplates, type BackendRuntimeTemplate } from '@/api/backends'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import { translateStatus } from '@/utils/status'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'
const { loadNodes, nodes: nodeItems, nodeLabel } = useNodeLabels()
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
const createVersions = ref<any[]>([])
const selected = ref<BackendRuntime | null>(null)
const createVisible = ref(false); const editVisible = ref(false); const detailVisible = ref(false)
const createForm = ref({ template_name: 'vllm-nvidia-docker', name: '', vendor: 'nvidia', image_name: '', backend_id: '', backend_version_id: '', display_name: '' })
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
const parameterValues = ref<any[]>([]); const parameterSchema = ref<any[]>([])

// Clone-to-user state
const cloneVisible = ref(false); const cloneSaving = ref(false); const cloneSource = ref<BackendRuntime | null>(null)
const cloneForm = reactive({ name: '', display_name: '', image_name: '', vendor: '' })
const cloneScalarOptions = reactive([
  { key: 'privileged', label: 'runtimes.privileged', warning: 'runtimes.privilegedRisk', enabled: false, value: 'true' },
  { key: 'ipc_mode', label: 'runtimes.ipcMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'uts_mode', label: 'runtimes.utsMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'network_mode', label: 'runtimes.networkMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'shm_size', label: 'runtimes.shmSize', warning: 'runtimes.resourceRisk', enabled: false, value: '16gb' },
])
const cloneListOptions = reactive([
  { key: 'devices', label: 'runtimes.devices', enabled: false, value: '' },
  { key: 'group_add', label: 'runtimes.groupAdd', enabled: false, value: '' },
  { key: 'security_options', label: 'runtimes.securityOpt', enabled: false, value: '' },
  { key: 'ulimits', label: 'runtimes.ulimits', enabled: false, value: '' },
  { key: 'env', label: 'runtimes.env', enabled: false, value: '' },
])
const cloneCustomArgs = reactive({ enabled: false, value: '' }); const cloneCustomEnv = reactive({ enabled: false, value: '' }); const cloneCustomDocker = reactive({ enabled: false, value: '' })

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
function showEdit(row: BackendRuntime) { selected.value = row; editingId = row.id; editForm.display_name = row.display_name; editForm.image_name = row.image_name; editForm.vendor = row.vendor; loadDockerJson(row); editVisible.value = true }
async function doEdit() { editing.value = true; try { await patchRuntime(editingId, buildPayload()); ElMessage.success(t('runtimes.saved')); editVisible.value = false; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.requestFailed')) } editing.value = false }
async function handleDelete(row: BackendRuntime) { try { await ElMessageBox.confirm(t('runtimes.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' }); await deleteRuntime(row.id); ElMessage.success(t('runtimes.deleted')); await refresh() } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.requestFailed')) } }

// Clone with pre-edit dialog
function showClone(row: BackendRuntime) {
  cloneSource.value = row
  const suffix = t('runtimes.customSuffix')
  cloneForm.name = `${row.name}${suffix}`
  cloneForm.display_name = `${row.display_name || row.name}${suffix}`
  cloneForm.image_name = row.image_name
  cloneForm.vendor = row.vendor
  // Load docker config into clone options
  const docker = row.docker_json || {}
  for (const opt of cloneScalarOptions) { const v = docker[opt.key]; opt.enabled = v !== undefined && v !== '' && v !== false; opt.value = typeof v === 'boolean' ? String(v) : (v || opt.value) }
  for (const opt of cloneListOptions) { const v = docker[opt.key]; opt.enabled = Array.isArray(v) ? v.length > 0 : !!v; opt.value = Array.isArray(v) ? v.map(formatListValue).join('\n') : '' }
  cloneCustomArgs.enabled = Array.isArray(row.args_override_json) && row.args_override_json.length > 0; cloneCustomArgs.value = Array.isArray(row.args_override_json) ? row.args_override_json.join('\n') : ''
  cloneVisible.value = true
}
function buildClonePayload() {
  const docker: Record<string, any> = {}
  for (const opt of cloneScalarOptions) { if (!opt.enabled) continue; docker[opt.key] = opt.key === 'privileged' ? opt.value === 'true' : opt.value }
  for (const opt of cloneListOptions) { if (!opt.enabled) continue; const lines = parseLines(opt.value); if (opt.key === 'devices') docker[opt.key] = lines.map(parseMapping); else if (opt.key === 'ulimits') docker[opt.key] = Object.fromEntries(lines.map(parseKeyValue)); else if (opt.key === 'env') (docker as any).default_env = Object.fromEntries(lines.map(parseKeyValue)); else docker[opt.key] = lines }
  return { name: cloneForm.name, display_name: cloneForm.display_name, image_name: cloneForm.image_name, vendor: cloneForm.vendor, docker_json: docker, args_override_json: cloneCustomArgs.enabled ? parseLines(cloneCustomArgs.value) : [], default_env_json: cloneCustomEnv.enabled ? Object.fromEntries(parseLines(cloneCustomEnv.value).map(parseKeyValue)) : cloneSource.value?.default_env_json, entrypoint_override_json: cloneSource.value?.entrypoint_override_json }
}
const cloneCommandPreview = computed(() => {
  if (!cloneSource.value) return ''
  const payload = buildClonePayload(); const docker = payload.docker_json; const parts = ['docker', 'run', '-d']
  if (docker.privileged) parts.push('--privileged')
  for (const key of ['ipc_mode', 'uts_mode', 'network_mode', 'shm_size']) { if (docker[key]) parts.push(`--${key.replace('_mode','').replace('_','-')}`, String(docker[key])) }
  for (const d of docker.devices || []) parts.push('--device', `${d.host_path}:${d.container_path}`)
  for (const g of docker.group_add || []) parts.push('--group-add', g)
  for (const s of docker.security_options || []) parts.push('--security-opt', s)
  for (const [k, v] of Object.entries(docker.ulimits || {})) parts.push('--ulimit', `${k}=${v}`)
  for (const [k, v] of Object.entries((docker as any).default_env || {})) parts.push('-e', `${k}=${v}`)
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

// Wizard
// Ported from original (loadDockerJson, buildPayload, commandPreview, parseLines, etc.)
function loadDockerJson(row: BackendRuntime) {
  const docker = row.docker_json || {}
  for (const opt of scalarOptions) { const v = docker[opt.key]; opt.enabled = v !== undefined && v !== '' && v !== false; opt.value = typeof v === 'boolean' ? String(v) : (v || opt.value) }
  for (const opt of listOptions) { const v = docker[opt.key]; opt.enabled = Array.isArray(v) ? v.length > 0 : !!v; opt.value = Array.isArray(v) ? v.map(formatListValue).join('\n') : '' }
  customArgs.enabled = Array.isArray(row.args_override_json) && row.args_override_json.length > 0; customArgs.value = Array.isArray(row.args_override_json) ? row.args_override_json.join('\n') : ''
  // Load structured parameter values
  parameterValues.value = Array.isArray(row.parameter_values_json) ? [...row.parameter_values_json] : []
  parameterSchema.value = Array.isArray(row.parameter_schema_json) ? [...row.parameter_schema_json] : []
}
const parameterEditorModel = computed({
  get: () => ({
    docker_json: buildPayload().docker_json,
    args_override_json: customArgs.enabled ? parseLines(customArgs.value) : [],
    default_env_json: {},
    parameter_values_json: parameterValues.value,
  }),
  set: (val: any) => {
    if (val.parameter_values_json) parameterValues.value = val.parameter_values_json
  },
})
function buildPayload() {
  const docker: Record<string, any> = {}
  for (const opt of scalarOptions) { if (!opt.enabled) continue; docker[opt.key] = opt.key === 'privileged' ? opt.value === 'true' : opt.value }
  for (const opt of listOptions) { if (!opt.enabled) continue; const lines = parseLines(opt.value); if (opt.key === 'devices' || opt.key === 'optional_devices' || opt.key === 'extra_mounts') docker[opt.key] = lines.map(parseMapping); else if (opt.key === 'ulimits') docker[opt.key] = Object.fromEntries(lines.map(parseKeyValue)); else if (opt.key === 'env') (docker as any).default_env = Object.fromEntries(lines.map(parseKeyValue)); else docker[opt.key] = lines }
  return { display_name: editForm.display_name, image_name: editForm.image_name, vendor: editForm.vendor, docker_json: docker, args_override_json: customArgs.enabled ? parseLines(customArgs.value) : [], parameter_values_json: parameterValues.value, parameter_schema_json: parameterSchema.value }
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
