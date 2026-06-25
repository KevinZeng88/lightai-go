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

    <el-drawer v-model="detailVisible" :title="selected?.display_name || selected?.name || ''" size="60%">
      <JsonViewer v-if="selected" :value="selected.config_set || {}" title="ConfigSet" max-height="520px" :searchable="true" />
      <JsonViewer v-if="selected" :value="selected.source_metadata || {}" title="Source Metadata" max-height="240px" :searchable="true" />
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'
import JsonViewer from '@/components/common/JsonViewer.vue'

const loading = ref(false)
const backends = ref<any[]>([])
const selected = ref<any | null>(null)
const detailVisible = computed({
  get: () => !!selected.value,
  set: (value: boolean) => { if (!value) selected.value = null },
})

async function load() {
  loading.value = true
  try {
    backends.value = await apiClient.get('/backends')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
