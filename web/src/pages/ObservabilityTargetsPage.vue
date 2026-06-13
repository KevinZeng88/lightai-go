<template>
  <div>
    <div class="page-header">
      <h2>{{ t('observability.targets') }}</h2>
      <el-button @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
    </div>

    <el-table :data="targets" v-loading="loading" size="small">
      <el-table-column :label="t('observability.targetAddress')">
        <template #default="{ row }">
          <span class="mono">{{ row.targets?.[0] || 'N/A' }}</span>
          <CopyButton :text="row.targets?.[0] || ''" v-if="row.targets?.[0]" />
        </template>
      </el-table-column>
      <el-table-column :label="t('observability.scrapePath')">
        <template #default="{ row }">{{ row.labels?.__metrics_path__ || '/metrics' }}</template>
      </el-table-column>
      <el-table-column :label="t('observability.labels')">
        <template #default="{ row }">
          <template v-for="(val, key) in row.labels" :key="key">
            <el-tag size="small" style="margin: 2px">{{ key }}: {{ val }}</el-tag>
          </template>
        </template>
      </el-table-column>
      <template #empty>{{ t('observability.noTargets') }}</template>
    </el-table>

    <el-collapse style="margin-top: 16px">
      <el-collapse-item :title="t('common.rawJson')">
        <pre class="raw-json">{{ JSON.stringify(targets, null, 2) }}</pre>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchMetricsTargets, type MetricsTarget } from '@/api/metrics'
import CopyButton from '@/components/CopyButton.vue'

const { t } = useI18n()
const targets = ref<MetricsTarget[]>([])
const loading = ref(false)

async function refresh() {
  loading.value = true
  try { targets.value = await fetchMetricsTargets() } catch { /* */ }
  loading.value = false
}

onMounted(refresh)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.page-header h2 { margin: 0; }
.mono { font-family: monospace; font-size: 12px; }
.raw-json {
  max-height: 400px;
  overflow: auto;
  font-size: 12px;
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
}
</style>
