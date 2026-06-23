<template>
  <div class="health-check-editor">
    <el-form label-position="top" size="small">
      <el-form-item :label="t('healthCheck.path') || 'Path'">
        <el-input v-model="model.path" placeholder="/health" />
      </el-form-item>
      <el-form-item :label="t('healthCheck.method') || 'Method'">
        <el-select v-model="model.method" style="width:100%">
          <el-option label="GET" value="GET" />
          <el-option label="POST" value="POST" />
        </el-select>
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="8">
          <el-form-item :label="t('healthCheck.timeoutSeconds') || 'Timeout (s)'">
            <el-input-number v-model="model.timeout_seconds" :min="1" :max="300" style="width:100%" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item :label="t('healthCheck.intervalSeconds') || 'Interval (s)'">
            <el-input-number v-model="model.interval_seconds" :min="1" :max="600" style="width:100%" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item :label="t('healthCheck.expectedStatus') || 'Expected Status'">
            <el-input-number v-model="model.expected_status" :min="100" :max="599" style="width:100%" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item :label="t('healthCheck.expectedBodyContains') || 'Expected Body Contains'">
        <el-input v-model="model.expected_body_contains" placeholder="" />
      </el-form-item>
      <el-form-item :label="t('healthCheck.readinessGraceSeconds') || 'Readiness Grace (s)'">
        <el-input-number v-model="model.readiness_grace_seconds" :min="0" :max="600" style="width:100%" />
      </el-form-item>
    </el-form>
    <div class="health-check-editor__raw">
      <el-collapse>
        <el-collapse-item :title="t('runnerConfigs.advancedJson') || 'Raw JSON'">
          <el-input v-model="rawJson" type="textarea" :rows="4" @change="onRawChange" />
          <div v-if="rawError" class="health-check-editor__error">{{ rawError }}</div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface HealthCheckModel {
  path: string
  method: string
  timeout_seconds: number
  interval_seconds: number
  expected_status: number
  expected_body_contains: string
  readiness_grace_seconds: number
  [key: string]: unknown
}

const props = defineProps<{
  modelValue: Record<string, unknown>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const model = ref<HealthCheckModel>(fromJSON(props.modelValue))
const rawError = ref('')

const rawJson = computed({
  get: () => JSON.stringify(toJSON(model.value), null, 2),
  set: () => {},
})

function fromJSON(obj: Record<string, unknown>): HealthCheckModel {
  return {
    path: String(obj?.path || '/health'),
    method: String(obj?.method || 'GET'),
    timeout_seconds: Number(obj?.timeout_seconds || 5),
    interval_seconds: Number(obj?.interval_seconds || 10),
    expected_status: Number(obj?.expected_status || 200),
    expected_body_contains: String(obj?.expected_body_contains || ''),
    readiness_grace_seconds: Number(obj?.readiness_grace_seconds || 30),
  }
}

function toJSON(m: HealthCheckModel): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  if (m.path) out.path = m.path
  if (m.method && m.method !== 'GET') out.method = m.method
  if (m.timeout_seconds && m.timeout_seconds !== 5) out.timeout_seconds = m.timeout_seconds
  if (m.interval_seconds && m.interval_seconds !== 10) out.interval_seconds = m.interval_seconds
  if (m.expected_status && m.expected_status !== 200) out.expected_status = m.expected_status
  if (m.expected_body_contains) out.expected_body_contains = m.expected_body_contains
  if (m.readiness_grace_seconds && m.readiness_grace_seconds !== 30) out.readiness_grace_seconds = m.readiness_grace_seconds
  return out
}

watch(model, (val) => {
  rawError.value = ''
  emit('update:modelValue', toJSON(val))
}, { deep: true })

watch(() => props.modelValue, (val) => {
  model.value = fromJSON(val)
}, { deep: true })

function onRawChange(val: string) {
  try {
    const parsed = JSON.parse(val || '{}')
    model.value = fromJSON(parsed)
    rawError.value = ''
  } catch (e: any) {
    rawError.value = e?.message || 'Invalid JSON'
  }
}
</script>

<style scoped>
.health-check-editor {
  padding: 4px 0;
}
.health-check-editor__raw {
  margin-top: 8px;
}
.health-check-editor__error {
  color: var(--el-color-danger);
  font-size: 12px;
  margin-top: 4px;
}
</style>
