<template>
  <div class="roles-page">
    <div class="page-header"><h2>{{ t('roles.title') }} ({{ items.length }})</h2>
      <div class="header-actions"><el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button></div>
    </div>
    <el-table :data="items" v-loading="loading" size="small">
      <el-table-column prop="name" :label="t('roles.name')" width="160" />
      <el-table-column prop="display_name" :label="t('roles.displayName')" width="160" />
      <el-table-column :label="t('roles.builtin')" width="80"><template #default="{row}"><el-tag size="small" :type="row.built_in?'warning':'info'">{{ row.built_in ? 'Built-in' : 'Custom' }}</el-tag></template></el-table-column>
      <el-table-column prop="description" :label="t('roles.description')" min-width="200" show-overflow-tooltip />
      <template #empty><el-empty :description="t('roles.noData')" /></template>
    </el-table>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'; import { useI18n } from 'vue-i18n'; import { RefreshRight } from '@element-plus/icons-vue'
import { fetchRoles, type Role } from '@/api/roles'
const { t } = useI18n()
const items = ref<Role[]>([]); const loading = ref(false)
async function refresh() { loading.value=true; try { items.value = await fetchRoles() } catch { items.value=[] } finally { loading.value=false } }
refresh()
</script>
