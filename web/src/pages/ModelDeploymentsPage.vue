<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('deployments.title') }}</h2>
      <div>
        <el-button type="primary" @click="createVisible = true">{{ $t('common.create') }}</el-button>
        <el-button @click="load">{{ $t('common.refresh') }}</el-button>
      </div>
    </div>

    <el-table :data="deployments" v-loading="loading" stripe @row-click="selected = $event">
      <el-table-column :label="$t('deployments.name')" min-width="220">
        <template #default="{ row }">{{ row.display_name || row.name || row.id }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.artifact')" min-width="220">
        <template #default="{ row }">{{ row.model_display_name || row.model_name || row.model_artifact_id }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.runtime')" min-width="260">
        <template #default="{ row }">{{ row.source_node_backend_runtime_display_name || row.source_node_backend_runtime_id }}</template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('common.status')" width="140" />
      <el-table-column :label="$t('common.actions')" min-width="300">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click.stop="editDeployment(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" @click.stop="viewRunPlan(row)">{{ $t('deployments.viewRunPlan') }}</el-button>
          <el-button size="small" @click.stop="dryRun(row)">{{ $t('deployments.dryRun') }}</el-button>
          <el-button size="small" type="primary" @click.stop="start(row)">{{ $t('deployments.start') }}</el-button>
          <el-button size="small" @click.stop="stop(row)">{{ $t('deployments.stop') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('deployments.createDeployment')" width="960px" :close-on-click-modal="false" destroy-on-close @closed="onWizardClosed">
      <DeploymentWizard
        v-if="createVisible"
        ref="wizardRef"
        :artifacts="artifacts"
        :node-runtimes="nodeRuntimes"
        :model-locations="modelLocations"
        :saving="saving"
        @save="createFromWizard"
        @cancel="createVisible = false"
        @refresh-data="load"
      />
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.name || ''" size="70%">
      <template v-if="selected">
        <div class="sticky-actions">
          <div>
            <strong>{{ selected.display_name || selected.name || selected.id }}</strong>
            <div class="action-meta">{{ selected.model_display_name || selected.model_name || selected.model_artifact_id }} / {{ selected.source_node_backend_runtime_display_name || selected.source_node_backend_runtime_id }}</div>
          </div>
          <div>
            <el-button @click="editDeployment(selected)">{{ $t('common.edit') }}</el-button>
            <el-button @click="viewRunPlan(selected)">{{ $t('deployments.viewRunPlan') }}</el-button>
          </div>
        </div>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('deployments.artifact')">{{ selected.model_display_name || selected.model_name || selected.model_artifact_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.runtime')">{{ selected.source_node_backend_runtime_display_name || selected.source_node_backend_runtime_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.status')">{{ selected.status }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.created')">{{ selected.created_at }}</el-descriptions-item>
        </el-descriptions>
        <!-- RunPlan / command preview -->
        <template v-if="lastDryRun?.command_preview">
          <el-divider content-position="left">{{ $t('deployments.finalRunPlan') }}</el-divider>
          <pre style="background:var(--el-fill-color);padding:12px;border-radius:4px;font-size:12px;overflow-x:auto;white-space:pre-wrap">{{ lastDryRun.command_preview }}</pre>
          <el-descriptions v-if="lastDryRun.resolved_image" :column="2" border size="small" style="margin-top:8px">
            <el-descriptions-item label="Resolved Image">{{ lastDryRun.resolved_image }}</el-descriptions-item>
            <el-descriptions-item v-if="lastDryRun.selected_node" label="Selected Node">{{ lastDryRun.selected_node }}</el-descriptions-item>
          </el-descriptions>
          <el-descriptions v-if="lastDryRun.run_plan?.device_binding" :column="2" border size="small" style="margin-top:8px">
            <el-descriptions-item :label="$t('deployments.gpuBindingGroup')">{{ lastDryRun.run_plan.device_binding.gpu_device_ids?.join(', ') || '-' }}</el-descriptions-item>
            <el-descriptions-item :label="$t('deployments.gpuVisibleEnv')">{{ lastDryRun.run_plan.device_binding.visible_env_key }}={{ lastDryRun.run_plan.device_binding.visible_env_value }}</el-descriptions-item>
            <el-descriptions-item v-if="lastDryRun.run_plan.device_binding.docker_gpu_option" label="Docker GPU" :span="2">--gpus "{{ lastDryRun.run_plan.device_binding.docker_gpu_option }}"</el-descriptions-item>
          </el-descriptions>
        </template>
        <ConfigEditView
          v-if="editing && deploymentEditView"
          :model-value="deploymentEditView"
          @update:patch="deploymentEditPatch = $event"
        />
        <div v-if="editing" style="margin-top:12px;text-align:right">
          <el-button @click="editing = false">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="savingEdit" @click="saveDeploymentEdit">{{ $t('common.save') }}</el-button>
        </div>
        <!-- Diagnostic section (collapsed by default) -->
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="$t('runtimes.advancedDiagnostics')">
            <JsonViewer :value="selected.config_set || {}" :title="$t('runtimes.rawConfigJson')" max-height="520px" :searchable="true" />
            <JsonViewer :value="selected.source_metadata || {}" :title="$t('runtimes.rawSourceMetadataJson')" max-height="260px" :searchable="true" />
            <JsonViewer v-if="lastDryRun" :value="lastDryRun" title="Dry Run Detail" max-height="420px" :searchable="true" />
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { createDeployment, dryRunDeployment, startDeployment, stopDeployment } from '@/api/deployments'
import { applyConfigEditPatch, getConfigEditView } from '@/api/configEdit'
import JsonViewer from '@/components/common/JsonViewer.vue'
import ConfigEditView from '@/components/config/ConfigEditView.vue'
import DeploymentWizard from '@/components/deployments/DeploymentWizard.vue'
import type { ConfigEditPatch, ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'
import { apiErrorMessage } from '@/utils/apiErrors'

const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const createVisible = ref(false)
const wizardRef = ref<any>(null)
const deployments = ref<any[]>([])
const artifacts = ref<any[]>([])
const nodeRuntimes = ref<any[]>([])
const modelLocations = ref<any[]>([])
const selected = ref<any | null>(null)
const lastDryRun = ref<any | null>(null)
const editing = ref(false)
const savingEdit = ref(false)
const deploymentEditView = ref<ConfigEditViewModel | null>(null)
const deploymentEditPatch = ref<ConfigEditPatch | null>(null)

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; lastDryRun.value = null; editing.value = false; deploymentEditView.value = null; deploymentEditPatch.value = null } },
})

async function load() {
  loading.value = true
  try {
    const [deploymentList, artifactList, runtimeList] = await Promise.all([
      apiClient.get('/deployments'),
      apiClient.get('/model-artifacts'),
      apiClient.get('/nodes/backend-runtimes/all'),
    ])
    deployments.value = Array.isArray(deploymentList) ? deploymentList : []
    artifacts.value = Array.isArray(artifactList) ? artifactList : []
    nodeRuntimes.value = Array.isArray(runtimeList) ? runtimeList : []
    // Derive model locations from artifacts (each artifact includes .locations)
    const locs: any[] = []
    for (const a of artifacts.value) {
      if (Array.isArray(a.locations)) {
        for (const l of a.locations) {
          locs.push({ ...l, model_artifact_id: a.id })
        }
      }
    }
    modelLocations.value = locs
  } finally {
    loading.value = false
  }
}

async function createFromWizard() {
  saving.value = true
  try {
    const payload = wizardRef.value?.buildPayload()
    if (!payload) { ElMessage.error(t('deployments.createBlocked')); return }
    await createDeployment(payload)
    createVisible.value = false
    ElMessage.success(t('deployments.created'))
    await load()
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  } finally {
    saving.value = false
  }
}

async function dryRun(row: any) {
  selected.value = row
  lastDryRun.value = await dryRunDeployment(row.id)
}

async function viewRunPlan(row: any) {
  selected.value = row
  lastDryRun.value = await dryRunDeployment(row.id)
}

async function editDeployment(row: any) {
  selected.value = row
  editing.value = true
  deploymentEditPatch.value = null
  deploymentEditView.value = await getConfigEditView({
    object_kind: 'deployment',
    object_id: row.id,
    layer: 'deployment',
    mode: 'edit',
  })
}

async function saveDeploymentEdit() {
  if (!selected.value) return
  savingEdit.value = true
  try {
    if (deploymentEditPatch.value) {
      await applyConfigEditPatch({
        object_kind: 'deployment',
        object_id: selected.value.id,
        layer: 'deployment',
        patch: deploymentEditPatch.value,
      })
    }
    ElMessage.success(t('common.saved'))
    editing.value = false
    await load()
    const updated = deployments.value.find((d: any) => d.id === selected.value?.id)
    if (updated) selected.value = updated
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  } finally {
    savingEdit.value = false
  }
}

async function start(row: any) {
  try {
    await startDeployment(row.id)
    ElMessage.success(t('deployments.started'))
    await load()
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  }
}

async function stop(row: any) {
  try {
    await stopDeployment(row.id)
    ElMessage.success(t('deployments.stopped'))
    await load()
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  }
}

function onWizardClosed() {
  // Reset wizard state so next create starts from clean step 1.
  if (wizardRef.value?.resetWizard) {
    wizardRef.value.resetWizard()
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
