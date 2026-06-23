<template>
  <div class="runtime-parameter-editor">
    <!-- High Risk Options -->
    <el-collapse v-model="activeSections">
      <el-collapse-item :title="t('runtimes.highRiskOptions')" name="highRisk">
        <div v-for="opt in scalarOptions" :key="opt.key" class="param-row">
          <el-checkbox v-model:checked="opt.enabled" :disabled="readonly">
            {{ t(opt.label) }}
          </el-checkbox>
          <el-input
            v-if="opt.enabled"
            v-model="opt.value"
            :disabled="readonly"
            size="small"
            class="param-input"
          />
          <span v-if="opt.enabled && opt.warning" class="param-warning">
            {{ t(opt.warning) }}
          </span>
        </div>
      </el-collapse-item>

      <!-- List Options -->
      <el-collapse-item :title="t('runtimes.listOptions')" name="listOptions">
        <div v-for="opt in listOptions" :key="opt.key" class="param-row">
          <el-checkbox v-model:checked="opt.enabled" :disabled="readonly">
            {{ t(opt.label) }}
          </el-checkbox>
          <el-input
            v-if="opt.enabled"
            v-model="opt.value"
            type="textarea"
            :rows="3"
            :disabled="readonly"
            :placeholder="opt.placeholder"
            class="param-textarea"
          />
        </div>
      </el-collapse-item>

      <!-- Custom Args -->
      <el-collapse-item :title="t('runtimes.customOptions')" name="custom">
        <div class="param-row">
          <el-checkbox v-model:checked="customArgs.enabled" :disabled="readonly">
            {{ t('runtimes.customArgs') }}
          </el-checkbox>
          <el-input
            v-if="customArgs.enabled"
            v-model="customArgs.value"
            type="textarea"
            :rows="3"
            :disabled="readonly"
            placeholder="--flag value (one per line)"
            class="param-textarea"
          />
        </div>
        <div class="param-row">
          <el-checkbox v-model:checked="customEnv.enabled" :disabled="readonly">
            {{ t('runtimes.customEnv') }}
          </el-checkbox>
          <el-input
            v-if="customEnv.enabled"
            v-model="customEnv.value"
            type="textarea"
            :rows="3"
            :disabled="readonly"
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

interface Props {
  modelValue: {
    docker_json?: Record<string, any>
    args_override_json?: string[]
    default_env_json?: Record<string, string>
    entrypoint_override_json?: string[]
    parameter_values_json?: any[]
  }
  readonly?: boolean
  vendor?: string
}

const props = withDefaults(defineProps<Props>(), {
  readonly: false,
  vendor: 'nvidia',
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
}

const scalarOptions = reactive<ParamOption[]>([
  { key: 'privileged', label: 'runtimes.privileged', warning: 'runtimes.privilegedRisk', enabled: false, value: 'true' },
  { key: 'ipc_mode', label: 'runtimes.ipcMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'uts_mode', label: 'runtimes.utsMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'network_mode', label: 'runtimes.networkMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'pid_mode', label: 'runtimes.pidMode', warning: 'runtimes.namespaceRisk', enabled: false, value: 'host' },
  { key: 'shm_size', label: 'runtimes.shmSize', warning: 'runtimes.resourceRisk', enabled: false, value: '16gb' },
])

const listOptions = reactive<ParamOption[]>([
  { key: 'devices', label: 'runtimes.devices', enabled: false, value: '', placeholder: '/dev/dri:/dev/dri' },
  { key: 'group_add', label: 'runtimes.groupAdd', enabled: false, value: '', placeholder: 'video' },
  { key: 'security_options', label: 'runtimes.securityOpt', enabled: false, value: '', placeholder: 'seccomp=unconfined' },
  { key: 'env', label: 'runtimes.env', enabled: false, value: '', placeholder: 'KEY=VALUE' },
  { key: 'extra_mounts', label: 'runtimes.extraMounts', enabled: false, value: '', placeholder: '/host/path:/container/path:ro' },
])

const customArgs = reactive({ enabled: false, value: '' })
const customEnv = reactive({ enabled: false, value: '' })

// Load from modelValue
function loadFromModel() {
  const docker = props.modelValue.docker_json || {}
  for (const opt of scalarOptions) {
    const v = docker[opt.key]
    opt.enabled = v !== undefined && v !== '' && v !== false
    opt.value = typeof v === 'boolean' ? String(v) : (v || opt.value)
  }
  for (const opt of listOptions) {
    const v = docker[opt.key]
    opt.enabled = Array.isArray(v) ? v.length > 0 : !!v
    opt.value = Array.isArray(v) ? v.join('\n') : ''
  }
  customArgs.enabled = Array.isArray(props.modelValue.args_override_json) && props.modelValue.args_override_json.length > 0
  customArgs.value = Array.isArray(props.modelValue.args_override_json) ? props.modelValue.args_override_json.join('\n') : ''
  const envObj = props.modelValue.default_env_json || {}
  customEnv.enabled = Object.keys(envObj).length > 0
  customEnv.value = Object.entries(envObj).map(([k, v]) => `${k}=${v}`).join('\n')
}

// Build output
function buildOutput() {
  const docker: Record<string, any> = {}
  for (const opt of scalarOptions) {
    if (!opt.enabled) continue
    docker[opt.key] = opt.key === 'privileged' ? opt.value === 'true' : opt.value
  }
  for (const opt of listOptions) {
    if (!opt.enabled) continue
    const lines = parseLines(opt.value)
    if (opt.key === 'env') {
      docker.default_env = Object.fromEntries(lines.map(parseKeyValue))
    } else {
      docker[opt.key] = lines
    }
  }
  const argsOverride = customArgs.enabled ? parseLines(customArgs.value) : []
  const envOverride = customEnv.enabled ? Object.fromEntries(parseLines(customEnv.value).map(parseKeyValue)) : {}

  emit('update:modelValue', {
    ...props.modelValue,
    docker_json: docker,
    args_override_json: argsOverride,
    default_env_json: envOverride,
  })
}

// Command preview
const commandPreview = computed(() => {
  const parts = ['docker', 'run', '-d']
  const docker = props.modelValue.docker_json || {}
  if (docker.privileged) parts.push('--privileged')
  if (docker.ipc_mode) parts.push('--ipc', docker.ipc_mode)
  if (docker.shm_size) parts.push('--shm-size', docker.shm_size)
  if (docker.devices) {
    for (const d of (Array.isArray(docker.devices) ? docker.devices : [])) {
      parts.push('--device', typeof d === 'string' ? d : `${d.host_path}:${d.container_path}`)
    }
  }
  parts.push('<image>')
  if (props.modelValue.args_override_json?.length) {
    parts.push(...props.modelValue.args_override_json)
  }
  return parts.join(' ')
})

// Watch for changes
watch([scalarOptions, listOptions, customArgs, customEnv], buildOutput, { deep: true })

onMounted(loadFromModel)
watch(() => props.modelValue, loadFromModel, { deep: true })

// Helpers
function parseLines(value: string): string[] {
  return value.split('\n').map(s => s.trim()).filter(s => s.length > 0)
}

function parseKeyValue(value: string): [string, string] {
  const idx = value.indexOf('=')
  if (idx < 0) return [value, '']
  return [value.substring(0, idx), value.substring(idx + 1)]
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
