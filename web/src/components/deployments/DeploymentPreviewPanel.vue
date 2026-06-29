<template>
  <div>
    <div v-if="!props.previewData && !props.loading" style="text-align:center;padding:40px">
      <el-button type="primary" @click="$emit('preview')" :loading="props.loading">
        {{ $t('deployments.previewRunPlan') }}
      </el-button>
    </div>

    <div v-loading="props.loading" v-if="props.previewData">
      <el-descriptions :column="1" border size="small" style="margin-bottom:16px">
        <el-descriptions-item :label="$t('deployments.canRun')">
          <el-tag :type="props.previewData.can_run ? 'success' : 'danger'">
            {{ props.previewData.can_run ? t('common.yes') : t('common.no') }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="props.previewData.preflight?.errors?.length" :label="$t('common.error')">
          <div v-for="(e, i) in props.previewData.preflight.errors" :key="i" style="color:var(--el-color-danger)">
            [{{ e.code }}] {{ preflightMessage(e) }}
          </div>
        </el-descriptions-item>
        <el-descriptions-item v-if="props.previewData.preflight?.warnings?.length" :label="$t('deployments.warnings')">
          <div v-for="(w, i) in props.previewData.preflight.warnings" :key="i" style="color:var(--el-color-warning)">
            [{{ w.code }}] {{ preflightMessage(w) }}
          </div>
        </el-descriptions-item>
        <el-descriptions-item v-if="props.previewData.lint?.findings?.length" :label="$t('deployments.lintFindings')">
          <div v-for="(f, i) in props.previewData.lint.findings" :key="i">
            <el-tag :type="f.severity === 'error' ? 'danger' : f.severity === 'warning' ? 'warning' : 'info'" size="small">
              {{ translateSeverity(f.severity) }}
            </el-tag>
            {{ lintMessage(f) }}
          </div>
        </el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">{{ $t('deployments.dockerPreview') }}</el-divider>
      <el-input :model-value="props.previewData.docker_preview" type="textarea" :rows="4" readonly />

      <template v-if="props.previewData.run_plan?.device_binding">
        <el-divider content-position="left">{{ $t('deployments.gpuBindingGroup') }}</el-divider>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('deployments.acceleratorIds')">
            {{ props.previewData.run_plan.device_binding.gpu_device_ids?.join(', ') || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.gpuVisibleEnv')">
            {{ props.previewData.run_plan.device_binding.visible_env_key }}={{ props.previewData.run_plan.device_binding.visible_env_value }}
          </el-descriptions-item>
          <el-descriptions-item v-if="props.previewData.run_plan.device_binding.docker_gpu_option" label="Docker GPU" :span="2">
            --gpus "{{ props.previewData.run_plan.device_binding.docker_gpu_option }}"
          </el-descriptions-item>
        </el-descriptions>
      </template>

      <el-divider content-position="left">{{ $t('deployments.finalRunPlan') }}</el-divider>
      <JsonViewer :value="props.previewData.run_plan || {}" :title="$t('deployments.finalRunPlan')" max-height="420px" :searchable="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import JsonViewer from '@/components/common/JsonViewer.vue'
import type { PreviewResult } from '@/api/deployments'

const { t } = useI18n()

const props = defineProps<{
  previewData: PreviewResult | null
  loading: boolean
}>()

defineEmits<{ preview: [] }>()

function preflightMessage(item: any): string {
  const code = item?.code || 'unknown'
  const key = `preflight.reason.${code}`
  const translated = t(key)
  return translated !== key ? translated : t('preflight.reason.unknown')
}

function translateSeverity(severity: string): string {
  const key = `status.${severity || 'unknown'}`
  const translated = t(key)
  return translated !== key ? translated : severity || t('common.unknown')
}

function lintMessage(item: any): string {
  const code = item?.code || item?.kind || 'unknown'
  const key = `runPlan.lint.${code}`
  const translated = t(key)
  return translated !== key ? translated : t('runPlan.lint.unknown')
}
</script>
