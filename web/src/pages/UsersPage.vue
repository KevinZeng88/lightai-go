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
    <el-dialog v-model="createVisible" :title="t('users.create')" width="400px">
      <el-form :model="createForm" label-width="100px" size="small">
        <el-form-item :label="t('users.username')" required><el-input v-model="createForm.username" /></el-form-item>
        <el-form-item :label="t('users.displayName')"><el-input v-model="createForm.display_name" /></el-form-item>
        <el-form-item :label="t('users.password')" required><el-input v-model="createForm.password" type="password" /></el-form-item>
        <el-form-item :label="t('users.isPlatformAdmin')"><el-switch v-model="createForm.is_platform_admin" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="createVisible=false">{{t('common.cancel')}}</el-button><el-button type="primary" @click="doCreate" :loading="creating">{{t('common.save')}}</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'; import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'; import { RefreshRight } from '@element-plus/icons-vue'
import { fetchUsers, createUser, type User } from '@/api/users'; import { useAuthStore } from '@/stores/auth'; import { formatDateTime } from '@/utils/format'
const { t } = useI18n(); const auth = useAuthStore()
const isPlatformAdmin = auth.user?.is_platform_admin || false
const items = ref<User[]>([]); const loading = ref(false)
async function refresh() { loading.value=true; try { items.value = await fetchUsers() } catch { items.value=[] } finally { loading.value=false } }
const createForm = ref({ username: '', display_name: '', password: '', is_platform_admin: false })
const createVisible = ref(false)
const creating = ref(false)
function openCreate() { createForm.value = { username: '', display_name: '', password: '', is_platform_admin: false }; createVisible.value = true }
async function doCreate() {
  creating.value = true
  try {
    await createUser({ username: createForm.value.username, display_name: createForm.value.display_name, password: createForm.value.password, is_platform_admin: createForm.value.is_platform_admin })
    ElMessage.success(t('users.created'))
    createVisible.value = false; refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { creating.value = false }
}
refresh()
</script>
