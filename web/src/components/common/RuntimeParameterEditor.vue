<template>
  <div class="runtime-parameter-editor">
    <el-collapse v-model="activeSections">
      <!-- High Risk / Docker Scalar Options -->
      <el-collapse-item :title="t('runtimes.highRiskOptions')" name="highRisk">
        <div v-for="opt in scalarOptions" :key="opt.key" class="param-row">
          <div class="param-header">
            <el-checkbox v-model:checked="opt.enabled" :disabled="readonly">
              {{ t(opt.label) }}
            </el-checkbox>
            <span v-if="opt.warning" class="param-warning">{{ t(opt.warning) }}</span>
          </div>
          <el-input
            v-model="opt.value"
            :disabled="!opt.enabled || readonly"
            size="small"
            class="param-input"
            :placeholder="opt.placeholder"
          />
          <span class="param-hint">{{ opt.hint }}</span>
        </div>
      </el-collapse-item>

      <!-- List Options -->
      <el-collapse-item :title="t('runtimes.listOptions')" name="listOptions">
        <div v-for="opt in listOptions" :key="opt.key" class="param-row">
          <div class="param-header">
            <el-checkbox v-model:checked="opt.enabled" :disabled="readonly">
              {{ t(opt.label) }}
            </el-checkbox>
          </div>
          <el-input
            v-model="opt.value"
            type="textarea"
            :rows="3"
            :disabled="!opt.enabled || readonly"
            :placeholder="opt.placeholder"
            class="param-textarea"
          />
        </div>
      </el-collapse-item>

      <!-- Backend Serving Args (dynamic from BackendVersion schema) -->
      <el-collapse-item v-if="backendParams.length > 0" :title="t('runtimes.backendServingArgs')" name="backendArgs">
        <el-alert type="info" :closable="false" style="margin-bottom:8px">
          {{ t('runtimes.backendArgsHint') }}
        </el-alert>
        <div v-for="param in backendParams" :key="param.key" class="param-row">
          <div class="param-header">
            <el-checkbox v-model:checked="param.enabled" :disabled="readonly">
              {{ param.cli_name }}
              <el-tag v-if="param.required" size="small" type="danger" style="margin-left:4px">required</el-tag>
            </el-checkbox>
          </div>
          <el-input
            v-model="param.value"
            :disabled="!param.enabled || readonly"
            size="small"
            class="param-input"
            :placeholder="param.default || param.type || ''"
          />
          <span class="param-hint">{{ param.name }}{{ param.alias ? ' / ' + param.alias : '' }}</span>
        </div>
      </el-collapse-item>

      <!-- Custom Args / Env -->
      <el-collapse-item :title="t('runtimes.customOptions')" name="custom">
        <div class="param-row">
          <div class="param-header">
            <el-checkbox v-model:checked="customArgs.enabled" :disabled="readonly">
              {{ t('runtimes.customArgs') }}
            </el-checkbox>
          </div>
          <el-input
            v-model="customArgs.value"
            type="textarea"
            :rows="3"
            :disabled="!customArgs.enabled || readonly"
            placeholder="--flag value (one per line)"
            class="param-textarea"
          />
        </div>
        <div class="param-row">
          <div class="param-header">
            <el-checkbox v-model:checked="customEnv.enabled" :disabled="readonly">
              {{ t('runtimes.customEnv') }}
            </el-checkbox>
          </div>
          <el-input
            v-model="customEnv.value"
            type="textarea"
            :rows="3"
            :disabled="!customEnv.enabled || readonly"
            placeholder="KEY=VALUE (one per line)"
            class="param-textarea"
          />
        </div>
      </el-collapse-item>

      <!-- Command Preview -->
      <el-collapse-item v-if="commandPreview" :title="t('runtimes.commandPreview')" name="preview">
        <pre class="command-preview">{{ commandPreview }}</pre>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface BackendParamDef {
  name: string
  alias?: string
  required?: boolean
  optional?: boolean
  default?: string
  value?: string
  type?: string
}

interface Props {
  modelValue: {
    docker_json?: Record<string, any>
    args_override_json?: string[]
    default_env_json?: Record<string, string>
    entrypoint_override_json?: string[]
    parameter_values_json?: any[]
  }
  backendSchema?: BackendParamDef[]
  readonly?: boolean
  vendor?: string
}

const props = withDefaults(defineProps<Props>(), {
  readonly: false,
  vendor: 'nvidia',
  backendSchema: () => [],
})

const emit = defineEmits(['update:modelValue'])

const activeSections = ref<string[]>(['highRisk'])

interface ParamOption {
  key: string
  label: string
  warning?: string
  enabled: boolean
  value: string
  placeholder?: string
  hint?: string
}

const scalarOptions = reactive<ParamOption[]>([
  { key: 'privileged', label: 'runtimes.privileged', warning: 'runtimes.privilegedRisk', enabled: false, value: 'true', placeholder: 'true/false', hint: 'Docker --privileged' },
  { key: 'ipc_mode', label: 'runtimes.ipcMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host', placeholder: 'host', hint: 'Docker --ipc' },
  { key: 'uts_mode', label: 'runtimes.utsMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host', placeholder: 'host', hint: 'Docker --uts' },
  { key: 'network_mode', label: 'runtimes.networkMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host', placeholder: 'bridge/host/none', hint: 'Docker --network' },
  { key: 'pid_mode', label: 'runtimes.pidMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host', placeholder: 'host', hint: 'Docker --pid' },
  { key: 'shm_size', label: 'runtimes.shmSize', warning: 'runtimes.resourceRisk', enabled: false, value: '16gb', placeholder: '16gb', hint: 'Docker --shm-size' },
])

const listOptions = reactive<ParamOption[]>([
  { key: 'devices', label: 'runtimes.devices', enabled: false, value: '', placeholder: '/dev/fuse:/dev/fuse', hint: 'Docker --device (host:container[:perms])' },
  { key: 'optional_devices', label: 'runtimes.optionalDevices', enabled: false, value: '', placeholder: '/dev/infiniband:/dev/infiniband', hint: 'Optional devices, failure tolerated' },
  { key: 'group_add', label: 'runtimes.groupAdd', enabled: false, value: '', placeholder: 'video\nrender', hint: 'Docker --group-add' },
  { key: 'security_options', label: 'runtimes.securityOpt', enabled: false, value: '', placeholder: 'seccomp=unconfined\napparmor=unconfined', hint: 'Docker --security-opt' },
  { key: 'cap_add', label: 'runtimes.capAdd', enabled: false, value: '', placeholder: 'SYS_ADMIN\nIPC_LOCK', hint: 'Docker --cap-add' },
  { key: 'device_cgroup_rules', label: 'runtimes.deviceCgroupRules', enabled: false, value: '', placeholder: 'c 195:* rmw', hint: 'Docker --device-cgroup-rule' },
  { key: 'extra_hosts', label: 'runtimes.extraHosts', enabled: false, value: '', placeholder: 'host.docker.internal:host-gateway', hint: 'Docker --add-host' },
  { key: 'ulimits', label: 'runtimes.ulimits', enabled: false, value: '', placeholder: 'memlock=-1\nnofile=65536:65536', hint: 'Docker --ulimit (KEY=VALUE per line)' },
  { key: 'extra_mounts', label: 'runtimes.extraMounts', enabled: false, value: '', placeholder: '/host/path:/container/path:ro', hint: 'Docker -v / --volume' },
])

const customArgs = reactive({ enabled: false, value: '' })
const customEnv = reactive({ enabled: false, value: '' })

// --- Dynamic backend serving args from BackendVersion schema ---
interface BackendParam {
  key: string
  name: string
  alias: string
  cli_name: string
  required: boolean
  enabled: boolean
  value: string
  default: string
  type: string
}

const backendParams = reactive<BackendParam[]>([])

function syncBackendParamsFromSchema() {
  const schema = props.backendSchema || []
  const existingValues = props.modelValue.parameter_values_json || []
  const existingMap = new Map<string, any>()
  for (const pv of existingValues) {
    existingMap.set(pv.key || pv.cli_name || '', pv)
  }

  // Rebuild backendParams from schema
  backendParams.length = 0
  for (const def of schema) {
    const key = (def.name || '').replace(/^-+/, '')
    const cliName = def.name || key
    const existing = existingMap.get(key) || existingMap.get(cliName)
    backendParams.push({
      key,
      name: def.name || key,
      alias: def.alias || '',
      cli_name: cliName,
      required: !!def.required,
      enabled: existing ? !!existing.enabled : !!def.required,
      value: existing?.value != null ? String(existing.value) : (def.default || def.value || ''),
      default: def.default || def.value || '',
      type: def.type || 'string',
    })
  }
}

// --- Sync guard with try/finally ---
let syncing = false

function loadFromModel() {
  if (syncing) return
  syncing = true
  try {
    const docker = props.modelValue.docker_json || {}
    for (const opt of scalarOptions) {
      const v = docker[opt.key]
      if (v !== undefined && v !== null) {
        opt.enabled = v !== '' && v !== false
        opt.value = typeof v === 'boolean' ? String(v) : String(v)
      } else {
        opt.enabled = false
      }
    }
    for (const opt of listOptions) {
      const v = docker[opt.key]
      if (v !== undefined && v !== null) {
        opt.enabled = Array.isArray(v) ? v.length > 0 : !!v
        opt.value = Array.isArray(v) ? v.map(formatListItem).join('\n') : (typeof v === 'object' ? JSON.stringify(v) : String(v || ''))
      } else {
        opt.enabled = false
      }
    }
    const argsArr = props.modelValue.args_override_json
    if (Array.isArray(argsArr)) {
      customArgs.enabled = argsArr.length > 0
      customArgs.value = argsArr.join('\n')
    }
    const envObj = props.modelValue.default_env_json
    if (envObj && typeof envObj === 'object') {
      const entries = Object.entries(envObj)
      customEnv.enabled = entries.length > 0
      customEnv.value = entries.map(([k, v]) => `${k}=${v}`).join('\n')
    }
    // Sync backend params from schema + existing values
    syncBackendParamsFromSchema()
  } finally {
    syncing = false
  }
}

function buildOutput() {
  if (syncing) return
  syncing = true
  try {
    const docker: Record<string, any> = {}
    for (const opt of scalarOptions) {
      if (opt.key === 'privileged') {
        docker[opt.key] = opt.value === 'true'
      } else if (opt.value !== '') {
        docker[opt.key] = opt.value
      }
    }
    for (const opt of listOptions) {
      const lines = parseLines(opt.value)
      if (opt.key === 'env') {
        docker.default_env = Object.fromEntries(lines.map(parseKeyValue))
      } else if (lines.length > 0) {
        docker[opt.key] = lines
      }
    }
    const argsOverride = parseLines(customArgs.value)
    const envOverride = Object.fromEntries(parseLines(customEnv.value).map(parseKeyValue))

    // Build parameter_values_json from backend params
    const paramValues = backendParams.map(p => ({
      key: p.key,
      cli_name: p.cli_name,
      alias: p.alias,
      name: p.name,
      type: p.type,
      enabled: p.enabled,
      value: p.value,
      default: p.default,
      required: p.required,
    }))

    emit('update:modelValue', {
      ...props.modelValue,
      docker_json: docker,
      args_override_json: argsOverride,
      default_env_json: envOverride,
      parameter_values_json: paramValues,
    })
  } finally {
    syncing = false
  }
}

// Command preview — show ALL enabled parameters
const commandPreview = computed(() => {
  const parts = ['docker', 'run', '-d']
  const docker = props.modelValue.docker_json || {}
  if (docker.privileged) parts.push('--privileged')
  if (docker.ipc_mode) parts.push('--ipc', String(docker.ipc_mode))
  if (docker.uts_mode) parts.push('--uts', String(docker.uts_mode))
  if (docker.network_mode) parts.push('--network', String(docker.network_mode))
  if (docker.pid_mode) parts.push('--pid', String(docker.pid_mode))
  if (docker.shm_size) parts.push('--shm-size', String(docker.shm_size))
  for (const d of (Array.isArray(docker.devices) ? docker.devices : [])) {
    parts.push('--device', typeof d === 'string' ? d : `${d.host_path}:${d.container_path || d.host_path}`)
  }
  for (const g of (Array.isArray(docker.group_add) ? docker.group_add : [])) {
    parts.push('--group-add', g)
  }
  for (const s of (Array.isArray(docker.security_options) ? docker.security_options : [])) {
    parts.push('--security-opt', s)
  }
  for (const c of (Array.isArray(docker.cap_add) ? docker.cap_add : [])) {
    parts.push('--cap-add', c)
  }
  for (const r of (Array.isArray(docker.device_cgroup_rules) ? docker.device_cgroup_rules : [])) {
    parts.push('--device-cgroup-rule', r)
  }
  for (const h of (Array.isArray(docker.extra_hosts) ? docker.extra_hosts : [])) {
    parts.push('--add-host', h)
  }
  for (const [k, v] of Object.entries(docker.ulimits || {})) {
    parts.push('--ulimit', `${k}=${v}`)
  }
  for (const v of (Array.isArray(docker.extra_mounts) ? docker.extra_mounts : [])) {
    parts.push('-v', typeof v === 'string' ? v : `${v.host_path}:${v.container_path || v.host_path}`)
  }
  const envObj = docker.default_env || {}
  for (const [k, v] of Object.entries(envObj)) {
    parts.push('-e', `${k}=${v}`)
  }
  // Backend serving args
  for (const p of backendParams) {
    if (p.enabled && p.value) {
      parts.push(p.cli_name, String(p.value))
    }
  }
  parts.push('<image>')
  if (props.modelValue.args_override_json?.length) {
    parts.push(...props.modelValue.args_override_json)
  }
  return parts.join(' ')
})

// Watch local changes → emit
watch([scalarOptions, listOptions, customArgs, customEnv, backendParams], () => { buildOutput() }, { deep: true })

onMounted(loadFromModel)

// Watch external modelValue changes → sync to local
let lastModelJson = ''
watch(() => props.modelValue, (newVal) => {
  const json = JSON.stringify(newVal)
  if (json !== lastModelJson) {
    lastModelJson = json
    loadFromModel()
  }
}, { deep: false })

// Watch backendSchema changes
watch(() => props.backendSchema, () => { syncBackendParamsFromSchema() }, { deep: false })

// Helpers
function parseLines(value: string): string[] {
  return value.split('\n').map(s => s.trim()).filter(s => s.length > 0)
}

function parseKeyValue(value: string): [string, string] {
  const idx = value.indexOf('=')
  if (idx < 0) return [value, '']
  return [value.substring(0, idx), value.substring(idx + 1)]
}

function formatListItem(v: any): string {
  if (typeof v === 'string') return v
  if (v?.host_path) return `${v.host_path}:${v.container_path || v.host_path}${v.permissions ? ':' + v.permissions : ''}`
  return JSON.stringify(v)
}
</script>

<style scoped>
.runtime-parameter-editor {
  width: 100%;
}
.param-row {
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.param-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.param-input {
  max-width: 300px;
}
.param-textarea {
  width: 100%;
}
.param-warning {
  font-size: 12px;
  color: var(--el-color-warning);
}
.param-hint {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}
.command-preview {
  background: var(--el-fill-color-darker);
  padding: 12px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
