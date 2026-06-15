<template>
  <div class="ma-page">
    <div class="page-header">
      <h2>{{ t('modelArtifacts.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate">{{ t('modelArtifacts.create') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" size="small" @row-click="openDetail" highlight-current-row>
      <el-table-column :label="t('modelArtifacts.name')" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="source_type" :label="t('modelArtifacts.sourceType')" width="100" />
      <el-table-column prop="path" :label="t('modelArtifacts.path')" min-width="200" show-overflow-tooltip />
      <el-table-column prop="format" :label="t('modelArtifacts.format')" width="90" />
      <el-table-column prop="architecture" :label="t('modelArtifacts.architecture')" width="100" />
      <el-table-column prop="size_label" :label="t('modelArtifacts.sizeLabel')" width="80" />
      <el-table-column :label="t('modelArtifacts.createdAt')" width="160">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="120" fixed="right">
        <template #default="{ row }">
          <el-button size="small" text @click.stop="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button size="small" text type="danger" @click.stop="confirmDelete(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
      <template #empty><el-empty :description="t('modelArtifacts.noData')" /></template>
    </el-table>

    <!-- Form Dialog -->
    <el-dialog v-model="dialogVisible" :title="editingId ? t('modelArtifacts.edit') : t('modelArtifacts.create')" width="560px" @close="resetForm">
      <el-form :model="form" label-width="120px" size="small">
        <el-form-item :label="t('modelArtifacts.name')" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.displayName')">
          <el-input v-model="form.display_name" />
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.path')">
          <el-input v-model="form.path" placeholder="/data/models/..." />
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.sourceType')">
          <el-select v-model="form.source_type" style="width:100%">
            <el-option label="local_path" value="local_path" />
            <el-option label="mounted_path" value="mounted_path" />
            <el-option label="remote_repo" value="remote_repo" />
            <el-option label="object_storage" value="object_storage" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.format')">
          <el-select v-model="form.format" style="width:100%">
            <el-option label="hf" value="hf" /><el-option label="gguf" value="gguf" /><el-option label="safetensors" value="safetensors" />
            <el-option label="onnx" value="onnx" /><el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.taskType')">
          <el-select v-model="form.task_type" style="width:100%">
            <el-option label="chat" value="chat" /><el-option label="completion" value="completion" /><el-option label="embedding" value="embedding" />
            <el-option label="vision" value="vision" /><el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.architecture')">
          <el-input v-model="form.architecture" placeholder="qwen / llama / deepseek / custom" />
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.sizeLabel')">
          <el-input v-model="form.size_label" placeholder="7B / 14B / 32B" />
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.quantization')">
          <el-select v-model="form.quantization" style="width:100%">
            <el-option label="fp16" value="fp16" /><el-option label="bf16" value="bf16" /><el-option label="fp8" value="fp8" />
            <el-option label="int8" value="int8" /><el-option label="int4" value="int4" /><el-option label="unknown" value="unknown" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.contextLength')">
          <el-input-number v-model="form.default_context_length" :min="0" style="width:100%" />
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.estimatedVram')">
          <el-input-number v-model="form.estimated_vram_bytes" :min="0" :step="1073741824" style="width:100%" />
        </el-form-item>
        <el-form-item :label="t('modelArtifacts.requiredGpuCount')">
          <el-input-number v-model="form.required_gpu_count" :min="1" :max="8" style="width:100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchModelArtifacts, createModelArtifact, updateModelArtifact, deleteModelArtifact, type ModelArtifact } from '@/api/modelArtifacts'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const items = ref<ModelArtifact[]>([])
const { loading, refresh } = useAutoRefresh(async () => { items.value = await fetchModelArtifacts() })
const dialogVisible = ref(false)
const editingId = ref('')
const saving = ref(false)

const defaultForm = () => ({
  name: '', display_name: '', path: '', source_type: 'local_path', format: 'custom',
  task_type: 'chat', architecture: 'custom', size_label: '', quantization: 'unknown',
  default_context_length: 0, estimated_vram_bytes: 0, required_gpu_count: 1
})
const form = ref(defaultForm())

function resetForm() { editingId.value = ''; form.value = defaultForm() }
function openCreate() { resetForm(); dialogVisible.value = true }
function openEdit(row: ModelArtifact) {
  editingId.value = row.id
  form.value = { ...defaultForm(), ...row }
  dialogVisible.value = true
}
function openDetail(row: ModelArtifact) { openEdit(row) }

async function save() {
  saving.value = true
  try {
    if (editingId.value) {
      await updateModelArtifact(editingId.value, form.value)
      ElMessage.success(t('modelArtifacts.updateSuccess'))
    } else {
      await createModelArtifact(form.value)
      ElMessage.success(t('modelArtifacts.createSuccess'))
    }
    dialogVisible.value = false
    refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { saving.value = false }
}

async function confirmDelete(row: ModelArtifact) {
  try {
    await ElMessageBox.confirm(t('modelArtifacts.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await deleteModelArtifact(row.id)
    ElMessage.success(t('modelArtifacts.deleteSuccess'))
    refresh()
  } catch { /* cancelled */ }
}
</script>
