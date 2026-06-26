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
            <ConfigEditView
              :model-value="editView"
              :readonly="!selected.is_editable"
              @update:patch="editPatch = $event"
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
            <JsonViewer :value="selected.config_set || {}" :title="$t('common.technicalConfig')" max-height="520px" :searchable="true" />
            <JsonViewer :value="selected.source_metadata || {}" title="Source Metadata" max-height="260px" :searchable="true" />
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>

    <el-dialog v-model="cloneDialogVisible" title="Clone runtime" width="520px">
      <el-form label-position="top">
        <el-form-item label="Display Name">
          <el-input v-model="cloneForm.display_name" />
        </el-form-item>
        <el-form-item label="Name">
          <el-input v-model="cloneForm.name" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cloneDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitCloneRuntime">{{ $t('runtimes.clone') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { listRuntimes } from '@/api/runtimes'
import { apiClient } from '@/api/client'
import { applyConfigEditPatch, getConfigEditView } from '@/api/configEdit'
import { toRuntimeTemplateDisplay, type RuntimeTemplateDisplay } from '@/utils/runtimeDisplay'
import type { ConfigEditPatch, ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'
import JsonViewer from '@/components/common/JsonViewer.vue'
import ConfigEditView from '@/components/config/ConfigEditView.vue'

const loading = ref(false)
const saving = ref(false)
const runtimes = ref<any[]>([])
const selected = ref<any | null>(null)
const editView = ref<ConfigEditViewModel | null>(null)
const editPatch = ref<ConfigEditPatch | null>(null)
const cloneDialogVisible = ref(false)
const cloneSource = ref<any | null>(null)
const cloneForm = ref<Record<string, any>>({ display_name: '', name: '' })

const displayRuntimes = computed(() => runtimes.value.map(toRuntimeTemplateDisplay))

const selectedDisplay = computed(() => {
  if (!selected.value) return null
  return toRuntimeTemplateDisplay(selected.value)
})

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; editView.value = null; editPatch.value = null } },
})

watch(selected, async (value) => {
  editView.value = null
  editPatch.value = null
  if (!value?.id) return
  editView.value = await getConfigEditView({
    object_kind: 'backend_runtime',
    object_id: value.id,
    layer: 'backend_runtime',
    mode: value.is_editable ? 'edit' : 'view',
  })
})

async function saveEdit() {
  if (!selected.value) return
  saving.value = true
  try {
    if (editPatch.value) {
      await applyConfigEditPatch({
        object_kind: 'backend_runtime',
        object_id: selected.value.id,
        layer: 'backend_runtime',
        patch: editPatch.value,
      })
    }
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
  cloneSource.value = row
  cloneForm.value = {
    display_name: `${row.display_name || row.name || 'Runtime'} Copy`,
    name: '',
  }
  cloneDialogVisible.value = true
}

async function submitCloneRuntime() {
  if (!cloneSource.value) return
  try {
    await apiClient.post(`/backend-runtimes/${cloneSource.value.id}/clone`, {
      display_name: cloneForm.value.display_name,
      name: cloneForm.value.name,
    })
    ElMessage.success('Cloned')
    cloneDialogVisible.value = false
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
