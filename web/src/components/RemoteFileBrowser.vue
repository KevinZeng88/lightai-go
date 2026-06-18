<template>
  <div class="file-browser">
    <!-- Root picker (shown when no root selected) -->
    <div v-if="!currentRoot" class="browser-picker">
      <div class="picker-label">{{ $t('fileBrowser.selectRoot') }}</div>
      <el-empty v-if="!rootsLoading && !roots.length" :description="$t('fileBrowser.noRoots')" />
      <div class="picker-row">
        <el-select v-model="selectedRootId" :placeholder="$t('fileBrowser.selectRoot')" style="flex:1" @change="onRootSelected" :loading="rootsLoading">
          <el-option v-for="r in roots" :key="r.id" :label="r.path" :value="r.id" />
        </el-select>
        <el-button :icon="Plus" size="small" @click="showAddRoot">{{ $t('fileBrowser.addRoot') }}</el-button>
      </div>
      <div v-if="roots.length" class="dynamic-roots">
        <el-tag v-for="r in roots" :key="r.id" closable size="small" @close="doRemoveRoot(r)">{{ r.path }}</el-tag>
      </div>
    </div>

    <!-- Add root dialog -->
    <el-dialog v-model="addRootVisible" :title="$t('fileBrowser.addRoot')" width="400px">
      <el-input v-model="newRootPath" :placeholder="$t('fileBrowser.addRootPlaceholder')" />
      <el-alert type="warning" :title="$t('fileBrowser.addRootWarning')" show-icon :closable="false" style="margin-top:8px" />
      <template #footer>
        <el-button @click="addRootVisible=false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!newRootPath" @click="doAddRoot">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <div v-if="error" class="browser-error">
      <el-alert type="error" :title="error" show-icon :closable="false" />
    </div>

    <template v-if="currentRoot">
      <div class="browser-toolbar">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item v-for="(seg, i) in breadcrumbs" :key="i" @click="navTo(i)">
            {{ seg.label }}
          </el-breadcrumb-item>
        </el-breadcrumb>
        <div>
          <el-button size="small" @click="switchRoot">{{ $t('fileBrowser.switchRoot') }}</el-button>
          <el-button :icon="Refresh" size="small" @click="refresh" :loading="loading">{{ $t('fileBrowser.refresh') }}</el-button>
        </div>
      </div>

      <el-table :data="entries" v-loading="loading" stripe max-height="400" @row-dblclick="onRowDblClick">
        <el-table-column :label="$t('fileBrowser.type')" width="70">
          <template #default="{ row }">
            <span>{{ row.is_dir ? '📁' : '📄' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" :label="$t('fileBrowser.name')" min-width="200" />
        <el-table-column :label="$t('fileBrowser.size')" width="120">
          <template #default="{ row }">{{ row.is_dir ? '-' : formatSize(row.size) }}</template>
        </el-table-column>
        <el-table-column prop="mod_time" :label="$t('fileBrowser.modTime')" width="180" />
        <el-table-column :label="$t('fileBrowser.select')" width="130">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="emitSelect(row)">
              {{ row.is_dir ? $t('fileBrowser.selectDirectory') : $t('fileBrowser.selectFile') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <div v-if="truncated" class="browser-truncated">
      <el-alert type="info" :closable="false" :title="$t('fileBrowser.truncated')" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Plus, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { apiClient } from '@/api/client'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{ nodeId: string; root?: string }>()
const emit = defineEmits<{ select: [entry: any] }>()

const loading = ref(false); const rootsLoading = ref(false)
const entries = ref<any[]>([]); const roots = ref<any[]>([])
const error = ref(''); const selectedRootId = ref('')
const currentRoot = ref('')
const currentRootId = ref('')
const currentPath = ref(''); const truncated = ref(false)

// Add root dialog
const addRootVisible = ref(false); const newRootPath = ref('')

const breadcrumbs = computed(() => {
  const parts: { label: string; path: string }[] = []
  parts.push({ label: currentRoot.value || '/', path: '' })
  if (currentPath.value) {
    const segs = currentPath.value.split('/').filter(Boolean)
    let acc = ''
    for (const s of segs) { acc = acc ? acc + '/' + s : s; parts.push({ label: s, path: acc }) }
  }
  return parts
})

async function fetchRoots() {
  if (!props.nodeId) return
  rootsLoading.value = true
  try {
    roots.value = await apiClient.get(`/nodes/${props.nodeId}/model-roots`)
    if (props.root) {
      const matched = roots.value.find((r) => r.path === props.root || r.root === props.root)
      if (matched) {
        currentRoot.value = matched.path
        currentRootId.value = matched.id
        selectedRootId.value = matched.id
        loadDir()
      }
    }
  } catch { roots.value = [] }
  rootsLoading.value = false
}

function showAddRoot() { newRootPath.value = ''; addRootVisible.value = true }
async function doAddRoot() {
  if (!newRootPath.value) return
  try {
    await apiClient.post(`/nodes/${props.nodeId}/model-roots`, { path: newRootPath.value })
    await fetchRoots()
    ElMessage.success(t('fileBrowser.rootAdded'))
    addRootVisible.value = false
  } catch (e: any) { ElMessage.error(e?.message || t('common.requestFailed')) }
}
async function doRemoveRoot(root: any) {
  try {
    await apiClient.delete(`/nodes/${props.nodeId}/model-roots/${root.id}`)
    await fetchRoots()
    ElMessage.success(t('fileBrowser.rootRemoved'))
  } catch (e: any) { ElMessage.error(e?.message || t('common.requestFailed')) }
}

function onRootSelected(rootId: string) {
  const root = roots.value.find((r) => r.id === rootId)
  if (!root) return
  currentRootId.value = root.id
  currentRoot.value = root.path
  currentPath.value = ''
  loadDir()
}
function switchRoot() { currentRoot.value = ''; currentRootId.value = ''; currentPath.value = ''; entries.value = []; selectedRootId.value = ''; fetchRoots() }

async function loadDir(path?: string) {
  if (!props.nodeId || !currentRoot.value) return
  loading.value = true; error.value = ''
  try {
    const params = new URLSearchParams(); params.set('root_id', currentRootId.value); params.set('path', path || currentPath.value || ''); params.set('limit', '200')
    const resp = await apiClient.get(`/nodes/${props.nodeId}/files?${params}`)
    entries.value = resp.entries || []
    truncated.value = resp.truncated || false
    if (resp.error === 'root_not_allowed') error.value = t('fileBrowser.rootNotAllowed')
    else if (resp.error === 'path traversal blocked') error.value = t('fileBrowser.pathBlocked')
    else if (resp.error) error.value = resp.error
  } catch (e: any) { error.value = e?.message || t('fileBrowser.noAccess'); entries.value = [] }
  loading.value = false
}

function navTo(index: number) {
  if (index === 0) { currentPath.value = ''; loadDir(''); return }
  const bp = breadcrumbs.value[index]; if (bp?.path) { currentPath.value = bp.path; loadDir(bp.path) }
}
function onRowDblClick(row: any) {
  if (!row.is_dir) return
  currentPath.value = currentPath.value ? currentPath.value + '/' + row.name : row.name; loadDir(currentPath.value)
}
function refresh() { loadDir() }

function emitSelect(row: any) {
  const rel = currentPath.value ? currentPath.value + '/' + row.name : row.name
  emit('select', {
    ...row,
    root_id: currentRootId.value,
    root: currentRoot.value,
    model_root: currentRoot.value,
    relative_path: rel,
    absolute_path: currentRoot.value + '/' + rel,
    path_type: row.is_dir ? 'directory' : 'file',
  })
}

function formatSize(bytes: number): string {
  if (!bytes || bytes === 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']; let i = 0; let sz = bytes
  while (sz >= 1024 && i < units.length - 1) { sz /= 1024; i++ }
  return sz.toFixed(1) + ' ' + units[i]
}

onMounted(() => { fetchRoots() })
watch(() => props.nodeId, () => { fetchRoots() })
</script>

<style scoped>
.file-browser { border: 1px solid var(--el-border-color); border-radius: 6px; padding: 12px; }
.browser-picker { padding: 12px 0; }
.picker-label { margin-bottom: 6px; font-weight: 500; color: var(--el-text-color-primary); }
.picker-row { display: flex; align-items: center; gap: 8px; }
.dynamic-roots { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
.browser-toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.browser-error { margin-bottom: 8px; }
.browser-truncated { margin-top: 8px; }
</style>
