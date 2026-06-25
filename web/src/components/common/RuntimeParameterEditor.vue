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
            <el-checkbox v-model="item.enabled" :disabled="readonly" @change="emitOutput">
              {{ itemLabel(item) }}
            </el-checkbox>
            <el-tag size="small" type="info">{{ item.kind }}</el-tag>
          </div>

          <el-input
            v-if="isMultiline(item)"
            v-model="item.textValue"
            type="textarea"
            :rows="3"
            :disabled="!item.enabled || readonly"
            class="param-textarea"
            @input="emitOutput"
          />
          <el-switch
            v-else-if="item.type === 'boolean'"
            v-model="item.boolValue"
            :disabled="!item.enabled || readonly"
            @change="emitOutput"
          />
          <el-input
            v-else
            v-model="item.textValue"
            :disabled="!item.enabled || readonly"
            size="small"
            class="param-input"
            @input="emitOutput"
          />

          <span class="param-hint">{{ renderHint(item) }}</span>
        </div>
      </el-collapse-item>

      <el-collapse-item title="ConfigSet" name="configset">
        <JsonViewer :value="configSetPreview" title="ConfigSet" max-height="420px" :searchable="true" />
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import JsonViewer from './JsonViewer.vue'

type ConfigItemView = {
  code: string
  category: string
  kind: string
  type: string
  enabled: boolean
  value: any
  defaultValue: any
  render: Record<string, any>
  supportLevel: string
  textValue: string
  boolValue: boolean
}

const props = withDefaults(defineProps<{
  modelValue: Record<string, any>
  readonly?: boolean
  backendSchema?: any[]
  vendor?: string
  helpBackend?: string
  helpVersion?: string
}>(), {
  readonly: false,
  backendSchema: () => [],
  vendor: 'nvidia',
  helpBackend: '',
  helpVersion: '',
})

const emit = defineEmits(['update:modelValue'])
const activeSections = ref<string[]>(['launcher', 'runtime_env', 'model_runtime'])
const editorItems = reactive<ConfigItemView[]>([])

const sourceConfigSet = computed(() => {
  const root = props.modelValue || {}
  if (root.config_set && typeof root.config_set === 'object') return root.config_set
  return root
})

const groupedItems = computed(() => {
  const groups = new Map<string, ConfigItemView[]>()
  for (const item of editorItems) {
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
    const value = item.value ?? item.default_value ?? ''
    editorItems.push({
      code,
      category: String(item.category || 'model_runtime'),
      kind: String(item.kind || 'cli_arg'),
      type: String(item.type || 'string'),
      enabled: Boolean(item.enabled),
      value,
      defaultValue: item.default_value,
      render: (item.render && typeof item.render === 'object') ? item.render : {},
      supportLevel: String(item.support_level || 'documented'),
      textValue: formatValue(value),
      boolValue: Boolean(value),
    })
  }
  editorItems.sort((a, b) => a.category.localeCompare(b.category) || a.code.localeCompare(b.code))
}

function emitOutput() {
  const set = buildConfigSet()
  emit('update:modelValue', {
    ...props.modelValue,
    config_set: set,
    config_overrides: {
      parameter_values: editorItems.map((item) => ({
        key: item.code,
        value: parsedValue(item),
        enabled: item.enabled,
      })),
    },
  })
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

function isMultiline(item: ConfigItemView) {
  return item.type === 'array' || item.type === 'lines' || item.type === 'object'
}

function itemLabel(item: ConfigItemView) {
  return item.code
}

function renderHint(item: ConfigItemView) {
  const flag = item.render?.flag || item.render?.env_name || ''
  return [flag, item.supportLevel].filter(Boolean).join(' | ')
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
.param-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.param-input, .param-textarea { width: 100%; }
.param-hint { display: block; margin-top: 4px; color: var(--el-text-color-secondary); font-size: 12px; }
</style>
