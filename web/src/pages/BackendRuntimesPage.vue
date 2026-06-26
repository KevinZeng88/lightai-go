<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runtimes.title') }}</h2>
      <el-button @click="load">{{ $t('common.refresh') }}</el-button>
    </div>

    <el-table :data="displayRuntimes" v-loading="loading" stripe @row-click="selected = $event.raw">
      <el-table-column :label="$t('runtimes.name')" min-width="220">
        <template #default="{ row }">{{ row.displayName }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.vendor')" width="120">
        <template #default="{ row }">{{ row.vendor }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.backend')" width="120">
        <template #default="{ row }">{{ row.backend }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.backendVersion')" width="120">
        <template #default="{ row }">{{ row.version || '-' }}</template>
      </el-table-column>
      <el-table-column prop="image" :label="$t('runtimes.image')" min-width="240" show-overflow-tooltip />
      <el-table-column :label="$t('runtimes.readyCount')" width="120">
        <template #default="{ row }">{{ row.readyCount }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.managedBy')" width="140">
        <template #default="{ row }">
          <el-tag :type="row.managedBy === 'user' ? 'success' : 'info'">
            {{ row.managedBy === 'user' ? $t('runtimes.userManaged') : $t('runtimes.systemManaged') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.actions')" width="100" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.managedBy === 'system'" size="small" @click.stop="cloneRuntime(row.raw)">
            {{ $t('runtimes.clone') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="detailVisible" :title="selectedDisplay?.displayName || selected?.display_name || selected?.name || ''" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runtimes.backend')">{{ selectedDisplay?.backend || selected.backend_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.backendVersion')">{{ selectedDisplay?.version || selected.backend_version_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.vendor')">{{ selected.vendor }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_ref }}</el-descriptions-item>
        </el-descriptions>
        <template v-if="selected.is_editable">
          <el-divider content-position="left">{{ $t('runtimes.structuredParameters') }}</el-divider>
          <div style="margin-bottom:12px">
            <HumanRuntimeParameterForm
              :config-set="selected.config_set || null"
              :backend-name="selected.backend_id"
              :vendor="selected.vendor"
              @update:output="onHumanParamOutput"
            />
          </div>
          <div style="margin-top: 12px; text-align: right">
            <el-button type="primary" :loading="saving" @click="saveEdit">
              {{ $t('common.save') }}
            </el-button>
          </div>
        </template>
        <template v-else>
          <el-alert type="info" :closable="false" style="margin:12px 0">
            {{ $t('runtimes.systemTemplateReadonly') || 'System template — clone to create an editable copy.' }}
          </el-alert>
        </template>
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="$t('runtimes.advancedDiagnostics') || 'Advanced Diagnostics'">
            <RuntimeParameterEditor
              :model-value="{ config_set: selected.config_set || {} }"
              :readonly="true"
              :vendor="selected.vendor"
              :layer="'backend_runtime'"
              :show-advanced="true"
            />
            <JsonViewer :value="selected.config_set || {}" :title="$t('common.technicalConfig')" max-height="520px" :searchable="true" />
            <JsonViewer :value="selected.source_metadata || {}" title="Source Metadata" max-height="260px" :searchable="true" />
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listRuntimes } from '@/api/runtimes'
import { apiClient } from '@/api/client'
import { toRuntimeTemplateDisplay, type RuntimeTemplateDisplay } from '@/utils/runtimeDisplay'
import JsonViewer from '@/components/common/JsonViewer.vue'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'
import HumanRuntimeParameterForm from '@/components/runtime/HumanRuntimeParameterForm.vue'
import type { RuntimeParamFormOutput } from '@/utils/runtimeParameterViewModel'

const loading = ref(false)
const saving = ref(false)
const runtimes = ref<any[]>([])
const selected = ref<any | null>(null)
const humanParamOutput = ref<RuntimeParamFormOutput>({})

const displayRuntimes = computed(() => runtimes.value.map(toRuntimeTemplateDisplay))

const selectedDisplay = computed(() => {
  if (!selected.value) return null
  return toRuntimeTemplateDisplay(selected.value)
})

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; humanParamOutput.value = {} } },
})

function onHumanParamOutput(output: RuntimeParamFormOutput) {
  humanParamOutput.value = output
}

async function saveEdit() {
  if (!selected.value) return
  saving.value = true
  try {
    const patchPayload: Record<string, any> = {}
    // Build config_set patch from human parameter output
    if (humanParamOutput.value?.parameter_values?.length || humanParamOutput.value?.docker_options) {
      const cs = selected.value.config_set ? JSON.parse(JSON.stringify(selected.value.config_set)) : { items: {} }
      cs.items = cs.items || {}
      for (const pv of (humanParamOutput.value.parameter_values || [])) {
        cs.items[pv.key] = { ...(cs.items[pv.key] || {}), value: pv.value, enabled: pv.enabled }
      }
      if (humanParamOutput.value.docker_options && Object.keys(humanParamOutput.value.docker_options).length) {
        const existingDocker = cs.items['launcher.docker_options']?.value || {}
        const merged = { ...(typeof existingDocker === 'object' ? existingDocker : {}), ...humanParamOutput.value.docker_options }
        cs.items['launcher.docker_options'] = { ...(cs.items['launcher.docker_options'] || {}), category: 'launcher', kind: 'docker_options', type: 'object', value: merged, enabled: true }
      }
      patchPayload.config_set = cs
    }
    await apiClient.patch(`/backend-runtimes/${selected.value.id}`, patchPayload)
    ElMessage.success('Saved')
    await load()
    const updated = runtimes.value.find(r => r.id === selected.value?.id)
    if (updated) selected.value = updated
  } catch (e: any) {
    ElMessage.error(e?.message || 'Save failed')
  } finally {
    saving.value = false
  }
}

async function cloneRuntime(row: any) {
  try {
    await apiClient.post(`/backend-runtimes/${row.id}/clone`)
    ElMessage.success('Cloned')
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || 'Clone failed')
  }
}

async function load() {
  loading.value = true
  try {
    runtimes.value = await listRuntimes()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
