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
      <!-- Boolean switch -->
      <el-switch
        v-if="field.widget === 'boolean' || field.type === 'boolean'"
        v-model="field.value"
        :disabled="!field.enabled || readonly || field.readonly"
        @change="$emit('change')"
      />

      <!-- Select / multi-select -->
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

      <!-- Number -->
      <el-input-number
        v-else-if="field.widget === 'number' || field.type === 'integer' || field.type === 'number'"
        v-model="field.value"
        :disabled="!field.enabled || readonly || field.readonly"
        size="small"
        class="field-input"
        @change="$emit('change')"
      />

      <!-- Raw JSON textarea -->
      <el-input
        v-else-if="field.widget === 'raw_json'"
        v-model="textValue"
        type="textarea"
        :rows="6"
        :disabled="!field.enabled || readonly || field.readonly"
        @input="onTextInput"
      />

      <!-- Key-value table (structured, replaces key_value_list textarea) -->
      <div v-else-if="field.widget === 'key_value_table'" class="kv-table-wrap">
        <el-table :data="kvRows" border size="small" max-height="260px">
          <el-table-column prop="key" :label="$t('configEdit.fields.key')" width="200" />
          <el-table-column prop="value" :label="$t('configEdit.fields.value')">
            <template #default="{ row, $index }">
              <template v-if="readonly || field.readonly">{{ row.value }}</template>
              <el-input v-else v-model="row.value" size="small" @input="onKeyValueTableChange" />
            </template>
          </el-table-column>
          <el-table-column v-if="!(readonly || field.readonly)" width="60">
            <template #default="{ $index }">
              <el-button size="small" type="danger" :icon="'Delete'" circle @click="removeKvRow($index)" />
            </template>
          </el-table-column>
        </el-table>
        <el-button v-if="!(readonly || field.readonly)" size="small" style="margin-top:6px" @click="addKvRow">
          + {{ $t('configEdit.actions.addRow') }}
        </el-button>
      </div>

      <!-- Device table -->
      <div v-else-if="field.widget === 'device_table'" class="kv-table-wrap">
        <el-table :data="deviceRows" border size="small" max-height="260px">
          <el-table-column prop="host_path" :label="$t('configEdit.fields.hostPath')" />
          <el-table-column prop="container_path" :label="$t('configEdit.fields.containerPath')" />
          <el-table-column prop="readonly" :label="$t('configEdit.fields.readonly')" width="80">
            <template #default="{ row }">{{ row.readonly ? $t('common.yes') : $t('common.no') }}</template>
          </el-table-column>
          <el-table-column v-if="!(readonly || field.readonly)" width="60">
            <template #default="{ $index }">
              <el-button size="small" type="danger" circle @click="removeDeviceRow($index)" />
            </template>
          </el-table-column>
        </el-table>
        <el-button v-if="!(readonly || field.readonly)" size="small" style="margin-top:6px" @click="addDeviceRow">
          + {{ $t('configEdit.actions.addRow') }}
        </el-button>
      </div>

      <!-- Mount form (model_mount) -->
      <div v-else-if="field.widget === 'mount_form'" class="mount-form">
        <div class="mount-row">
          <span class="mount-label">{{ $t('configEdit.fields.containerPath') }}:</span>
          <el-input v-if="!(readonly || field.readonly)" v-model="mountData.container_path" size="small" @input="onMountChange" />
          <span v-else>{{ mountData.container_path || '-' }}</span>
        </div>
        <div class="mount-row">
          <span class="mount-label">{{ $t('configEdit.fields.hostPath') }}:</span>
          <el-input v-if="!(readonly || field.readonly)" v-model="mountData.host_path" size="small" @input="onMountChange" />
          <span v-else>{{ mountData.host_path || '-' }}</span>
        </div>
        <div class="mount-row">
          <span class="mount-label">{{ $t('configEdit.fields.readonly') }}:</span>
          <el-switch v-if="!(readonly || field.readonly)" v-model="mountData.readonly" size="small" @change="onMountChange" />
          <span v-else>{{ mountData.readonly ? $t('common.yes') : $t('common.no') }}</span>
        </div>
      </div>

      <!-- Health check form -->
      <div v-else-if="field.widget === 'health_check_form'" class="health-form">
        <div class="health-row">
          <span class="health-label">{{ $t('configEdit.fields.healthPath') }}:</span>
          <el-input v-model="healthData.path" :disabled="readonly || field.readonly" size="small" @input="onHealthChange" />
        </div>
        <div class="health-row">
          <span class="health-label">{{ $t('configEdit.fields.healthPort') }}:</span>
          <el-input-number v-model="healthData.port" :disabled="readonly || field.readonly" size="small" @change="onHealthChange" />
        </div>
        <div class="health-row">
          <span class="health-label">{{ $t('configEdit.fields.healthTimeout') }}:</span>
          <el-input-number v-model="healthData.timeout" :disabled="readonly || field.readonly" size="small" @change="onHealthChange" />
        </div>
        <div class="health-row">
          <span class="health-label">{{ $t('configEdit.fields.healthInterval') }}:</span>
          <el-input-number v-model="healthData.interval" :disabled="readonly || field.readonly" size="small" @change="onHealthChange" />
        </div>
        <div class="health-row">
          <span class="health-label">{{ $t('configEdit.fields.healthRetries') }}:</span>
          <el-input-number v-model="healthData.retries" :disabled="readonly || field.readonly" size="small" @change="onHealthChange" />
        </div>
      </div>

      <!-- Port form with {{container_port}} handling -->
      <div v-else-if="field.widget === 'port_form'" class="port-form">
        <div v-if="isTemplatePort" class="readonly-hint">
          {{ $t('configEdit.placeholders.deploymentContainerPort') }}
        </div>
        <div v-else class="port-row">
          <span class="port-label">{{ $t('configEdit.fields.containerPort') }}:</span>
          <el-input-number v-if="!(readonly || field.readonly)" v-model="portData.container_port" size="small" @change="onPortChange" />
          <span v-else>{{ portData.container_port || '-' }}</span>
          <span class="port-label">{{ $t('configEdit.fields.hostPort') }}:</span>
          <el-input-number v-if="!(readonly || field.readonly)" v-model="portData.host_port" size="small" @change="onPortChange" />
          <span v-else>{{ portData.host_port || '-' }}</span>
        </div>
      </div>

      <!-- Readonly summary for advanced/capabilities fields -->
      <div v-else-if="field.widget === 'readonly_summary'" class="readonly-summary">
        <span v-if="summaryText">{{ summaryText }}</span>
        <el-tag v-else size="small" type="info">{{ $t('configEdit.fields.noDetails') }}</el-tag>
      </div>

      <!-- Textarea for generic strings and legacy list widgets -->
      <el-input
        v-else-if="field.widget === 'textarea' || field.widget === 'string_list' || field.widget === 'device_list' || field.widget === 'key_value_list'"
        v-model="textValue"
        type="textarea"
        :rows="field.widget === 'key_value_list' ? 3 : 2"
        :disabled="!field.enabled || readonly || field.readonly"
        :placeholder="field.widget === 'key_value_list' ? 'KEY=value' : ''"
        @input="onLegacyListInput"
      />

      <!-- Default: plain input, but handle objects gracefully -->
      <div v-else class="default-value">
        <el-input
          v-if="isScalarValue"
          v-model="field.value"
          :disabled="!field.enabled || readonly || field.readonly"
          size="small"
          class="field-input"
          @input="$emit('change')"
        />
        <span v-else class="readonly-hint">{{ formattedDisplayValue }}</span>
      </div>
    </div>
    <div v-if="field.help" class="field-help">{{ field.help }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { ConfigEditField } from '@/utils/configEditView'

const TEMPLATE_PORT_MARKER = '{{container_port}}'

const props = defineProps<{
  field: ConfigEditField
  readonly?: boolean
}>()

const emit = defineEmits<{ change: [] }>()

// -- Scalar check for default widget --
const isScalarValue = computed(() => {
  const v = props.field.value
  if (v === null || v === undefined) return true
  const t = typeof v
  return t === 'string' || t === 'number' || t === 'boolean'
})

const formattedDisplayValue = computed(() => {
  const v = props.field.value
  if (v === null || v === undefined) return '-'
  if (typeof v === 'object') {
    try { return JSON.stringify(v) } catch { return String(v) }
  }
  return String(v)
})

// -- Text value (for textarea / raw_json / string_list widgets) --
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

function onLegacyListInput(value: string) {
  const w = props.field.widget
  if (w === 'key_value_list') {
    const out: Record<string, string> = {}
    for (const line of value.split('\n')) {
      const idx = line.indexOf('=')
      if (idx <= 0) continue
      out[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
    }
    props.field.value = out
  } else {
    // string_list, device_list, or other list widgets
    props.field.value = value.split('\n').map(v => v.trim()).filter(Boolean)
  }
  emit('change')
}

// -- Key-value table --
const kvRows = ref<{ key: string; value: string }[]>([])

function initKvRows() {
  const v = props.field.value
  if (Array.isArray(v)) {
    // may be array of {key, value} pairs
    kvRows.value = v.map((e: any) => ({
      key: e?.key ?? e?.Key ?? '',
      value: e?.value ?? e?.Value ?? String(e ?? ''),
    }))
  } else if (v && typeof v === 'object') {
    kvRows.value = Object.entries(v as Record<string, any>).map(([key, val]) => ({
      key,
      value: val === null || val === undefined ? '' : String(val),
    }))
  } else {
    kvRows.value = []
  }
}

function onKeyValueTableChange() {
  props.field.value = Object.fromEntries(kvRows.value.map((r: { key: string; value: string }) => [r.key, r.value]))
  emit('change')
}

function addKvRow() {
  kvRows.value.push({ key: '', value: '' })
  onKeyValueTableChange()
}

function removeKvRow(index: number) {
  kvRows.value.splice(index, 1)
  onKeyValueTableChange()
}

// -- Device table --
const deviceRows = ref<any[]>([])

function initDeviceRows() {
  const v = props.field.value
  if (Array.isArray(v)) {
    deviceRows.value = v.map((d: any) => ({
      host_path: d?.host_path ?? d?.HostPath ?? d?.source ?? '',
      container_path: d?.container_path ?? d?.ContainerPath ?? d?.target ?? '',
      readonly: Boolean(d?.readonly ?? d?.Readonly ?? false),
    }))
  } else {
    deviceRows.value = []
  }
}

function onDeviceTableChange() {
  props.field.value = deviceRows.value.map((d: { host_path: string; container_path: string }) => ({
    host_path: d.host_path,
    container_path: d.container_path,
  }))
  emit('change')
}

function addDeviceRow() {
  deviceRows.value.push({ host_path: '', container_path: '', readonly: false })
  onDeviceTableChange()
}

function removeDeviceRow(index: number) {
  deviceRows.value.splice(index, 1)
  onDeviceTableChange()
}

// -- Mount form --
const mountData = reactive<{ container_path: string; host_path: string; readonly: boolean }>({
  container_path: '',
  host_path: '',
  readonly: false,
})

function initMountData() {
  const v = props.field.value
  if (v && typeof v === 'object') {
    mountData.container_path = (v as any).container_path ?? (v as any).containerPath ?? ''
    mountData.host_path = (v as any).host_path ?? (v as any).hostPath ?? (v as any).source_path ?? (v as any).source ?? ''
    mountData.readonly = Boolean((v as any).readonly ?? (v as any).Readonly ?? false)
  }
}

function onMountChange() {
  props.field.value = { ...mountData }
  emit('change')
}

// -- Health check form --
const healthData = reactive<{ path: string; port: number; timeout: number; interval: number; retries: number }>({
  path: '',
  port: 0,
  timeout: 30,
  interval: 10,
  retries: 3,
})

function initHealthData() {
  const v = props.field.value
  if (v && typeof v === 'object') {
    healthData.path = (v as any).path ?? (v as any).Path ?? ''
    healthData.port = Number((v as any).port ?? (v as any).Port ?? 0)
    healthData.timeout = Number((v as any).timeout ?? (v as any).Timeout ?? 30)
    healthData.interval = Number((v as any).interval ?? (v as any).Interval ?? 10)
    healthData.retries = Number((v as any).retries ?? (v as any).Retries ?? 3)
  }
}

function onHealthChange() {
  props.field.value = { ...healthData }
  emit('change')
}

// -- Port form --
const portData = reactive<{ container_port: number | null; host_port: number | null }>({
  container_port: null,
  host_port: null,
})

const isTemplatePort = computed(() => {
  const v = props.field.value
  return typeof v === 'string' && v.includes('{{')
})

function initPortData() {
  const v = props.field.value
  if (v && typeof v === 'object') {
    portData.container_port = (v as any).container_port ?? (v as any).containerPort ?? null
    portData.host_port = (v as any).host_port ?? (v as any).hostPort ?? null
  } else if (typeof v === 'number') {
    portData.container_port = v
  }
}

function onPortChange() {
  props.field.value = { ...portData }
  emit('change')
}

// -- Readonly summary --
const summaryText = computed(() => {
  const v = props.field.value
  if (v === null || v === undefined) return ''
  if (typeof v === 'object') {
    // For capabilities-like objects, extract a readable summary.
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      const entries = Object.entries(v as Record<string, any>)
      if (entries.length === 0) return ''
      // If all values are booleans, list the true ones.
      const allBool = entries.every(([, val]) => typeof val === 'boolean')
      if (allBool) {
        const trues = entries.filter(([, val]) => val).map(([k]) => k)
        return trues.length > 0 ? trues.join(', ') : ''
      }
      // Otherwise show key count.
      return `${entries.length} items`
    }
    return JSON.stringify(v)
  }
  return String(v)
})

// -- Initialize reactive data from field.value --
function initAll() {
  if (props.field.widget === 'key_value_table') initKvRows()
  else if (props.field.widget === 'device_table') initDeviceRows()
  else if (props.field.widget === 'mount_form') initMountData()
  else if (props.field.widget === 'health_check_form') initHealthData()
  else if (props.field.widget === 'port_form') initPortData()
}

// Watch for field changes (e.g. when editView loads)
watch(() => props.field.value, () => {
  initAll()
}, { immediate: false })

initAll()
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

.kv-table-wrap, .mount-form, .health-form, .port-form, .readonly-summary, .default-value {
  width: 100%;
}
.mount-row, .health-row, .port-row {
  display: flex; align-items: center; gap: 8px; margin-bottom: 4px;
}
.mount-label, .health-label, .port-label {
  font-weight: 500; min-width: 100px; font-size: 13px;
}
.readonly-hint {
  color: var(--el-text-color-secondary); font-style: italic; font-size: 13px;
}
.readonly-summary {
  color: var(--el-text-color-regular); font-size: 13px; word-break: break-word;
}

@media (max-width: 760px) {
  .config-field { grid-template-columns: 1fr; }
  .field-help { grid-column: 1; }
}
</style>
