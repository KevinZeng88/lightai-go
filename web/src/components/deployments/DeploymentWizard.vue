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
          <el-empty :description="$t('deployments.noArtifacts') || 'No model artifacts available'">
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
          <el-empty :description="$t('runnerConfigs.noConfigs') || 'No node runtime configs available. Enable one first.'">
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
              {{ showAllRuntimes ? $t('runnerConfigs.showDeployableOnly') || 'Deployable only' : $t('runnerConfigs.showAll') || 'Show all (' + props.nodeRuntimes.length + ')' }}
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
        @update:overrides="form.config_overrides = $event"
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
}

const compatibilityError = ref('')

function checkNodeCompatibility(): boolean {
  compatibilityError.value = ''
  if (!form.model_artifact_id || !form.node_backend_runtime_id) return true

  const nbr = selectedNBR.value
  if (!nbr) return true

  const nbrNodeId = nbr.node_id
  const locs = (props.modelLocations || []).filter(
    (l: any) => l.model_artifact_id === form.model_artifact_id
  )
  const hasLocationOnNode = locs.some(
    (l: any) => l.node_id === nbrNodeId &&
      (l.verification_status === 'verified' || l.verification_status === 'warning' || l.verification_status === 'manually_accepted')
  )

  if (!hasLocationOnNode) {
    compatibilityError.value = t('deployments.nodeMismatch') ||
      'This model has no verified location on the selected runtime node. Choose an NBR on the same node, or add a model location for this node.'
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
    })
    if (activeStep.value === 3) activeStep.value = 4
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
  if (form.served_model_name) {
    const hasSMN = overrides.parameter_values.some((p: any) => p.key === 'backend.common.served_model_name')
    if (!hasSMN) {
      overrides.parameter_values.push({ key: 'backend.common.served_model_name', value: form.served_model_name, enabled: true })
    }
  }
  return {
    name: form.name || form.display_name || 'deployment',
    display_name: form.display_name,
    model_artifact_id: form.model_artifact_id,
    node_backend_runtime_id: form.node_backend_runtime_id,
    service_json: { host_port: form.host_port, container_port: form.container_port, served_model_name: form.served_model_name },
    config_overrides: overrides,
  }
}

defineExpose({ buildPayload, form })
</script>

<style scoped>
.deployment-wizard { max-width: 900px; margin: 0 auto; }
.wizard-content { margin: 24px 0; min-height: 300px; }
.wizard-footer { display: flex; justify-content: flex-end; gap: 8px; }
</style>
