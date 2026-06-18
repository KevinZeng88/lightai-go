<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runnerConfigs.title') }}</h2>
      <el-button type="primary" @click="startWizard">{{ $t('runnerConfigs.newConfig') }}</el-button>
    </div>

    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column prop="name" :label="$t('runnerConfigs.name')" min-width="160" />
      <el-table-column :label="$t('modelLocations.node')" width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.node_label || row.node_id }}</template>
      </el-table-column>
      <el-table-column :label="$t('runnerConfigs.runnerType')" width="100">
        <template #default="{ row }">{{ row.runner_type === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : (row.runner_type || '-') }}</template>
      </el-table-column>
      <el-table-column :label="$t('nodeRuntime.status')" width="100">
        <template #default="{ row }"><el-tag :type="row.status==='ready'?'success':'warning'" size="small">{{ translateStatus(row.status, t) }}</el-tag></template>
      </el-table-column>
      <el-table-column :label="$t('common.actions')" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" type="danger" @click="doDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Wizard dialog -->
    <el-dialog v-model="wizardVisible" :title="$t('runnerConfigs.wizardTitle')" width="800px" :close-on-click-modal="false">
      <el-steps :active="step" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('runnerConfigs.selectRunnerType')" />
        <el-step :title="$t('runnerConfigs.selectTemplate')" />
        <el-step :title="$t('runnerConfigs.selectNode')" />
        <el-step :title="$t('runnerConfigs.selectImage')" />
        <el-step :title="$t('runnerConfigs.create')" />
      </el-steps>

      <div v-if="step===0">
        <el-select v-model="wizRunnerType" :placeholder="$t('runnerConfigs.selectRunnerType')" style="width:100%" @change="onWizAutoNext">
          <el-option label="Docker" value="docker" />
        </el-select>
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizRunnerType" @click="step=1">{{ $t('common.next') }}</el-button></div>
      </div>

      <div v-if="step===1">
        <el-select v-model="wizTemplateId" :placeholder="$t('runnerConfigs.selectTemplate')" style="width:100%" filterable @change="onWizTemplateSelected">
          <el-option v-for="t in templates" :key="t.id" :label="`${t.name} (${t.vendor})`" :value="t.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizTemplateId" @click="step=2">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===2">
        <el-select v-model="wizNodeId" :placeholder="$t('runnerConfigs.selectNode')" style="width:100%" filterable @change="onWizAutoNext">
          <el-option v-for="n in nodeItems" :key="n.id" :label="n.label" :value="n.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizNodeId" @click="step=wizRunnerType==='docker'?3:4">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===3 && wizRunnerType==='docker'">
        <DockerImagePicker v-if="wizNodeId" :node-id="wizNodeId" @select="(img:any) => wizImageRef = img.image_ref || img.image_ref" />
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=2">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizImageRef" @click="step=4">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===4">
        <el-form label-width="120px">
          <el-form-item :label="$t('runnerConfigs.template')"><span>{{ wizTemplateId }}</span></el-form-item>
          <el-form-item :label="$t('runnerConfigs.runnerType')"><span>{{ wizRunnerType === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : wizRunnerType }}</span></el-form-item>
          <el-form-item :label="$t('modelLocations.node')"><span>{{ wizNodeId }}</span></el-form-item>
          <el-form-item v-if="wizImageRef" :label="$t('runnerConfigs.selectImage')"><span>{{ wizImageRef }}</span></el-form-item>
        </el-form>
        <div v-if="wizCheckResult" style="margin-top:8px">
          <el-alert :type="wizCheckResult.status==='ready'?'success':'warning'" :title="translateStatus(wizCheckResult.status, t)" :description="translateStatusReason(wizCheckResult.status_reason, t)" show-icon :closable="false" />
        </div>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=wizRunnerType==='docker'?3:2">{{ $t('common.prev') }}</el-button>
          <el-button @click="doCheck" :loading="checking">{{ $t('runnerConfigs.check') }}</el-button>
          <el-button type="primary" :disabled="!wizCheckResult || wizCheckResult.status === 'unknown'" @click="doCreateConfig" :loading="saving">{{ $t('runnerConfigs.create') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Detail drawer -->
    <el-drawer v-model="detailVisible" :title="$t('common.detail')" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runnerConfigs.name')">{{ selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelLocations.node')">{{ selected.node_label || selected.node_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.runnerType')">{{ selected.runner_type === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : (selected.runner_type || '-') }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.status')">
            <el-tag :type="selected.status==='ready'?'success':'warning'" size="small">{{ translateStatus(selected.status, t) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.imageRef')">{{ selected.image_ref || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.template')">{{ selected.template_name || '-' }}</el-descriptions-item>
        </el-descriptions>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import { listRuntimes } from '@/api/runtimes'
import DockerImagePicker from '@/components/DockerImagePicker.vue'
import { translateStatus, translateStatusReason } from '@/utils/status'
import { useWizardAutoAdvance } from '@/composables/useWizardAutoAdvance'
const { loadNodes, nodes: nodeItems, nodeLabel } = useNodeLabels()
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

const loading = ref(false); const saving = ref(false); const checking = ref(false)
const items = ref<any[]>([]); const templates = ref<any[]>([]); const selected = ref<any>(null); const detailVisible = ref(false)
const detailNodeRuntimes = ref<any[]>([])

// Wizard
const wizardVisible = ref(false); const step = ref(0)
const wizTemplateId = ref(''); const wizRunnerType = ref('docker')
const wizNodeId = ref(''); const wizImageRef = ref('')
const wizConfigName = ref(''); const wizCheckResult = ref<any>(null)

const { onSelectAutoNext: onWizAutoNext } = useWizardAutoAdvance(step, () => { step.value++ })

onMounted(async () => { await loadRefs(); await refresh() })

async function refresh() {
  loading.value = true
  try {
    // Collect NodeBackendRuntime records from all nodes
    const nbrList: any[] = []
    for (const n of nodeItems.value) {
      try {
        const nbrs = await apiClient.get(`/nodes/${n.id}/backend-runtimes`)
        if (Array.isArray(nbrs)) {
          for (const nbr of nbrs) {
            nbrList.push({ ...nbr, _node_label: n.label, _node_id: n.id })
          }
        }
      } catch {}
    }
    items.value = nbrList.map((nbr: any) => ({
      id: nbr.id,
      name: nbr.backend_runtime?.name || nbr.backend_runtime_id,
      template_name: nbr.backend_runtime?.name || '-',
      runner_type: nbr.runner_type || 'docker',
      node_count: 1,
      ready_count: nbr.status === 'ready' ? 1 : 0,
      status: nbr.status,
      node_id: nbr._node_id,
      node_label: nbr._node_label,
      image_ref: nbr.image_ref,
      backend_runtime_id: nbr.backend_runtime_id,
    }))
  } catch {}
  loading.value = false
}

async function loadRefs() {
  try { templates.value = await listRuntimes() } catch { templates.value = [] }
  loadNodes()
}

function startWizard() { wizardVisible.value = true; step.value = 0; wizTemplateId.value = ''; wizRunnerType.value = 'docker'; wizNodeId.value = ''; wizImageRef.value = ''; wizConfigName.value = ''; wizCheckResult.value = null; loadRefs() }

function onWizTemplateSelected(templateId: string) {
  const template = templates.value.find((t: any) => t.id === templateId)
  if (!template) return
  const suffix = t('runnerConfigs.customSuffix')
  const baseName = `${template.name}${suffix}`
  // Auto-append number if name conflicts with existing configs
  const existingNames = new Set(items.value.map((c: any) => c.name))
  let candidate = baseName
  let counter = 2
  while (existingNames.has(candidate)) {
    candidate = `${baseName} ${counter}`
    counter++
  }
  wizConfigName.value = candidate
  // Auto-advance: this step has only one select control
  step.value = 2
}

async function doCheck() {
  checking.value = true
  try {
    wizCheckResult.value = await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/check`, { backend_runtime_id: wizTemplateId.value, image_ref: wizImageRef.value || '', image_present: !!wizImageRef.value, docker_available: wizRunnerType.value === 'docker' })
  } catch (e: any) { wizCheckResult.value = { status: 'unknown', status_reason: e?.message || 'check failed' } }
  checking.value = false
}

async function doCreateConfig() {
  saving.value = true
  try {
    // Enable the selected template on the selected node (creates NodeBackendRuntime only, no BackendRuntime clone)
    await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/enable`, { backend_runtime_id: wizTemplateId.value, image_ref: wizImageRef.value, image_present: !!wizImageRef.value, docker_available: wizRunnerType.value === 'docker' })
    ElMessage.success(t('runnerConfigs.created')); wizardVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function showDetail(row: any) {
  selected.value = row
  detailVisible.value = true
}

async function doDelete(row: any) {
  try {
    await ElMessageBox.confirm(t('runnerConfigs.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' })
    // Delete the NodeBackendRuntime record (node-level config only; template is preserved)
    await apiClient.delete(`/nodes/${row.node_id}/backend-runtimes/${row.id}`)
    ElMessage.success(t('runnerConfigs.deleted')); await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
</style>
