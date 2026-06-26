<template>
  <div class="human-runtime-form">
    <el-collapse v-model="openGroups">
      <el-collapse-item v-for="group in visibleGroups" :key="group.key" :title="group.label" :name="group.key">
        <div v-for="field in group.fields" :key="field.key" class="param-field">
          <el-form-item :label="field.label">
            <el-input-number
              v-if="field.type === 'number'"
              v-model="fieldValues[field.key]"
              :placeholder="field.placeholder"
              style="width:100%"
              @change="emitOutput"
            />
            <el-switch
              v-else-if="field.type === 'boolean'"
              v-model="fieldValues[field.key]"
              @change="emitOutput"
            />
            <el-input
              v-else
              v-model="fieldValues[field.key]"
              :placeholder="field.placeholder"
              @input="emitOutput"
            />
            <span v-if="field.unit" class="field-unit">{{ field.unit }}</span>
            <span v-if="field.help" class="field-help">{{ field.help }}</span>
          </el-form-item>
        </div>
      </el-collapse-item>

      <el-collapse-item v-if="unmappedConfigItems.length" :title="$t('runtimes.advancedDiagnostics') || 'Advanced'" name="advanced">
        <el-alert type="info" :closable="false" style="margin-bottom:8px">
          {{ $t('runnerConfigs.unmappedFieldsNote') || 'These fields are not mapped to the simplified form. They are preserved but only editable in Advanced mode.' }}
        </el-alert>
        <div v-for="item in unmappedConfigItems" :key="item.code" class="param-field">
          <el-form-item :label="item.code">
            <el-input v-model="advancedValues[item.code]" :disabled="readonly" @input="emitAdvancedOutput" />
          </el-form-item>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getHumanFieldsForBackend,
  isInternalKey,
  buildParamFormOutput,
  type HumanRuntimeField,
  type RuntimeParamFormOutput,
} from '@/utils/runtimeParameterViewModel'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  configSet: Record<string, any> | null
  backendName?: string
  vendor?: string
  readonly?: boolean
}>(), {
  backendName: '',
  vendor: 'nvidia',
  readonly: false,
})

const emit = defineEmits<{
  'update:output': [value: RuntimeParamFormOutput]
}>()

const humanFields = computed(() => getHumanFieldsForBackend(props.backendName))

const fieldValues = reactive<Record<string, any>>({})
const advancedValues = reactive<Record<string, any>>({})
const openGroups = ref<string[]>(['basic', 'gpu', 'backend_common'])

const groupLabels: Record<string, string> = {
  basic: 'Basic',
  gpu: 'GPU',
  backend_common: 'Backend Common',
  backend_vllm: 'vLLM',
  backend_sglang: 'SGLang',
  backend_llamacpp: 'llama.cpp',
}

const visibleGroups = computed(() => {
  const groups = new Map<string, HumanRuntimeField[]>()
  for (const f of humanFields.value) {
    if (!groups.has(f.group)) groups.set(f.group, [])
    groups.get(f.group)!.push(f)
  }
  return Array.from(groups.entries()).map(([key, fields]) => ({
    key,
    label: groupLabels[key] || key,
    fields,
  }))
})

interface ConfigItemSimple { code: string; value: any }

const unmappedConfigItems = computed((): ConfigItemSimple[] => {
  if (!props.configSet?.items) return []
  const items: ConfigItemSimple[] = []
  const knownKeys = new Set(humanFields.value.flatMap(f => f.mapsTo.map(m => m.internalKey)))
  for (const [code, raw] of Object.entries(props.configSet.items)) {
    if (isInternalKey(code)) continue
    if (knownKeys.has(code)) continue
    const item = raw as any
    items.push({ code, value: item?.value ?? item?.default_value ?? '' })
  }
  return items
})

// Initialize field values from config set
watch(() => props.configSet, initValues, { immediate: true })

function initValues() {
  for (const f of humanFields.value) {
    const val = resolveFieldValue(f)
    if (val !== undefined) fieldValues[f.key] = val
    else if (f.defaultValue !== undefined && fieldValues[f.key] === undefined) fieldValues[f.key] = f.defaultValue
  }
  // Init advanced values
  for (const item of unmappedConfigItems.value) {
    const val = item.value
    if (val !== undefined && advancedValues[item.code] === undefined) {
      advancedValues[item.code] = typeof val === 'string' ? val : String(val)
    }
  }
}

function resolveFieldValue(field: HumanRuntimeField): unknown {
  for (const m of field.mapsTo) {
    const v = resolveInternalKey(m.internalKey)
    if (v !== undefined && v !== null && v !== '') {
      if (m.transform === 'number') {
        const n = Number(v)
        if (!isNaN(n)) return n
      }
      return v
    }
  }
  return undefined
}

function resolveInternalKey(key: string): unknown {
  if (!props.configSet?.items) return undefined
  const item = props.configSet.items[key]
  if (!item) return undefined
  return item.value ?? item.default_value
}

function emitOutput() {
  const fields = humanFields.value.map(f => ({ ...f, value: fieldValues[f.key] }))
  emit('update:output', buildParamFormOutput(fields))
}

function emitAdvancedOutput() {
  emitOutput()
}
</script>

<style scoped>
.human-runtime-form { width: 100%; }
.param-field { padding: 4px 0; }
.field-unit { color: var(--el-text-color-secondary); font-size: 12px; margin-left: 8px; }
.field-help { color: var(--el-text-color-secondary); font-size: 12px; display: block; margin-top: 2px; }
</style>
