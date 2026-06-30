<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runnerConfigs.title') }}</h2>
      <div>
        <el-button type="primary" @click="createVisible = true">{{ $t('common.create') }}</el-button>
        <el-button @click="load">{{ $t('common.refresh') }}</el-button>
      </div>
    </div>

    <el-table :data="configs" v-loading="loading" stripe @row-click="openDetail">
      <el-table-column prop="display_name" :label="$t('runnerConfigs.name')" min-width="220" />
      <el-table-column prop="node_id" :label="$t('deployments.node')" min-width="180" />
      <el-table-column :label="$t('deployments.runtime')" min-width="240">
        <template #default="{ row }">{{ row.backend_runtime?.display_name || row.backend_runtime?.name || row.backend_runtime_id }}</template>
      </el-table-column>
      <el-table-column prop="image_ref" :label="$t('runtimes.image')" min-width="260" show-overflow-tooltip />
      <el-table-column :label="$t('common.status')" width="150">
        <template #default="{ row }">
          <el-tag :type="getStatusType(row.status)">{{ translateStatus(row.status || '', t) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.actions')" width="220">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click.stop="openDetail(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" @click.stop="check(row)">{{ $t('runnerConfigs.check') }}</el-button>
          <el-button size="small" type="danger" @click.stop="deleteNBR(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('runnerConfigs.create')" width="960px" :close-on-click-modal="false" destroy-on-close>
      <NodeRuntimeConfigWizard ref="nbrWizardRef" @completed="onNBRCreated" @cancel="createVisible = false" />
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.id || ''" size="65%">
      <template v-if="selected">
        <div class="sticky-actions">
          <div>
            <strong>{{ selected.display_name || selected.id }}</strong>
            <div class="action-meta">{{ selected.node_id }} / {{ selected.backend_runtime?.display_name || selected.backend_runtime?.name || selected.backend_runtime_id }}</div>
          </div>
          <div class="detail-actions">
            <el-tooltip :content="configViewLevelHelpText" placement="top">
              <el-segmented v-model="configViewLevel" :options="configViewOptions" size="small" />
            </el-tooltip>
          </div>
          <div>
            <el-button type="danger" @click="deleteNBR(selected)">{{ $t('common.delete') }}</el-button>
            <el-button type="primary" :loading="saving" @click="saveNBREdit">{{ $t('common.save') }}</el-button>
          </div>
        </div>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('deployments.node')">{{ selected.node_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.runtime')">{{ selected.backend_runtime?.display_name || selected.backend_runtime?.name || selected.backend_runtime_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_ref }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.status')">
            <el-tag :type="getStatusType(selected.status)">{{ translateStatus(selected.status || '', t) }}</el-tag>
            <span v-if="selected.disabled_reason" style="margin-left:8px;color:var(--el-color-warning);font-size:12px">{{ translateStatusReason(selected.disabled_reason, t) }}</span>
          </el-descriptions-item>
        </el-descriptions>
        <el-divider content-position="left">{{ $t('runtimes.structuredParameters') }}</el-divider>
        <ConfigEditView
          v-if="nbrEditView"
          :model-value="nbrEditView"
          @update:patch="onNBREditPatch"
        />
        <el-empty v-else :description="$t('common.noData')" />
        <template v-if="configViewLevel === 'developer'">
          <JsonViewer :value="selected.config_set || {}" :title="$t('runtimes.rawConfigJson')" max-height="520px" :searchable="true" />
          <JsonViewer :value="selected.source_metadata || {}" :title="$t('runtimes.rawSourceMetadataJson')" max-height="260px" :searchable="true" />
        </template>

        <!-- Probe Summary -->
        <el-divider content-position="left">{{ $t('nodeRuntimeProbe.title') }}</el-divider>
        <el-descriptions v-if="probeSummary" :column="2" border size="small">
          <el-descriptions-item :label="$t('nodeRuntimeProbe.imageRef')">{{ probeSummary.image_ref || selected.image_ref || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.status')">
            <el-tag :type="probeSummary.image_present ? 'success' : 'warning'">{{ probeSummary.image_present ? $t('runnerConfigs.checkReady') : $t('runnerConfigs.checkNotReady') }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.image_id" :label="$t('nodeRuntimeProbe.imageId')" :span="2">{{ probeSummary.image_id_truncated }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.cuda_version" label="CUDA_VERSION">{{ probeSummary.cuda_version }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.nvidia_constraint" label="NVIDIA_REQUIRE_CUDA">
            <el-tooltip :content="probeSummary.nvidia_constraint" placement="top"><span style="color:var(--el-color-info);font-size:12px">{{ $t('common.yes') }}</span></el-tooltip>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntimeProbe.backendMatch')">
            <el-tag :type="probeSummary.backend_confirmed ? 'success' : 'warning'">{{ translateStatus(probeSummary.backend_match_status || '', t) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.match_detail" :label="$t('nodeRuntimeProbe.matchDetail')" :span="2">{{ probeSummary.match_detail }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.runner_type" :label="$t('runnerConfigs.runnerType')">{{ probeSummary.runner_type }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.confidence" :label="$t('artifacts.confidence')">{{ probeSummary.confidence }}</el-descriptions-item>
          <el-descriptions-item :label="$t('preflight.errors')">
            <el-tag :type="probeSummary.blocking ? 'danger' : 'success'">{{ probeSummary.blocking ? $t('common.yes') : $t('common.no') }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <el-empty v-else :description="$t('common.noData')" :image-size="40" />

        <!-- Raw probe evidence (collapsed by default) -->
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="$t('runtimes.advancedDiagnostics')">
            <JsonViewer :value="selected.probe_results_json || {}" :title="$t('nodeRuntimeProbe.title')" max-height="260px" :searchable="true" />
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { getConfigEditView, applyConfigEditPatch } from '@/api/configEdit'
import { apiErrorMessage } from '@/utils/apiErrors'
import { getStatusType, translateStatus, translateStatusReason } from '@/utils/status'
import { configEditViewLevelHelp, configEditViewLevelOptions, type ConfigEditViewLevel } from '@/utils/configEditDisplay'
import JsonViewer from '@/components/common/JsonViewer.vue'
import ConfigEditView from '@/components/config/ConfigEditView.vue'
import NodeRuntimeConfigWizard from '@/components/deployments/NodeRuntimeConfigWizard.vue'
import type { ConfigEditPatch, ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'

const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const createVisible = ref(false)
const configs = ref<any[]>([])
const selected = ref<any | null>(null)
const nbrEditView = ref<ConfigEditViewModel | null>(null)
const nbrEditPatch = ref<ConfigEditPatch | null>(null)
const configViewLevel = ref<ConfigEditViewLevel>('advanced')
const configViewOptions = computed(() => configEditViewLevelOptions(t))
const configViewLevelHelpText = computed(() => configEditViewLevelHelp(t))

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; nbrEditView.value = null; nbrEditPatch.value = null } },
})

const probeSummary = computed(() => {
  const probe = selected.value?.probe_results_json
  if (!probe || typeof probe !== 'object' || Object.keys(probe).length === 0) return null
  const l2 = probe?.level2
  const l3 = probe?.level3
  const psd = probe?.process_start_detection
  const summary: Record<string, any> = {}

  // Image status
  summary.image_present = !!(l2?.inspect_success || probe?.level1?.image_present)
  summary.image_ref = selected.value?.image_ref || ''
  summary.runner_type = selected.value?.runner_type || 'docker'

  // Image ID (truncated)
  const imageId = l2?.image_id || ''
  summary.image_id = imageId
  if (imageId && imageId.length > 20) {
    summary.image_id_truncated = imageId.substring(0, 20) + '…'
  } else {
    summary.image_id_truncated = imageId
  }

  // Extract CUDA version from level2 env
  const envList = l2?.env
  if (Array.isArray(envList)) {
    const cudaEnv = envList.find((e: string) => e.startsWith('CUDA_VERSION='))
    if (cudaEnv) {
      summary.cuda_version = cudaEnv.split('=')[1]
    }
    const nvidiaReq = envList.find((e: string) => e.startsWith('NVIDIA_REQUIRE_CUDA='))
    if (nvidiaReq) {
      summary.nvidia_constraint = nvidiaReq.split('=')[1]
    }
  }

  // Backend match
  summary.backend_match_status = l3?.backend_match_status || 'not_checked'
  summary.backend_confirmed = !!l3?.confirmed_match
  summary.match_detail = l3?.match_detail || ''
  summary.match_method = l3?.match_method || ''

  // Process start detection
  summary.confidence = psd?.confidence || 'low'
  summary.start_status = psd?.status || 'unknown'

  // Blocking
  summary.blocking = !!(l3?.blocking || (probe?.level4?.blocking))

  return summary
})

function openDetail(row: any) {
  selected.value = row
}

watch([selected, configViewLevel], async ([value]) => {
  nbrEditView.value = null
  nbrEditPatch.value = null
  if (!value?.id) return
  nbrEditView.value = await getConfigEditView({
    object_kind: 'node_backend_runtime',
    object_id: value.id,
    layer: 'node_backend_runtime',
    mode: 'edit',
    view_level: configViewLevel.value,
  })
})

function onNBREditPatch(patch: ConfigEditPatch) {
  nbrEditPatch.value = patch
}

async function saveNBREdit() {
  if (!selected.value) return
  saving.value = true
  try {
    if (nbrEditPatch.value) {
      await applyConfigEditPatch({
        object_kind: 'node_backend_runtime',
        object_id: selected.value.id,
        layer: 'node_backend_runtime',
        patch: nbrEditPatch.value,
      })
    }
    ElMessage.success(t('common.saved'))
    await load()
    const updated = configs.value.find(r => r.id === selected.value?.id)
    if (updated) {
      selected.value = updated
    } else if (selected.value?.id) {
      nbrEditView.value = await getConfigEditView({
        object_kind: 'node_backend_runtime',
        object_id: selected.value.id,
        layer: 'node_backend_runtime',
        mode: 'edit',
        view_level: configViewLevel.value,
      })
    }
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  } finally {
    saving.value = false
  }
}

async function load() {
  loading.value = true
  try {
    configs.value = await apiClient.get('/nodes/backend-runtimes/all')
  } finally {
    loading.value = false
  }
}

async function onNBRCreated() {
  createVisible.value = false
  await load()
}

async function check(row: any) {
  try {
    const result = await apiClient.post(`/nodes/${row.node_id}/backend-runtimes/${row.id}/check-request`, {})
    if (result?.deployable) {
      ElMessage.success(t('runnerConfigs.checkPassedWithImage', { image: result.checked_image_ref || result.image_ref || '-' }))
    } else {
      ElMessage.warning(translateStatusReason(result?.status_reason || '', t) || translateStatus(result?.status || '', t))
    }
    await load()
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'runnerConfigs.checkFailed'))
  }
}

async function deleteNBR(row: any) {
  if (!row?.id || !row?.node_id) return
  const name = row.display_name || row.name || row.id
  try {
    await ElMessageBox.confirm(t('runnerConfigs.deleteConfirm', { name }), t('common.confirm'), {
      type: 'warning',
      confirmButtonText: t('common.delete'),
      cancelButtonText: t('common.cancel'),
    })
  } catch {
    return
  }
  try {
    await apiClient.delete(`/nodes/${row.node_id}/backend-runtimes/${row.id}`)
    ElMessage.success(t('runnerConfigs.deleted'))
    if (selected.value?.id === row.id) {
      detailVisible.value = false
    }
    await load()
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  }
}

onMounted(load)
</script>

<style scoped>
.sticky-actions {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  padding: 10px 0;
  margin-bottom: 12px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.action-meta {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
