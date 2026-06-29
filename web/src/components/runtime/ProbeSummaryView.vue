<template>
  <div v-if="summary" class="probe-summary-view" data-testid="probe-summary-view">
    <el-descriptions :column="2" border size="small">
      <el-descriptions-item :label="labels.imageRef">{{ summary.image_ref || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="labels.imageStatus">
        <el-tag :type="summary.image_present ? 'success' : 'warning'">
          {{ summary.image_present ? labels.ready : labels.notReady }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item v-if="summary.image_id_truncated" :label="labels.imageId" :span="2">
        {{ summary.image_id_truncated }}
      </el-descriptions-item>
      <el-descriptions-item v-if="summary.cuda_version" :label="labels.cudaVersion">
        {{ summary.cuda_version }}
      </el-descriptions-item>
      <el-descriptions-item v-if="summary.nvidia_constraint" :label="labels.nvidiaConstraint">
        <span class="info-hint">{{ labels.yes }}</span>
      </el-descriptions-item>
      <el-descriptions-item :label="labels.backendMatch">
        <el-tag :type="summary.backend_confirmed ? 'success' : 'warning'">
          {{ translateStatus(summary.backend_match_status || '', t) }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item v-if="summary.match_detail" :label="labels.matchDetail" :span="2">
        {{ summary.match_detail }}
      </el-descriptions-item>
      <el-descriptions-item v-if="summary.runner_type" :label="labels.runnerType">
        {{ summary.runner_type }}
      </el-descriptions-item>
      <el-descriptions-item v-if="summary.confidence" :label="labels.confidence">
        {{ summary.confidence }}
      </el-descriptions-item>
      <el-descriptions-item :label="labels.blocking">
        <el-tag :type="summary.blocking ? 'danger' : 'success'">
          {{ summary.blocking ? labels.yes : labels.no }}
        </el-tag>
      </el-descriptions-item>
    </el-descriptions>

    <!-- Raw probe evidence (collapsed by default) -->
    <el-collapse v-if="rawProbe && Object.keys(rawProbe).length > 0" class="raw-probe-collapse" data-testid="raw-probe-collapse">
      <el-collapse-item :title="labels.rawDiagnostics">
        <pre class="raw-probe-pre" data-testid="raw-probe-content">{{ JSON.stringify(rawProbe, null, 2) }}</pre>
      </el-collapse-item>
    </el-collapse>
  </div>
  <el-empty v-else :description="labels.noData" :image-size="40" data-testid="probe-summary-empty" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { translateStatus } from '@/utils/status'

const { t } = useI18n()

export interface ProbeResults {
  level1?: Record<string, any>
  level2?: Record<string, any>
  level3?: Record<string, any>
  level4?: Record<string, any>
  process_start_detection?: Record<string, any>
  [key: string]: any
}

interface ProbeSummary {
  image_present: boolean
  image_ref: string
  runner_type: string
  image_id: string
  image_id_truncated: string
  cuda_version: string
  nvidia_constraint: string
  backend_match_status: string
  backend_confirmed: boolean
  match_detail: string
  match_method: string
  confidence: string
  start_status: string
  blocking: boolean
}

const props = withDefaults(defineProps<{
  probeResults?: ProbeResults | null
  runnerType?: string
  imageRef?: string
  labels?: Record<string, string>
}>(), {
  labels: () => ({}),
})

const labels = computed<Record<string, string>>(() => ({
  imageRef: t('nodeRuntimeProbe.imageRef'),
  imageStatus: t('nodeRuntime.status'),
  ready: t('status.ready'),
  notReady: t('runnerConfigs.checkNotReady'),
  imageId: t('nodeRuntimeProbe.imageId'),
  cudaVersion: 'CUDA_VERSION',
  nvidiaConstraint: 'NVIDIA_REQUIRE_CUDA',
  yes: t('common.yes'),
  no: t('common.no'),
  backendMatch: t('nodeRuntimeProbe.backendMatch'),
  matchDetail: t('nodeRuntimeProbe.matchDetail'),
  runnerType: t('runnerConfigs.runnerType'),
  confidence: t('artifacts.confidence'),
  blocking: t('preflight.errors'),
  rawDiagnostics: t('nodeRuntimeProbe.imageMetadata'),
  noData: t('common.noData'),
  ...props.labels,
}))

const summary = computed<ProbeSummary | null>(() => {
  const probe = props.probeResults
  if (!probe || typeof probe !== 'object' || Object.keys(probe).length === 0) return null
  const l2 = probe.level2
  const l3 = probe.level3
  const psd = probe.process_start_detection

  const imageId = (l2?.image_id as string) || ''
  return {
    image_present: !!(l2?.inspect_success || probe?.level1?.image_present),
    image_ref: props.imageRef || '',
    runner_type: props.runnerType || 'docker',
    image_id: imageId,
    image_id_truncated: imageId.length > 20 ? imageId.substring(0, 20) + '…' : imageId,
    cuda_version: extractEnvVar(l2?.env, 'CUDA_VERSION'),
    nvidia_constraint: extractEnvVar(l2?.env, 'NVIDIA_REQUIRE_CUDA'),
    backend_match_status: (l3?.backend_match_status as string) || 'not_checked',
    backend_confirmed: !!(l3?.confirmed_match),
    match_detail: (l3?.match_detail as string) || '',
    match_method: (l3?.match_method as string) || '',
    confidence: (psd?.confidence as string) || 'low',
    start_status: (psd?.status as string) || 'unknown',
    blocking: !!(l3?.blocking || probe?.level4?.blocking),
  }
})

const rawProbe = computed(() => props.probeResults || null)

function extractEnvVar(envList: any, prefix: string): string {
  if (!Array.isArray(envList)) return ''
  const found = envList.find((e: string) => e.startsWith(prefix + '='))
  return found ? found.split('=')[1] : ''
}
</script>

<style scoped>
.probe-summary-view { width: 100%; }
.raw-probe-collapse { margin-top: 12px; }
.raw-probe-pre {
  background: var(--el-fill-color);
  padding: 12px;
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
  white-space: pre-wrap;
  max-height: 300px;
  overflow-y: auto;
}
.info-hint { color: var(--el-color-info); font-size: 12px; }
</style>
