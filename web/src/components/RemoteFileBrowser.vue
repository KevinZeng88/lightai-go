<template>
  <div class="file-browser">
    <div v-if="error" class="browser-error">
      <el-alert type="error" :title="error" show-icon :closable="false" />
    </div>

    <div class="browser-toolbar">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item v-for="(seg, i) in breadcrumbs" :key="i" @click="navTo(i)">
          {{ seg.label }}
        </el-breadcrumb-item>
      </el-breadcrumb>
      <el-button :icon="Refresh" size="small" @click="refresh" :loading="loading">{{ $t('fileBrowser.refresh') }}</el-button>
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
          <el-button size="small" type="primary" @click="$emit('select', row)">
            {{ row.is_dir ? $t('fileBrowser.selectDirectory') : $t('fileBrowser.selectFile') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="truncated" class="browser-truncated">
      <el-alert type="info" :closable="false" title="Directory listing truncated (max entries reached)" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { apiClient } from '@/api/client'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  nodeId: string
  root?: string
}>()

defineEmits<{
  select: [entry: any]
}>()

const loading = ref(false)
const entries = ref<any[]>([])
const error = ref('')
const currentRoot = ref(props.root || '')
const currentPath = ref('')
const truncated = ref(false)

const breadcrumbs = computed(() => {
  const parts: { label: string; path: string }[] = []
  if (currentRoot.value) {
    parts.push({ label: currentRoot.value, path: '' })
  }
  if (currentPath.value) {
    const segs = currentPath.value.split('/').filter(Boolean)
    let acc = ''
    for (const s of segs) {
      acc = acc ? acc + '/' + s : s
      parts.push({ label: s, path: acc })
    }
  }
  return parts
})

async function loadDir(path?: string) {
  if (!props.nodeId) return
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams()
    if (currentRoot.value) params.set('root', currentRoot.value)
    params.set('path', path || currentPath.value || '')
    params.set('limit', '200')
    const resp = await apiClient.get(`/nodes/${props.nodeId}/files?${params}`)
    entries.value = resp.entries || []
    truncated.value = resp.truncated || false
    if (resp.error) error.value = resp.error
  } catch (e: any) {
    error.value = e?.message || t('fileBrowser.noAccess')
    entries.value = []
  } finally {
    loading.value = false
  }
}

function navTo(index: number) {
  if (index === 0) { currentPath.value = ''; loadDir(''); return }
  const bp = breadcrumbs.value[index]
  if (bp) { currentPath.value = bp.path; loadDir(bp.path) }
}

function onRowDblClick(row: any) {
  if (!row.is_dir) return
  currentPath.value = currentPath.value ? currentPath.value + '/' + row.name : row.name
  loadDir(currentPath.value)
}

function refresh() { loadDir() }

function formatSize(bytes: number): string {
  if (!bytes || bytes === 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let sz = bytes
  while (sz >= 1024 && i < units.length - 1) { sz /= 1024; i++ }
  return sz.toFixed(1) + ' ' + units[i]
}

watch(() => props.nodeId, () => {
  if (props.nodeId) loadDir()
}, { immediate: true })

watch(() => props.root, (r) => {
  if (r) { currentRoot.value = r; currentPath.value = ''; loadDir('') }
})
</script>

<style scoped>
.file-browser { border: 1px solid var(--el-border-color); border-radius: 6px; padding: 12px; }
.browser-toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.browser-error { margin-bottom: 8px; }
.browser-truncated { margin-top: 8px; }
</style>
