<template>
  <div class="deployment-wizard">
    <el-steps :active="activeStep" align-center finish-status="success">
      <el-step :title="$t('deployments.wizardStepModel')" />
      <el-step :title="$t('deployments.wizardStepRuntime')" />
      <el-step :title="$t('deployments.wizardStepService')" />
      <el-step :title="$t('deployments.wizardStepOverrides')" />
      <el-step :title="$t('deployments.wizardStepPreview')" />
    </el-steps>

    <div class="wizard-content">
      <!-- Step 0: Model -->
      <div v-if="activeStep === 0">
        <div v-if="!hasArtifacts" style="text-align:center;padding:40px">
          <el-empty :description="$t('deployments.noArtifacts')">
            <el-button @click="$emit('refreshData')">{{ $t('common.refresh') }}</el-button>
          </el-empty>
        </div>
        <ModelSelector
          v-else
          :artifacts="artifacts"
          :model-value="form.model_artifact_id"
          @update:model-value="form.model_artifact_id = $event"
        />
      </div>

      <!-- Step 1: Node Runtime Config -->
      <div v-if="activeStep === 1">
        <div v-if="!hasRuntimes" style="text-align:center;padding:40px">
          <el-empty :description="$t('runnerConfigs.noConfigs')">
            <el-button @click="$emit('refreshData')">{{ $t('common.refresh') }}</el-button>
          </el-empty>
        </div>
        <div v-else>
          <NodeRuntimeSelector
            :node-runtimes="deployableRuntimes"
            :model-value="form.node_backend_runtime_id"
            @update:model-value="onNBRSelected($event)"
          />
          <div style="margin-top:8px; text-align:right">
            <el-button v-if="deployableRuntimes.length < props.nodeRuntimes.length" size="small" @click="showAllRuntimes = !showAllRuntimes">
              {{ showAllRuntimes ? $t('runnerConfigs.showDeployableOnly') : $t('runnerConfigs.showAll') }}
            </el-button>
            <el-button size="small" @click="$emit('refreshData')">{{ $t('common.refresh') }}</el-button>
          </div>
        </div>
      </div>

      <!-- Step 2: Service -->
      <DeploymentServiceEditor
        v-if="activeStep === 2"
        v-model:host-port="form.host_port"
        v-model:container-port="form.container_port"
        v-model:served-model-name="form.served_model_name"
      />

      <!-- Step 3: Overrides -->
      <DeploymentOverrideEditor
        v-if="activeStep === 3"
        :nbr-config-set="selectedNBRConfigSet"
        :nbr-id="form.node_backend_runtime_id"
        @update:overrides="form.config_overrides = $event"
        @update:patch="form.editable_config_patch = $event"
      />

      <!-- Step 4: Preview -->
      <DeploymentPreviewPanel
        v-if="activeStep === 4"
        :preview-data="previewData"
        :loading="previewLoading"
        @preview="doPreview"
      />
    </div>

    <el-alert
      v-if="compatibilityError"
      type="error"
      :title="compatibilityError"
      show-icon
      :closable="false"
      style="margin-bottom:12px"
    />

    <div class="wizard-footer">
      <el-button v-if="activeStep > 0" @click="activeStep--">{{ $t('common.prev') }}</el-button>
      <el-button v-if="activeStep < 4" type="primary" @click="nextStep">{{ $t('common.next') }}</el-button>
      <el-button v-if="activeStep === 4" type="primary" :loading="saving" @click="$emit('save')">
        {{ $t('deployments.saveConfig') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelSelector from './ModelSelector.vue'
import NodeRuntimeSelector from './NodeRuntimeSelector.vue'
import DeploymentServiceEditor from './DeploymentServiceEditor.vue'
import DeploymentOverrideEditor from './DeploymentOverrideEditor.vue'
import DeploymentPreviewPanel from './DeploymentPreviewPanel.vue'
import { previewDeployment, type PreviewResult } from '@/api/deployments'
import type { ConfigEditPatch } from '@/utils/configEditView'
import { apiErrorMessage } from '@/utils/apiErrors'

const { t } = useI18n()

const props = defineProps<{
  artifacts: any[]
  nodeRuntimes: any[]
  modelLocations?: any[]
  saving?: boolean
}>()

const emit = defineEmits<{
  save: []
  refreshData: []
}>()

const activeStep = ref(0)
const previewLoading = ref(false)
const previewData = ref<PreviewResult | null>(null)

const form = reactive({
  name: '',
  display_name: '',
  model_artifact_id: '',
  node_backend_runtime_id: '',
  host_port: 8000,
  container_port: 8000,
  served_model_name: '',
  config_overrides: {} as Record<string, any>,
  editable_config_patch: null as ConfigEditPatch | null,
})

const showAllRuntimes = ref(false)

const deployableRuntimes = computed(() => {
  if (showAllRuntimes.value) return props.nodeRuntimes
  return props.nodeRuntimes.filter((r: any) =>
    r.deployable === true || r.status === 'ready' || r.status === 'ready_with_warnings'
  )
})

const hasArtifacts = computed(() => props.artifacts && props.artifacts.length > 0)
const hasRuntimes = computed(() => props.nodeRuntimes && props.nodeRuntimes.length > 0)
const selectedArtifact = computed(() =>
  props.artifacts.find((artifact: any) => artifact.id === form.model_artifact_id)
)

function isNBRDeployable(nbr: any): boolean {
  if (!nbr) return false
  if (nbr.deployable === true) return true
  if (nbr.status === 'ready' || nbr.status === 'ready_with_warnings') return true
  return false
}

const selectedNBR = computed(() =>
  props.nodeRuntimes.find((r: any) => r.id === form.node_backend_runtime_id)
)

const selectedNBRConfigSet = computed(() => {
  return selectedNBR.value?.config_set || null
})

function onNBRSelected(nbrID: string) {
  form.node_backend_runtime_id = nbrID
  const nbr = props.nodeRuntimes.find((r: any) => r.id === nbrID)
  const port = configSetValue(nbr?.config_set, 'service.container_port')
  const numericPort = Number(port)
  if (Number.isFinite(numericPort) && numericPort > 0) {
    form.container_port = numericPort
    form.host_port = numericPort
  }
}

const compatibilityError = ref('')

function validateServiceConfig(): boolean {
  compatibilityError.value = ''
  if (!Number.isFinite(Number(form.host_port)) || form.host_port < 1 || form.host_port > 65535) {
    compatibilityError.value = t('deployments.invalidHostPort')
    return false
  }
  if (!Number.isFinite(Number(form.container_port)) || form.container_port < 1 || form.container_port > 65535) {
    compatibilityError.value = t('deployments.invalidContainerPort')
    return false
  }
  return true
}

function checkNodeCompatibility(): boolean {
  compatibilityError.value = ''
  if (!form.model_artifact_id || !form.node_backend_runtime_id) return true

  const nbr = selectedNBR.value
  if (!nbr) return true

  const nbrNodeId = nbr.node_id
  const explicitLocations = (props.modelLocations || []).filter(
    (l: any) => l.model_artifact_id === form.model_artifact_id
  )
  const artifactLocations = Array.isArray(selectedArtifact.value?.locations)
    ? selectedArtifact.value.locations
    : []
  const locs = explicitLocations.length > 0 ? explicitLocations : artifactLocations
  const deployableVerificationStatuses = ['verified', 'warning', 'manually_accepted']
  const deployableMatchStatuses = ['exact_match', 'probable_match', 'manual_attested']
  const hasLocationOnNode = locs.some(
    (l: any) => l.node_id === nbrNodeId &&
      deployableVerificationStatuses.includes(l.verification_status) &&
      deployableMatchStatuses.includes(l.match_status)
  )

  if (!hasLocationOnNode) {
    const visibleLocations = locs.map((l: any) =>
      `id=${l.id || ''} node_id=${l.node_id || ''} verification_status=${l.verification_status || ''} match_status=${l.match_status || ''} last_error=${l.last_error || ''}`
    ).join('; ') || '<none>'
    const artifactName = selectedArtifact.value?.display_name || selectedArtifact.value?.name || form.model_artifact_id
    compatibilityError.value = `${t('deployments.nodeMismatch')} model_artifact_id=${form.model_artifact_id} model=${artifactName} node_id=${nbrNodeId} visibleLocations=${visibleLocations}`
    return false
  }
  return true
}

function nextStep() {
  const s = activeStep.value
  if (s === 0 && !form.model_artifact_id) return
  if (s === 1) {
    if (!form.node_backend_runtime_id) return
    if (!isNBRDeployable(selectedNBR.value)) return
    if (!checkNodeCompatibility()) return
  }
  if (s === 2) {
    if (!validateServiceConfig()) return
    activeStep.value = 3
    return
  }
  if (s === 3 && activeStep.value < 4) {
    doPreview()
    return
  }
  if (s < 4) activeStep.value++
}

async function doPreview() {
  if (!checkNodeCompatibility()) return
  previewLoading.value = true
  try {
    previewData.value = await previewDeployment({
      name: form.name || 'preview',
      model_artifact_id: form.model_artifact_id,
      node_backend_runtime_id: form.node_backend_runtime_id,
      service_json: { host_port: form.host_port, container_port: form.container_port, served_model_name: form.served_model_name },
      config_overrides: form.config_overrides,
      editable_config_patch: form.editable_config_patch,
    })
    if (activeStep.value === 3) activeStep.value = 4
  } catch (e: any) {
    compatibilityError.value = apiErrorMessage(e, t, 'common.requestFailed')
  } finally {
    previewLoading.value = false
  }
}

function buildPayload() {
  // Guard: reject non-deployable NBR
  if (!isNBRDeployable(selectedNBR.value)) {
    return null
  }
  // Guard: reject node mismatch
  if (!checkNodeCompatibility()) {
    return null
  }
  const overrides: Record<string, any> = { parameter_values: [] }
  if (form.config_overrides?.parameter_values) {
    overrides.parameter_values = form.config_overrides.parameter_values
  }
  return {
    name: form.name || form.display_name || 'deployment',
    display_name: form.display_name,
    model_artifact_id: form.model_artifact_id,
    node_backend_runtime_id: form.node_backend_runtime_id,
    service_json: { host_port: form.host_port, container_port: form.container_port, served_model_name: form.served_model_name },
    config_overrides: overrides,
    editable_config_patch: form.editable_config_patch,
  }
}

function configSetValue(configSet: any, key: string): any {
  const item = configSet?.items?.[key]
  const value = item?.value
  if (value?.effective_value !== undefined && value?.effective_value !== null) return value.effective_value
  if (value?.default_value !== undefined && value?.default_value !== null) return value.default_value
  return undefined
}

defineExpose({ buildPayload, form, resetWizard: () => { activeStep.value = 0 } })
</script>

<style scoped>
.deployment-wizard { max-width: 900px; margin: 0 auto; }
.wizard-content { margin: 24px 0; min-height: 300px; }
.wizard-footer { display: flex; justify-content: flex-end; gap: 8px; }
</style>
