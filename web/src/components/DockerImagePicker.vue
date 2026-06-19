<template>
  <div class="image-picker">
    <div v-if="error" class="picker-error">
      <el-alert type="error" :title="error" show-icon :closable="false" />
    </div>

    <div class="picker-toolbar">
      <el-input v-model="query" :placeholder="$t('dockerImages.search')" size="small" style="width: 240px" clearable @keyup.enter="search" />
      <el-button :icon="Refresh" size="small" @click="search" :loading="loading">{{ $t('common.refresh') }}</el-button>
      <span class="picker-manual">
        <el-input v-model="manualRef" :placeholder="$t('dockerImages.manualInput')" size="small" style="width: 240px" clearable />
        <el-button size="small" type="primary" :disabled="!manualRef" @click="selectManual">
          {{ $t('dockerImages.select') }}
        </el-button>
      </span>
    </div>

    <el-alert v-if="selectedRef" class="selected-alert" type="success" :title="$t('dockerImages.selectedImage')" :description="selectedRef" show-icon :closable="false" />

    <el-table :data="images" v-loading="loading" stripe max-height="350" @row-click="selectRow" highlight-current-row :row-class-name="rowClassName">
      <el-table-column width="54">
        <template #default="{ row }"><el-icon v-if="imageRef(row) === selectedRef" color="var(--el-color-success)"><Check /></el-icon></template>
      </el-table-column>
      <el-table-column prop="repository" :label="$t('dockerImages.repository')" min-width="160" />
      <el-table-column prop="tag" :label="$t('dockerImages.tag')" width="120" />
      <el-table-column :label="$t('dockerImages.imageId')" width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ (row.image_id || '').slice(7, 19) }}</template>
      </el-table-column>
      <el-table-column prop="created_at" :label="$t('dockerImages.created')" width="160" />
      <el-table-column prop="size" :label="$t('dockerImages.size')" width="100" />
      <el-table-column :label="$t('common.actions')" width="80">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click.stop="selectRow(row)">{{ $t('dockerImages.select') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="images.length === 0 && !loading && !error" class="picker-empty">
      {{ $t('dockerImages.noImages') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Check, Refresh } from '@element-plus/icons-vue'
import { apiClient } from '@/api/client'

const props = defineProps<{ nodeId: string }>()
const emit = defineEmits<{ select: [image: any] }>()

const loading = ref(false)
const images = ref<any[]>([])
const error = ref('')
const query = ref('')
const manualRef = ref('')
const selectedRef = ref('')

async function search() {
  if (!props.nodeId) return
  loading.value = true; error.value = ''
  try {
    const params = new URLSearchParams()
    if (query.value) params.set('query', query.value)
    params.set('limit', '100')
    const resp = await apiClient.get(`/nodes/${props.nodeId}/docker-images?${params}`)
    images.value = resp.images || []
  } catch (e: any) {
    error.value = e?.message || 'agent unreachable'
    images.value = []
  } finally { loading.value = false }
}

function imageRef(row: any) {
  if (row.image_ref) return row.image_ref
  if (row.repository && row.tag) return `${row.repository}:${row.tag}`
  return row.repository || row.image_id || ''
}

function selectRow(row: any) {
  const ref = imageRef(row)
  selectedRef.value = ref
  manualRef.value = ref
  emit('select', { ...row, image_ref: ref })
}

function selectManual() {
  selectedRef.value = manualRef.value
  emit('select', { image_ref: manualRef.value })
}

function rowClassName({ row }: { row: any }) {
  return imageRef(row) === selectedRef.value ? 'selected-image-row' : ''
}

onMounted(() => { if (props.nodeId) search() })
</script>

<style scoped>
.image-picker { border: 1px solid var(--el-border-color); border-radius: 6px; padding: 12px; }
.picker-toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; }
.picker-manual { display: flex; align-items: center; gap: 4px; margin-left: auto; }
.picker-error { margin-bottom: 8px; }
.selected-alert { margin-bottom: 8px; }
.picker-empty { text-align: center; padding: 24px; color: var(--el-text-color-secondary); }
:deep(.selected-image-row) { background: var(--el-color-success-light-9); }
</style>
