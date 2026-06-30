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
          <div v-for="(e, i) in normalizedErrors" :key="i" class="preview-issue preview-issue-error">
            <strong>[{{ e.code }}]</strong> {{ preflightMessage(e) }}
            <div class="issue-meta">
              <span v-if="e.key">key: {{ e.key }}</span>
              <span v-if="issuePath(e)">path: {{ issuePath(e) }}</span>
              <span v-if="e.source">source: {{ e.source }}</span>
              <span>blocking: {{ e.blocking ? t('common.yes') : t('common.no') }}</span>
            </div>
          </div>
        </el-descriptions-item>
        <el-descriptions-item v-if="props.previewData.preflight?.warnings?.length" :label="$t('deployments.warnings')">
          <div v-for="(w, i) in normalizedWarnings" :key="i" class="preview-issue preview-issue-warning">
            <strong>[{{ w.code }}]</strong> {{ preflightMessage(w) }}
            <div class="issue-meta">
              <span v-if="w.key">key: {{ w.key }}</span>
              <span v-if="issuePath(w)">path: {{ issuePath(w) }}</span>
              <span v-if="w.source">source: {{ w.source }}</span>
              <span>blocking: {{ w.blocking ? t('common.yes') : t('common.no') }}</span>
            </div>
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
          <el-descriptions-item label="mode">
            {{ props.previewData.run_plan.device_binding.selection_mode || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="vendor">
            {{ props.previewData.run_plan.device_binding.vendor || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.acceleratorIds')">
            {{ props.previewData.run_plan.device_binding.gpu_device_ids?.join(', ') || props.previewData.run_plan.device_binding.accelerator_ids?.join(', ') || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="source">
            {{ props.previewData.run_plan.device_binding.source || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="patch target" :span="2">
            {{ props.previewData.run_plan.device_binding.patch_target || '-' }}
          </el-descriptions-item>
          <el-descriptions-item
            v-for="(item, i) in props.previewData.run_plan.device_binding.injection_preview || []"
            :key="i"
            :label="item.key"
            :span="2"
          >
            {{ item.docker_effect ? item.docker_effect + ' ' : '' }}{{ item.value }}
            <span class="source-muted">({{ item.source }})</span>
          </el-descriptions-item>
        </el-descriptions>
      </template>

      <template v-if="sourceEntries.length">
        <el-divider content-position="left">{{ $t('deployments.runPlanSourceNote') }}</el-divider>
        <el-table :data="sourceEntries" size="small" border max-height="260">
          <el-table-column prop="target" label="target" width="130" />
          <el-table-column prop="key" label="key" min-width="180" />
          <el-table-column prop="effective_source" label="source" width="160" />
          <el-table-column prop="patch_target" label="patch target" min-width="180" />
          <el-table-column prop="docker_effect" label="effect" min-width="150" />
        </el-table>
      </template>

      <el-divider content-position="left">{{ $t('deployments.finalRunPlan') }}</el-divider>
      <JsonViewer :value="props.previewData.run_plan || {}" :title="$t('deployments.finalRunPlan')" max-height="420px" :searchable="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import JsonViewer from '@/components/common/JsonViewer.vue'
import type { PreviewResult } from '@/api/deployments'

const { t } = useI18n()

const props = defineProps<{
  previewData: PreviewResult | null
  loading: boolean
}>()

defineEmits<{ preview: [] }>()

const normalizedErrors = computed(() => dedupeIssues(props.previewData?.preflight?.errors || []))
const normalizedWarnings = computed(() => dedupeIssues(props.previewData?.preflight?.warnings || []))
const sourceEntries = computed(() => flattenSourceMap(props.previewData?.run_plan?.parameter_source_map))

function preflightMessage(item: any): string {
  const direct = item?.message || item?.reason
  if (direct) return String(direct)
  const code = item?.code || 'unknown'
  const key = `preflight.reason.${code}`
  const translated = t(key)
  return translated !== key ? translated : code
}

function issuePath(item: any): string {
  if (Array.isArray(item?.path)) return item.path.join('.')
  return item?.field || ''
}

function dedupeIssues(items: any[]): any[] {
  const seen = new Set<string>()
  const out: any[] = []
  for (const item of items || []) {
    const key = `${item?.code || ''}|${item?.key || ''}|${issuePath(item)}|${item?.reason || item?.message || ''}|${item?.source || ''}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(item)
  }
  return out
}

function flattenSourceMap(sourceMap: any): any[] {
  if (!sourceMap) return []
  const groups = ['image', 'args', 'env', 'mounts', 'ports', 'devices', 'docker_options', 'health_check', 'resource_controls', 'system_generated']
  return groups.flatMap(group => (sourceMap[group] || []).map((item: any) => ({ ...item, target: item.target || group })))
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

<style scoped>
.preview-issue { margin-bottom: 8px; }
.preview-issue-error { color: var(--el-color-danger); }
.preview-issue-warning { color: var(--el-color-warning); }
.issue-meta { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 2px; font-size: 12px; color: var(--el-text-color-secondary); }
.source-muted { color: var(--el-text-color-secondary); margin-left: 6px; }
</style>
