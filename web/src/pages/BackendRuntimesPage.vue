<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runtimes.title') }}</h2>
      <el-button @click="load">{{ $t('common.refresh') }}</el-button>
    </div>

    <el-table :data="runtimes" v-loading="loading" stripe @row-click="selected = $event">
      <el-table-column :label="$t('runtimes.name')" min-width="220">
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="backend_id" :label="$t('runtimes.backend')" min-width="160" />
      <el-table-column prop="backend_version_id" :label="$t('runtimes.backendVersion')" min-width="200" />
      <el-table-column prop="vendor" :label="$t('runtimes.vendor')" width="120" />
      <el-table-column prop="image_ref" :label="$t('runtimes.image')" min-width="260" show-overflow-tooltip />
      <el-table-column prop="deployable_count" :label="$t('runtimes.readyCount')" width="120" />
      <el-table-column :label="$t('runtimes.managedBy')" width="140">
        <template #default="{ row }">
          <el-tag :type="row.is_editable ? 'success' : 'info'">
            {{ row.is_editable ? $t('runtimes.userManaged') : $t('runtimes.systemManaged') }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.name || ''" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runtimes.backend')">{{ selected.backend_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.backendVersion')">{{ selected.backend_version_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.vendor')">{{ selected.vendor }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.image')">{{ selected.image_ref }}</el-descriptions-item>
        </el-descriptions>
        <JsonViewer :value="selected.config_set || {}" title="ConfigSet" max-height="520px" :searchable="true" />
        <JsonViewer :value="selected.source_metadata || {}" title="Source Metadata" max-height="260px" :searchable="true" />
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listRuntimes } from '@/api/runtimes'
import JsonViewer from '@/components/common/JsonViewer.vue'

const loading = ref(false)
const runtimes = ref<any[]>([])
const selected = ref<any | null>(null)
const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) selected.value = null },
})

async function load() {
  loading.value = true
  try {
    runtimes.value = await listRuntimes()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
