<template>
  <div class="runtime-parameter-editor">
    <el-collapse v-model="activeSections">
      <el-collapse-item
        v-for="[category, items] in groupedItems"
        :key="category"
        :title="categoryTitle(category)"
        :name="category"
      >
        <div v-for="item in items" :key="item.code" class="param-row">
          <div class="param-header">
            <el-checkbox
              v-model="item.enabled"
              :disabled="readonly || item.required"
              @change="onItemChanged(item)"
            >
              <span class="param-label">{{ itemLabel(item) }}</span>
              <el-tag v-if="item.required" size="small" type="danger" effect="plain" class="param-tag">required</el-tag>
            </el-checkbox>
            <el-tag size="small" type="info">{{ item.kind }}</el-tag>
            <el-tag v-if="showSource && item.sourceLayer" size="small" type="success" effect="plain">{{ item.sourceLayer }}</el-tag>
            <span v-if="item.validationError" class="param-error">{{ item.validationError }}</span>
          </div>

          <div v-if="showSource && item.baseValue !== undefined && item.value !== item.baseValue" class="param-diff">
            <span class="diff-base">{{ formatDisplayValue(item.baseValue) }}</span>
            <span class="diff-arrow">→</span>
            <span class="diff-override">{{ formatDisplayValue(item.value) }}</span>
          </div>

          <el-input
            v-if="isMultiline(item)"
            v-model="item.textValue"
            type="textarea"
            :rows="3"
            :disabled="!item.enabled || readonly"
            class="param-textarea"
            @input="onItemChanged(item)"
          />
          <el-switch
            v-else-if="item.type === 'boolean'"
            v-model="item.boolValue"
            :disabled="!item.enabled || readonly"
            @change="onItemChanged(item)"
          />
          <el-select
            v-else-if="item.type === 'select' || item.type === 'multi_select'"
            v-model="item.selectValue"
            :multiple="item.type === 'multi_select'"
            :disabled="!item.enabled || readonly"
            size="small"
            class="param-input"
            @change="onItemChanged(item)"
          >
            <el-option
              v-for="option in item.options"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <el-input
            v-else
            v-model="item.textValue"
            :placeholder="item.placeholder"
            :disabled="!item.enabled || readonly"
            size="small"
            class="param-input"
            @input="onItemChanged(item)"
          />

          <span class="param-hint">{{ renderHint(item) }}</span>
        </div>
      </el-collapse-item>

      <el-collapse-item v-if="showAdvanced && advancedItems.length" title="Advanced" name="advanced">
        <div v-for="item in advancedItems" :key="item.code" class="param-row">
          <div class="param-header">
            <el-checkbox v-model="item.enabled" :disabled="readonly || item.required" @change="onItemChanged(item)">
              {{ itemLabel(item) }}
            </el-checkbox>
            <el-tag size="small" type="info">{{ item.kind }}</el-tag>
          </div>
          <el-input
            v-if="isMultiline(item)"
            v-model="item.textValue" type="textarea" :rows="3"
            :disabled="!item.enabled || readonly" class="param-textarea" @input="onItemChanged(item)"
          />
          <el-switch
            v-else-if="item.type === 'boolean'"
            v-model="item.boolValue" :disabled="!item.enabled || readonly" @change="onItemChanged(item)"
          />
          <el-select
            v-else-if="item.type === 'select' || item.type === 'multi_select'"
            v-model="item.selectValue" :multiple="item.type === 'multi_select'"
            :disabled="!item.enabled || readonly" size="small" class="param-input" @change="onItemChanged(item)"
          >
            <el-option v-for="option in item.options" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
          <el-input
            v-else v-model="item.textValue" :disabled="!item.enabled || readonly"
            :placeholder="item.placeholder" size="small" class="param-input" @input="onItemChanged(item)"
          />
          <span class="param-hint">{{ renderHint(item) }}</span>
        </div>
      </el-collapse-item>

      <el-collapse-item :title="$t('common.parameterConfiguration') || 'Parameter Configuration'" name="configset">
        <JsonViewer :value="configSetPreview" :title="$t('common.parameterConfiguration') || 'Parameter Configuration'" max-height="420px" :searchable="true" />
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
// Diagnostic/dev-only legacy editor. Normal runtime and deployment flows use
// ConfigEditView plus semantic config projection instead of page-private mappings.
import { computed, reactive, ref, watch } from 'vue'
import JsonViewer from './JsonViewer.vue'

type ConfigItemView = {
  code: string
  category: string
  kind: string
  type: string
  required: boolean
  enabled: boolean
  value: any
  defaultValue: any
  render: Record<string, any>
  extensions: Record<string, any>
  constraints: Record<string, any>
  order: number
  visibility: string
  readonly: boolean
  advanced: boolean
  options: Array<{ label: string, value: any }>
  placeholder: string
  supportLevel: string
  textValue: string
  boolValue: boolean
  selectValue: any
  sourceLayer: string
  baseValue: any
  validationError: string
}

const props = withDefaults(defineProps<{
  modelValue: Record<string, any>
  readonly?: boolean
  backendSchema?: any[]
  vendor?: string
  helpBackend?: string
  helpVersion?: string
  /** layer context: 'backend_runtime' | 'node_backend_runtime' | 'deployment' */
  layer?: string
  /** inherited values from parent layer, for source/diff display */
  baseValues?: Record<string, any>[]
  /** show source/diff column */
  showSource?: boolean
  /** show advanced parameters in a separate collapsible section */
  showAdvanced?: boolean
}>(), {
  readonly: false,
  backendSchema: () => [],
  vendor: 'nvidia',
  helpBackend: '',
  helpVersion: '',
  layer: 'backend_runtime',
  baseValues: () => [],
  showSource: false,
  showAdvanced: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, any>]
  validate: [errors: string[]]
}>()

const activeSections = ref<string[]>(['launcher', 'runtime_env', 'model_runtime'])
const editorItems = reactive<ConfigItemView[]>([])
const advancedItems = computed(() => editorItems.filter(i => i.advanced || i.supportLevel === 'advanced' || i.category === 'advanced'))

const sourceConfigSet = computed(() => {
  const root = props.modelValue || {}
  if (root.config_set && typeof root.config_set === 'object') return root.config_set
  return root
})

const baseValueMap = computed(() => {
  const map = new Map<string, any>()
  for (const bv of props.baseValues || []) {
    if (bv && typeof bv === 'object') {
      for (const [k, v] of Object.entries(bv)) {
        map.set(k, v)
      }
    }
  }
  return map
})

const groupedItems = computed(() => {
  const groups = new Map<string, ConfigItemView[]>()
  const nonAdvanced = editorItems.filter(i => !i.advanced && i.supportLevel !== 'advanced' && i.category !== 'advanced')
  for (const item of nonAdvanced) {
    if (!groups.has(item.category)) groups.set(item.category, [])
    groups.get(item.category)!.push(item)
  }
  return Array.from(groups.entries())
})

const configSetPreview = computed(() => buildConfigSet())

watch(() => sourceConfigSet.value, loadFromModel, { immediate: true, deep: true })

function loadFromModel() {
  const items = sourceConfigSet.value?.items || {}
  editorItems.splice(0, editorItems.length)
  for (const [code, raw] of Object.entries(items)) {
    const item = raw as Record<string, any>
    if (!shouldShowItem(item)) continue
    const value = item.value ?? item.default_value ?? ''
    const baseVal = baseValueMap.value.get(code)
    const required = Boolean(item.required)
    const sourceLayer = item.source ? String(item.source) : (baseVal !== undefined && baseVal !== value ? 'override' : '')
    const render = (item.render && typeof item.render === 'object') ? item.render as Record<string, any> : {}
    const extensions = (item.extensions && typeof item.extensions === 'object') ? item.extensions as Record<string, any> : {}
    const constraints = (item.constraints && typeof item.constraints === 'object') ? item.constraints as Record<string, any> : ((render.constraints && typeof render.constraints === 'object') ? render.constraints as Record<string, any> : {})
    const options = normalizeOptions(render.options || constraints.options || extensions.options || item.options)
    editorItems.push({
      code,
      category: String(item.category || 'model_runtime'),
      kind: String(item.kind || 'cli_arg'),
      type: String(item.type || 'string'),
      required,
      enabled: required ? true : Boolean(item.enabled),
      value,
      defaultValue: item.default_value,
      render,
      extensions,
      constraints,
      order: Number.isFinite(Number(item.order)) ? Number(item.order) : 9999,
      visibility: String(item.visibility || 'visible'),
      readonly: Boolean(item.readonly),
      advanced: Boolean(item.advanced),
      options,
      placeholder: String(render.placeholder || extensions.placeholder || ''),
      supportLevel: String(item.support_level || 'documented'),
      textValue: formatValue(value),
      boolValue: Boolean(value),
      selectValue: item.type === 'multi_select' ? (Array.isArray(value) ? value : []) : value,
      sourceLayer,
      baseValue: baseVal,
      validationError: '',
    })
  }
  editorItems.sort((a, b) => categoryOrder(a.category) - categoryOrder(b.category) || groupName(a).localeCompare(groupName(b)) || a.order - b.order || a.code.localeCompare(b.code))
}

function onItemChanged(item: ConfigItemView) {
  // Auto-enable required items
  if (item.required && !item.enabled) {
    item.enabled = true
  }
  // Validate
  const errs = validateItem(item)
  item.validationError = errs.length > 0 ? errs[0] : ''
  emitOutput()
  emitValidation()
}

function emitOutput() {
  const set = buildConfigSet()
  const parameterValues = editorItems.map((item) => ({
    key: item.code,
    value: parsedValue(item),
    enabled: item.enabled,
  }))

  emit('update:modelValue', {
    ...props.modelValue,
    config_set: set,
    config_overrides: { parameter_values: parameterValues },
  })
}

function emitValidation() {
  const errs: string[] = []
  for (const item of editorItems) {
    const ve = validateItem(item)
    if (ve.length > 0) errs.push(`${itemLabel(item)}: ${ve.join('; ')}`)
  }
  emit('validate', errs)
}

function validateItem(item: ConfigItemView): string[] {
  const errs: string[] = []
  if (!item.enabled) return errs
  const val = item.textValue

  if (item.type === 'integer') {
    const n = Number.parseInt(val, 10)
    if (val !== '' && !Number.isFinite(n)) errs.push('must be an integer')
    else if (Number.isFinite(n)) {
      const constraints = item.constraints
      if (constraints) {
        if (constraints.min !== undefined && n < constraints.min) errs.push(`min ${constraints.min}`)
        if (constraints.max !== undefined && n > constraints.max) errs.push(`max ${constraints.max}`)
      }
    }
  }
  if (item.type === 'number') {
    const n = Number.parseFloat(val)
    if (val !== '' && !Number.isFinite(n)) errs.push('must be a number')
    else if (Number.isFinite(n)) {
      const constraints = item.constraints
      if (constraints) {
        if (constraints.min !== undefined && n < constraints.min) errs.push(`min ${constraints.min}`)
        if (constraints.max !== undefined && n > constraints.max) errs.push(`max ${constraints.max}`)
      }
    }
  }
  return errs
}

function buildConfigSet() {
  const root = sourceConfigSet.value || {}
  const out: Record<string, any> = {
    ...root,
    items: { ...(root.items || {}) },
  }
  for (const item of editorItems) {
    out.items[item.code] = {
      ...(out.items[item.code] || {}),
      value: parsedValue(item),
      enabled: item.enabled,
    }
  }
  return out
}

function parsedValue(item: ConfigItemView) {
  if (item.type === 'boolean') return item.boolValue
  if (item.type === 'select' || item.type === 'multi_select') return item.selectValue
  if (item.type === 'integer') {
    const n = Number.parseInt(item.textValue, 10)
    return Number.isFinite(n) ? n : item.textValue
  }
  if (item.type === 'number') {
    const n = Number.parseFloat(item.textValue)
    return Number.isFinite(n) ? n : item.textValue
  }
  if (item.type === 'array' || item.type === 'lines') {
    return item.textValue.split('\n').map((line) => line.trim()).filter(Boolean)
  }
  if (item.type === 'object') {
    try { return JSON.parse(item.textValue || '{}') } catch { return item.textValue }
  }
  return item.textValue
}

function formatValue(value: any) {
  if (Array.isArray(value)) return value.join('\n')
  if (value && typeof value === 'object') return JSON.stringify(value, null, 2)
  return value == null ? '' : String(value)
}

function formatDisplayValue(value: any) {
  if (value === undefined) return '(none)'
  if (Array.isArray(value)) return value.join(', ')
  if (value && typeof value === 'object') return JSON.stringify(value)
  return value == null ? '(empty)' : String(value)
}

function isMultiline(item: ConfigItemView) {
  return item.type === 'array' || item.type === 'lines' || item.type === 'object'
}

function itemLabel(item: ConfigItemView) {
  return item.render?.label || item.extensions?.label || item.code
}

function renderHint(item: ConfigItemView) {
  const flag = item.render?.flag || item.render?.env_name || ''
  const help = item.render?.help || item.extensions?.help || ''
  return [flag, help, item.supportLevel].filter(Boolean).join(' | ')
}

function groupName(item: ConfigItemView) {
  return String(item.render?.group || item.extensions?.group || item.category)
}

function shouldShowItem(item: Record<string, any>) {
  const visibility = String(item.visibility || '')
  if (visibility === 'internal' || visibility === 'hidden') return false
  const code = String(item.code || '')
  if (code.startsWith('internal.') || code.startsWith('resolver.') || code.startsWith('source_metadata.')) return false
  return true
}

function normalizeOptions(raw: any): Array<{ label: string, value: any }> {
  if (!Array.isArray(raw)) return []
  return raw.map((option) => {
    if (option && typeof option === 'object') {
      const record = option as Record<string, any>
      const value = record.value ?? record.key ?? record.label
      return { label: String(record.label ?? value), value }
    }
    return { label: String(option), value: option }
  })
}

function categoryOrder(category: string) {
  if (category === 'launcher') return 10
  if (category === 'runtime_env') return 20
  if (category === 'model_runtime') return 30
  if (category === 'advanced') return 90
  return 80
}

function categoryTitle(category: string) {
  if (category === 'runtime_env') return 'Runtime Environment'
  if (category === 'model_runtime') return 'Model Runtime'
  if (category === 'launcher') return 'Launcher'
  return category
}
</script>

<style scoped>
.runtime-parameter-editor { width: 100%; }
.param-row { padding: 8px 0; border-bottom: 1px solid var(--el-border-color-lighter); }
.param-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; flex-wrap: wrap; }
.param-label { font-weight: 500; }
.param-tag { margin-left: 4px; }
.param-input, .param-textarea { width: 100%; }
.param-hint { display: block; margin-top: 4px; color: var(--el-text-color-secondary); font-size: 12px; }
.param-error { color: var(--el-color-danger); font-size: 12px; margin-left: auto; }
.param-diff { display: flex; align-items: center; gap: 6px; padding: 2px 0; font-size: 12px; color: var(--el-text-color-secondary); }
.diff-base { color: var(--el-text-color-placeholder); text-decoration: line-through; }
.diff-arrow { color: var(--el-color-warning); }
.diff-override { color: var(--el-color-primary); font-weight: 500; }
</style>
