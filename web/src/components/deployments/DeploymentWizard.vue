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
      <ModelSelector
        v-if="activeStep === 0"
        :artifacts="artifacts"
        :model-value="form.model_artifact_id"
        @update:model-value="form.model_artifact_id = $event"
      />

      <NodeRuntimeSelector
        v-if="activeStep === 1"
        :node-runtimes="deployableRuntimes"
        :model-value="form.node_backend_runtime_id"
        @update:model-value="onNBRSelected($event)"
      />

      <DeploymentServiceEditor
        v-if="activeStep === 2"
        v-model:host-port="form.host_port"
        v-model:container-port="form.container_port"
        v-model:served-model-name="form.served_model_name"
      />

      <DeploymentOverrideEditor
        v-if="activeStep === 3"
        :nbr-config-set="selectedNBRConfigSet"
        @update:overrides="form.config_overrides = $event"
      />

      <DeploymentPreviewPanel
        v-if="activeStep === 4"
        :preview-data="previewData"
        :loading="previewLoading"
        @preview="doPreview"
      />
    </div>

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
import ModelSelector from './ModelSelector.vue'
import NodeRuntimeSelector from './NodeRuntimeSelector.vue'
import DeploymentServiceEditor from './DeploymentServiceEditor.vue'
import DeploymentOverrideEditor from './DeploymentOverrideEditor.vue'
import DeploymentPreviewPanel from './DeploymentPreviewPanel.vue'
import { previewDeployment, type PreviewResult } from '@/api/deployments'

const props = defineProps<{
  artifacts: any[]
  nodeRuntimes: any[]
  saving?: boolean
}>()

const emit = defineEmits<{
  save: []
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

const deployableRuntimes = computed(() =>
  props.nodeRuntimes.filter((r: any) => r.deployable === true)
)

const selectedNBRConfigSet = computed(() => {
  const nbr = props.nodeRuntimes.find((r: any) => r.id === form.node_backend_runtime_id)
  return nbr?.config_set || null
})

function onNBRSelected(nbrID: string) {
  form.node_backend_runtime_id = nbrID
}

function nextStep() {
  const s = activeStep.value
  if (s === 0 && !form.model_artifact_id) return
  if (s === 1 && !form.node_backend_runtime_id) return
  if (s === 3 && activeStep.value < 4) {
    doPreview()
    return
  }
  if (s < 4) activeStep.value++
}

async function doPreview() {
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
