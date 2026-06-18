<template>
  <div class="page-container">
    <h2>{{ $t('backends.title') }}</h2>
    <el-table :data="backends" v-loading="loading" stripe>
      <el-table-column prop="name" :label="$t('backends.name')" width="120" />
      <el-table-column prop="display_name" :label="$t('backends.displayName')" width="150" />
      <el-table-column prop="default_version" :label="$t('backends.defaultVersion')" width="100" />
      <el-table-column prop="parameter_format" :label="$t('backends.paramFormat')" width="100" />
      <el-table-column :label="$t('backends.actions')" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" @click="showVersions(row)">{{ $t('backends.versions') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="detailVisible" :title="$t('backends.detail')" width="700px">
      <div v-if="selected">
        <p><strong>{{ $t('backends.name') }}:</strong> {{ selected.name }}</p>
        <p><strong>{{ $t('backends.displayName') }}:</strong> {{ selected.display_name }}</p>
        <p><strong>{{ $t('backends.defaultVersion') }}:</strong> {{ selected.default_version }}</p>
        <p><strong>{{ $t('backends.paramFormat') }}:</strong> {{ selected.parameter_format }}</p>
        <p><strong>{{ $t('backends.commonParams') }}:</strong> {{ JSON.stringify(selected.common_parameters_json) }}</p>
        <p><strong>Protocol:</strong> {{ JSON.stringify(selected.protocol_json) }}</p>
      </div>
    </el-dialog>

    <el-dialog v-model="versionsVisible" :title="$t('backends.versions')" width="980px">
      <div style="margin-bottom:12px;text-align:right">
        <el-button @click="reloadCatalog" :loading="syncing">{{ $t('backends.reloadCatalog') }}</el-button>
        <el-button type="primary" @click="showVersionCreate">{{ $t('backends.addVersion') }}</el-button>
      </div>
      <el-table :data="versions" stripe>
        <el-table-column prop="version" label="Version" width="100" />
        <el-table-column prop="display_name" :label="$t('backends.displayName')" width="150" />
        <el-table-column prop="managed_by" :label="$t('runtimes.managedBy')" width="100" />
        <el-table-column :label="$t('backends.isDefault')" width="100">
          <template #default="{ row }">{{ row.is_default ? '✓' : '' }}</template>
        </el-table-column>
        <el-table-column prop="default_container_port" :label="$t('backends.port')" width="80" />
        <el-table-column :label="$t('common.actions')" width="220">
          <template #default="{ row }">
            <el-button v-if="row.managed_by === 'system'" size="small" @click="cloneVersion(row)">{{ $t('backends.cloneVersion') }}</el-button>
            <el-button size="small" :disabled="row.readonly || row.managed_by === 'system'" @click="showVersionEdit(row)">{{ $t('common.edit') }}</el-button>
            <el-button size="small" type="danger" :disabled="row.readonly || row.managed_by === 'system'" @click="deleteVersion(row)">{{ $t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="versionEditVisible" :title="versionEditingId ? $t('backends.editVersion') : $t('backends.addVersion')" width="820px">
      <el-form label-position="top">
        <div class="version-grid">
          <el-form-item :label="$t('backends.versionName')"><el-input v-model="versionForm.version" /></el-form-item>
          <el-form-item :label="$t('backends.displayName')"><el-input v-model="versionForm.display_name" /></el-form-item>
          <el-form-item :label="$t('backends.protocol')"><el-input v-model="versionForm.protocol" /></el-form-item>
          <el-form-item :label="$t('backends.port')"><el-input-number v-model="versionForm.default_container_port" :min="1" :max="65535" style="width:100%" /></el-form-item>
        </div>
        <el-form-item :label="$t('backends.description')"><el-input v-model="versionForm.description" /></el-form-item>
        <el-form-item :label="$t('backends.imageCandidates')"><el-input v-model="versionForm.image_candidates_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.defaultImages')"><el-input v-model="versionForm.default_images_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.defaultEndpoints')"><el-input v-model="versionForm.default_endpoints_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.argsSchema')"><el-input v-model="versionForm.default_args_schema_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.defaultArgs')"><el-input v-model="versionForm.default_args_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.defaultEnv')"><el-input v-model="versionForm.env_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.healthCheck')"><el-input v-model="versionForm.health_check_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.dockerOptions')"><el-input v-model="versionForm.docker_options_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.capabilities')"><el-input v-model="versionForm.capabilities_json" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="$t('backends.modelMount')"><el-input v-model="versionForm.model_mount_json" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="versionEditVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveVersion" :loading="savingVersion">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { cloneBackendVersion, createBackendVersion, deleteBackendVersion, listBackends, listBackendVersions, patchBackendVersion, reloadBackendCatalog, type InferenceBackend, type BackendVersion } from '@/api/backends'

const { t } = useI18n()
const loading = ref(false)
const backends = ref<InferenceBackend[]>([])
const selected = ref<InferenceBackend | null>(null)
const detailVisible = ref(false)
const versionsVisible = ref(false)
const versions = ref<BackendVersion[]>([])
const syncing = ref(false)
const versionEditVisible = ref(false)
const savingVersion = ref(false)
const versionEditingId = ref('')
const versionForm = ref<any>({ version: '', display_name: '', protocol: '', description: '', default_container_port: 8000, image_candidates_json: '[]', default_images_json: '{}', default_endpoints_json: '{}', default_args_schema_json: '[]', default_args_json: '[]', env_json: '{}', health_check_json: '{}', docker_options_json: '{}', capabilities_json: '{}', model_mount_json: '{}' })

onMounted(async () => {
  loading.value = true
  try { backends.value = await listBackends() } catch (e: any) { console.error(e) }
  loading.value = false
})

async function showDetail(row: InferenceBackend) {
  selected.value = row
  detailVisible.value = true
}

async function showVersions(row: InferenceBackend) {
  try { versions.value = await listBackendVersions(row.id) } catch (e: any) { console.error(e) }
  selected.value = row
  versionsVisible.value = true
}

function showVersionCreate() {
  versionEditingId.value = ''
  versionForm.value = { version: '', display_name: '', protocol: 'openai-compatible', description: '', default_container_port: 8000, image_candidates_json: '[]', default_images_json: '{}', default_endpoints_json: '{}', default_args_schema_json: '[]', default_args_json: '[]', env_json: '{}', health_check_json: '{}', docker_options_json: '{}', capabilities_json: '{}', model_mount_json: '{}' }
  versionEditVisible.value = true
}

function showVersionEdit(row: BackendVersion) {
  versionEditingId.value = row.id
  versionForm.value = {
    version: row.version,
    display_name: row.display_name,
    protocol: row.protocol || '',
    description: row.description || '',
    default_container_port: row.default_container_port || 8000,
    image_candidates_json: JSON.stringify(row.image_candidates_json || [], null, 2),
    default_images_json: JSON.stringify(row.default_images_json || {}, null, 2),
    default_endpoints_json: JSON.stringify(row.default_endpoints_json || {}, null, 2),
    default_args_schema_json: JSON.stringify(row.default_args_schema_json || [], null, 2),
    default_args_json: JSON.stringify(row.default_args_json || [], null, 2),
    env_json: JSON.stringify(row.env_json || {}, null, 2),
    health_check_json: JSON.stringify(row.health_check_json || {}, null, 2),
    docker_options_json: JSON.stringify(row.docker_options_json || {}, null, 2),
    capabilities_json: JSON.stringify(row.capabilities_json || {}, null, 2),
    model_mount_json: JSON.stringify(row.model_mount_json || {}, null, 2),
  }
  versionEditVisible.value = true
}

async function cloneVersion(row: BackendVersion) {
  try {
    const cloned = await cloneBackendVersion(row.id)
    await refreshVersions()
    showVersionEdit(cloned)
    ElMessage.success(t('backends.clonedVersion'))
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}

async function saveVersion() {
  if (!selected.value) return
  savingVersion.value = true
  try {
    const payload = {
      version: versionForm.value.version,
      display_name: versionForm.value.display_name,
      protocol: versionForm.value.protocol,
      description: versionForm.value.description,
      default_container_port: versionForm.value.default_container_port,
      image_candidates_json: parseJSON(versionForm.value.image_candidates_json),
      default_images_json: parseJSON(versionForm.value.default_images_json),
      default_endpoints_json: parseJSON(versionForm.value.default_endpoints_json),
      default_args_schema_json: parseJSON(versionForm.value.default_args_schema_json),
      default_args_json: parseJSON(versionForm.value.default_args_json),
      env_json: parseJSON(versionForm.value.env_json),
      health_check_json: parseJSON(versionForm.value.health_check_json),
      docker_options_json: parseJSON(versionForm.value.docker_options_json),
      capabilities_json: parseJSON(versionForm.value.capabilities_json),
      model_mount_json: parseJSON(versionForm.value.model_mount_json),
    }
    if (versionEditingId.value) await patchBackendVersion(versionEditingId.value, payload)
    else await createBackendVersion(selected.value.id, payload)
    ElMessage.success(t('backends.savedVersion'))
    versionEditVisible.value = false
    await refreshVersions()
  } catch (e: any) { ElMessage.error(e?.message || t('backends.invalidVersionForm')) }
  savingVersion.value = false
}

async function deleteVersion(row: BackendVersion) {
  try {
    await deleteBackendVersion(row.id)
    ElMessage.success(t('backends.deletedVersion'))
    await refreshVersions()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}

async function reloadCatalog() {
  syncing.value = true
  try {
    const result = await reloadBackendCatalog()
    ElMessage.success(t('backends.reloadCatalogDone', { count: result.versions ?? 0 }))
    await refreshVersions()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  syncing.value = false
}

async function refreshVersions() {
  if (!selected.value) return
  versions.value = await listBackendVersions(selected.value.id)
}

function parseJSON(text: string) {
  return JSON.parse(text || '{}')
}

// JSON.stringify used directly via globalThis
</script>

<style scoped>
.version-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
@media (max-width: 900px) { .version-grid { grid-template-columns: 1fr; } }
</style>
