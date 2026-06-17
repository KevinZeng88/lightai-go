<template>
  <div class="page-container">
    <h2>{{ $t('runtimes.title') }}</h2>
    <el-button type="primary" @click="showCreate">{{ $t('runtimes.createFromTemplate') }}</el-button>
    <el-table :data="runtimes" v-loading="loading" stripe style="margin-top:12px">
      <el-table-column prop="name" :label="$t('runtimes.name')" width="180" />
      <el-table-column prop="vendor" :label="$t('runtimes.vendor')" width="80" />
      <el-table-column prop="runtime_type" :label="$t('runtimes.type')" width="80" />
      <el-table-column prop="image_name" :label="$t('runtimes.image')" min-width="200" />
      <el-table-column :label="$t('common.actions')" width="260">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create from template dialog -->
    <el-dialog v-model="createVisible" :title="$t('runtimes.createFromTemplate')" width="500px">
      <el-form :model="createForm" label-width="120px">
        <el-form-item :label="$t('runtimes.templateName')">
          <el-select v-model="createForm.template_name" placeholder="Select template">
            <el-option v-for="t in templates" :key="t.name" :label="t.name" :value="t.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('runtimes.name')">
          <el-input v-model="createForm.name" />
        </el-form-item>
        <el-form-item :label="$t('runtimes.vendor')">
          <el-input v-model="createForm.vendor" />
        </el-form-item>
        <el-form-item :label="$t('runtimes.image')">
          <el-input v-model="createForm.image_name" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doCreate" :loading="creating">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Edit dialog -->
    <el-dialog v-model="editVisible" :title="$t('common.edit')" width="500px">
      <el-form :model="editForm" label-width="120px">
        <el-form-item :label="$t('runtimes.displayName')">
          <el-input v-model="editForm.display_name" />
        </el-form-item>
        <el-form-item :label="$t('runtimes.image')">
          <el-input v-model="editForm.image_name" />
        </el-form-item>
        <el-form-item :label="$t('runtimes.vendor')">
          <el-input v-model="editForm.vendor" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doEdit" :loading="editing">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Detail dialog -->
    <el-dialog v-model="detailVisible" :title="$t('common.detail')" width="600px">
      <div v-if="selected">
        <p v-for="(v,k) in selected" :key="k"><strong>{{ k }}:</strong> {{ typeof v === 'object' ? JSON.stringify(v) : v }}</p>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listRuntimes, getRuntime, createRuntimeFromTemplate, patchRuntime, deleteRuntime, type BackendRuntime } from '@/api/runtimes'
import { listRuntimeTemplates, type BackendRuntimeTemplate } from '@/api/backends'

const loading = ref(false); const creating = ref(false); const editing = ref(false)
const runtimes = ref<BackendRuntime[]>([]); const templates = ref<BackendRuntimeTemplate[]>([])
const selected = ref<any>(null)
const createVisible = ref(false); const editVisible = ref(false); const detailVisible = ref(false)
const createForm = ref({ template_name: 'vllm-nvidia-docker', name: '', vendor: 'nvidia', image_name: '', backend_name: 'vllm', backend_version: '0.8.5', display_name: '' })
const editForm = ref({ display_name: '', image_name: '', vendor: '' })
let editingId = ''

onMounted(async () => { await refresh() })

async function refresh() {
  loading.value = true
  try { runtimes.value = await listRuntimes(); templates.value = await listRuntimeTemplates() } catch (e: any) { console.error(e) }
  loading.value = false
}

function showCreate() { createVisible.value = true }
async function doCreate() {
  creating.value = true
  try {
    await createRuntimeFromTemplate(createForm.value)
    ElMessage.success('Created')
    createVisible.value = false
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  creating.value = false
}

function showEdit(row: BackendRuntime) {
  editingId = row.id
  editForm.value = { display_name: row.display_name, image_name: row.image_name, vendor: row.vendor }
  editVisible.value = true
}
async function doEdit() {
  editing.value = true
  try {
    await patchRuntime(editingId, editForm.value)
    ElMessage.success('Saved')
    editVisible.value = false
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Failed') }
  editing.value = false
}

function showDetail(row: BackendRuntime) { selected.value = row; detailVisible.value = true }

async function handleDelete(row: BackendRuntime) {
  try {
    await ElMessageBox.confirm(`Delete ${row.name}?`, 'Confirm', { type: 'warning' })
    await deleteRuntime(row.id)
    ElMessage.success('Deleted')
    await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || 'Failed') }
}

const JSON = { stringify: (o: any) => globalThis.JSON.stringify(o, null, 2) }
</script>
