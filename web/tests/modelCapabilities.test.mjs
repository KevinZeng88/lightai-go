import {
  inferModelCapabilities,
  recommendedTestMode,
  formatTestFailure,
} from '../src/utils/modelCapabilities.js'

let failed = 0

function check(name, condition, details = '') {
  if (!condition) {
    failed += 1
    console.error(`FAIL: ${name}${details ? ` (${details})` : ''}`)
  } else {
    console.log(`PASS: ${name}`)
  }
}

const qwen = {
  name: 'Qwen3-0.6B-Instruct-2512',
  display_name: 'Qwen3-0.6B-Instruct-2512',
  task_type: 'chat',
  locations: [
    { discovered_metadata_json: { architectures: ['Qwen3ForCausalLM'] } },
  ],
}
const qwenCaps = inferModelCapabilities(qwen)
check('Qwen Instruct infers chat capability', qwenCaps.some((c) => c.id === 'chat'))
check('Qwen Instruct chat source is inferred from name or task', qwenCaps.some((c) => c.id === 'chat' && c.source !== 'unknown'))
check('Qwen Instruct defaults to chat completion', recommendedTestMode(qwen) === 'chat')

const embedding = {
  name: 'bge-large-zh-v1.5',
  task_type: 'embedding',
  locations: [{ discovered_metadata_json: { model_type: 'bert' } }],
}
check('embedding-like model infers embedding', inferModelCapabilities(embedding).some((c) => c.id === 'embedding'))
check('embedding-like model defaults to auto because UI only supports chat/completion', recommendedTestMode(embedding) === 'auto')

const plainCausal = {
  name: 'base-causal',
  task_type: '',
  locations: [{ discovered_metadata_json: { architectures: ['LlamaForCausalLM'] } }],
}
check('causal LLM infers completion', inferModelCapabilities(plainCausal).some((c) => c.id === 'completion'))
check('completion-only model defaults to completion', recommendedTestMode(plainCausal) === 'completion')

const chatFailure = formatTestFailure({
  ok: false,
  mode: 'chat',
  endpoint: 'http://127.0.0.1:8000/v1/chat/completions',
  http_status: 404,
  message: 'not found',
})
check('chat failure mentions endpoint', chatFailure.includes('/v1/chat/completions'), chatFailure)
check('chat failure mentions HTTP status', chatFailure.includes('404'), chatFailure)
check('chat failure is specific to Chat Completion', chatFailure.includes('Chat Completion'), chatFailure)

const stoppedFailure = formatTestFailure({
  ok: false,
  reason_code: 'instance_not_running',
  current_state: 'stopped',
  message: 'instance is stopped',
})
check('not-running failure includes current state', stoppedFailure.includes('stopped'), stoppedFailure)

if (failed > 0) {
  process.exit(1)
}
