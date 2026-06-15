<template>
  <div class="rt-page">
    <div class="page-header">
      <h2>{{ t('runTemplates.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate">{{ t('runTemplates.create') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" size="small" @row-click="openDetail" highlight-current-row>
      <el-table-column :label="t('runTemplates.name')" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="runtime_type" :label="t('runTemplates.runtimeType')" width="90" />
      <el-table-column prop="vendor" :label="t('runTemplates.vendor')" width="90" />
      <el-table-column prop="backend_type" :label="t('runTemplates.backendType')" width="100" />
      <el-table-column :label="t('runTemplates.argsTemplate')" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">{{ (row.args_template || []).join(' ') }}</template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="210" fixed="right">
        <template #default="{ row }">
          <el-button size="small" text @click.stop="openRenderPreview(row)">{{ t('runTemplates.renderPreview') }}</el-button>
          <el-button size="small" text @click.stop="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button size="small" text type="danger" @click.stop="confirmDelete(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
      <template #empty><el-empty :description="t('runTemplates.noData')" /></template>
    </el-table>

    <!-- Form Dialog -->
    <el-dialog v-model="dialogVisible" :title="editingId ? t('runTemplates.edit') : t('runTemplates.create')" width="560px" @close="resetForm">
      <el-form :model="form" label-width="120px" size="small">
        <el-form-item :label="t('runTemplates.name')" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('runTemplates.displayName')"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item :label="t('runTemplates.runtimeType')">
          <el-select v-model="form.runtime_type" style="width:100%"><el-option label="docker" value="docker" /></el-select>
        </el-form-item>
        <el-form-item :label="t('runTemplates.vendor')">
          <el-select v-model="form.vendor" style="width:100%">
            <el-option label="nvidia" value="nvidia" /><el-option label="metax" value="metax" /><el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('runTemplates.backendType')">
          <el-select v-model="form.backend_type" style="width:100%">
            <el-option label="vllm" value="vllm" /><el-option label="sglang" value="sglang" /><el-option label="llama_cpp" value="llama_cpp" />
            <el-option label="mindie" value="mindie" /><el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('runTemplates.requiredVars')">
          <el-input v-model="form.required_variables_str" placeholder="MODEL_PATH,GPU_IDS (comma separated)" />
        </el-form-item>
        <el-form-item :label="t('runTemplates.argsTemplate')">
          <el-input v-model="form.args_template_str" type="textarea" :rows="4" placeholder="--model ${MODEL_PATH}&#10;--served-model-name ${SERVED_MODEL_NAME}" />
        </el-form-item>
        <el-form-item :label="t('runTemplates.description')">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Render Preview Drawer -->
    <el-drawer v-model="previewVisible" :title="t('runTemplates.renderPreviewTitle')" size="600px">
      <div v-if="previewResult">
        <el-alert v-if="!previewResult.valid" :title="t('modelDeployments.validationFailed')" type="error" :closable="false" style="margin-bottom:12px">
          <ul style="margin:4px 0;padding-left:20px"><li v-for="(e,i) in previewResult.errors" :key="i">{{ e }}</li></ul>
        </el-alert>
        <el-alert v-if="previewResult.warnings?.length" type="warning" :closable="false" style="margin-bottom:12px">
          <ul style="margin:4px 0;padding-left:20px"><li v-for="(w,i) in previewResult.warnings" :key="i">{{ w }}</li></ul>
        </el-alert>
        <div v-if="previewResult.equivalent_command_preview" style="margin-bottom:16px">
          <h4>{{ t('runTemplates.commandPreview') }}</h4>
          <el-input :model-value="previewResult.equivalent_command_preview" type="textarea" :rows="8" readonly style="font-family:monospace;font-size:12px" />
        </div>
      </div>
      <el-empty v-else :description="t('common.loading')" />
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchRunTemplates, createRunTemplate, updateRunTemplate, deleteRunTemplate, renderPreview, type RunTemplate, type RenderPreviewResponse } from '@/api/runTemplates'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t } = useI18n()
const items = ref<RunTemplate[]>([])
const { loading, refresh } = useAutoRefresh(async () => { items.value = await fetchRunTemplates() })
const dialogVisible = ref(false)
const editingId = ref('')
const saving = ref(false)
const previewVisible = ref(false)
const previewResult = ref<RenderPreviewResponse | null>(null)

const defaultForm = () => ({
  name: '', display_name: '', runtime_type: 'docker', vendor: 'nvidia', backend_type: 'vllm',
  required_variables_str: '', args_template_str: '', description: ''
})
const form = ref(defaultForm())

function resetForm() { editingId.value = ''; form.value = defaultForm() }
function openCreate() { resetForm(); dialogVisible.value = true }
function openEdit(row: RunTemplate) {
  editingId.value = row.id
  form.value = {
    name: row.name, display_name: row.display_name, runtime_type: row.runtime_type,
    vendor: row.vendor, backend_type: row.backend_type,
    required_variables_str: (row.required_variables || []).join(','),
    args_template_str: (row.args_template || []).join('\n'),
    description: row.description,
  }
  dialogVisible.value = true
}
function openDetail(row: RunTemplate) { openEdit(row) }

async function openRenderPreview(row: RunTemplate) {
  previewVisible.value = true
  previewResult.value = null
  try {
    previewResult.value = await renderPreview(row.id, {})
    ElMessage.success(t('runTemplates.previewSuccess'))
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
}

async function save() {
  saving.value = true
  try {
    const payload: any = {
      name: form.value.name, display_name: form.value.display_name,
      runtime_type: form.value.runtime_type, vendor: form.value.vendor, backend_type: form.value.backend_type,
      required_variables: form.value.required_variables_str.split(',').map((s: string) => s.trim()).filter(Boolean),
      args_template: form.value.args_template_str.split('\n').filter(Boolean),
      description: form.value.description,
    }
    if (editingId.value) {
      await updateRunTemplate(editingId.value, payload)
      ElMessage.success(t('runTemplates.updateSuccess'))
    } else {
      await createRunTemplate(payload)
      ElMessage.success(t('runTemplates.createSuccess'))
    }
    dialogVisible.value = false
    refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { saving.value = false }
}

async function confirmDelete(row: RunTemplate) {
  try {
    await ElMessageBox.confirm(t('runTemplates.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await deleteRunTemplate(row.id)
    ElMessage.success(t('runTemplates.deleteSuccess'))
    refresh()
  } catch { /* cancelled */ }
}
</script>
