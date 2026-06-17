<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runtimes.title') }}</h2>
      <el-button type="primary" @click="showCreate">{{ $t('runtimes.createFromTemplate') }}</el-button>
    </div>

    <el-table :data="runtimes" v-loading="loading" stripe>
      <el-table-column prop="name" :label="$t('runtimes.name')" min-width="180" />
      <el-table-column prop="vendor" :label="$t('runtimes.vendor')" width="100" />
      <el-table-column prop="runtime_type" :label="$t('runtimes.type')" width="100" />
      <el-table-column prop="image_name" :label="$t('runtimes.image')" min-width="220" />
      <el-table-column :label="$t('runtimes.managedBy')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.is_editable ? 'success' : 'info'">
            {{ row.is_editable ? $t('runtimes.userManaged') : $t('runtimes.systemManaged') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.actions')" width="280">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" :disabled="!row.is_editable" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" type="danger" :disabled="!row.is_editable" @click="handleDelete(row)">
            {{ $t('common.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

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
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doCreate" :loading="creating">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

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

      <el-divider />
      <h3>{{ $t('runtimes.commandPreview') }}</h3>
      <pre class="preview">{{ commandPreview }}</pre>

      <template #footer>
        <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doEdit" :loading="editing" :disabled="selected && !selected.is_editable">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="detailVisible" :title="$t('common.detail')" width="760px">
      <el-descriptions v-if="selected" :column="1" border>
        <el-descriptions-item :label="$t('runtimes.name')">{{ selected.name }}</el-descriptions-item>
        <el-descriptions-item :label="$t('runtimes.displayName')">{{ selected.display_name }}</el-descriptions-item>
        <el-descriptions-item :label="$t('runtimes.vendor')">{{ selected.vendor }}</el-descriptions-item>
        <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_name }}</el-descriptions-item>
        <el-descriptions-item :label="$t('runtimes.dockerOptions')"><pre>{{ JSON.stringify(selected.docker_json, null, 2) }}</pre></el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { ElCheckbox, ElInput, ElMessage, ElMessageBox } from 'element-plus'
import { listRuntimes, createRuntimeFromTemplate, patchRuntime, deleteRuntime, type BackendRuntime } from '@/api/runtimes'
import { listRuntimeTemplates, type BackendRuntimeTemplate } from '@/api/backends'
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
const loading = ref(false)
const creating = ref(false)
const editing = ref(false)
const runtimes = ref<BackendRuntime[]>([])
const templates = ref<BackendRuntimeTemplate[]>([])
const selected = ref<BackendRuntime | null>(null)
const createVisible = ref(false)
const editVisible = ref(false)
const detailVisible = ref(false)
const createForm = ref({ template_name: 'vllm-nvidia-docker', name: '', vendor: 'nvidia', image_name: '', backend_name: 'vllm', backend_version: 'openai-latest', display_name: '' })
const editForm = reactive({ display_name: '', image_name: '', vendor: '' })
let editingId = ''

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

const customArgs = reactive({ enabled: false, value: '' })
const customEnv = reactive({ enabled: false, value: '' })
const customDocker = reactive({ enabled: false, value: '' })

onMounted(refresh)

async function refresh() {
  loading.value = true
  try {
    runtimes.value = await listRuntimes()
    templates.value = await listRuntimeTemplates()
  } finally {
    loading.value = false
  }
}

function showCreate() { createVisible.value = true }

async function doCreate() {
  creating.value = true
  try {
    await createRuntimeFromTemplate(createForm.value)
    ElMessage.success(t('runtimes.created'))
    createVisible.value = false
    await refresh()
  } catch (e: any) {
    ElMessage.error(e?.message || t('common.requestFailed'))
  } finally {
    creating.value = false
  }
}

function showEdit(row: BackendRuntime) {
  selected.value = row
  editingId = row.id
  editForm.display_name = row.display_name
  editForm.image_name = row.image_name
  editForm.vendor = row.vendor
  loadDockerJson(row)
  editVisible.value = true
}

function showDetail(row: BackendRuntime) {
  selected.value = row
  detailVisible.value = true
}

async function doEdit() {
  editing.value = true
  try {
    await patchRuntime(editingId, buildPayload())
    ElMessage.success(t('runtimes.saved'))
    editVisible.value = false
    await refresh()
  } catch (e: any) {
    ElMessage.error(e?.message || t('common.requestFailed'))
  } finally {
    editing.value = false
  }
}

async function handleDelete(row: BackendRuntime) {
  try {
    await ElMessageBox.confirm(t('runtimes.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' })
    await deleteRuntime(row.id)
    ElMessage.success(t('runtimes.deleted'))
    await refresh()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || t('common.requestFailed'))
  }
}

function loadDockerJson(row: BackendRuntime) {
  const docker = row.docker_json || {}
  for (const opt of scalarOptions) {
    const v = docker[opt.key]
    opt.enabled = v !== undefined && v !== '' && v !== false
    opt.value = typeof v === 'boolean' ? String(v) : (v || opt.value)
  }
  for (const opt of listOptions) {
    const v = docker[opt.key]
    opt.enabled = Array.isArray(v) ? v.length > 0 : !!v
    opt.value = Array.isArray(v) ? v.map(formatListValue).join('\n') : ''
  }
  customArgs.enabled = Array.isArray(row.args_override_json) && row.args_override_json.length > 0
  customArgs.value = Array.isArray(row.args_override_json) ? row.args_override_json.join('\n') : ''
  customEnv.enabled = false
  customEnv.value = ''
  customDocker.enabled = false
  customDocker.value = ''
}

function buildPayload() {
  const docker: Record<string, any> = {}
  for (const opt of scalarOptions) {
    if (!opt.enabled) continue
    docker[opt.key] = opt.key === 'privileged' ? opt.value === 'true' : opt.value
  }
  for (const opt of listOptions) {
    if (!opt.enabled) continue
    const lines = parseLines(opt.value)
    if (opt.key === 'devices' || opt.key === 'optional_devices' || opt.key === 'extra_mounts') {
      docker[opt.key] = lines.map(parseMapping)
    } else if (opt.key === 'ulimits') {
      docker[opt.key] = Object.fromEntries(lines.map(parseKeyValue))
    } else if (opt.key === 'env') {
      docker.default_env = Object.fromEntries(lines.map(parseKeyValue))
    } else {
      docker[opt.key] = lines
    }
  }
  if (customDocker.enabled) {
    try { Object.assign(docker, JSON.parse(customDocker.value)) } catch {}
  }
  const defaultEnv = customEnv.enabled ? Object.fromEntries(parseLines(customEnv.value).map(parseKeyValue)) : undefined
  return {
    display_name: editForm.display_name,
    image_name: editForm.image_name,
    vendor: editForm.vendor,
    docker_json: docker,
    args_override_json: customArgs.enabled ? parseLines(customArgs.value) : [],
    ...(defaultEnv ? { default_env_json: defaultEnv } : {}),
  }
}

const commandPreview = computed(() => {
  const payload = buildPayload()
  const docker = payload.docker_json
  const parts = ['docker', 'run', '-d']
  if (docker.privileged) parts.push('--privileged')
  for (const key of ['ipc_mode', 'uts_mode', 'network_mode', 'pid_mode', 'shm_size']) {
    if (docker[key]) parts.push(`--${key.replace('_mode', '').replace('_', '-')}`, String(docker[key]))
  }
  for (const d of docker.devices || []) parts.push('--device', `${d.host_path}:${d.container_path}`)
  for (const g of docker.group_add || []) parts.push('--group-add', g)
  for (const s of docker.security_options || []) parts.push('--security-opt', s)
  for (const [k, v] of Object.entries(docker.ulimits || {})) parts.push('--ulimit', `${k}=${v}`)
  for (const [k, v] of Object.entries(payload.default_env_json || {})) parts.push('-e', `${k}=${v}`)
  parts.push(payload.image_name || '<image>')
  parts.push(...payload.args_override_json)
  return parts.join(' ')
})

function parseLines(value: string) {
  return Array.from(new Set(value.split('\n').map(v => v.trim()).filter(Boolean)))
}
function parseKeyValue(value: string): [string, string] {
  const idx = value.indexOf('=')
  return idx > 0 ? [value.slice(0, idx), value.slice(idx + 1)] : [value, '']
}
function parseMapping(value: string) {
  const [host, container = host, permissions = 'rwm'] = value.split(':')
  return { host_path: host, container_path: container, permissions }
}
function formatListValue(value: any) {
  if (typeof value === 'string') return value
  if (value.host_path) return `${value.host_path}:${value.container_path || value.host_path}`
  return JSON.stringify(value)
}

const JSON = globalThis.JSON
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.form-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.option-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.textarea-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.option-block, .textarea-block { border: 1px solid var(--el-border-color); border-radius: 6px; padding: 10px; }
.risk-text { color: var(--el-color-danger); font-size: 12px; line-height: 1.4; margin-top: 4px; }
.preview { background: var(--el-fill-color-light); border: 1px solid var(--el-border-color); border-radius: 6px; padding: 12px; white-space: pre-wrap; word-break: break-all; }
@media (max-width: 900px) {
  .form-grid, .option-grid, .textarea-grid { grid-template-columns: 1fr; }
}
</style>
