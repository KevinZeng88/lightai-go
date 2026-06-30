<template>
  <div class="nbr-wizard">
    <el-steps :active="activeStep" align-center finish-status="success">
      <el-step :title="$t('runnerConfigs.wizardStepNode')" />
      <el-step :title="$t('runnerConfigs.wizardStepTemplate')" />
      <el-step :title="$t('runnerConfigs.wizardStepImage')" />
      <el-step :title="$t('runnerConfigs.wizardStepCheck')" />
    </el-steps>

    <WizardActionBar
      :active-step="activeStep"
      :total-steps="4"
      :can-prev="activeStep > 0"
      :can-next="actionCanProceed"
      :primary-label="primaryActionLabel"
      :primary-loading="primaryActionLoading"
      :next-disabled-reason="actionDisabledReason"
      :secondary-actions="secondaryActions"
      layout="sticky-top"
      @cancel="emit('cancel')"
      @prev="activeStep--"
      @primary="onPrimaryAction"
      @secondary="onSecondaryAction"
    />

    <div class="wizard-content">
      <!-- Step 0: Select Node -->
      <div v-if="activeStep === 0">
        <NodeSelectorTable
          :nodes="nodes"
          :loading="nodesLoading"
          :error="nodesError"
          :label="$t('nodeSelector.selectRuntimeNode')"
          @select="onNodeSelected"
          @refresh="loadNodes"
        />
      </div>

      <!-- Step 1: Select Runtime Template -->
      <div v-if="activeStep === 1">
        <div v-if="runtimesLoading" style="text-align:center;padding:40px">
          <el-icon class="is-loading" :size="32"><Loading /></el-icon>
          <p>{{ $t('common.loading') }}</p>
        </div>
        <div v-else-if="runtimesError" style="text-align:center;padding:40px">
          <el-result icon="error" :title="$t('common.error')" :sub-title="runtimesError">
            <template #extra><el-button @click="loadRuntimes">{{ $t('common.refresh') }}</el-button></template>
          </el-result>
        </div>
        <div v-else-if="!runtimes.length" style="text-align:center;padding:40px">
          <el-empty :description="$t('runtimes.noRuntimes')">
            <el-button @click="loadRuntimes">{{ $t('common.refresh') }}</el-button>
          </el-empty>
        </div>
        <el-table v-else :data="displayRuntimes" highlight-current-row @current-change="onDisplayRuntimeSelected" max-height="400">
          <el-table-column :label="$t('runtimes.name')" min-width="200">
            <template #default="{ row }">{{ row.displayName }}</template>
          </el-table-column>
          <el-table-column :label="$t('runtimes.backend')" width="120">
            <template #default="{ row }">{{ row.backendDisplay }}</template>
          </el-table-column>
          <el-table-column :label="$t('runtimes.vendor')" width="100">
            <template #default="{ row }">{{ row.vendorDisplay }}</template>
          </el-table-column>
          <el-table-column :label="$t('runtimes.backendVersion')" width="100">
            <template #default="{ row }">{{ row.versionDisplay || '-' }}</template>
          </el-table-column>
          <el-table-column prop="image" :label="$t('runtimes.image')" min-width="240" show-overflow-tooltip />
          <el-table-column :label="$t('runtimes.source')" width="100">
            <template #default="{ row }">{{ row.sourceType === 'user' ? $t('runtimes.userConfig') : $t('runtimes.builtinTemplate') }}</template>
          </el-table-column>
        </el-table>
        <div v-if="runtimes.length" style="margin-top:12px; text-align:right">
          <el-button size="small" @click="loadRuntimes">{{ $t('common.refresh') }}</el-button>
        </div>
        <div v-if="selectedRuntime" style="margin-top:12px">
          <el-tag type="success">{{ $t('runnerConfigs.selectedTemplate') }}: {{ selectedRuntimeDisplay?.displayName || selectedRuntime.display_name || selectedRuntime.name }}</el-tag>
        </div>
      </div>

      <!-- Step 2: Config name, image, and parameters -->
      <div v-if="activeStep === 2">
        <el-form label-position="top">
          <el-form-item :label="$t('runnerConfigs.configName')">
            <el-input v-model="form.display_name" :placeholder="defaultConfigName" />
          </el-form-item>
          <el-form-item :label="$t('runtimes.image')">
            <DockerImagePicker
              v-if="selectedNode?.id"
              :node-id="selectedNode.id"
              :initial-ref="form.image_ref || selectedRuntime?.image_ref || ''"
              @select="onImageSelected"
            />
            <el-input
              v-else
              v-model="form.image_ref"
              :placeholder="selectedRuntime?.image_ref || $t('runnerConfigs.selectImage')"
            />
          </el-form-item>
        </el-form>
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="$t('runtimes.structuredParameters')">
            <ConfigEditView
              v-if="runtimeEditView"
              :model-value="runtimeEditView"
              @update:patch="onSchemaParamOutput"
            />
            <el-empty v-else :description="$t('common.noData')" />
          </el-collapse-item>
        </el-collapse>
      </div>

      <!-- Step 3: Save and check -->
      <div v-if="activeStep === 3">
        <h4>{{ $t('runnerConfigs.summary') }}</h4>
        <el-descriptions :column="2" border size="small" style="margin-bottom:16px">
          <el-descriptions-item :label="$t('deployments.node')">{{ selectedNode?.name || selectedNode?.id || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.runtime')">{{ selectedRuntime?.display_name || selectedRuntime?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.configName')">{{ form.display_name || defaultConfigName }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ form.image_ref || selectedRuntime?.image_ref || '-' }}</el-descriptions-item>
        </el-descriptions>

        <!-- Error display -->
        <el-alert v-if="wizardError" type="error" :title="wizardError" show-icon :closable="false" style="margin-bottom:12px" />

        <!-- Check result display -->
        <div v-if="checkResult" style="margin-bottom:16px">
          <el-alert
            :type="checkResult.deployable ? 'success' : 'warning'"
            :title="checkResultTitle"
            :description="checkResultDescription"
            show-icon :closable="false"
          />
          <div v-if="checkResult.warnings?.length" style="margin-top:8px">
            <div v-for="(w, i) in checkResult.warnings" :key="i" style="color:var(--el-color-warning);font-size:12px">{{ w }}</div>
          </div>
        </div>

        <el-alert v-if="checkResult?.deployable" type="success" :title="$t('runnerConfigs.finish')" show-icon :closable="false" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { getConfigEditView } from '@/api/configEdit'
import { listRuntimes } from '@/api/runtimes'
import { toRuntimeTemplateDisplay, type RuntimeTemplateDisplay } from '@/utils/runtimeDisplay'
import { apiErrorMessage } from '@/utils/apiErrors'
import { translateStatus, translateStatusReason } from '@/utils/status'
import NodeSelectorTable from '@/components/common/NodeSelectorTable.vue'
import WizardActionBar from '@/components/common/WizardActionBar.vue'
import ConfigEditView from '@/components/config/ConfigEditView.vue'
import DockerImagePicker from '@/components/DockerImagePicker.vue'
import type { ConfigEditPatch, ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'

const { t } = useI18n()

const emit = defineEmits<{
  completed: []
  cancel: []
}>()

const activeStep = ref(0)
const savingState = ref<'idle' | 'saving' | 'save_failed' | 'checking' | 'check_failed' | 'checked_ready'>('idle')
const wizardError = ref('')
const nodesLoading = ref(false)
const nodesError = ref('')
const runtimesLoading = ref(false)
const runtimesError = ref('')
const nodes = ref<any[]>([])
const runtimes = ref<any[]>([])
const displayRuntimes = computed(() => runtimes.value.map(toRuntimeTemplateDisplay))
const selectedNode = ref<any>(null)
const selectedRuntime = ref<any>(null)
const selectedRuntimeDisplay = computed(() => {
  if (!selectedRuntime.value) return null
  return toRuntimeTemplateDisplay(selectedRuntime.value)
})
const paramOverrides = ref<ConfigEditPatch | null>(null)
const runtimeEditView = ref<ConfigEditViewModel | null>(null)
const checkResult = ref<any>(null)

const checkResultTitle = computed(() => {
  if (!checkResult.value) return ''
  return translateStatus(checkResult.value.status || '', t)
})

const checkResultDescription = computed(() => {
  if (!checkResult.value) return ''
  if (checkResult.value.deployable) {
    return t('runnerConfigs.checkPassedWithImage', { image: checkResult.value.checked_image_ref || checkResult.value.image_ref || '-' })
  }
  return translateStatusReason(checkResult.value.status_reason || '', t)
})

const form = reactive({
  display_name: '',
  image_ref: '',
})

const defaultConfigName = computed(() => {
  const host = selectedNode.value?.name || selectedNode.value?.hostname || selectedNode.value?.id || 'node'
  const display = selectedRuntimeDisplay.value
  const vendorName = display?.vendorDisplay || selectedRuntime.value?.vendor || 'unknown'
  const backendName = display?.backendDisplay || (selectedRuntime.value?.backend_id || '').replace(/^backend\./, '')
  return `${host} / ${vendorName} / ${backendName}`
})

const canProceed = computed(() => {
  const s = activeStep.value
  if (s === 0) return !!selectedNode.value
  if (s === 1) return !!selectedRuntime.value
  return true
})

const cannotProceedReason = computed(() => {
  const s = activeStep.value
  if (s === 0 && !selectedNode.value) return t('runnerConfigs.selectNode')
  if (s === 1 && !selectedRuntime.value) return t('runnerConfigs.selectTemplate')
  return ''
})

const primaryActionLabel = computed(() => {
  if (activeStep.value < 3) return t('common.next')
  if (checkResult.value?.deployable) return t('runnerConfigs.finish')
  return t('runnerConfigs.saveAndCheck')
})

const primaryActionLoading = computed(() => savingState.value === 'checking')

const secondaryActions = computed(() => {
  if (activeStep.value !== 3 || checkResult.value?.deployable) return []
  return [{
    key: 'save-only',
    label: t('runnerConfigs.saveOnly'),
    type: 'default' as const,
    loading: savingState.value === 'saving',
  }]
})

const actionCanProceed = computed(() => {
  if (activeStep.value < 3) return canProceed.value
  return savingState.value !== 'saving' && savingState.value !== 'checking'
})

const actionDisabledReason = computed(() => {
  if (activeStep.value < 3) return cannotProceedReason.value
  return ''
})

function resetWizard() {
  activeStep.value = 0
  selectedNode.value = null
  selectedRuntime.value = null
  form.display_name = ''
  form.image_ref = ''
  paramOverrides.value = null
  runtimeEditView.value = null
  checkResult.value = null
  wizardError.value = ''
  savingState.value = 'idle'
}

function onNodeSelected(node: any) {
  selectedNode.value = node
}

function onDisplayRuntimeSelected(displayRow: RuntimeTemplateDisplay | null) {
  if (!displayRow) return
  selectedRuntime.value = displayRow.raw
  form.image_ref = displayRow.raw.image_ref || ''
}

function onRuntimeSelected(row: any) {
  // Keep for backward compat; new code uses onDisplayRuntimeSelected.
  if (!row) return
  selectedRuntime.value = row
  form.image_ref = row.image_ref || ''
}

function onImageSelected(image: any) {
  form.image_ref = image?.image_ref || ''
}

watch(selectedRuntime, async (runtime) => {
  paramOverrides.value = null
  runtimeEditView.value = null
  if (!runtime?.id) return
  runtimeEditView.value = await getConfigEditView({
    object_kind: 'backend_runtime',
    object_id: runtime.id,
    layer: 'node_backend_runtime',
    mode: 'enable',
    view_level: 'advanced',
  })
})

function onSchemaParamOutput(output: ConfigEditPatch) {
  paramOverrides.value = output
}

function nextStep() {
  if (activeStep.value < 3) activeStep.value++
}

function onPrimaryAction() {
  if (activeStep.value < 3) {
    nextStep()
    return
  }
  if (checkResult.value?.deployable) {
    finish()
    return
  }
  doSaveAndCheck()
}

function onSecondaryAction(key: string) {
  if (key === 'save-only') doSave()
}

async function loadNodes() {
  nodesLoading.value = true
  nodesError.value = ''
  try {
    nodes.value = await apiClient.get('/nodes')
  } catch (e: any) {
    nodesError.value = apiErrorMessage(e, t)
  } finally {
    nodesLoading.value = false
  }
}

async function loadRuntimes() {
  runtimesLoading.value = true
  runtimesError.value = ''
  try {
    runtimes.value = await listRuntimes()
  } catch (e: any) {
    runtimesError.value = apiErrorMessage(e, t)
  } finally {
    runtimesLoading.value = false
  }
}

async function doSave() {
  await saveAndMaybeCheck(false)
}

async function doSaveAndCheck() {
  await saveAndMaybeCheck(true)
}

async function saveAndMaybeCheck(andCheck: boolean) {
  if (!selectedNode.value || !selectedRuntime.value) return
  wizardError.value = ''
  savingState.value = 'saving'
  try {
    const payload: Record<string, any> = {
      backend_runtime_id: selectedRuntime.value.id,
      display_name: form.display_name || defaultConfigName.value,
      image_ref: form.image_ref || undefined,
    }
    if (paramOverrides.value) {
      payload.editable_config_patch = paramOverrides.value
    }
    const enableResp = await apiClient.post(`/nodes/${selectedNode.value.id}/backend-runtimes/enable`, payload)
    const nbrId = enableResp?.id

    if (!andCheck) {
      ElMessage.success(t('common.saved'))
      checkResult.value = null
      savingState.value = 'idle'
      return
    }

    if (!nbrId) {
      wizardError.value = t('runnerConfigs.missingNBRID')
      savingState.value = 'save_failed'
      return
    }

    savingState.value = 'checking'
    try {
      const checkResp = await apiClient.post(`/nodes/${selectedNode.value.id}/backend-runtimes/${nbrId}/check-request`, {})
      checkResult.value = checkResp
      if (checkResp?.deployable) {
        savingState.value = 'checked_ready'
      } else {
        savingState.value = 'idle'
      }
    } catch (e: any) {
      wizardError.value = apiErrorMessage(e, t, 'runnerConfigs.checkFailed')
      savingState.value = 'check_failed'
    }
  } catch (e: any) {
    wizardError.value = apiErrorMessage(e, t, 'common.requestFailed')
    savingState.value = 'save_failed'
  }
}

function finish() {
  emit('completed')
}

// Load on init
loadNodes()
loadRuntimes()

defineExpose({ resetWizard, saving: savingState })
</script>

<style scoped>
.nbr-wizard { max-width: 900px; margin: 0 auto; }
.wizard-content { margin: 24px 0; min-height: 300px; }
</style>
