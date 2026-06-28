<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runtimes.title') }}</h2>
      <el-button @click="load">{{ $t('common.refresh') }}</el-button>
    </div>

    <el-table :data="displayRuntimes" v-loading="loading" stripe @row-click="openDetail($event.raw)">
      <el-table-column :label="$t('runtimes.name')" min-width="220">
        <template #default="{ row }">{{ row.displayName }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.vendor')" width="120">
        <template #default="{ row }">{{ row.vendorDisplay }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.backend')" width="120">
        <template #default="{ row }">{{ row.backendDisplay }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.backendVersion')" width="120">
        <template #default="{ row }">{{ row.versionDisplay || '-' }}</template>
      </el-table-column>
      <el-table-column prop="image" :label="$t('runtimes.image')" min-width="240" show-overflow-tooltip />
      <el-table-column :label="$t('runtimes.readyCount')" width="120">
        <template #default="{ row }">{{ row.readyCount }}</template>
      </el-table-column>
      <el-table-column :label="$t('runtimes.managedBy')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.sourceType === 'user' ? 'success' : 'info'">
            {{ row.sourceType === 'user' ? $t('runtimes.userConfig') : $t('runtimes.builtinTemplate') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.actions')" width="280" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click.stop="openDetail(row.raw)">{{ $t('common.view') }}</el-button>
          <el-button v-if="row.sourceType === 'user'" size="small" type="warning" @click.stop="openEdit(row.raw)">{{ $t('common.edit') }}</el-button>
          <el-button v-if="row.sourceType === 'builtin'" size="small" type="primary" @click.stop="cloneRuntime(row.raw)">
            {{ $t('runtimes.clone') }}
          </el-button>
          <template v-if="row.sourceType === 'user'">
            <el-button size="small" @click.stop="renameRuntime(row.raw)">{{ $t('common.rename') || 'Rename' }}</el-button>
            <el-button size="small" type="danger" @click.stop="confirmDeleteRuntime(row.raw)">{{ $t('common.delete') }}</el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="detailVisible" :title="selectedDisplay?.displayName || selected?.display_name || selected?.name || ''" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runtimes.backend')">{{ selectedDisplay?.backendDisplay || selected.backend_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.backendVersion')">{{ selectedDisplay?.versionDisplay || selected.backend_version_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.vendor')">{{ selectedDisplay?.vendorDisplay || selected.vendor }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_ref }}</el-descriptions-item>
        </el-descriptions>

        <!-- Config parameters: readonly detail by default -->
        <el-divider content-position="left">{{ editing ? $t('runtimes.structuredParameters') : $t('runtimes.configParametersReadonly') }}</el-divider>
        <div style="margin-bottom:12px">
          <ConfigEditView
            :model-value="editView"
            :readonly="!editing || !selected.is_editable"
            @update:patch="editPatch = $event"
          />
        </div>
        <template v-if="selected.is_editable">
          <div style="margin-top: 12px; text-align: right">
            <template v-if="!editing">
              <el-button type="primary" @click="startEditing">{{ $t('common.edit') }}</el-button>
            </template>
            <template v-else>
              <el-button @click="cancelEditing">{{ $t('common.cancel') }}</el-button>
              <el-button type="primary" :loading="saving" @click="saveEdit">{{ $t('common.save') }}</el-button>
            </template>
          </div>
        </template>
        <template v-else>
          <el-alert type="info" :closable="false" style="margin:12px 0">
            {{ $t('runtimes.systemTemplateReadonly') }}
          </el-alert>
        </template>

        <!-- Source Summary -->
        <el-divider content-position="left">{{ $t('runtimes.sourceSummary') }}</el-divider>
        <el-descriptions v-if="sourceSummary" :column="2" border size="small">
          <el-descriptions-item v-if="sourceSummary.source_type" :label="$t('configEdit.source.sourceType')">{{ sourceSummary.source_type }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.source_backend" :label="$t('configEdit.source.sourceBackend')">{{ sourceSummary.source_backend }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.source_version" :label="$t('configEdit.source.sourceVersion')">{{ sourceSummary.source_version }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.source_template" :label="$t('configEdit.source.sourceTemplate')">{{ sourceSummary.source_template }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.copy_semantics" :label="$t('configEdit.source.copySemantics')">{{ sourceSummary.copy_semantics }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.source_checksum" :label="$t('configEdit.source.sourceChecksum')" :span="2">{{ sourceSummary.source_checksum }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.loaded_from" :label="$t('configEdit.source.loadedFrom')" :span="2">{{ sourceSummary.loaded_from }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.loaded_at" :label="$t('configEdit.source.loadedAt')">{{ sourceSummary.loaded_at }}</el-descriptions-item>
          <el-descriptions-item v-if="sourceSummary.updated_at" :label="$t('configEdit.source.updatedAt')">{{ sourceSummary.updated_at }}</el-descriptions-item>
        </el-descriptions>
        <el-empty v-else :description="$t('common.noData')" :image-size="40" />

        <!-- Developer Diagnostics (collapsed by default) -->
        <el-collapse style="margin-top:12px">
          <el-collapse-item :title="$t('runtimes.advancedDiagnostics')">
            <JsonViewer :value="selected.config_set || {}" :title="$t('runtimes.rawConfigJson')" max-height="520px" :searchable="true" />
            <JsonViewer :value="selected.source_metadata || {}" :title="$t('runtimes.rawSourceMetadataJson')" max-height="260px" :searchable="true" />
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>

    <!-- Clone Dialog -->
    <el-dialog v-model="cloneDialogVisible" :title="$t('runtimes.cloneRuntimeTitle')" width="520px">
      <el-form label-position="top">
        <el-form-item :label="$t('runtimes.displayName')">
          <el-input v-model="cloneForm.display_name" />
        </el-form-item>
        <el-form-item :label="$t('runtimes.technicalName')">
          <el-input v-model="cloneForm.name" :placeholder="$t('runtimes.technicalNamePlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cloneDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="cloning" @click="submitCloneRuntime">{{ $t('runtimes.clone') }}</el-button>
      </template>
    </el-dialog>

    <!-- Rename Dialog -->
    <el-dialog v-model="renameDialogVisible" :title="$t('runtimes.renameTitle')" width="520px">
      <el-form label-position="top">
        <el-form-item :label="$t('runtimes.displayName')">
          <el-input v-model="renameForm.display_name" />
        </el-form-item>
        <el-form-item :label="$t('runtimes.technicalName')">
          <el-input v-model="renameForm.name" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renameDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="renaming" @click="submitRenameRuntime">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Delete Confirm Dialog -->
    <el-dialog v-model="deleteDialogVisible" :title="$t('common.delete')" width="480px">
      <p>{{ deleteMessage }}</p>
      <template #footer>
        <el-button @click="deleteDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="danger" :loading="deleting" @click="submitDeleteRuntime">{{ $t('common.delete') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { listRuntimes } from '@/api/runtimes'
import { apiClient } from '@/api/client'
import { applyConfigEditPatch, getConfigEditView } from '@/api/configEdit'
import { toRuntimeTemplateDisplay, type RuntimeTemplateDisplay } from '@/utils/runtimeDisplay'
import type { ConfigEditPatch, ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'
import JsonViewer from '@/components/common/JsonViewer.vue'
import ConfigEditView from '@/components/config/ConfigEditView.vue'

const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const cloning = ref(false)
const renaming = ref(false)
const deleting = ref(false)
const runtimes = ref<any[]>([])
const selected = ref<any | null>(null)
const editView = ref<ConfigEditViewModel | null>(null)
const editPatch = ref<ConfigEditPatch | null>(null)
const cloneDialogVisible = ref(false)
	const editing = ref(false)
const cloneSource = ref<any | null>(null)
const cloneForm = ref<Record<string, any>>({ display_name: '', name: '' })
const renameDialogVisible = ref(false)
const renameTarget = ref<any | null>(null)
const renameForm = ref<Record<string, any>>({ display_name: '', name: '' })
const deleteDialogVisible = ref(false)
const deleteTarget = ref<any | null>(null)

const displayRuntimes = computed(() => runtimes.value.map(toRuntimeTemplateDisplay))

const selectedDisplay = computed(() => {
  if (!selected.value) return null
  return toRuntimeTemplateDisplay(selected.value)
})

const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; editView.value = null; editPatch.value = null; editing.value = false } },
})

const deleteMessage = computed(() => {
  if (!deleteTarget.value) return ''
  const name = deleteTarget.value.display_name || deleteTarget.value.name || deleteTarget.value.id
  return t('runtimes.deleteConfirmRuntime', { name })
})

// Parse source_metadata into a display-friendly summary.
const sourceSummary = computed(() => {
  if (!selected.value) return null
  const sm = selected.value.source_metadata
  if (!sm || typeof sm !== 'object') return null
  const out: Record<string, any> = {}
  if (sm.source_type) out.source_type = sm.source_type
  if (sm.source_backend || sm.backend_id) out.source_backend = sm.source_backend || sm.backend_id
  if (sm.source_version || sm.backend_version_id) out.source_version = sm.source_version || sm.backend_version_id
  if (sm.source_template || sm.template_name) out.source_template = sm.source_template || sm.template_name
  if (sm.copy_semantics || sm.semantics) out.copy_semantics = sm.copy_semantics || sm.semantics
  if (sm.checksum || sm.source_checksum) out.source_checksum = sm.checksum || sm.source_checksum
  if (sm.loaded_from) out.loaded_from = sm.loaded_from
  if (sm.loaded_at) out.loaded_at = sm.loaded_at
  if (sm.updated_at) out.updated_at = sm.updated_at
  return Object.keys(out).length > 0 ? out : null
})

function openDetail(row: any) {
  selected.value = row
}

function openEdit(row: any) {
  selected.value = row
  // After view loads, enter edit mode
  editing.value = true
}

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

function startEditing() {
  editing.value = true
}

function cancelEditing() {
  editing.value = false
  // Reload view to discard changes
  if (selected.value?.id) {
    getConfigEditView({ object_kind: 'backend_runtime', object_id: selected.value.id, layer: 'backend_runtime', mode: 'edit' }).then(v => { editView.value = v; editPatch.value = null })
  }
}

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
    ElMessage.success(t('common.saved'))
    editing.value = false
    await load()
    const updated = runtimes.value.find(r => r.id === selected.value?.id)
    if (updated) selected.value = updated
  } catch (e: any) {
    ElMessage.error(e?.message || 'Save failed')
  } finally {
    saving.value = false
  }
}

function cloneRuntime(row: any) {
  cloneSource.value = row
  const display = toRuntimeTemplateDisplay(row)
  cloneForm.value = {
    display_name: `${display.displayName}${t('runtimes.customSuffix')}`,
    name: '',
  }
  cloneDialogVisible.value = true
}

async function submitCloneRuntime() {
  if (!cloneSource.value) return
  cloning.value = true
  try {
    const res = await apiClient.post(`/backend-runtimes/${cloneSource.value.id}/clone`, {
      display_name: cloneForm.value.display_name,
      name: cloneForm.value.name,
    })
    ElMessage.success(t('runtimes.cloned'))
    cloneDialogVisible.value = false
    await load()
    // Auto-select the newly created runtime if we have an id.
    const newId = res?.data?.id || res?.id
    if (newId) {
      const found = runtimes.value.find(r => r.id === newId)
      if (found) selected.value = found
    }
  } catch (e: any) {
    ElMessage.error(e?.message || 'Clone failed')
  } finally {
    cloning.value = false
  }
}

function renameRuntime(row: any) {
  renameTarget.value = row
  renameForm.value = {
    display_name: row.display_name || '',
    name: row.name || '',
  }
  renameDialogVisible.value = true
}

async function submitRenameRuntime() {
  if (!renameTarget.value) return
  renaming.value = true
  try {
    await apiClient.patch(`/backend-runtimes/${renameTarget.value.id}`, {
      display_name: renameForm.value.display_name,
      name: renameForm.value.name,
    })
    ElMessage.success(t('common.saved'))
    renameDialogVisible.value = false
    await load()
    const updated = runtimes.value.find(r => r.id === renameTarget.value?.id)
    if (updated) selected.value = updated
  } catch (e: any) {
    ElMessage.error(e?.message || 'Rename failed')
  } finally {
    renaming.value = false
  }
}

function confirmDeleteRuntime(row: any) {
  deleteTarget.value = row
  deleteDialogVisible.value = true
}

async function submitDeleteRuntime() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await apiClient.delete(`/backend-runtimes/${deleteTarget.value.id}`)
    ElMessage.success(t('runtimes.deleted'))
    deleteDialogVisible.value = false
    if (selected.value?.id === deleteTarget.value.id) {
      selected.value = null
    }
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || 'Delete failed')
  } finally {
    deleting.value = false
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
