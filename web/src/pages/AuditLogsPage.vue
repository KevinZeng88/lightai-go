<template>
  <div class="audit-page">
    <div class="page-header">
      <h2>{{ t('audit.title') }} ({{ entries.length }})</h2>
      <div class="header-actions">
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <div class="toolbar">
      <el-select v-model="filterAction" clearable size="small" :placeholder="t('audit.filterAction')" style="width:160px">
        <el-option label="start" value="start" /><el-option label="stop" value="stop" /><el-option label="dry_run" value="dry_run" />
        <el-option label="created" value="created" /><el-option label="deleted" value="deleted" /><el-option label="updated" value="updated" />
        <el-option label="sweep" value="sweep" /><el-option label="transfer" value="transfer" />
      </el-select>
      <el-select v-model="filterType" clearable size="small" :placeholder="t('audit.filterType')" style="width:160px">
        <el-option label="model_deployment" value="model_deployment" /><el-option label="model_instance" value="model_instance" />
        <el-option label="model_artifact" value="model_artifact" /><el-option label="runtime_environment" value="runtime_environment" />
        <el-option label="run_template" value="run_template" /><el-option label="gpu_lease" value="gpu_lease" />
        <el-option label="tenant" value="tenant" /><el-option label="user" value="user" /><el-option label="node" value="node" />
      </el-select>
    </div>
    <el-alert v-if="errorMessage" type="error" :title="errorMessage" show-icon closable @close="errorMessage=''" style="margin-bottom:12px" />
    <el-table :data="entries" v-loading="loading" size="small" highlight-current-row>
      <el-table-column :label="t('audit.time')" width="160">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column prop="action" :label="t('audit.action')" width="100" />
      <el-table-column prop="entity_type" :label="t('audit.entityType')" width="140" />
      <el-table-column prop="entity_id" :label="t('audit.entityId')" width="160" show-overflow-tooltip />
      <el-table-column :label="t('audit.operator')" width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.operator_user_id ? row.operator_user_id.substring(0,12) : '-' }}</template>
      </el-table-column>
      <el-table-column prop="detail" :label="t('audit.detail')" min-width="200" show-overflow-tooltip />
      <template #empty><el-empty :description="t('audit.noData')" /></template>
    </el-table>
    <div class="pagination" style="margin-top:12px;text-align:right">
      <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total" layout="prev,next" small @current-change="loadData" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchAuditLogs, type AuditLogEntry } from '@/api/auditLogs'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const entries = ref<AuditLogEntry[]>([])
const loading = ref(false)
const total = ref(0)
const errorMessage = ref('')
const page = ref(1)
const pageSize = 50
const filterAction = ref('')
const filterType = ref('')

async function loadData() {
  loading.value = true
  try {
    const resp = await fetchAuditLogs({
      action: filterAction.value || undefined,
      entity_type: filterType.value || undefined,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    entries.value = resp.entries || []
    total.value = resp.total || 0
  } catch (e: any) { entries.value = []; errorMessage.value = e?.message || String(e) }
  finally { loading.value = false }
}
const refresh = loadData
watch([filterAction, filterType], () => { page.value = 1; loadData() })
loadData()
</script>
