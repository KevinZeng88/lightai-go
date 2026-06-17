<template>
  <div class="page-container">
    <h2>{{ $t('deployments.title') }}</h2>
    <el-button type="primary" @click="showCreate">{{ $t('common.create') }}</el-button>
    <el-table :data="items" v-loading="loading" stripe style="margin-top:12px">
      <el-table-column prop="name" :label="$t('deployments.name')" width="150" />
      <el-table-column prop="status" :label="$t('deployments.status')" width="100" />
      <el-table-column prop="model_artifact_id" :label="$t('deployments.artifact')" width="200" />
      <el-table-column prop="backend_runtime_id" :label="$t('deployments.runtime')" width="200" />
      <el-table-column :label="$t('common.actions')" width="320">
        <template #default="{ row }">
          <el-button size="small" @click="doDryRun(row)">{{ $t('deployments.dryRun') }}</el-button>
          <el-button size="small" type="success" @click="doStart(row)">{{ $t('deployments.start') }}</el-button>
          <el-button size="small" type="warning" @click="doStop(row)">{{ $t('deployments.stop') }}</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('common.create')" width="500px">
      <el-form :model="createForm" label-width="140px">
        <el-form-item :label="$t('deployments.name')"><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item :label="$t('deployments.artifact')"><el-input v-model="createForm.model_artifact_id" /></el-form-item>
        <el-form-item :label="$t('deployments.runtime')"><el-input v-model="createForm.backend_runtime_id" /></el-form-item>
        <el-form-item :label="$t('deployments.nodeId')"><el-input v-model="createForm.node_id" /></el-form-item>
        <el-form-item :label="$t('deployments.gpuIds')"><el-input v-model="createForm.gpu_ids" placeholder='["gpu-id"]' /></el-form-item>
        <el-form-item :label="$t('deployments.hostPort')"><el-input v-model.number="createForm.host_port" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doCreate" :loading="saving">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="dryRunVisible" title="Dry Run Result" width="700px">
      <pre v-if="dryRunResult" style="white-space:pre-wrap;max-height:400px;overflow:auto">{{ JSON.stringify(dryRunResult, null, 2) }}</pre>
    </el-dialog>

    <el-dialog v-model="runPlanVisible" title="RunPlan / Docker Preview" width="700px">
      <pre v-if="runPlanData" style="white-space:pre-wrap;max-height:400px;overflow:auto">{{ runPlanData }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'

const loading = ref(false); const saving = ref(false)
const items = ref<any[]>([]); const createVisible = ref(false); const dryRunVisible = ref(false); const runPlanVisible = ref(false)
const dryRunResult = ref<any>(null); const runPlanData = ref('')
const createForm = ref({ name: '', model_artifact_id: '', backend_runtime_id: '', node_id: '', gpu_ids: '[]', host_port: 8000, placement_json: '{}', service_json: '{}', parameters_json: '{}', env_overrides_json: '{}' })

onMounted(async () => { await refresh() })
async function refresh() { loading.value = true; try { items.value = await apiClient.get('/api/v1/model-deployments') } catch (e: any) {} loading.value = false }

function showCreate() { createVisible.value = true }
async function doCreate() {
  saving.value = true
  try {
    createForm.value.placement_json = JSON.stringify({ node_id: createForm.value.node_id, gpu_ids: JSON.parse(createForm.value.gpu_ids || '[]') })
    createForm.value.service_json = JSON.stringify({ host_port: createForm.value.host_port })
    await apiClient.post('/api/v1/model-deployments', createForm.value)
    ElMessage.success('Created'); createVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  saving.value = false
}

async function doDryRun(row: any) {
  try { dryRunResult.value = await apiClient.post(`/api/v1/model-deployments/${row.id}/dry-run`, {}) } catch (e: any) {}
  dryRunVisible.value = true
}

async function doStart(row: any) {
  try {
    const res = await apiClient.post(`/api/v1/model-deployments/${row.id}/start`, {})
    runPlanData.value = res.docker_preview || JSON.stringify(res, null, 2)
    runPlanVisible.value = true
    ElMessage.success('Started')
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
}

async function doStop(row: any) {
  try { await apiClient.post(`/api/v1/model-deployments/${row.id}/stop`, {}); ElMessage.success('Stopped'); await refresh() } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
}

async function handleDelete(row: any) {
  try { await ElMessageBox.confirm(`Delete ${row.name}?`, 'Confirm', { type: 'warning' }); await apiClient.delete(`/api/v1/model-deployments/${row.id}`); ElMessage.success('Deleted'); await refresh() } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || 'Failed') }
}

// JSON used directly
</script>
