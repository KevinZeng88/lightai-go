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
      <el-table-column :label="$t('deployments.runtime')" min-width="240">
        <template #default="{ row }">{{ row.backend_runtime?.display_name || row.backend_runtime?.name || row.backend_runtime_id }}</template>
      </el-table-column>
      <el-table-column prop="image_ref" :label="$t('runtimes.image')" min-width="260" show-overflow-tooltip />
      <el-table-column prop="status" :label="$t('common.status')" width="140" />
      <el-table-column :label="$t('common.actions')" width="160">
        <template #default="{ row }">
          <el-button size="small" @click.stop="check(row)">{{ $t('runnerConfigs.check') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="$t('runnerConfigs.create')" width="960px" :close-on-click-modal="false" destroy-on-close>
      <NodeRuntimeConfigWizard ref="nbrWizardRef" @completed="onNBRCreated" />
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.id || ''" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('deployments.node')">{{ selected.node_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('deployments.runtime')">{{ selected.backend_runtime?.display_name || selected.backend_runtime?.name || selected.backend_runtime_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_ref }}</el-descriptions-item>
          <el-descriptions-item :label="$t('common.status')">
            <el-tag :type="selected.deployable ? 'success' : 'warning'">{{ selected.status }}</el-tag>
            <span v-if="selected.disabled_reason" style="margin-left:8px;color:var(--el-color-warning);font-size:12px">{{ selected.disabled_reason }}</span>
          </el-descriptions-item>
        </el-descriptions>
        <el-divider content-position="left">{{ $t('runtimes.structuredParameters') }}</el-divider>
        <RuntimeParameterEditor
          :model-value="nbrEditState"
          :vendor="selected.backend_runtime?.vendor || 'nvidia'"
          :layer="'node_backend_runtime'"
          :base-values="baseParamValues"
          :show-source="true"
          :show-advanced="true"
          @update:model-value="onNBREditUpdate"
        />
        <div style="margin-top:12px;text-align:right">
          <el-button type="primary" :loading="saving" @click="saveNBREdit">{{ $t('common.save') }}</el-button>
        </div>
        <JsonViewer :value="selected.config_set || {}" title="ConfigSet" max-height="520px" :searchable="true" />
        <JsonViewer :value="selected.source_metadata || {}" title="Source Metadata" max-height="260px" :searchable="true" />
        <JsonViewer :value="selected.probe_results_json || {}" title="Probe Results" max-height="260px" :searchable="true" />
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { listRuntimes } from '@/api/runtimes'
import JsonViewer from '@/components/common/JsonViewer.vue'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'
import NodeRuntimeConfigWizard from '@/components/deployments/NodeRuntimeConfigWizard.vue'

const loading = ref(false)
const saving = ref(false)
const createVisible = ref(false)
const configs = ref<any[]>([])
const nodes = ref<any[]>([])
const runtimes = ref<any[]>([])
const selected = ref<any | null>(null)
const nbrEditState = ref<Record<string, any>>({})

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; nbrEditState.value = {} } },
})

const baseParamValues = computed(() => {
  const rtId = selected.value?.backend_runtime_id
  if (!rtId) return []
  const rt = runtimes.value.find((r: any) => r.id === rtId)
  if (rt?.config_set?.items) {
    const items = rt.config_set.items
    const vals: Record<string, any>[] = []
    for (const [k, v] of Object.entries(items)) {
      vals.push({ [k]: (v as any)?.value ?? (v as any)?.default_value })
    }
    return vals
  }
  return []
})

function onNBREditUpdate(val: Record<string, any>) {
  nbrEditState.value = val
}

async function saveNBREdit() {
  if (!selected.value) return
  saving.value = true
  try {
    await apiClient.patch(`/nodes/${selected.value.node_id}/backend-runtimes/${selected.value.id}`, nbrEditState.value)
    ElMessage.success('Saved')
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || 'Save failed')
  } finally {
    saving.value = false
  }
}

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

async function onNBRCreated() {
  createVisible.value = false
  await load()
}

async function check(row: any) {
  await apiClient.post(`/nodes/${row.node_id}/backend-runtimes/${row.id}/check-request`, {})
  ElMessage.success('Check requested')
  await load()
}

onMounted(load)
</script>
