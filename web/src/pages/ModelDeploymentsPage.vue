<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('deployments.title') }}</h2>
      <div>
        <el-button type="primary" @click="startWizard">{{ $t('startWizard.title') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column :label="$t('deployments.name')" width="180">
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('deployments.status')" width="120">
        <template #default="{ row }">
          <el-tag :type="deploymentStatusType(row.status)" size="small">{{ deploymentStatusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('deployments.artifact')" width="200">
	        <template #default="{ row }">{{ modelName(row.model_artifact_id) }}</template>
	      </el-table-column>
      <el-table-column :label="$t('deployments.backend')" width="140">
        <template #default="{ row }">{{ runtimeContext(row).backend || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.backendVersion')" width="150">
        <template #default="{ row }">{{ runtimeContext(row).version || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.runtime')" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">{{ runtimeContext(row).runtime || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.image')" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">{{ runtimeContext(row).image || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.node')" width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ deploymentContext(row).node || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.endpoint')" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ deploymentContext(row).endpoint || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('deployments.recentError')" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ deploymentContext(row).lastError || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('common.actions')" width="320" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" @click="doDryRun(row)">{{ $t('deployments.viewRunPlan') }}</el-button>
          <el-button size="small" type="success" :disabled="isRunBlocked(row.status)" @click="doStart(row)">{{ $t('deployments.runExisting') }}</el-button>
          <el-button size="small" type="warning" @click="doStop(row)">{{ $t('deployments.stop') }}</el-button>
          <el-button size="small" @click="doRestart(row)">{{ $t('deployments.restart') }}</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create Dialog -->
    <el-dialog v-model="createVisible" :title="$t('common.create')" width="500px">
      <el-form :model="createForm" label-width="140px">
        <el-form-item :label="$t('deployments.name')"><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item :label="$t('deployments.artifact')"><el-input v-model="createForm.model_artifact_id" /></el-form-item>
        <el-form-item :label="$t('deployments.runtime')">
          <el-select v-model="createForm.node_backend_runtime_id" filterable style="width:100%" :placeholder="$t('startWizard.selectRuntime')">
            <el-option v-for="nbr in allNBRs" :key="nbr.id" :label="formatNBRLabel(nbr)" :value="nbr.id" :disabled="!isNBRDeployable(nbr)">
              <div style="display:flex;justify-content:space-between;align-items:center;width:100%">
                <span style="flex:1;overflow:hidden;text-overflow:ellipsis">{{ formatNBRLabel(nbr) }}</span>
                <el-tag :type="nbrStatusTagType(nbr.status)" size="small" style="margin-left:8px;flex-shrink:0">{{ nbrStatusText(nbr.status) }}</el-tag>
              </div>
              <div v-if="!isNBRDeployable(nbr) && nbr.disabled_reason" style="font-size:11px;color:var(--el-color-danger);margin-top:2px">{{ nbr.disabled_reason }}</div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.nodeId')"><el-input v-model="createForm.node_id" /></el-form-item>
        <el-form-item :label="$t('deployments.hostPort')"><el-input v-model.number="createForm.host_port" /></el-form-item>
        <el-form-item :label="$t('deployments.containerPort')"><el-input v-model.number="createForm.container_port" /></el-form-item>
        <el-form-item :label="$t('deployments.appPort')"><el-input v-model.number="createForm.app_port" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doCreate" :loading="saving">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="dryRunVisible" :title="$t('deployments.finalRunPlan')" width="860px">
      <template v-if="dryRunResult">
        <!-- Parameter source groups -->
        <el-alert type="info" :closable="false" style="margin-bottom:12px">
          {{ $t('deployments.runPlanSourceNote') }}
        </el-alert>
        <el-descriptions :column="1" border size="small" style="margin-top:8px">
          <template #title>{{ $t('deployments.nbrTemplateGroup') }}</template>
          <el-descriptions-item :label="$t('runtimes.image')">{{ dryRunSummary(dryRunResult).image || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.command')">{{ dryRunSummary(dryRunResult).command || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.detailEnv')">{{ dryRunSummary(dryRunResult).env || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.dockerOptions')">{{ dryRunSummary(dryRunResult).dockerOptions || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="1" border size="small" style="margin-top:12px">
          <template #title>{{ $t('deployments.modelLocationGroup') }}</template>
          <el-descriptions-item :label="$t('runtimes.extraMounts')">{{ dryRunSummary(dryRunResult).volumes || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.servedModelName')">{{ dryRunSummary(dryRunResult).servedModelName || '-' }} <span class="text-muted" v-if="dryRunSummary(dryRunResult).servedModelNameSource">({{ dryRunSummary(dryRunResult).servedModelNameSource }})</span></el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="1" border size="small" style="margin-top:12px">
          <template #title>{{ $t('deployments.portInjectionGroup') }}</template>
          <el-descriptions-item :label="$t('runnerConfigs.ports')">{{ dryRunSummary(dryRunResult).ports || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="1" border size="small" style="margin-top:12px">
          <template #title>{{ $t('deployments.gpuBindingGroup') }}</template>
          <el-descriptions-item :label="$t('runtimes.devices')">{{ dryRunSummary(dryRunResult).devices || '-' }} <span class="text-muted" v-if="dryRunSummary(dryRunResult).gpuSource">{{ dryRunSummary(dryRunResult).gpuSource }}</span></el-descriptions-item>
          <el-descriptions-item v-if="dryRunSummary(dryRunResult).gpuEnv" :label="$t('deployments.gpuVisibleEnv')">{{ dryRunSummary(dryRunResult).gpuEnv }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="1" border size="small" style="margin-top:12px">
          <template #title>{{ $t('deployments.backendServiceGroup') }}</template>
          <el-descriptions-item :label="$t('backends.healthCheck')">{{ dryRunSummary(dryRunResult).health || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="1" border size="small" style="margin-top:12px">
          <template #title>{{ $t('deployments.finalCommandGroup') }}</template>
          <el-descriptions-item :label="$t('runtimes.commandPreview')">{{ dryRunSummary(dryRunResult).preview || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="$t('runnerConfigs.advancedJson')">
            <JsonViewer :value="dryRunResult" title="Dry Run Response" max-height="500px" :allow-download="true" :searchable="true" />
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" :title="$t('deployments.editDeployment')" width="600px">
      <el-form :model="editForm" label-width="160px">
        <el-form-item :label="$t('deployments.name')">
          <span>{{ editForm.original_name }}</span>
          <el-tag size="small" type="info" style="margin-left:8px">{{ $t('common.readonly') }}</el-tag>
        </el-form-item>
        <el-form-item :label="$t('deployments.displayName')">
          <el-input v-model="editForm.display_name" />
        </el-form-item>
        <el-form-item :label="$t('deployments.artifact')">
          <el-select v-model="editForm.model_artifact_id" filterable style="width:100%">
            <el-option v-for="m in models" :key="m.id" :label="`${m.display_name || m.name}`" :value="m.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.runtime')">
          <el-select v-model="editForm.backend_runtime_id" filterable style="width:100%">
            <el-option v-for="r in runtimes" :key="r.id" :label="`${r.display_name || r.name} (${r.vendor})`" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.hostPort')">
          <el-input v-model.number="editForm.host_port" />
        </el-form-item>
        <el-form-item :label="$t('deployments.containerPort')">
          <el-input v-model.number="editForm.container_port" />
        </el-form-item>
        <el-form-item :label="$t('deployments.appPort')">
          <el-input v-model.number="editForm.app_port" />
        </el-form-item>
        <el-divider>{{ $t('deployments.existingOverrides') }}</el-divider>
        <el-alert :title="$t('deployments.overrideHint')" type="info" show-icon :closable="false" style="margin-bottom:8px" />
        <el-form-item :label="$t('runtimes.detailEnv')">
          <el-input v-model="editForm.env_overrides_text" type="textarea" :rows="3" :placeholder="$t('runnerConfigs.keyValueLines')" />
        </el-form-item>
        <el-divider>{{ $t('deployments.structuredParameters') }}</el-divider>
        <RuntimeParameterEditor v-model="editParameterModel" />
      </el-form>
      <template #footer>
        <div style="display:flex;justify-content:space-between;width:100%">
          <div>
            <el-button v-if="editForm.source_template_name" @click="doTemplateSyncPreview" size="small">{{ $t('deployments.previewTemplateDiff') }}</el-button>
            <el-button v-if="editForm.source_template_name" @click="doTemplateSyncApply" size="small" type="warning">{{ $t('deployments.applyTemplateChanges') }}</el-button>
          </div>
          <div>
            <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="doEdit" :loading="saving">{{ $t('common.save') }}</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="syncPreviewVisible" :title="$t('deployments.templateSyncPreview')" width="700px">
      <div v-if="syncPreviewData">
        <el-alert v-if="syncPreviewData.template_changed" type="warning" :closable="false" style="margin-bottom:12px">
          {{ $t('deployments.templateChanged') }}: {{ syncPreviewData.source_template_name }}
        </el-alert>
        <el-alert v-else type="success" :closable="false" style="margin-bottom:12px">
          {{ $t('deployments.templateUnchanged') }}
        </el-alert>
        <el-table v-if="syncPreviewData.diffs?.length" :data="syncPreviewData.diffs" stripe size="small">
          <el-table-column prop="field" :label="$t('deployments.syncField')" width="200" />
          <el-table-column :label="$t('deployments.syncDeployValue')">
            <template #default="{ row }"><code>{{ JSON.stringify(row.deploy_value) }}</code></template>
          </el-table-column>
          <el-table-column :label="$t('deployments.syncTemplateValue')">
            <template #default="{ row }"><code>{{ JSON.stringify(row.template_value) }}</code></template>
          </el-table-column>
        </el-table>
        <div v-if="!syncPreviewData.diffs?.length" style="padding:12px;color:var(--el-text-color-secondary)">
          {{ $t('deployments.noDiffs') }}
        </div>
      </div>
      <template #footer>
        <el-button @click="syncPreviewVisible = false">{{ $t('common.cancel') }}</el-button>
      </template>
    </el-dialog>

    <!-- Start Wizard -->
    <el-dialog v-model="wizardVisible" :title="$t('startWizard.title')" width="800px" :close-on-click-modal="false">
      <el-steps :active="wizardStep" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('startWizard.selectModel')" />
        <el-step :title="$t('startWizard.selectBackend')" />
        <el-step :title="$t('startWizard.selectVersion')" />
        <el-step :title="$t('startWizard.selectRuntime')" />
        <el-step :title="$t('startWizard.preflight')" />
        <el-step :title="$t('startWizard.start')" />
      </el-steps>

      <div v-if="wizardStep === 0">
        <el-select v-model="wizardModelId" :placeholder="$t('startWizard.selectModel')" style="width:100%" filterable @change="onWizAutoNext">
          <el-option v-for="m in models" :key="m.id" :label="`${m.name} (${m.format})`" :value="m.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizardModelId" @click="wizardStep=1">{{ $t('common.next') }}</el-button></div>
      </div>

      <div v-if="wizardStep === 1">
        <el-select v-model="wizardBackendId" :placeholder="$t('startWizard.selectBackend')" style="width:100%" filterable @change="onBackendSelected">
          <el-option v-for="b in backends" :key="b.id" :label="b.display_name || b.name" :value="b.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardBackendId" @click="wizardStep=2">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 2">
        <el-select v-model="wizardVersionId" :placeholder="$t('startWizard.selectVersion')" style="width:100%" filterable @change="wizardNBRId=''; if($event) { onVersionSelected($event) }">
          <el-option v-for="v in versions" :key="v.id" :label="v.display_name || v.version" :value="v.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardVersionId" @click="wizardStep=3">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 3">
        <div v-if="wizardVersionId && !nbrsLoaded" style="color:var(--el-text-color-secondary);margin-bottom:8px">
          {{ $t('common.loading') }}
        </div>
        <el-select v-model="wizardNBRId" :placeholder="$t('startWizard.selectRuntime')" style="width:100%" filterable @change="onNBRAutoPreflight">
          <el-option
            v-for="nbr in wizardNBRs"
            :key="nbr.id"
            :label="formatNBRLabel(nbr)"
            :value="nbr.id"
            :disabled="!isNBRDeployable(nbr)"
          >
            <div style="display:flex;justify-content:space-between;align-items:center;width:100%">
              <span style="flex:1;overflow:hidden;text-overflow:ellipsis">{{ formatNBRLabel(nbr) }}</span>
              <el-tag :type="nbrStatusTagType(nbr.status)" size="small" style="margin-left:8px;flex-shrink:0">{{ nbrStatusText(nbr.status) }}</el-tag>
            </div>
            <div v-if="!isNBRDeployable(nbr) && nbr.disabled_reason" style="font-size:11px;color:var(--el-color-danger);margin-top:2px">{{ nbr.disabled_reason }}</div>
          </el-option>
        </el-select>
        <el-alert v-if="wizardVersionId && wizardNBRs.length === 0 && nbrsLoaded" type="warning" :closable="false" style="margin-top:8px">
          <template #title>{{ $t('startWizard.noReadyNBR') }}</template>
          <template #default>
            <div style="font-size:12px;margin-top:4px">
              <router-link to="/runner-configs">{{ $t('startWizard.goToRunnerConfigs') }}</router-link>
            </div>
          </template>
        </el-alert>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=2">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardNBRId" @click="doPreflight">{{ $t('startWizard.preflight') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 4" v-loading="preflightLoading">
        <el-alert v-if="preflightResult" :type="preflightResult.can_run ? 'success' : 'warning'" :closable="false">
          {{ preflightResult.can_run ? $t('preflight.canRun') : $t('preflight.noNodes') }}
        </el-alert>
        <el-table v-if="preflightResult?.candidate_nodes?.length" :data="preflightResult.candidate_nodes" stripe size="small" style="margin-top:8px" @row-click="onPreflightNodeClick" highlight-current-row>
          <el-table-column :label="$t('modelLocations.node')">
            <template #default="{ row }">{{ nodeLabel(row.node_id) }}</template>
          </el-table-column>
          <el-table-column prop="status" :label="$t('preflight.canRun')" width="80" />
        </el-table>
        <div v-if="preflightResult?.errors?.length" style="margin-top:8px">
          <el-alert v-for="(e, idx) in preflightResult.errors" :key="idx" type="error" :closable="false">
            <template #title>
              {{ preflightErrorText(e) }}
            </template>
            <template v-if="e.context" #default>
              <div style="font-size:12px;color:var(--el-color-info);margin-top:4px">
                <span v-if="e.context.node_id">node: {{ e.context.node_id }}</span>
                <span v-if="e.context.artifact_id"> | artifact: {{ e.context.artifact_id }}</span>
                <span v-if="e.context.runtime_id"> | runtime: {{ e.context.runtime_id }}</span>
              </div>
            </template>
          </el-alert>
        </div>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=3">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!preflightResult?.can_run" @click="wizardStep=5">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="wizardStep === 5">
        <el-form label-width="120px">
          <el-form-item :label="$t('modelLocations.node')"><el-input v-model="wizardStartNode" disabled /></el-form-item>
          <el-form-item :label="$t('deployments.hostPort')"><el-input v-model.number="wizardHostPort" /></el-form-item>
          <el-form-item :label="$t('deployments.containerPort')"><el-input v-model.number="wizardContainerPort" /></el-form-item>
          <el-form-item :label="$t('deployments.appPort')"><el-input v-model.number="wizardAppPort" /></el-form-item>
        </el-form>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=4">{{ $t('common.prev') }}</el-button>
          <el-button @click="doWizardPreview" :loading="wizardStarting">{{ $t('deployments.previewRunPlan') }}</el-button>
          <el-button @click="doWizardSave" :loading="wizardStarting">{{ $t('deployments.saveConfig') }}</el-button>
          <el-button type="primary" @click="doWizardStart" :loading="wizardStarting">{{ $t('deployments.saveAndRun') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="runPlanVisible" :title="$t('deployments.finalRunPlan')" width="820px">
      <el-descriptions v-if="runPlanData" :column="1" border size="small">
        <el-descriptions-item :label="$t('runtimes.commandPreview')">{{ runPlanData }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import { useWizardAutoAdvance } from '@/composables/useWizardAutoAdvance'
import { translateStatus } from '@/utils/status'
import JsonViewer from '@/components/common/JsonViewer.vue'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'

const { t } = useI18n()

const loading = ref(false); const saving = ref(false)
const items = ref<any[]>([]); const models = ref<any[]>([]); const runtimes = ref<any[]>([]); const backends = ref<any[]>([]); const versions = ref<any[]>([]); const instances = ref<any[]>([])
const allNBRs = ref<any[]>([]); const nbrsLoaded = ref(false)
const createVisible = ref(false); const dryRunVisible = ref(false); const runPlanVisible = ref(false)
const dryRunResult = ref<any>(null); const runPlanData = ref('')
const editVisible = ref(false); const selectedEditRow = ref<any>(null)
const editForm = ref({ display_name: '', model_artifact_id: '', backend_runtime_id: '', host_port: 8000, container_port: 0, app_port: 0, original_name: '', source_template_name: '', source_backend_runtime_id: '', copied_at: '', env_overrides_text: '' })
const editParameterValues = ref<any[]>([])
const editDisabledParameters = ref<any[]>([])
const editParameterModel = computed({
  get: () => ({ docker_json: {}, args_override_json: [], default_env_json: {}, parameter_values_json: editParameterValues.value }),
  set: (val: any) => { if (val.parameter_values_json) editParameterValues.value = val.parameter_values_json },
})
const createForm = ref({ name: '', model_artifact_id: '', node_backend_runtime_id: '', node_id: '', accelerator_ids: '[]', host_port: 8000, container_port: 0, app_port: 0, placement_json: '{}', service_json: '{}', env_overrides_json: '{}' })

// Wizard state
const wizardVisible = ref(false); const wizardStep = ref(0)
const wizardModelId = ref(''); const wizardBackendId = ref(''); const wizardVersionId = ref(''); const wizardNBRId = ref('')
const wizardStartNode = ref(''); const wizardHostPort = ref(8004); const wizardContainerPort = ref(0); const wizardAppPort = ref(0); const wizardDeploymentId = ref('')
const preflightLoading = ref(false); const preflightResult = ref<any>(null)
const wizardStarting = ref(false)

const { onSelectAutoNext: onWizAutoNext } = useWizardAutoAdvance(wizardStep, () => { wizardStep.value++ })

onMounted(async () => { await refresh(); await loadRefs() })
async function refresh() {
  loading.value = true
  try {
    items.value = await apiClient.get('/deployments')
    try { instances.value = await apiClient.get('/model-instances') } catch { instances.value = [] }
  } catch (e: any) { console.error('deployments refresh failed', e) }
  loading.value = false
}
const { loadNodes, nodeLabel, nodes: nodeItems } = useNodeLabels()

async function loadRefs() {
  try { models.value = await apiClient.get('/model-artifacts') } catch { models.value = [] }
  try { runtimes.value = await apiClient.get('/backend-runtimes') } catch { runtimes.value = [] }
  try { backends.value = await apiClient.get('/backends') } catch { backends.value = [] }
  await loadNodes()
  await loadAllNBRs()
}

async function loadAllNBRs() {
  nbrsLoaded.value = false
  try {
    const nodeIds = nodeItems.value.map((n: any) => n.id)
    const results: any[] = []
    for (const nid of nodeIds) {
      try {
        const nbrs = await apiClient.get(`/nodes/${nid}/backend-runtimes`)
        for (const nbr of (nbrs || [])) {
          results.push({ ...nbr, _node_id: nid })
        }
      } catch { /* node may not have any NBRs */ }
    }
    allNBRs.value = results
  } catch { allNBRs.value = [] }
  nbrsLoaded.value = true
}

// Wizard NBRs filtered by selected version's backend_runtime_ids
const wizardNBRs = computed(() => {
  if (!wizardVersionId.value) return []
  // Find BR templates that belong to this version
  const versionRuntimeIds = new Set(
    runtimes.value
      .filter((r: any) => r.backend_version_id === wizardVersionId.value)
      .map((r: any) => r.id)
  )
  return allNBRs.value.filter((nbr: any) => versionRuntimeIds.has(nbr.backend_runtime_id))
})

// NBR display helpers
function formatNBRLabel(nbr: any): string {
  const node = nbr._node_id || nbr.node_id || ''
  const host = nodeLabel(node)
  const br = runtimes.value.find((r: any) => r.id === nbr.backend_runtime_id)
  const brName = br?.display_name || br?.name || nbr.backend_runtime_id || ''
  const img = nbr.image_ref || ''
  const parts = [host, brName]
  if (img) parts.push(img)
  return parts.join(' / ')
}

function isNBRDeployable(nbr: any): boolean {
  // API must return deployable field; missing means broken contract.
  return nbr.deployable === true
}

// Resolve model display name from artifact ID using the already-loaded models cache.
	function modelName(id: string): string {
	  if (!id) return '-'
	  const m = models.value.find((m: any) => m.id === id)
	  if (m) return m.display_name || m.name || id.slice(0, 8) + '...'
	  // Unknown model: show short id prefix
	  return id.length > 12 ? id.slice(0, 12) + '...' : id
	}

	function nbrStatusTagType(status: string): string {
  if (status === 'ready') return 'success'
  if (status === 'ready_with_warnings') return 'warning'
  if (status === 'needs_check') return 'warning'
  if (status === 'missing_image' || status === 'unsupported_device' || status === 'error') return 'danger'
  if (status === 'disabled') return 'info'
  return 'info'
}

function nbrStatusText(status: string): string {
  return translateStatus(status, t)
}

function asObject(value: any): Record<string, any> {
  if (!value) return {}
  if (typeof value === 'object' && !Array.isArray(value)) return value
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {}
    } catch { return {} }
  }
  return {}
}

function runtimeContext(row: any) {
  const rt = runtimes.value.find((r: any) => r.id === row.backend_runtime_id)
  const nbr = allNBRs.value.find((n: any) => n.id === row.source_node_backend_runtime_id)
  const snapshot = asObject(row.config_snapshot_json)
  const image = snapshot.image_name || snapshot.image || nbr?.image_ref || rt?.image_name || ''
  return {
    backend: rt?.backend_id || '',
    version: rt?.backend_version_id || row.source_template_version || '',
    runtime: nbr ? formatNBRLabel(nbr) : (row.source_template_name || rt?.display_name || rt?.name || row.backend_runtime_id),
    image,
  }
}

function deploymentContext(row: any) {
  const list = instances.value.filter((it: any) => it.deployment_id === row.id)
  const latest = list[0] || {}
  const placement = asObject(row.placement_json)
  return {
    node: latest.node_id ? nodeLabel(latest.node_id) : nodeLabel(placement.node_id || ''),
    endpoint: latest.endpoint_url || '',
    lastError: latest.last_error || '',
  }
}

function summarizeList(value: any): string {
  if (Array.isArray(value)) return value.map((v) => typeof v === 'object' ? JSON.stringify(v) : String(v)).join('\n')
  if (value && typeof value === 'object') return Object.entries(value).map(([k, v]) => `${k}=${typeof v === 'object' ? JSON.stringify(v) : v}`).join('\n')
  return value ? String(value) : ''
}

function asArray(value: any): any[] {
  if (Array.isArray(value)) return value
  if (value == null || value === '') return []
  return [value]
}

function parseKeyValueLines(value: string): Record<string, string> {
  const out: Record<string, string> = {}
  for (const raw of (value || '').split('\n')) {
    const line = raw.trim()
    if (!line) continue
    const idx = line.indexOf('=')
    if (idx > 0) out[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  }
  return out
}

function dryRunSummary(result: any) {
  const plan = asObject(result?.run_plan_json || result?.plan_json || result?.plan || result?.resolved_run_plan)
  const docker = asObject(plan.docker || plan.docker_run_spec || plan)
  // Derive served model name and source.
  const sn = plan.served_model_name || docker.served_model_name || ''
  const source = (result?.served_model_name_source)
    || (sn && sn === plan.model_name ? 'derived from artifact name' : '')
  // GPU source note.
  const gpuIDs = asArray(plan.gpu_device_ids || docker.gpu_device_ids || [])
  const gpuEnvKey = plan.gpu_visible_env_key || docker.gpu_visible_env_key || ''
  const gpuEnvVal = gpuEnvKey && docker.env ? (typeof docker.env === 'object' && !Array.isArray(docker.env) ? docker.env[gpuEnvKey] : '') : ''
  const gpuSource = gpuIDs.length > 0
    ? `来自 placement / accelerator_ids；若用户未手动选择，则来自当前单节点默认调度选择`
    : ''
  return {
    image: docker.image || plan.image || '',
    command: summarizeList([...asArray(docker.entrypoint), ...asArray(docker.command), ...asArray(docker.args)].filter(Boolean)),
    env: summarizeList(docker.env || plan.env),
    volumes: summarizeList(docker.volumes || plan.volumes),
    ports: summarizeList(docker.ports || plan.ports),
    devices: summarizeList(docker.devices || plan.devices),
    health: plan.health_check ? (typeof plan.health_check === 'object' ? JSON.stringify(plan.health_check) : String(plan.health_check)) : (docker.health_check ? JSON.stringify(docker.health_check) : ''),
    preview: result?.docker_preview || result?.command_preview || '',
    servedModelName: sn || '-',
    servedModelNameSource: sn ? `来自模型显示名 / deployment served model name 派生` : '',
    gpuSource,
    gpuEnv: gpuEnvKey ? `${gpuEnvKey}=${gpuEnvVal} (vendor binding 注入)` : '',
    dockerOptions: [
      plan.privileged ? '--privileged' : '',
      plan.ipc_mode ? `--ipc ${plan.ipc_mode}` : '',
      plan.shm_size ? `--shm-size ${plan.shm_size}` : '',
    ].filter(Boolean).join(', ') || '-',
  }
}

function onNBRAutoPreflight() {
  if (!wizardNBRId.value) return
  // Auto-set node from NBR
  const nbr = allNBRs.value.find((n: any) => n.id === wizardNBRId.value)
  if (nbr) {
    wizardStartNode.value = nbr._node_id || nbr.node_id || ''
  }
  doPreflight()
}

function showCreate() { createVisible.value = true }
async function doCreate() {
  saving.value = true
  try {
    const payload: any = { ...createForm.value }
    // If node_backend_runtime_id is set, resolve it for placement
    if (payload.node_backend_runtime_id) {
      const nbr = allNBRs.value.find((n: any) => n.id === payload.node_backend_runtime_id)
      if (nbr) {
        payload.placement_json = JSON.stringify({ node_id: nbr._node_id || nbr.node_id || createForm.value.node_id, accelerator_ids: JSON.parse(createForm.value.accelerator_ids || '[]') })
      } else {
        payload.placement_json = JSON.stringify({ node_id: createForm.value.node_id, accelerator_ids: JSON.parse(createForm.value.accelerator_ids || '[]') })
      }
    } else {
      payload.placement_json = JSON.stringify({ node_id: createForm.value.node_id, accelerator_ids: JSON.parse(createForm.value.accelerator_ids || '[]') })
    }
    payload.service_json = JSON.stringify(servicePayload(createForm.value.host_port, createForm.value.container_port, createForm.value.app_port))
    // Remove legacy fields not in create payload
    delete payload.accelerator_ids
    await apiClient.post('/deployments', payload)
    ElMessage.success(t('deployments.created')); createVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function doDryRun(row: any) {
  try { dryRunResult.value = await apiClient.post(`/deployments/${row.id}/dry-run`, {}) } catch (e: any) {}
  dryRunVisible.value = true
}
async function doStart(row: any) {
  try { const res = await apiClient.post(`/deployments/${row.id}/start`, {}); runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2); runPlanVisible.value = true; ElMessage.success(t('deployments.started')); await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function doStop(row: any) {
  try { await apiClient.post(`/deployments/${row.id}/stop`, {}); ElMessage.success(t('deployments.stopped')); await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function doRestart(row: any) {
  try { await apiClient.post(`/deployments/${row.id}/stop`, {}); const res = await apiClient.post(`/deployments/${row.id}/start`, {}); runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2); runPlanVisible.value = true; ElMessage.success(t('deployments.restarted')); await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function handleDelete(row: any) {
  try { await ElMessageBox.confirm(t('deployments.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' }); await apiClient.delete(`/deployments/${row.id}`); ElMessage.success(t('deployments.deleted')); await refresh() } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}

// ---- Deployment Edit ----
function showEdit(row: any) {
  selectedEditRow.value = row
  editForm.value.original_name = row.name || row.display_name || ''
  editForm.value.display_name = row.display_name || ''
  editForm.value.model_artifact_id = row.model_artifact_id || ''
  editForm.value.backend_runtime_id = row.source_node_backend_runtime_id || row.backend_runtime_id || ''
  editForm.value.source_template_name = row.source_template_name || ''
  editForm.value.source_backend_runtime_id = row.source_backend_runtime_id || ''
  editForm.value.copied_at = row.copied_at || ''
  try {
    const svc = typeof row.service_json === 'string' ? JSON.parse(row.service_json) : (row.service_json || {})
    editForm.value.host_port = svc.host_port || 8000
    editForm.value.container_port = svc.container_port || 0
    editForm.value.app_port = svc.app_port || 0
  } catch {
    editForm.value.host_port = 8000
    editForm.value.container_port = 0
    editForm.value.app_port = 0
  }
  const envOverrides = asObject(row.env_overrides_json)
  editForm.value.env_overrides_text = Object.entries(envOverrides).map(([k, v]) => `${k}=${v}`).join('\n')
  // Load structured parameter values and disabled tombstones
  editParameterValues.value = Array.isArray(row.parameter_values_json) ? [...row.parameter_values_json] : []
  editDisabledParameters.value = Array.isArray(row.disabled_parameters_json) ? [...row.disabled_parameters_json] : []
  editVisible.value = true
}

async function doEdit() {
  if (!selectedEditRow.value) return
  saving.value = true
  try {
    const payload: any = {
      display_name: editForm.value.display_name,
      model_artifact_id: editForm.value.model_artifact_id,
      service_json: servicePayload(editForm.value.host_port, editForm.value.container_port, editForm.value.app_port),
      env_overrides_json: parseKeyValueLines(editForm.value.env_overrides_text),
      parameter_values_json: editParameterValues.value,
      disabled_parameters_json: editDisabledParameters.value,
    }
    await apiClient.patch(`/deployments/${selectedEditRow.value.id}`, payload)
    ElMessage.success(t('deployments.saved'))
    editVisible.value = false
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

// ---- Start Wizard ----

// Map preflight error code to i18n-keyed user-facing text.
function preflightErrorText(e: any): string {
  if (!e || typeof e !== 'object') return String(e)
  if (e.code) {
    const codeMap: Record<string, string> = {
      model_location_missing: 'preflight.reason.modelLocationMissing',
      node_backend_runtime_not_ready: 'preflight.reason.nbrNotReady',
      node_offline: 'preflight.reason.nodeOffline',
      backend_version_mismatch: 'preflight.reason.backendVersionMismatch',
      docker_image_missing: 'preflight.reason.dockerImageMissing',
      runtime_disabled: 'preflight.reason.runtimeDisabled',
    }
    const i18nKey = codeMap[e.code]
    if (i18nKey) return t(i18nKey)
    return `[${e.code}] ${e.message || ''}`
  }
  return typeof e === 'string' ? e : (e.message || JSON.stringify(e))
}

async function onBackendSelected() {
  wizardVersionId.value = ''
  wizardNBRId.value = ''
  try { versions.value = await apiClient.get(`/backends/${wizardBackendId.value}/versions`) } catch { versions.value = [] }
  if (wizardBackendId.value) wizardStep.value = 2
}

async function onVersionSelected(versionId: string) {
  if (!versionId) return
  try {
    const v = versions.value.find((ver: any) => ver.id === versionId)
    if (v?.default_container_port && v.default_container_port > 0) {
      wizardContainerPort.value = v.default_container_port
      if (wizardAppPort.value === 0) wizardAppPort.value = v.default_container_port
    }
  } catch { /* keep defaults */ }
  // Ensure NBRs are loaded for this version
  if (allNBRs.value.length === 0) await loadAllNBRs()
  wizardStep.value = 3
}

function startWizard() {
  wizardVisible.value = true; wizardStep.value = 0
  wizardModelId.value = ''; wizardBackendId.value = ''; wizardVersionId.value = ''; wizardNBRId.value = ''
  versions.value = []; preflightResult.value = null; wizardStartNode.value = ''; wizardDeploymentId.value = ''
  wizardContainerPort.value = 0; wizardAppPort.value = 0
  loadRefs()
}

async function doPreflight() {
  preflightLoading.value = true; wizardStep.value = 4
  try {
    // Use node_backend_runtime_id if we selected an NBR
    const preflightPayload: any = { model_artifact_id: wizardModelId.value, host_port: wizardHostPort.value }
    if (wizardNBRId.value) {
      preflightPayload.node_backend_runtime_id = wizardNBRId.value
    }
    preflightResult.value = await apiClient.post('/deployments/preflight', preflightPayload)
    if (preflightResult.value?.candidate_nodes?.length) {
      wizardStartNode.value = preflightResult.value.candidate_nodes[0].node_id
    } else if (wizardStartNode.value) {
      // Keep the NBR's node even if preflight shows no candidates
    }
  } catch (e: any) { preflightResult.value = { can_run: false, errors: [e?.message || 'preflight failed'], candidate_nodes: [] } }
  preflightLoading.value = false
}

function onPreflightNodeClick(row: any) { wizardStartNode.value = row.node_id }

async function doWizardStart() {
  wizardStarting.value = true
  try {
    const deploy = await ensureWizardDeployment()
    const res = await apiClient.post(`/deployments/${deploy.id}/start`, {})
    runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2)
    runPlanVisible.value = true; wizardVisible.value = false
    ElMessage.success(t('deployments.started')); await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardStarting.value = false
}

async function doWizardSave() {
  wizardStarting.value = true
  try { await ensureWizardDeployment(); ElMessage.success(t('deployments.saved')); wizardVisible.value = false; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardStarting.value = false
}

async function doWizardPreview() {
  wizardStarting.value = true
  try { const deploy = await ensureWizardDeployment(); dryRunResult.value = await apiClient.post(`/deployments/${deploy.id}/dry-run`, {}); dryRunVisible.value = true; await refresh() } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardStarting.value = false
}

async function ensureWizardDeployment() {
  if (wizardDeploymentId.value) return { id: wizardDeploymentId.value }
  const name = `wizard-${Date.now()}`
  const payload: any = {
    name, display_name: name, model_artifact_id: wizardModelId.value,
    placement_json: { node_id: wizardStartNode.value, accelerator_ids: [] },
    service_json: servicePayload(wizardHostPort.value, wizardContainerPort.value, wizardAppPort.value),
    parameters_json: {}, env_overrides_json: {},
  }
  // Send node_backend_runtime_id when we have one selected
  if (wizardNBRId.value) {
    payload.node_backend_runtime_id = wizardNBRId.value
  }
  const deploy = await apiClient.post('/deployments', payload)
  wizardDeploymentId.value = deploy.id
  return deploy
}

function servicePayload(hostPort: number, containerPort?: number, appPort?: number) {
  const payload: any = { host_port: hostPort }
  if (containerPort && containerPort > 0) payload.container_port = containerPort
  if (appPort && appPort > 0) payload.app_port = appPort
  payload.health_port = hostPort
  payload.api_test_port = hostPort
  return payload
}

// ---- Template Sync ----
const syncPreviewVisible = ref(false)
const syncPreviewData = ref<any>(null)

async function doTemplateSyncPreview() {
  if (!selectedEditRow.value) return
  try {
    syncPreviewData.value = await apiClient.post(`/deployments/${selectedEditRow.value.id}/template-sync/preview`, {})
    syncPreviewVisible.value = true
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}

async function doTemplateSyncApply() {
  if (!selectedEditRow.value) return
  try {
    await ElMessageBox.confirm(
      t('deployments.syncConfirm'),
      t('deployments.applyTemplateChanges'),
      { type: 'warning', confirmButtonText: t('common.yes'), cancelButtonText: t('common.no') }
    )
    const res = await apiClient.post(`/deployments/${selectedEditRow.value.id}/template-sync/apply`, { strategy: 'preserve_overrides' })
    ElMessage.success(t('deployments.synced'))
    editVisible.value = false
    await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}

function isRunBlocked(status: string) { return ['starting', 'pending', 'provisioning', 'running', 'healthy', 'stopping'].includes(status) }
function deploymentStatusType(status: string) { if (['running', 'healthy'].includes(status)) return 'success'; if (['failed'].includes(status)) return 'danger'; if (['starting', 'pending', 'provisioning', 'stopping'].includes(status)) return 'warning'; return 'info' }
function deploymentStatusText(status: string) { return t(`deployments.status_${status || 'unknown'}`) }
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
</style>
