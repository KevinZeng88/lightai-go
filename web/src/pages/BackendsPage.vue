<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('backends.title') }}</h2>
      <el-button @click="load">{{ $t('common.refresh') }}</el-button>
    </div>

    <el-table :data="backends" v-loading="loading" stripe @row-click="selected = $event">
      <el-table-column prop="display_name" :label="$t('backends.name')" min-width="180" />
      <el-table-column prop="name" label="ID" min-width="160" />
      <el-table-column prop="status" :label="$t('common.status')" width="120" />
      <el-table-column prop="managed_by" label="Managed By" width="140" />
    </el-table>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.name || ''" size="72%">
      <template v-if="selected">
        <el-tabs>
          <el-tab-pane :label="$t('backends.title')">
            <JsonViewer :value="selected.config_set || {}" :title="$t('runtimes.rawConfigJson')" max-height="520px" :searchable="true" />
            <JsonViewer :value="selected.source_metadata || {}" :title="$t('runtimes.rawSourceMetadataJson')" max-height="240px" :searchable="true" />
          </el-tab-pane>
          <el-tab-pane :label="$t('backends.versions')">
            <div class="version-toolbar">
              <el-button size="small" @click="loadVersions(selected.id)">{{ $t('common.refresh') }}</el-button>
              <el-button size="small" type="primary" @click="newVersion">{{ $t('backends.addVersion') }}</el-button>
            </div>
            <el-table :data="versions" v-loading="versionsLoading" stripe highlight-current-row @row-click="selectVersion">
              <el-table-column prop="display_name" :label="$t('backends.versionName')" min-width="180" />
              <el-table-column prop="version" :label="$t('backends.software')" width="150" />
              <el-table-column prop="managed_by" :label="$t('runtimes.managedBy')" width="120" />
              <el-table-column :label="$t('common.readonly')" width="100">
                <template #default="{ row }">{{ row.readonly ? $t('common.yes') : $t('common.no') }}</template>
              </el-table-column>
              <el-table-column :label="$t('common.actions')" width="170" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" @click.stop="cloneVersion(row)">Clone</el-button>
                  <el-button v-if="!row.readonly" size="small" type="danger" @click.stop="removeVersion(row)">Delete</el-button>
                </template>
              </el-table-column>
            </el-table>

            <el-divider content-position="left">{{ versionForm.id ? $t('backends.versionEditor') : $t('backends.selectOrCreateVersion') }}</el-divider>
            <el-form v-if="versionForm.id || versionForm.creating" label-position="top">
              <el-form-item :label="$t('backends.versionName')">
                <el-input v-model="versionForm.version" :disabled="versionReadonly" />
              </el-form-item>
              <el-form-item :label="$t('backends.versionDisplayName')">
                <el-input v-model="versionForm.display_name" :disabled="versionReadonly" />
              </el-form-item>
              <el-form-item :label="$t('backends.versionDescription')">
                <el-input v-model="versionForm.description" type="textarea" :rows="2" :disabled="versionReadonly" />
              </el-form-item>
              <ConfigEditView
                v-if="versionEditView"
                :model-value="versionEditView"
                :readonly="versionReadonly"
                @update:patch="versionEditPatch = $event"
              />
              <el-collapse v-if="!versionReadonly" style="margin-top:12px">
                <el-collapse-item :title="$t('backends.addParameter')" name="add-param">
                  <el-alert type="info" :closable="false" style="margin-bottom:12px">
                    {{ $t('backends.addParameterHint') }}
                  </el-alert>
                  <el-form label-position="top" class="param-grid">
                    <el-form-item :label="$t('backends.paramCode')"><el-input v-model="newParam.code" placeholder="model_runtime.custom_parameter" /></el-form-item>
                    <el-form-item :label="$t('backends.paramLabel')"><el-input v-model="newParam.label" /></el-form-item>
                    <el-form-item :label="$t('backends.paramHelp')"><el-input v-model="newParam.help" /></el-form-item>
                    <el-form-item :label="$t('backends.paramCategory')"><el-input v-model="newParam.category" /></el-form-item>
                    <el-form-item :label="$t('backends.paramGroup')"><el-input v-model="newParam.group" /></el-form-item>
                    <el-form-item :label="$t('backends.paramKind')"><el-input v-model="newParam.kind" /></el-form-item>
                    <el-form-item :label="$t('backends.paramType')"><el-input v-model="newParam.type" /></el-form-item>
                    <el-form-item :label="$t('backends.paramFlag')"><el-input v-model="newParam.flag" /></el-form-item>
                    <el-form-item :label="$t('backends.paramDefaultValue')"><el-input v-model="newParam.value" /></el-form-item>
                    <el-form-item :label="$t('backends.paramOrder')"><el-input v-model.number="newParam.order" /></el-form-item>
                    <el-form-item :label="$t('backends.paramEnabled')"><el-switch v-model="newParam.enabled" /></el-form-item>
                    <el-form-item :label="$t('backends.paramRequired')"><el-switch v-model="newParam.required" /></el-form-item>
                  </el-form>
                  <el-button size="small" @click="addParameter">{{ $t('backends.addParameterButton') }}</el-button>
                </el-collapse-item>
              </el-collapse>
              <div v-if="!versionReadonly" class="version-actions">
                <el-button type="primary" :loading="savingVersion" @click="saveVersion">Save Version</el-button>
              </div>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  cloneBackendVersion,
  createBackendVersion,
  deleteBackendVersion,
  listBackendVersions,
  listBackends,
  patchBackendVersion,
} from '@/api/backends'
import { applyConfigEditPatch, getConfigEditView } from '@/api/configEdit'
import { apiErrorMessage } from '@/utils/apiErrors'
import type { ConfigEditPatch, ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'
import JsonViewer from '@/components/common/JsonViewer.vue'
import ConfigEditView from '@/components/config/ConfigEditView.vue'

const { t } = useI18n()
const loading = ref(false)
const backends = ref<any[]>([])
const selected = ref<any | null>(null)
const versions = ref<any[]>([])
const versionsLoading = ref(false)
const savingVersion = ref(false)
const versionEditorModel = ref<Record<string, any>>({ config_set: {} })
const versionEditView = ref<ConfigEditViewModel | null>(null)
const versionEditPatch = ref<ConfigEditPatch | null>(null)
const versionForm = reactive<Record<string, any>>({ id: '', creating: false, version: '', display_name: '', description: '', readonly: true })
const newParam = reactive({
  code: 'model_runtime.custom_parameter',
  label: 'Custom Parameter',
  help: '',
  category: 'model_runtime',
  group: 'Custom',
  kind: 'cli_arg',
  type: 'string',
  flag: '--fake-new-param',
  value: '',
  order: 340,
  enabled: false,
  required: false,
})
const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) { selected.value = null; versions.value = []; resetVersionForm() } },
})
const versionReadonly = computed(() => Boolean(versionForm.readonly) && !versionForm.creating)

watch(selected, (backend) => {
  resetVersionForm()
  if (backend?.id) loadVersions(backend.id)
})

async function load() {
  loading.value = true
  try {
    backends.value = await listBackends()
  } finally {
    loading.value = false
  }
}

async function loadVersions(backendId: string) {
  versionsLoading.value = true
  try {
    versions.value = await listBackendVersions(backendId)
  } finally {
    versionsLoading.value = false
  }
}

function resetVersionForm() {
  Object.assign(versionForm, { id: '', creating: false, version: '', display_name: '', description: '', readonly: true })
  versionEditorModel.value = { config_set: {} }
  versionEditView.value = null
  versionEditPatch.value = null
}

async function selectVersion(row: any) {
  Object.assign(versionForm, {
    id: row.id,
    creating: false,
    version: row.version || '',
    display_name: row.display_name || row.version || '',
    description: row.description || '',
    readonly: Boolean(row.readonly),
  })
  versionEditorModel.value = { config_set: row.config_set ? JSON.parse(JSON.stringify(row.config_set)) : {} }
  versionEditPatch.value = null
  versionEditView.value = await getConfigEditView({
    object_kind: 'backend_version',
    object_id: row.id,
    layer: 'backend_version',
    mode: row.readonly ? 'view' : 'edit',
    view_level: 'normal',
  })
}

function newVersion() {
  if (!selected.value) return
  Object.assign(versionForm, {
    id: '',
    creating: true,
    version: 'custom-version',
    display_name: 'Custom Version',
    description: '',
    readonly: false,
  })
  versionEditorModel.value = { config_set: selected.value.config_set ? JSON.parse(JSON.stringify(selected.value.config_set)) : { items: {} } }
  versionEditView.value = null
  versionEditPatch.value = null
}

async function cloneVersion(row: any) {
  try {
    const cloned = await cloneBackendVersion(row.id)
    ElMessage.success(t('backends.clonedVersion'))
    if (selected.value?.id) await loadVersions(selected.value.id)
    await selectVersion(cloned)
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  }
}

async function removeVersion(row: any) {
  try {
    await deleteBackendVersion(row.id)
    ElMessage.success(t('backends.deletedVersion'))
    if (selected.value?.id) await loadVersions(selected.value.id)
    resetVersionForm()
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  }
}

function addParameter() {
  const code = newParam.code.trim()
  if (!code) return
  if (isLegacyUserConfigKey(code)) {
    ElMessage.error(t('backends.legacyKeyRejected'))
    return
  }
  const configSet = versionEditorModel.value.config_set || { items: {} }
  configSet.items = configSet.items || {}
  configSet.items[code] = {
    code,
    category: newParam.category || 'model_runtime',
    kind: newParam.kind || 'cli_arg',
    type: newParam.type || 'string',
    enabled: newParam.enabled,
    required: newParam.required,
    value: parseParamValue(newParam.value, newParam.type),
    default_value: parseParamValue(newParam.value, newParam.type),
    render: {
      label: newParam.label || code,
      help: newParam.help || '',
      group: newParam.group || newParam.category || 'model_runtime',
      flag: newParam.flag || undefined,
      target: newParam.kind === 'env' ? 'env' : 'cli',
      style: newParam.type === 'boolean' ? 'flag_if_true' : 'flag_space_value',
    },
    order: Number(newParam.order) || 9999,
    support_level: 'documented',
  }
  versionEditorModel.value = { ...versionEditorModel.value, config_set: configSet }
}

function isLegacyUserConfigKey(code: string): boolean {
  return code.startsWith('backend.arg.') ||
    code.startsWith('backend.common.') ||
    code === 'launcher.listen_host' ||
    code === 'launcher.container_port'
}

function parseParamValue(value: string, type: string) {
  if (type === 'integer') {
    const n = Number.parseInt(value, 10)
    return Number.isFinite(n) ? n : value
  }
  if (type === 'number') {
    const n = Number.parseFloat(value)
    return Number.isFinite(n) ? n : value
  }
  if (type === 'boolean') return value === 'true'
  return value
}

async function saveVersion() {
  if (!selected.value) return
  savingVersion.value = true
  const payload = {
    version: versionForm.version,
    display_name: versionForm.display_name,
    description: versionForm.description,
    config_set: versionEditorModel.value.config_set || { items: {} },
  }
  try {
    const saved = versionForm.creating
      ? await createBackendVersion(selected.value.id, payload)
      : await patchBackendVersion(versionForm.id, payload)
    ElMessage.success(t('backends.savedVersion'))
    await loadVersions(selected.value.id)
    if (!versionForm.creating && versionEditPatch.value) {
      await applyConfigEditPatch({
        object_kind: 'backend_version',
        object_id: versionForm.id,
        layer: 'backend_version',
        patch: versionEditPatch.value,
      })
    }
    await selectVersion(saved)
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e, t, 'common.requestFailed'))
  } finally {
    savingVersion.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.version-toolbar { display: flex; justify-content: flex-end; gap: 8px; margin-bottom: 12px; }
.version-actions { margin-top: 12px; text-align: right; }
.param-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px 12px; }
@media (max-width: 900px) {
  .param-grid { grid-template-columns: 1fr; }
}
</style>
