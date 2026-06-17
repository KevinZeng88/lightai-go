<template>
  <div class="page-container">
    <h2>{{ $t('instances.title') }}</h2>
    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="200" />
      <el-table-column prop="deployment_id" :label="$t('instances.deployment')" width="200" />
      <el-table-column prop="actual_state" :label="$t('instances.state')" width="100" />
      <el-table-column prop="node_id" :label="$t('instances.node')" width="150" />
      <el-table-column prop="container_id" :label="$t('instances.container')" width="150" />
      <el-table-column prop="host_port" :label="$t('instances.port')" width="80" />
      <el-table-column prop="endpoint_url" :label="$t('instances.endpoint')" min-width="200" />
      <el-table-column :label="$t('common.actions')" width="150">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="detailVisible" :title="$t('instances.detail')" width="600px">
      <div v-if="selected">
        <p v-for="(v,k) in selected" :key="k"><strong>{{ k }}:</strong> {{ typeof v === 'object' ? JSON.stringify(v) : v }}</p>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiClient } from '@/api/client'

const loading = ref(false); const items = ref<any[]>([]); const selected = ref<any>(null); const detailVisible = ref(false)

onMounted(async () => { await refresh() })
async function refresh() { loading.value = true; try { items.value = await apiClient.get('/api/v1/model-instances') } catch (e: any) {} loading.value = false }

function showDetail(row: any) { selected.value = row; detailVisible.value = true }

// JSON used directly
</script>
