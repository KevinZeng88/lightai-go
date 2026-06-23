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
// Phase 3: Without persisted caps and without allowInference, return empty.
const qwenCaps = inferModelCapabilities(qwen)
check('Qwen without persisted caps returns empty (inference disabled by default)', qwenCaps.length === 0)
// With allowInference (wizard preview), inference works.
const qwenInferred = inferModelCapabilities(qwen, { allowInference: true })
check('Qwen with allowInference infers chat capability', qwenInferred.some((c) => c.id === 'chat'))
check('Qwen with allowInference chat source is inferred', qwenInferred.some((c) => c.id === 'chat' && c.source !== 'unknown'))
// recommendedTestMode uses allowInference:true internally (wizard context).
check('Qwen recommendedTestMode is chat', recommendedTestMode(qwen) === 'chat')

const embedding = {
  name: 'bge-large-zh-v1.5',
  task_type: 'embedding',
  locations: [{ discovered_metadata_json: { model_type: 'bert' } }],
}
check('embedding-like model infers embedding', inferModelCapabilities(embedding, { allowInference: true }).some((c) => c.id === 'embedding'))
check('embedding-like model defaults to auto because UI only supports chat/completion', recommendedTestMode(embedding) === 'auto')

const plainCausal = {
  name: 'base-causal',
  task_type: '',
  locations: [{ discovered_metadata_json: { architectures: ['LlamaForCausalLM'] } }],
}
check('causal LLM infers completion', inferModelCapabilities(plainCausal, { allowInference: true }).some((c) => c.id === 'completion'))
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

// Phase 2: Persisted capabilities tests.
const qwenWithCapabilities = {
  ...qwen,
  capabilities: ['chat', 'completion'],
  capability_sources: { chat: 'user_override', completion: 'scan' },
  default_test_mode: 'chat',
}
const qwenPersistedCaps = inferModelCapabilities(qwenWithCapabilities)
check('persisted capabilities override inference', qwenPersistedCaps.some((c) => c.id === 'chat' && c.source === 'user_override'))
check('persisted capability source is user_override for chat', qwenPersistedCaps.find((c) => c.id === 'chat')?.source === 'user_override')
check('persisted capability source is scan for completion', qwenPersistedCaps.find((c) => c.id === 'completion')?.source === 'scan')
check('persisted default_test_mode=chat returns chat', recommendedTestMode(qwenWithCapabilities) === 'chat')

// Phase 3: Empty persisted capabilities without allowInference returns empty.
const qwenEmptyCaps = { ...qwen, capabilities: [] }
check('empty persisted caps without allowInference returns empty', inferModelCapabilities(qwenEmptyCaps).length === 0)
// With allowInference, empty persisted still falls back to inference.
check('empty persisted caps with allowInference falls back', inferModelCapabilities(qwenEmptyCaps, { allowInference: true }).some((c) => c.id === 'chat'))

// default_test_mode='completion' returns completion even without capability.
const qwenCompletionTestMode = { ...qwen, capabilities: [], default_test_mode: 'completion' }
check('default_test_mode=completion returns completion', recommendedTestMode(qwenCompletionTestMode) === 'completion')

// default_test_mode='auto' falls back to inference.
const qwenAutoTestMode = { ...qwen, default_test_mode: 'auto' }
check('default_test_mode=auto falls back to inference', recommendedTestMode(qwenAutoTestMode) === 'chat')

if (failed > 0) {
  process.exit(1)
}
