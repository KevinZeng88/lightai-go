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
      <el-table-column prop="source_node_backend_runtime_id" :label="$t('deployments.runtime')" min-width="260" />
      <el-table-column prop="status" :label="$t('common.status')" width="140" />
      <el-table-column :label="$t('common.actions')" width="260">
        <template #default="{ row }">
          <el-button size="small" @click.stop="dryRun(row)">{{ $t('deployments.dryRun') }}</el-button>
          <el-button size="small" type="primary" @click.stop="start(row)">{{ $t('deployments.start') }}</el-button>
          <el-button size="small" @click.stop="stop(row)">{{ $t('deployments.stop') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('deployments.title')" width="680px">
      <el-form label-position="top">
        <el-form-item :label="$t('deployments.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="$t('deployments.artifact')">
          <el-select v-model="form.model_artifact_id" style="width:100%" filterable>
            <el-option v-for="artifact in artifacts" :key="artifact.id" :label="artifact.display_name || artifact.name" :value="artifact.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.runtime')">
          <el-select v-model="form.node_backend_runtime_id" style="width:100%" filterable>
            <el-option v-for="runtime in deployableRuntimes" :key="runtime.id" :label="runtime.display_name || runtime.id" :value="runtime.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.hostPort')"><el-input-number v-model="form.host_port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="$t('deployments.servedModelName')"><el-input v-model="form.served_model_name" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="create">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.name || ''" size="70%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('deployments.artifact')">{{ selected.model_artifact_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.runtime')">{{ selected.source_node_backend_runtime_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.status')">{{ selected.status }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.created')">{{ selected.created_at }}</el-descriptions-item>
        </el-descriptions>
        <JsonViewer :value="selected.config_set || {}" title="Deployment ConfigSet" max-height="520px" :searchable="true" />
        <JsonViewer :value="selected.source_metadata || {}" title="Source Metadata" max-height="260px" :searchable="true" />
        <JsonViewer v-if="lastDryRun" :value="lastDryRun" title="Last Dry Run" max-height="420px" :searchable="true" />
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { createDeployment, dryRunDeployment, startDeployment, stopDeployment } from '@/api/deployments'
import JsonViewer from '@/components/common/JsonViewer.vue'

const loading = ref(false)
const saving = ref(false)
const createVisible = ref(false)
const deployments = ref<any[]>([])
const artifacts = ref<any[]>([])
const nodeRuntimes = ref<any[]>([])
const selected = ref<any | null>(null)
const lastDryRun = ref<any | null>(null)
const form = reactive({
  name: '',
  model_artifact_id: '',
  node_backend_runtime_id: '',
  host_port: 8000,
  served_model_name: '',
})

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; lastDryRun.value = null } },
})

const deployableRuntimes = computed(() => nodeRuntimes.value.filter((runtime) => runtime.deployable))

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
  } finally {
    loading.value = false
  }
}

async function create() {
  saving.value = true
  try {
    const overrides = form.served_model_name
      ? { parameter_values: [{ key: 'backend.common.served_model_name', value: form.served_model_name, enabled: true }] }
      : { parameter_values: [] }
    await createDeployment({
      name: form.name,
      model_artifact_id: form.model_artifact_id,
      node_backend_runtime_id: form.node_backend_runtime_id,
      service_json: { host_port: form.host_port },
      config_overrides: overrides,
    })
    createVisible.value = false
    ElMessage.success('Saved')
    await load()
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
