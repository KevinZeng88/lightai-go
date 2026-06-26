export type HumanFieldGroup = 'basic' | 'gpu' | 'backend_common' | 'backend_vllm' | 'backend_sglang' | 'backend_llamacpp' | 'advanced'

export interface HumanRuntimeField {
  key: string
  label: string
  group: HumanFieldGroup
  type: 'string' | 'number' | 'boolean' | 'select'
  placeholder?: string
  unit?: string
  defaultValue?: unknown
  value?: unknown
  enabled?: boolean
  required?: boolean
  help?: string
  mapsTo: HumanFieldMapping[]
  visibility?: {
    backends?: string[]
    vendors?: string[]
  }
}

export interface HumanFieldMapping {
  internalKey: string
  target: 'config_set_value' | 'parameter_values' | 'docker' | 'env'
  transform?: 'number' | 'string' | 'boolean'
}

export interface RuntimeParamFormOutput {
  config_set_patch?: Record<string, any>
  parameter_values?: Array<{ key: string; value: any; enabled: boolean }>
  docker_options?: Record<string, unknown>
  env?: Record<string, string>
}

// ── Known human-field definitions ──

const HUMAN_FIELDS: HumanRuntimeField[] = [
  // Basic
  {
    key: 'shm_size', label: 'Shared Memory', group: 'basic', type: 'string',
    placeholder: '16gb', help: 'Docker --shm-size for the container',
    mapsTo: [{ internalKey: 'shm_size', target: 'docker', transform: 'string' }],
  },
  {
    key: 'health_timeout', label: 'Health Check Timeout (s)', group: 'basic', type: 'number',
    placeholder: '120', defaultValue: 120,
    mapsTo: [{ internalKey: 'runtime.health.timeout_seconds', target: 'config_set_value', transform: 'number' }],
  },
  // GPU
  {
    key: 'gpu_devices', label: 'GPU Devices', group: 'gpu', type: 'string',
    placeholder: 'all', defaultValue: 'all',
    mapsTo: [{ internalKey: 'runtime.gpu.devices', target: 'env', transform: 'string' }],
  },
  // Backend Common
  {
    key: 'served_model_name', label: 'Served Model Name', group: 'backend_common', type: 'string',
    placeholder: 'my-model', mapsTo: [{ internalKey: 'backend.common.served_model_name', target: 'parameter_values', transform: 'string' }],
  },
  {
    key: 'context_length', label: 'Context Length', group: 'backend_common', type: 'number',
    placeholder: '4096',
    mapsTo: [{ internalKey: 'backend.arg.max_model_len', target: 'parameter_values', transform: 'number' },
             { internalKey: 'backend.arg.context_length', target: 'parameter_values', transform: 'number' },
             { internalKey: 'backend.arg.ctx_size', target: 'parameter_values', transform: 'number' }],
  },
  // vLLM
  {
    key: 'vllm_gpu_memory_util', label: 'GPU Memory Utilization', group: 'backend_vllm', type: 'number',
    placeholder: '0.9', defaultValue: 0.9, unit: '0.1-0.95',
    mapsTo: [{ internalKey: 'backend.arg.gpu_memory_utilization', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['vllm'] },
  },
  {
    key: 'vllm_max_model_len', label: 'Max Model Length', group: 'backend_vllm', type: 'number',
    mapsTo: [{ internalKey: 'backend.arg.max_model_len', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['vllm'] },
  },
  {
    key: 'vllm_max_num_seqs', label: 'Max Num Seqs', group: 'backend_vllm', type: 'number',
    mapsTo: [{ internalKey: 'backend.arg.max_num_seqs', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['vllm'] },
  },
  {
    key: 'vllm_max_num_batched_tokens', label: 'Max Batched Tokens', group: 'backend_vllm', type: 'number',
    mapsTo: [{ internalKey: 'backend.arg.max_num_batched_tokens', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['vllm'] },
  },
  // SGLang
  {
    key: 'sglang_mem_fraction', label: 'Memory Fraction Static', group: 'backend_sglang', type: 'number',
    placeholder: '0.9', defaultValue: 0.9, unit: '0.1-0.95',
    mapsTo: [{ internalKey: 'backend.arg.mem_fraction_static', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['sglang'] },
  },
  {
    key: 'sglang_max_running', label: 'Max Running Requests', group: 'backend_sglang', type: 'number',
    mapsTo: [{ internalKey: 'backend.arg.max_running_requests', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['sglang'] },
  },
  // llama.cpp
  {
    key: 'llamacpp_ctx_size', label: 'Context Size', group: 'backend_llamacpp', type: 'number',
    placeholder: '2048',
    mapsTo: [{ internalKey: 'backend.arg.ctx_size', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['llamacpp'] },
  },
  {
    key: 'llamacpp_n_gpu_layers', label: 'GPU Layers', group: 'backend_llamacpp', type: 'number',
    mapsTo: [{ internalKey: 'backend.arg.n_gpu_layers', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['llamacpp'] },
  },
  {
    key: 'llamacpp_batch_size', label: 'Batch Size', group: 'backend_llamacpp', type: 'number',
    mapsTo: [{ internalKey: 'backend.arg.batch_size', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['llamacpp'] },
  },
  {
    key: 'llamacpp_threads', label: 'Threads', group: 'backend_llamacpp', type: 'number',
    mapsTo: [{ internalKey: 'backend.arg.threads', target: 'parameter_values', transform: 'number' }],
    visibility: { backends: ['llamacpp'] },
  },
]

// ── Internal keys that must NOT appear in normal forms ──

const HIDDEN_INTERNAL_PREFIXES = [
  'launcher.command',
  'launcher.args',
  'launcher.docker_options',
  'launcher.entrypoint',
  'runtime_env.',
  'runtime.env',
  'internal.',
  'resolver.',
  'source_metadata.',
  'MODEL_CONTAINER_PATH',
  'MODEL_CONTAINER_DIR',
]

// ── Public API ──

export function getHumanFieldsForBackend(backendName: string | undefined): HumanRuntimeField[] {
  if (!backendName) return HUMAN_FIELDS.filter(f => f.group === 'basic' || f.group === 'gpu' || f.group === 'backend_common')
  const name = backendName.replace(/^backend\./, '').toLowerCase()
  return HUMAN_FIELDS.filter(f => {
    if (f.group === 'basic' || f.group === 'gpu' || f.group === 'backend_common') return true
    if (!f.visibility?.backends) return false
    return f.visibility.backends.some(b => b.toLowerCase() === name)
  })
}

export function isInternalKey(key: string): boolean {
  return HIDDEN_INTERNAL_PREFIXES.some(prefix => key.startsWith(prefix))
}

export function buildParamFormOutput(fields: HumanRuntimeField[]): RuntimeParamFormOutput {
  const out: RuntimeParamFormOutput = {
    parameter_values: [],
    docker_options: {} as Record<string, unknown>,
    env: {},
    config_set_patch: {},
  }

  for (const f of fields) {
    if (f.value === undefined || f.value === null || f.value === '') continue
    for (const m of f.mapsTo) {
      switch (m.target) {
        case 'parameter_values':
          out.parameter_values!.push({ key: m.internalKey, value: f.value, enabled: f.enabled !== false })
          break
        case 'docker':
          out.docker_options![m.internalKey] = f.value
          break
        case 'env':
          out.env![m.internalKey] = String(f.value)
          break
        case 'config_set_value':
          out.config_set_patch![m.internalKey] = f.value
          break
      }
    }
  }

  // Clean up empty sections
  if (!out.parameter_values?.length) out.parameter_values = undefined
  if (!Object.keys(out.docker_options!).length) out.docker_options = undefined
  if (!Object.keys(out.env!).length) out.env = undefined
  if (!Object.keys(out.config_set_patch!).length) out.config_set_patch = undefined

  return out
}
