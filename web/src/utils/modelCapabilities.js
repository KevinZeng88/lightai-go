const CAPABILITY_LABELS = {
  chat: { zh: '对话', en: 'Chat' },
  completion: { zh: '文本补全', en: 'Completion' },
  embedding: { zh: '向量', en: 'Embedding' },
  rerank: { zh: '重排', en: 'Rerank' },
  vision: { zh: '视觉', en: 'Vision' },
  image_generation: { zh: '图像生成', en: 'Image Generation' },
  asr: { zh: '语音识别', en: 'ASR' },
  tts: { zh: '语音合成', en: 'TTS' },
  classification: { zh: '分类', en: 'Classification' },
  tool_calling: { zh: '工具调用', en: 'Tool Calling' },
  structured_output: { zh: '结构化输出', en: 'Structured Output' },
}

function normalizeText(value) {
  if (Array.isArray(value)) return value.join(' ')
  if (value == null) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function metadataList(model) {
  const metas = []
  if (model?.metadata && typeof model.metadata === 'object') metas.push(model.metadata)
  if (model?.discovered_metadata_json && typeof model.discovered_metadata_json === 'object') metas.push(model.discovered_metadata_json)
  if (Array.isArray(model?.locations)) {
    for (const loc of model.locations) {
      if (loc?.discovered_metadata_json && typeof loc.discovered_metadata_json === 'object') metas.push(loc.discovered_metadata_json)
    }
  }
  return metas
}

function explicitCapabilitySet(model) {
  const out = new Set()
  const values = []
  for (const key of ['capabilities']) {
    const raw = model?.[key]
    if (Array.isArray(raw)) values.push(...raw)
    else if (raw && typeof raw === 'object') values.push(...Object.keys(raw).filter((k) => raw[k]))
    else if (typeof raw === 'string') {
      try {
        const parsed = JSON.parse(raw)
        if (Array.isArray(parsed)) values.push(...parsed)
        else if (parsed && typeof parsed === 'object') values.push(...Object.keys(parsed).filter((k) => parsed[k]))
      } catch {
        values.push(...raw.split(/[,\s]+/))
      }
    }
  }
  for (const value of values) {
    const normalized = normalizeText(value).toLowerCase().replace(/[-\s]/g, '_')
    if (normalized.includes('chat')) out.add('chat')
    if (normalized.includes('completion')) out.add('completion')
    if (normalized.includes('embedding')) out.add('embedding')
    if (normalized.includes('rerank') || normalized.includes('ranker')) out.add('rerank')
    if (normalized.includes('vision') || normalized.includes('vlm') || normalized.includes('multimodal')) out.add('vision')
    if (normalized.includes('tool')) out.add('tool_calling')
    if (normalized.includes('structured') || normalized.includes('json_schema')) out.add('structured_output')
  }
  return out
}

function addCapability(map, id, source, confidence, reason) {
  const current = map.get(id)
  const rank = { high: 3, medium: 2, low: 1 }
  if (!current || rank[confidence] > rank[current.confidence]) {
    map.set(id, { id, label: CAPABILITY_LABELS[id] || { zh: id, en: id }, source, confidence, reason })
  }
}

// inferModelCapabilities returns the capability list for a model.
// Persisted capabilities (model.capabilities) always take precedence.
// When persisted caps are empty, behavior depends on allowInference:
//   - false (default): return empty — no silent regex guessing.
//     Saved models should show "not configured" rather than guessed capabilities.
//   - true: run name/metadata-based regex inference (wizard preview use only).
export function inferModelCapabilities(model, { allowInference = false } = {}) {
  const caps = new Map()

  // Phase 2: Prefer persisted capabilities from backend.
  const persisted = model?.capabilities
  if (Array.isArray(persisted) && persisted.length > 0) {
    const sources = model?.capability_sources || {}
    for (const id of persisted) {
      const source = sources[id] || 'user_override'
      addCapability(caps, id, source, 'high', source === 'scan' ? 'scan metadata' : source === 'inferred' ? 'inferred' : 'user configured')
    }
    return Array.from(caps.values())
  }

  // Inference is opt-in: only used for wizard temporary preview, not for saved models.
  if (!allowInference) {
    return Array.from(caps.values())
  }

  // Legacy path: infer from model fields and scan metadata (wizard-only).
  const explicit = explicitCapabilitySet(model)
  for (const id of explicit) {
    addCapability(caps, id, 'explicit', 'high', 'capabilities')
  }

  const nameText = [
    model?.name,
    model?.display_name,
    model?.path,
    model?.task_type,
    model?.format,
    model?.architecture,
  ].map(normalizeText).join(' ').toLowerCase()

  const metas = metadataList(model)
  const metaText = metas.map(normalizeText).join(' ').toLowerCase()

  if (metas.some((m) => normalizeText(m?.tokenizer_config?.chat_template || m?.chat_template).trim() !== '')) {
    addCapability(caps, 'chat', 'metadata', 'high', 'tokenizer_config.chat_template')
  }
  if (/\b(instruct|chat|assistant|conversation)\b/i.test(nameText)) {
    addCapability(caps, 'chat', 'inferred', 'medium', 'model name')
  }
  if (/\bchat\b/i.test(nameText)) {
    addCapability(caps, 'chat', 'inferred', 'medium', 'task_type')
  }
  if (/forcausallm|causal\s*lm|causal_lm|llm/.test(metaText) || /\bchat\b|\bcompletion\b/.test(nameText)) {
    addCapability(caps, 'completion', 'inferred', 'medium', 'causal LLM')
  }
  if (/embedding|sentence-transformers|bge|e5|gte|embed/.test(nameText + ' ' + metaText)) {
    addCapability(caps, 'embedding', explicit.has('embedding') ? 'explicit' : 'inferred', explicit.has('embedding') ? 'high' : 'medium', 'embedding model pattern')
  }
  if (/rerank|reranker|cross-encoder|cross_encoder/.test(nameText + ' ' + metaText)) {
    addCapability(caps, 'rerank', explicit.has('rerank') ? 'explicit' : 'inferred', explicit.has('rerank') ? 'high' : 'medium', 'rerank model pattern')
  }
  if (/vision|vlm|multimodal|image_text|llava|qwen.*vl/.test(nameText + ' ' + metaText)) {
    addCapability(caps, 'vision', explicit.has('vision') ? 'explicit' : 'inferred', explicit.has('vision') ? 'high' : 'low', 'vision model pattern')
  }

  return Array.from(caps.values())
}

export function recommendedTestMode(model) {
  // Phase 2: Prefer persisted default_test_mode.
  const dtm = model?.default_test_mode
  if (dtm && dtm !== 'auto') return dtm
  // Fall back to inference (wizard preview — opt-in inference enabled).
  const caps = inferModelCapabilities(model, { allowInference: true })
  const ids = new Set(caps.map((c) => c.id))
  if (ids.has('chat')) return 'chat'
  if (ids.has('completion')) return 'completion'
  return 'auto'
}

export function capabilityLabel(capability, locale = 'zh-CN') {
  const label = capability?.label || CAPABILITY_LABELS[capability?.id]
  if (!label) return capability?.id || ''
  return locale === 'en-US' ? label.en : label.zh
}

export function testModeLabel(mode, locale = 'zh-CN') {
  const zh = { auto: '自动', chat: 'Chat Completion', completion: 'Text Completion', embedding: 'Embedding', rerank: 'Rerank' }
  const en = { auto: 'Auto', chat: 'Chat Completion', completion: 'Text Completion', embedding: 'Embedding', rerank: 'Rerank' }
  return (locale === 'en-US' ? en : zh)[mode] || mode
}

export function formatTestFailure(result) {
  const code = result?.reason_code || ''
  const mode = result?.mode === 'completion' ? 'Completion' : result?.mode === 'chat' ? 'Chat Completion' : '模型测试'
  const endpoint = result?.endpoint || (result?.mode === 'completion' ? '/v1/completions' : result?.mode === 'chat' ? '/v1/chat/completions' : '')
  const status = result?.http_status || result?.status || ''
  const requested = result?.requested_model || result?.model || ''
  const available = result?.available_models || []
  const hint = result?.hint || ''

  // Structured reason_code handling (preferred over raw HTTP status inspection).
  switch (code) {
  case 'instance_not_running': {
    const state = result?.current_state || result?.state || ''
    return `实例未运行：当前状态 ${state || 'unknown'}`
  }
  case 'model_id_not_resolved': {
    let msg = '模型 ID 解析失败'
    if (requested) msg += `；请求模型 ${requested}`
    if (available.length > 0) msg += `；可用模型 ${available.join(', ')}`
    if (hint) msg += `；${hint}`
    return msg
  }
  case 'chat_endpoint_failed':
  case 'completion_endpoint_failed':
  case 'embedding_endpoint_failed':
  case 'rerank_endpoint_failed': {
    const backendError = result?.error_body || result?.raw_response || ''
    let msg = `${mode} 请求失败`
    if (endpoint) msg += `：接口 ${endpoint}`
    if (status) msg += `，HTTP 状态 ${status}`
    if (requested) msg += `，请求模型 ${requested}`
    if (available.length > 0) msg += `，可用模型 ${available.join(', ')}`
    if (backendError) {
      const short = typeof backendError === 'string' ? backendError.substring(0, 200) : ''
      if (short) msg += `，后端错误 ${short}`
    }
    if (hint) msg += `，提示：${hint}`
    return msg
  }
  case 'backend_capability_missing':
    return '后端能力未声明，无法确认该模型是否可运行。'
  case 'format_mismatch':
  case 'task_mismatch':
  case 'path_mode_mismatch':
  case 'architecture_blocked':
  case 'not_deployable':
    return result?.message || result?.reason || code
  default:
    break
  }

  // Fallback: use raw HTTP status and endpoint info.
  if (result?.http_status === 404) {
    const backendError = result?.error_body || result?.raw_response || ''
    let msg = `${mode} 请求失败`
    if (endpoint) msg += `：接口 ${endpoint}`
    if (status) msg += `，HTTP 状态 ${status}`
    if (requested) msg += `，请求模型 ${requested}`
    if (available.length > 0) msg += `，可用模型 ${available.join(', ')}`
    if (backendError) {
      const short = typeof backendError === 'string' ? backendError.substring(0, 200) : ''
      if (short) msg += `，后端错误 ${short}`
    }
    if (hint) msg += `，提示：${hint}`
    return msg
  }

  const statusText = status ? `，HTTP 状态 ${status}` : ''
  const endpointText = endpoint ? `接口 ${endpoint}` : '接口未知'
  const resultSummary = result?.message || result?.error || ''
  const summaryText = resultSummary ? `，错误摘要 ${resultSummary}` : ''
  return `${mode} 请求失败：${endpointText}${statusText}${summaryText}`
}
