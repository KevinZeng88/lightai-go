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
      <el-table-column prop="status" :label="$t('common.status')" width="140" />
      <el-table-column :label="$t('common.actions')" width="220">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click.stop="openDetail(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" @click.stop="check(row)">{{ $t('runnerConfigs.check') }}</el-button>
          <el-button size="small" type="danger" @click.stop="deleteNBR(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('runnerConfigs.create')" width="960px" :close-on-click-modal="false" destroy-on-close>
      <NodeRuntimeConfigWizard ref="nbrWizardRef" @completed="onNBRCreated" />
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.id || ''" size="65%">
      <template v-if="selected">
        <div class="sticky-actions">
          <div>
            <strong>{{ selected.display_name || selected.id }}</strong>
            <div class="action-meta">{{ selected.node_id }} / {{ selected.backend_runtime?.display_name || selected.backend_runtime?.name || selected.backend_runtime_id }}</div>
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
          <el-descriptions-item :label="$t('common.status')">
            <el-tag :type="selected.deployable ? 'success' : 'warning'">{{ selected.status }}</el-tag>
            <span v-if="selected.disabled_reason" style="margin-left:8px;color:var(--el-color-warning);font-size:12px">{{ selected.disabled_reason }}</span>
          </el-descriptions-item>
        </el-descriptions>
        <el-divider content-position="left">{{ $t('runtimes.structuredParameters') }}</el-divider>
        <ConfigEditView
          v-if="nbrEditView"
          :model-value="nbrEditView"
          @update:patch="onNBREditPatch"
        />
        <el-empty v-else :description="$t('common.noData')" />
        <JsonViewer :value="selected.config_set || {}" :title="$t('runtimes.rawConfigJson')" max-height="520px" :searchable="true" />
        <JsonViewer :value="selected.source_metadata || {}" :title="$t('runtimes.rawSourceMetadataJson')" max-height="260px" :searchable="true" />

        <!-- Probe Summary -->
        <el-divider content-position="left">{{ $t('nodeRuntimeProbe.title') }}</el-divider>
        <el-descriptions v-if="probeSummary" :column="2" border size="small">
          <el-descriptions-item :label="$t('nodeRuntimeProbe.imageRef') || 'Image Ref'">{{ probeSummary.image_ref || selected.image_ref || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.status') || 'Status'">
            <el-tag :type="probeSummary.image_present ? 'success' : 'warning'">{{ probeSummary.image_present ? ($t('runnerConfigs.checkReady') || 'Ready') : ($t('runnerConfigs.checkNotReady') || 'Not Ready') }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.image_id" :label="$t('nodeRuntimeProbe.imageId') || 'Image ID'" :span="2">{{ probeSummary.image_id_truncated }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.cuda_version" :label="'CUDA Version'">{{ probeSummary.cuda_version }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.nvidia_constraint" :label="'NVIDIA CUDA Constraint'">
            <el-tooltip :content="probeSummary.nvidia_constraint" placement="top"><span style="color:var(--el-color-info);font-size:12px">{{ $t('common.yes') }}</span></el-tooltip>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntimeProbe.backendMatch') || 'Backend Match'">
            <el-tag :type="probeSummary.backend_confirmed ? 'success' : 'warning'">{{ probeSummary.backend_match_status || '-' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.match_detail" :label="$t('nodeRuntimeProbe.matchDetail') || 'Match Detail'" :span="2">{{ probeSummary.match_detail }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.runner_type" :label="$t('runnerConfigs.runnerType') || 'Runner Type'">{{ probeSummary.runner_type }}</el-descriptions-item>
          <el-descriptions-item v-if="probeSummary.confidence" :label="'Confidence'">{{ probeSummary.confidence }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.checkReady') || 'Blocking'">
            <el-tag :type="probeSummary.blocking ? 'danger' : 'success'">{{ probeSummary.blocking ? ($t('common.yes') || 'Yes') : ($t('common.no') || 'No') }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <el-empty v-else :description="$t('common.noData') || 'No probe data'" :image-size="40" />

        <!-- Raw probe evidence (collapsed by default) -->
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="$t('runtimes.advancedDiagnostics') || 'Raw Probe Evidence'">
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

watch(selected, async (value) => {
  nbrEditView.value = null
  nbrEditPatch.value = null
  if (!value?.id) return
  nbrEditView.value = await getConfigEditView({
    object_kind: 'node_backend_runtime',
    object_id: value.id,
    layer: 'node_backend_runtime',
    mode: 'edit',
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
    ElMessage.success('Saved')
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
      })
    }
  } catch (e: any) {
    ElMessage.error(e?.message || 'Save failed')
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
  await apiClient.post(`/nodes/${row.node_id}/backend-runtimes/${row.id}/check-request`, {})
  ElMessage.success('Check requested')
  await load()
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
    ElMessage.error(e?.message || e?.response?.data?.error || 'Delete failed')
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
