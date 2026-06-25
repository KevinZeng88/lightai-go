<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runnerConfigs.title') }}</h2>
      <div>
        <el-button type="primary" @click="createVisible = true">{{ $t('common.create') }}</el-button>
        <el-button @click="load">{{ $t('common.refresh') }}</el-button>
      </div>
    </div>

    <el-table :data="configs" v-loading="loading" stripe @row-click="selected = $event">
      <el-table-column prop="display_name" :label="$t('runnerConfigs.name')" min-width="220" />
      <el-table-column prop="node_id" :label="$t('deployments.node')" min-width="180" />
      <el-table-column prop="backend_runtime_id" :label="$t('deployments.runtime')" min-width="240" />
      <el-table-column prop="image_ref" :label="$t('runtimes.image')" min-width="260" show-overflow-tooltip />
      <el-table-column prop="status" :label="$t('common.status')" width="140" />
      <el-table-column :label="$t('common.actions')" width="160">
        <template #default="{ row }">
          <el-button size="small" @click.stop="check(row)">{{ $t('runnerConfigs.check') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('runnerConfigs.create')" width="640px">
      <el-form label-position="top">
        <el-form-item :label="$t('deployments.node')">
          <el-select v-model="form.node_id" style="width:100%" filterable>
            <el-option v-for="node in nodes" :key="node.id" :label="node.name || node.id" :value="node.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('deployments.runtime')">
          <el-select v-model="form.runtime_id" style="width:100%" filterable>
            <el-option v-for="runtime in runtimes" :key="runtime.id" :label="runtime.display_name || runtime.name" :value="runtime.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('runtimes.displayName')"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item :label="$t('runtimes.image')"><el-input v-model="form.image_ref" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="enable">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.id || ''" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('deployments.node')">{{ selected.node_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.runtime')">{{ selected.backend_runtime_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_ref }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.status')">{{ selected.status }}</el-descriptions-item>
        </el-descriptions>
        <JsonViewer :value="selected.config_set || {}" title="ConfigSet" max-height="520px" :searchable="true" />
        <JsonViewer :value="selected.source_metadata || {}" title="Source Metadata" max-height="260px" :searchable="true" />
        <JsonViewer :value="selected.probe_results_json || {}" title="Probe Results" max-height="260px" :searchable="true" />
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { listRuntimes } from '@/api/runtimes'
import JsonViewer from '@/components/common/JsonViewer.vue'

const loading = ref(false)
const saving = ref(false)
const createVisible = ref(false)
const configs = ref<any[]>([])
const nodes = ref<any[]>([])
const runtimes = ref<any[]>([])
const selected = ref<any | null>(null)
const form = reactive({ node_id: '', runtime_id: '', display_name: '', image_ref: '' })

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) selected.value = null },
})

async function load() {
  loading.value = true
  try {
    const [nodeList, runtimeList, configList] = await Promise.all([
      apiClient.get('/nodes'),
      listRuntimes(),
      apiClient.get('/nodes/backend-runtimes/all'),
    ])
    nodes.value = Array.isArray(nodeList) ? nodeList : []
    runtimes.value = Array.isArray(runtimeList) ? runtimeList : []
    configs.value = Array.isArray(configList) ? configList : []
  } finally {
    loading.value = false
  }
}

async function enable() {
  if (!form.node_id || !form.runtime_id) return
  saving.value = true
  try {
    await apiClient.post(`/nodes/${form.node_id}/backend-runtimes/enable`, {
      backend_runtime_id: form.runtime_id,
      display_name: form.display_name,
      image_ref: form.image_ref,
    })
    createVisible.value = false
    ElMessage.success('Saved')
    await load()
  } finally {
    saving.value = false
  }
}

async function check(row: any) {
  await apiClient.post(`/nodes/${row.node_id}/backend-runtimes/${row.id}/check-request`, {})
  ElMessage.success('Check requested')
  await load()
}

onMounted(load)
</script>
