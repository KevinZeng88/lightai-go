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
      <el-table-column prop="display_name" :label="$t('deployments.name')" min-width="220" />
      <el-table-column prop="model_artifact_id" :label="$t('deployments.artifact')" min-width="220" />
      <el-table-column :label="$t('deployments.runtime')" min-width="260">
        <template #default="{ row }">{{ row.source_node_backend_runtime_display_name || row.source_node_backend_runtime_id }}</template>
      </el-table-column>
      <el-table-column prop="status" :label="$t('common.status')" width="140" />
      <el-table-column :label="$t('common.actions')" width="260">
        <template #default="{ row }">
          <el-button size="small" @click.stop="dryRun(row)">{{ $t('deployments.dryRun') }}</el-button>
          <el-button size="small" type="primary" @click.stop="start(row)">{{ $t('deployments.start') }}</el-button>
          <el-button size="small" @click.stop="stop(row)">{{ $t('deployments.stop') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('deployments.createDeployment') || $t('deployments.title')" width="960px" :close-on-click-modal="false">
      <DeploymentWizard
        ref="wizardRef"
        :artifacts="artifacts"
        :node-runtimes="nodeRuntimes"
        :model-locations="modelLocations"
        :saving="saving"
        @save="createFromWizard"
        @refresh-data="load"
      />
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.name || ''" size="70%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('deployments.artifact')">{{ selected.model_artifact_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.runtime')">{{ selected.source_node_backend_runtime_display_name || selected.source_node_backend_runtime_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.status')">{{ selected.status }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.created')">{{ selected.created_at }}</el-descriptions-item>
        </el-descriptions>
        <JsonViewer :value="selected.config_set || {}" :title="$t('runtimes.rawConfigJson')" max-height="520px" :searchable="true" />
        <JsonViewer :value="selected.source_metadata || {}" :title="$t('runtimes.rawSourceMetadataJson')" max-height="260px" :searchable="true" />
        <JsonViewer v-if="lastDryRun" :value="lastDryRun" title="Last Dry Run" max-height="420px" :searchable="true" />
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { createDeployment, dryRunDeployment, startDeployment, stopDeployment } from '@/api/deployments'
import JsonViewer from '@/components/common/JsonViewer.vue'
import DeploymentWizard from '@/components/deployments/DeploymentWizard.vue'

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

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; lastDryRun.value = null } },
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
    if (!payload) { ElMessage.error('Cannot create deployment: check the compatibility errors above'); return }
    await createDeployment(payload)
    createVisible.value = false
    ElMessage.success('Saved')
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || 'Create failed')
  } finally {
    saving.value = false
  }
}

async function dryRun(row: any) {
  selected.value = row
  lastDryRun.value = await dryRunDeployment(row.id)
}

async function start(row: any) {
  await startDeployment(row.id)
  ElMessage.success('Started')
  await load()
}

async function stop(row: any) {
  await stopDeployment(row.id)
  ElMessage.success('Stopped')
  await load()
}

onMounted(load)
</script>
