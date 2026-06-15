<template>
  <div class="re-page">
    <div class="page-header">
      <h2>{{ t('runtimeEnvs.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate">{{ t('runtimeEnvs.create') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" size="small" @row-click="openDetail" highlight-current-row>
      <el-table-column :label="t('runtimeEnvs.name')" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="runtime_type" :label="t('runtimeEnvs.runtimeType')" width="90" />
      <el-table-column prop="vendor" :label="t('runtimeEnvs.vendor')" width="90" />
      <el-table-column prop="backend_type" :label="t('runtimeEnvs.backendType')" width="100" />
      <el-table-column prop="default_port" :label="t('runtimeEnvs.defaultPort')" width="100" />
      <el-table-column :label="t('runtimeEnvs.description')" min-width="150" show-overflow-tooltip prop="description" />
      <el-table-column :label="t('common.actions')" width="120" fixed="right">
        <template #default="{ row }">
          <el-button size="small" text @click.stop="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button size="small" text type="danger" @click.stop="confirmDelete(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
      <template #empty><el-empty :description="t('runtimeEnvs.noData')" /></template>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingId ? t('runtimeEnvs.edit') : t('runtimeEnvs.create')" width="640px" @close="resetForm">
      <el-form :model="form" label-width="140px" size="small">
        <el-form-item :label="t('runtimeEnvs.name')" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('runtimeEnvs.displayName')"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item :label="t('runtimeEnvs.runtimeType')">
          <el-select v-model="form.runtime_type" style="width:100%">
            <el-option label="docker" value="docker" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.vendor')">
          <el-select v-model="form.vendor" style="width:100%">
            <el-option label="nvidia" value="nvidia" /><el-option label="metax" value="metax" /><el-option label="cpu" value="cpu" /><el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.backendType')">
          <el-select v-model="form.backend_type" style="width:100%">
            <el-option label="vllm" value="vllm" /><el-option label="sglang" value="sglang" /><el-option label="llama_cpp" value="llama_cpp" />
            <el-option label="mindie" value="mindie" /><el-option label="ollama" value="ollama" /><el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.defaultPort')"><el-input-number v-model="form.default_port" :min="1" :max="65535" style="width:100%" /></el-form-item>
        <el-form-item :label="t('runtimeEnvs.healthCheckPath')"><el-input v-model="form.health_check_path" /></el-form-item>
        <el-form-item :label="t('runtimeEnvs.description')"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
        <el-divider>{{ t('runtimeEnvs.docker') }}</el-divider>
        <el-form-item :label="t('runtimeEnvs.image')"><el-input v-model="form.docker.image" placeholder="vllm/vllm-openai:latest" /></el-form-item>
        <el-form-item label="Volumes">
          <el-checkbox v-model="form.docker.volumes_enabled" size="small" style="margin-right:8px">{{ t('runtimeEnvs.enabled') }}</el-checkbox>
          <el-input v-model="form.docker.volumes" :disabled="!form.docker.volumes_enabled" type="textarea" :rows="2" placeholder="/host/path:/container/path:ro&#10;/host/path2:/container/path2" />
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.ipcMode')">
          <el-checkbox v-model="form.docker.ipc_mode_enabled" size="small" style="margin-right:8px">{{ t('runtimeEnvs.enabled') }}</el-checkbox>
          <el-input v-model="form.docker.ipc_mode" :disabled="!form.docker.ipc_mode_enabled" placeholder="host" style="width:200px" />
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.shmSize')">
          <el-checkbox v-model="form.docker.shm_size_enabled" size="small" style="margin-right:8px">{{ t('runtimeEnvs.enabled') }}</el-checkbox>
          <el-input v-model="form.docker.shm_size" :disabled="!form.docker.shm_size_enabled" placeholder="8gb" style="width:200px" />
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.privileged')">
          <el-checkbox v-model="form.docker.privileged_enabled" size="small" style="margin-right:8px">{{ t('runtimeEnvs.enabled') }}</el-checkbox>
          <el-switch v-model="form.docker.privileged" :disabled="!form.docker.privileged_enabled" />
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.networkMode')">
          <el-checkbox v-model="form.docker.network_mode_enabled" size="small" style="margin-right:8px">{{ t('runtimeEnvs.enabled') }}</el-checkbox>
          <el-input v-model="form.docker.network_mode" :disabled="!form.docker.network_mode_enabled" placeholder="host" style="width:200px" />
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.securityOptions')">
          <el-checkbox v-model="form.docker.security_options_enabled" size="small" style="margin-right:8px">{{ t('runtimeEnvs.enabled') }}</el-checkbox>
          <el-input v-model="form.docker.security_options" :disabled="!form.docker.security_options_enabled" placeholder="seccomp=unconfined" />
        </el-form-item>
        <el-form-item :label="t('runtimeEnvs.gpuVisibleEnvKey')">
          <el-input v-model="form.docker.gpu_visible_env_key" placeholder="CUDA_VISIBLE_DEVICES" />
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
import { fetchRuntimeEnvironments, createRuntimeEnvironment, updateRuntimeEnvironment, deleteRuntimeEnvironment, type RuntimeEnvironment } from '@/api/runtimeEnvironments'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t } = useI18n()
const items = ref<RuntimeEnvironment[]>([])
const { loading, refresh } = useAutoRefresh(async () => { items.value = await fetchRuntimeEnvironments() })
const dialogVisible = ref(false)
const editingId = ref('')
const saving = ref(false)

function defaultDocker(): any {
  return { image: '', ipc_mode: '', ipc_mode_enabled: false, shm_size: '', shm_size_enabled: false,
    privileged: false, privileged_enabled: false, network_mode: '', network_mode_enabled: false,
    security_options: '', security_options_enabled: false, gpu_visible_env_key: 'CUDA_VISIBLE_DEVICES',
    volumes: '', volumes_enabled: false }
}
const defaultForm = () => ({
  name: '', display_name: '', runtime_type: 'docker', vendor: 'nvidia', backend_type: 'vllm',
  default_port: 8000, health_check_path: '/health', description: '', docker: defaultDocker()
})
const form = ref(defaultForm())

function resetForm() { editingId.value = ''; form.value = defaultForm() }
function openCreate() { resetForm(); dialogVisible.value = true }
function openEdit(row: RuntimeEnvironment) {
  editingId.value = row.id
  const d = row.docker || {}
  form.value = {
    ...defaultForm(), name: row.name, display_name: row.display_name, runtime_type: row.runtime_type,
    vendor: row.vendor, backend_type: row.backend_type, default_port: row.default_port,
    health_check_path: row.health_check_path, description: row.description,
    docker: { ...defaultDocker(), image: d.image || '', gpu_visible_env_key: d.gpu_visible_env_key || 'CUDA_VISIBLE_DEVICES',
      ipc_mode: typeof d.ipc_mode === 'object' ? (d.ipc_mode as any).value : (d.ipc_mode || ''),
      ipc_mode_enabled: typeof d.ipc_mode === 'object' ? !!(d.ipc_mode as any).enabled : !!d.ipc_mode,
      shm_size: typeof d.shm_size === 'object' ? (d.shm_size as any).value : (d.shm_size || ''),
      shm_size_enabled: typeof d.shm_size === 'object' ? !!(d.shm_size as any).enabled : !!d.shm_size,
      privileged: typeof d.privileged === 'object' ? (d.privileged as any).value : d.privileged,
      privileged_enabled: typeof d.privileged === 'object' ? !!(d.privileged as any).enabled : (d.privileged !== undefined),
      network_mode: typeof d.network_mode === 'object' ? (d.network_mode as any).value : (d.network_mode || ''),
      network_mode_enabled: typeof d.network_mode === 'object' ? !!(d.network_mode as any).enabled : !!d.network_mode,
    }
  }
  dialogVisible.value = true
}
function openDetail(row: RuntimeEnvironment) { openEdit(row) }

function buildDockerPayload(): any {
  const d = form.value.docker
  // Parse volumes from textarea format: /host:/container[:ro]
  const volLines = d.volumes_enabled ? (d.volumes || '').split('\n').filter((l: string) => l.trim()) : []
  const volumes = volLines.map((line: string) => {
    const parts = line.trim().split(':')
    return { host_path: parts[0], container_path: parts[1] || parts[0], readonly: parts[2] === 'ro' }
  })
  return {
    image: d.image,
    ipc_mode: d.ipc_mode_enabled ? { enabled: true, value: d.ipc_mode } : { enabled: false },
    shm_size: d.shm_size_enabled ? { enabled: true, value: d.shm_size } : { enabled: false },
    privileged: d.privileged_enabled ? { enabled: true, value: d.privileged } : { enabled: false },
    network_mode: d.network_mode_enabled ? { enabled: true, value: d.network_mode } : { enabled: false },
    security_options: d.security_options_enabled ? { enabled: true, value: d.security_options } : { enabled: false },
    gpu_visible_env_key: d.gpu_visible_env_key || 'CUDA_VISIBLE_DEVICES',
    devices: d.volumes_enabled && volumes.length ? { enabled: true, value: volumes.map((v: any) => ({host_path: v.host_path, container_path: v.container_path, permissions: 'ro'})) } : { enabled: false },
  }
}

async function save() {
  saving.value = true
  try {
    const payload: any = { ...form.value, docker: buildDockerPayload() }
    delete payload.docker.ipc_mode_enabled; delete payload.docker.shm_size_enabled
    delete payload.docker.privileged_enabled; delete payload.docker.network_mode_enabled
    delete payload.docker.security_options_enabled
    
    if (editingId.value) {
      await updateRuntimeEnvironment(editingId.value, payload)
      ElMessage.success(t('runtimeEnvs.updateSuccess'))
    } else {
      await createRuntimeEnvironment(payload)
      ElMessage.success(t('runtimeEnvs.createSuccess'))
    }
    dialogVisible.value = false
    refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { saving.value = false }
}

async function confirmDelete(row: RuntimeEnvironment) {
  try {
    await ElMessageBox.confirm(t('runtimeEnvs.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await deleteRuntimeEnvironment(row.id)
    ElMessage.success(t('runtimeEnvs.deleteSuccess'))
    refresh()
  } catch { /* cancelled */ }
}
</script>
