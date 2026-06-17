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

    <el-dialog v-model="versionsVisible" :title="$t('backends.versions')" width="800px">
      <el-table :data="versions" stripe>
        <el-table-column prop="version" label="Version" width="100" />
        <el-table-column prop="display_name" :label="$t('backends.displayName')" width="150" />
        <el-table-column :label="$t('backends.isDefault')" width="100">
          <template #default="{ row }">{{ row.is_default ? '✓' : '' }}</template>
        </el-table-column>
        <el-table-column prop="default_container_port" :label="$t('backends.port')" width="80" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listBackends, listBackendVersions, type InferenceBackend, type BackendVersion } from '@/api/backends'

const loading = ref(false)
const backends = ref<InferenceBackend[]>([])
const selected = ref<InferenceBackend | null>(null)
const detailVisible = ref(false)
const versionsVisible = ref(false)
const versions = ref<BackendVersion[]>([])

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

// JSON.stringify used directly via globalThis
</script>
