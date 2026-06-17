<template>
  <div class="page-container">
    <h2>{{ $t('artifacts.title') }}</h2>
    <el-button type="primary" @click="showCreate">{{ $t('common.create') }}</el-button>
    <el-table :data="items" v-loading="loading" stripe style="margin-top:12px">
      <el-table-column prop="name" :label="$t('artifacts.name')" width="150" />
      <el-table-column prop="format" :label="$t('artifacts.format')" width="100" />
      <el-table-column prop="task_type" :label="$t('artifacts.taskType')" width="100" />
      <el-table-column prop="size_label" :label="$t('artifacts.size')" width="80" />
      <el-table-column prop="path" :label="$t('artifacts.path')" min-width="200" />
      <el-table-column :label="$t('common.actions')" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingId ? $t('common.edit') : $t('common.create')" width="500px">
      <el-form :model="form" label-width="140px">
        <el-form-item :label="$t('artifacts.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="$t('artifacts.path')"><el-input v-model="form.path" /></el-form-item>
        <el-form-item :label="$t('artifacts.format')"><el-select v-model="form.format" filterable allow-create style="width:100%"><el-option v-for="o in formatOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
        <el-form-item :label="$t('artifacts.taskType')"><el-select v-model="form.task_type" filterable allow-create style="width:100%"><el-option v-for="o in taskTypeOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
        <el-form-item :label="$t('artifacts.architecture')"><el-select v-model="form.architecture" filterable allow-create style="width:100%"><el-option v-for="o in architectureOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
        <el-form-item :label="$t('artifacts.size')"><el-input v-model="form.size_label" /></el-form-item>
        <el-form-item :label="$t('artifacts.quantization')"><el-select v-model="form.quantization" filterable allow-create style="width:100%"><el-option v-for="o in quantOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doSave" :loading="saving">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'

const loading = ref(false); const saving = ref(false)
const items = ref<any[]>([]); const dialogVisible = ref(false)
const form = ref({ name: '', path: '', format: 'gguf', task_type: 'chat', architecture: 'qwen', size_label: '', quantization: 'Q4_K_M', source_type: 'local_path', display_name: '' })
let editingId = ''

// REVIEW-027: Recommended options + custom input for model metadata fields.
const formatOptions = ['gguf', 'safetensors', 'pt', 'onnx', 'other']
const taskTypeOptions = ['chat', 'completion', 'embedding', 'rerank', 'image', 'audio', 'other']
const architectureOptions = ['qwen', 'llama', 'glm', 'deepseek', 'baichuan', 'mistral', 'other']
const quantOptions = ['Q4_K_M', 'Q5_K_M', 'Q8_0', 'FP16', 'BF16', 'FP8', 'INT8', 'INT4', 'none', 'other']

onMounted(async () => { await refresh() })
async function refresh() {
  loading.value = true
  try { items.value = await apiClient.get('/api/v1/model-artifacts') } catch (e: any) { console.error(e) }
  loading.value = false
}

function showCreate() { editingId = ''; form.value = { name: '', path: '', format: 'custom', task_type: 'chat', architecture: 'custom', size_label: '', quantization: 'unknown', source_type: 'local_path', display_name: '' }; dialogVisible.value = true }
function showEdit(row: any) { editingId = row.id; Object.assign(form.value, row); dialogVisible.value = true }

async function doSave() {
  saving.value = true
  try {
    if (editingId) {
      await apiClient.patch(`/api/v1/model-artifacts/${editingId}`, form.value)
    } else {
      await apiClient.post('/api/v1/model-artifacts', form.value)
    }
    ElMessage.success('Saved'); dialogVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  saving.value = false
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`Delete ${row.name}?`, 'Confirm', { type: 'warning' })
    await apiClient.delete(`/api/v1/model-artifacts/${row.id}`)
    ElMessage.success('Deleted'); await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || 'Failed') }
}
</script>
