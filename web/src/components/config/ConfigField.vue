<template>
  <div class="config-field" :class="{ disabled: !field.enabled, readonly: readonly || field.readonly }">
    <div class="field-header">
      <el-checkbox
        v-if="field.has_enable && !field.required"
        v-model="field.enabled"
        :disabled="readonly || field.readonly"
        @change="$emit('change')"
      />
      <span class="field-label">{{ field.label }}</span>
      <el-tag v-if="field.required" size="small" type="danger" effect="plain">required</el-tag>
    </div>
    <div class="field-control">
      <el-switch
        v-if="field.widget === 'boolean' || field.type === 'boolean'"
        v-model="field.value"
        :disabled="!field.enabled || readonly || field.readonly"
        @change="$emit('change')"
      />
      <el-select
        v-else-if="field.widget === 'select' || field.widget === 'multi_select'"
        v-model="field.value"
        :multiple="field.widget === 'multi_select'"
        :disabled="!field.enabled || readonly || field.readonly"
        size="small"
        class="field-input"
        @change="$emit('change')"
      >
        <el-option v-for="option in field.options || []" :key="String(option.value)" :label="option.label" :value="option.value" />
      </el-select>
      <el-input-number
        v-else-if="field.widget === 'number' || field.type === 'integer' || field.type === 'number'"
        v-model="field.value"
        :disabled="!field.enabled || readonly || field.readonly"
        size="small"
        class="field-input"
        @change="$emit('change')"
      />
      <el-input
        v-else-if="field.widget === 'textarea' || field.widget === 'raw_json'"
        v-model="textValue"
        type="textarea"
        :rows="field.widget === 'raw_json' ? 6 : 3"
        :disabled="!field.enabled || readonly || field.readonly"
        @input="onTextInput"
      />
      <el-input
        v-else-if="field.widget === 'string_list' || field.widget === 'device_list'"
        v-model="textValue"
        type="textarea"
        :rows="2"
        :disabled="!field.enabled || readonly || field.readonly"
        @input="onListInput"
      />
      <el-input
        v-else-if="field.widget === 'key_value_list'"
        v-model="textValue"
        type="textarea"
        :rows="3"
        :disabled="!field.enabled || readonly || field.readonly"
        placeholder="KEY=value"
        @input="onKeyValueInput"
      />
      <el-input
        v-else
        v-model="field.value"
        :disabled="!field.enabled || readonly || field.readonly"
        size="small"
        class="field-input"
        @input="$emit('change')"
      />
    </div>
    <div v-if="field.help" class="field-help">{{ field.help }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ConfigEditField } from '@/utils/configEditView'

const props = defineProps<{
  field: ConfigEditField
  readonly?: boolean
}>()

const emit = defineEmits<{ change: [] }>()

const textValue = computed({
  get() {
    if (props.field.widget === 'raw_json') return JSON.stringify(props.field.value ?? {}, null, 2)
    if (Array.isArray(props.field.value)) return props.field.value.join('\n')
    if (props.field.value && typeof props.field.value === 'object') {
      return Object.entries(props.field.value).map(([k, v]) => `${k}=${v}`).join('\n')
    }
    return props.field.value == null ? '' : String(props.field.value)
  },
  set(value: string) {
    props.field.value = value
  },
})

function onTextInput(value: string) {
  if (props.field.widget === 'raw_json') {
    try {
      props.field.value = JSON.parse(value || '{}')
    } catch {
      props.field.value = value
    }
  } else {
    props.field.value = value
  }
  emit('change')
}

function onListInput(value: string) {
  props.field.value = value.split('\n').map(v => v.trim()).filter(Boolean)
  emit('change')
}

function onKeyValueInput(value: string) {
  const out: Record<string, string> = {}
  for (const line of value.split('\n')) {
    const idx = line.indexOf('=')
    if (idx <= 0) continue
    out[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  }
  props.field.value = out
  emit('change')
}
</script>

<style scoped>
.config-field {
  display: grid;
  grid-template-columns: minmax(180px, 260px) minmax(0, 1fr);
  gap: 8px 14px;
  align-items: start;
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.field-header { display: flex; align-items: center; gap: 8px; min-height: 28px; }
.field-label { font-weight: 500; overflow-wrap: anywhere; }
.field-input { width: 100%; }
.field-help { grid-column: 2; color: var(--el-text-color-secondary); font-size: 12px; }
.disabled .field-label { color: var(--el-text-color-secondary); }
@media (max-width: 760px) {
  .config-field { grid-template-columns: 1fr; }
  .field-help { grid-column: 1; }
}
</style>
