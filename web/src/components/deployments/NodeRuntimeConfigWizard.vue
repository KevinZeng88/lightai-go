<template>
  <div class="nbr-wizard">
    <el-steps :active="activeStep" align-center finish-status="success">
      <el-step :title="$t('runnerConfigs.wizardStepNode')" />
      <el-step :title="$t('runnerConfigs.wizardStepTemplate')" />
      <el-step :title="$t('runnerConfigs.wizardStepImage')" />
      <el-step :title="$t('runnerConfigs.wizardStepCheck')" />
    </el-steps>

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
          <el-empty :description="$t('runtimes.noRuntimes') || 'No runtime templates'">
            <el-button @click="loadRuntimes">{{ $t('common.refresh') }}</el-button>
          </el-empty>
        </div>
        <el-table v-else :data="runtimes" highlight-current-row @current-change="onRuntimeSelected" max-height="400">
          <el-table-column :label="$t('runtimes.name')" min-width="200">
            <template #default="{ row }">{{ row.display_name || row.name }}</template>
          </el-table-column>
          <el-table-column prop="backend_id" :label="$t('runtimes.backend')" width="120" />
          <el-table-column prop="vendor" :label="$t('runtimes.vendor')" width="100" />
          <el-table-column prop="image_ref" :label="$t('runtimes.image')" min-width="240" show-overflow-tooltip />
        </el-table>
        <div v-if="runtimes.length" style="margin-top:12px; text-align:right">
          <el-button size="small" @click="loadRuntimes">{{ $t('common.refresh') }}</el-button>
        </div>
        <div v-if="selectedRuntime" style="margin-top:12px">
          <el-tag type="success">{{ $t('runnerConfigs.selectedTemplate') }}: {{ selectedRuntime.display_name || selectedRuntime.name }}</el-tag>
        </div>
      </div>

      <!-- Step 2: Config name, image, and parameters -->
      <div v-if="activeStep === 2">
        <el-form label-position="top">
          <el-form-item :label="$t('runnerConfigs.configName')">
            <el-input v-model="form.display_name" :placeholder="defaultConfigName" />
          </el-form-item>
          <el-form-item :label="$t('runtimes.image')">
            <el-input v-model="form.image_ref" :placeholder="selectedRuntime?.image_ref || ''" />
          </el-form-item>
        </el-form>
        <el-divider content-position="left">{{ $t('runtimes.structuredParameters') }}</el-divider>
        <RuntimeParameterEditor
          v-if="runtimeConfigForEditor"
          v-model="editorModel"
          :vendor="selectedRuntime?.vendor || 'nvidia'"
          :layer="'node_backend_runtime'"
          :show-advanced="true"
          @update:model-value="onSchemaParamOutput"
        />
        <el-empty v-else :description="$t('common.noData')" />
      </div>

      <!-- Step 3: Save and check -->
      <div v-if="activeStep === 3">
        <h4>{{ $t('runnerConfigs.summary') || 'Summary' }}</h4>
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
            :title="checkResult.status"
            :description="checkResult.status_reason || ''"
            show-icon :closable="false"
          />
          <div v-if="checkResult.warnings?.length" style="margin-top:8px">
            <div v-for="(w, i) in checkResult.warnings" :key="i" style="color:var(--el-color-warning);font-size:12px">{{ w }}</div>
          </div>
        </div>

        <div style="text-align:center;display:flex;gap:8px;justify-content:center">
          <el-button type="primary" :loading="savingState === 'saving'" @click="doSave">
            {{ $t('runnerConfigs.saveOnly') }}
          </el-button>
          <el-button type="primary" :loading="savingState === 'checking'" @click="doSaveAndCheck">
            {{ $t('runnerConfigs.saveAndCheck') }}
          </el-button>
          <el-button
            v-if="checkResult?.deployable"
            type="success"
            @click="finish"
          >
            {{ $t('runnerConfigs.finish') }}
          </el-button>
        </div>
      </div>
    </div>

    <div class="wizard-footer">
      <el-button v-if="activeStep > 0" @click="activeStep--">{{ $t('common.prev') }}</el-button>
      <el-button v-if="activeStep < 3" type="primary" @click="nextStep" :disabled="!canProceed">
        {{ $t('common.next') }}
      </el-button>
      <span v-if="!canProceed && activeStep < 3" style="color:var(--el-color-warning);font-size:12px;margin-left:8px">
        {{ cannotProceedReason }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { listRuntimes } from '@/api/runtimes'
import NodeSelectorTable from '@/components/common/NodeSelectorTable.vue'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'

const { t } = useI18n()

const emit = defineEmits<{
  completed: []
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
const selectedNode = ref<any>(null)
const selectedRuntime = ref<any>(null)
const paramOverrides = ref<Record<string, any>>({})
const editorModel = ref<Record<string, any>>({ config_set: {} })
const checkResult = ref<any>(null)

const form = reactive({
  display_name: '',
  image_ref: '',
})

const defaultConfigName = computed(() => {
  const host = selectedNode.value?.name || selectedNode.value?.hostname || selectedNode.value?.id || 'node'
  const vendor = selectedRuntime.value?.vendor || 'unknown'
  const backend = selectedRuntime.value?.backend_id || ''
  const name = backend.replace(/^backend\./, '')
  return `${host} / ${vendor} / ${name}`
})

const runtimeConfigForEditor = computed(() => {
  if (!selectedRuntime.value) return null
  return { config_set: selectedRuntime.value.config_set || {} }
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

function resetWizard() {
  activeStep.value = 0
  selectedNode.value = null
  selectedRuntime.value = null
  form.display_name = ''
  form.image_ref = ''
  paramOverrides.value = {}
  editorModel.value = { config_set: {} }
  checkResult.value = null
  wizardError.value = ''
  savingState.value = 'idle'
}

function onNodeSelected(node: any) {
  selectedNode.value = node
}

function onRuntimeSelected(row: any) {
  if (!row) return
  selectedRuntime.value = row
  form.image_ref = row.image_ref || ''
}

watch(selectedRuntime, (runtime) => {
  editorModel.value = { config_set: runtime?.config_set ? JSON.parse(JSON.stringify(runtime.config_set)) : {} }
})

function onSchemaParamOutput(output: Record<string, any>) {
  paramOverrides.value = output
}

function nextStep() {
  if (activeStep.value < 3) activeStep.value++
}

async function loadNodes() {
  nodesLoading.value = true
  nodesError.value = ''
  try {
    nodes.value = await apiClient.get('/nodes')
  } catch (e: any) {
    nodesError.value = e?.message || 'Failed to load nodes'
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
    runtimesError.value = e?.message || 'Failed to load runtimes'
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
    if (paramOverrides.value?.config_set) {
      payload.config_set = paramOverrides.value.config_set
    }
    const enableResp = await apiClient.post(`/nodes/${selectedNode.value.id}/backend-runtimes/enable`, payload)
    const nbrId = enableResp?.id

    if (!andCheck) {
      ElMessage.success('Saved')
      checkResult.value = null
      savingState.value = 'idle'
      return
    }

    if (!nbrId) {
      wizardError.value = 'Enable succeeded but no NBR ID returned'
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
      wizardError.value = e?.message || 'Check failed'
      savingState.value = 'check_failed'
    }
  } catch (e: any) {
    wizardError.value = e?.message || 'Save failed'
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
.wizard-footer { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
</style>
