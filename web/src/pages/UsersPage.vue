<template>
  <div class="users-page">
    <div class="page-header"><h2>{{ t('users.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate" v-if="isPlatformAdmin">{{ t('users.create') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" size="small">
      <el-table-column prop="username" :label="t('users.username')" min-width="120" />
      <el-table-column prop="display_name" :label="t('users.displayName')" width="140" />
      <el-table-column :label="t('users.status')" width="90"><template #default="{row}"><el-tag size="small" :type="row.status==='active'?'success':'danger'">{{ row.status }}</el-tag></template></el-table-column>
      <el-table-column :label="t('users.platformAdmin')" width="110"><template #default="{row}">{{ row.is_platform_admin ? 'Yes' : '-' }}</template></el-table-column>
      <el-table-column :label="t('users.createdAt')" width="160"><template #default="{row}">{{ formatDateTime(row.created_at) }}</template></el-table-column>
      <template #empty><el-empty :description="t('users.noData')" /></template>
    </el-table>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'; import { useI18n } from 'vue-i18n'; import { RefreshRight } from '@element-plus/icons-vue'
import { fetchUsers, type User } from '@/api/users'; import { useAuthStore } from '@/stores/auth'; import { formatDateTime } from '@/utils/format'
const { t } = useI18n(); const auth = useAuthStore()
const isPlatformAdmin = auth.user?.is_platform_admin || false
const items = ref<User[]>([]); const loading = ref(false)
async function refresh() { loading.value=true; try { items.value = await fetchUsers() } catch { items.value=[] } finally { loading.value=false } }
function openCreate() {}
refresh()
</script>
