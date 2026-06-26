<template>
  <div>
    <div v-if="!props.previewData && !props.loading" style="text-align:center;padding:40px">
      <el-button type="primary" @click="$emit('preview')" :loading="props.loading">
        {{ $t('deployments.previewRunPlan') || 'Preview RunPlan' }}
      </el-button>
    </div>

    <div v-loading="props.loading" v-if="props.previewData">
      <el-descriptions :column="1" border size="small" style="margin-bottom:16px">
        <el-descriptions-item :label="$t('deployments.canRun') || 'Can Run'">
          <el-tag :type="props.previewData.can_run ? 'success' : 'danger'">
            {{ props.previewData.can_run ? 'Yes' : 'No' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="props.previewData.preflight?.errors?.length" :label="$t('common.error')">
          <div v-for="(e, i) in props.previewData.preflight.errors" :key="i" style="color:var(--el-color-danger)">
            [{{ e.code }}] {{ e.message }}
          </div>
        </el-descriptions-item>
        <el-descriptions-item v-if="props.previewData.preflight?.warnings?.length" :label="$t('deployments.warnings') || 'Warnings'">
          <div v-for="(w, i) in props.previewData.preflight.warnings" :key="i" style="color:var(--el-color-warning)">
            [{{ w.code }}] {{ w.message }}
          </div>
        </el-descriptions-item>
        <el-descriptions-item v-if="props.previewData.lint?.findings?.length" :label="$t('deployments.lintFindings') || 'Lint'">
          <div v-for="(f, i) in props.previewData.lint.findings" :key="i">
            <el-tag :type="f.severity === 'error' ? 'danger' : f.severity === 'warning' ? 'warning' : 'info'" size="small">
              {{ f.severity }}
            </el-tag>
            {{ f.message }}
          </div>
        </el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">{{ $t('deployments.dockerPreview') || 'Docker Command' }}</el-divider>
      <el-input :model-value="props.previewData.docker_preview" type="textarea" :rows="4" readonly />

      <el-divider content-position="left">{{ $t('deployments.finalRunPlan') || 'Run Plan' }}</el-divider>
      <JsonViewer :value="props.previewData.run_plan || {}" :title="$t('deployments.finalRunPlan') || 'Run Plan'" max-height="420px" :searchable="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import JsonViewer from '@/components/common/JsonViewer.vue'
import type { PreviewResult } from '@/api/deployments'

const props = defineProps<{
  previewData: PreviewResult | null
  loading: boolean
}>()

defineEmits<{ preview: [] }>()
</script>
