<template>
  <div class="roles-page">
    <div class="page-header"><h2>{{ t('roles.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate" v-if="isPlatformAdmin">{{ t('roles.createRole') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-alert v-if="errorMessage" type="error" :title="errorMessage" show-icon closable @close="errorMessage=''" style="margin-bottom:12px" />
    <el-table :data="items" v-loading="loading" size="small">
      <el-table-column prop="name" :label="t('roles.name')" width="160" />
      <el-table-column prop="display_name" :label="t('roles.displayName')" width="160" />
      <el-table-column :label="t('roles.builtin')" width="80"><template #default="{row}"><el-tag size="small" :type="row.built_in?'warning':'info'">{{ row.built_in ? 'Built-in' : 'Custom' }}</el-tag></template></el-table-column>
      <el-table-column prop="description" :label="t('roles.description')" min-width="200" show-overflow-tooltip />
      <el-table-column label="" width="240" fixed="right">
        <template #default="{row}">
          <el-button size="small" type="primary" link @click="openPermissions(row)">{{ t('roles.editPermissions') }}</el-button>
          <el-button size="small" type="danger" link @click="confirmDelete(row)" v-if="!row.built_in && isPlatformAdmin">{{ t('roles.delete') }}</el-button>
        </template>
      </el-table-column>
      <template #empty><el-empty :description="t('roles.noData')" /></template>
    </el-table>
    <el-dialog v-model="createVisible" :title="t('roles.createRole')" width="400px">
      <el-form :model="createForm" label-width="100px" size="small">
        <el-form-item :label="t('roles.name')" required><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item :label="t('roles.displayName')"><el-input v-model="createForm.display_name" /></el-form-item>
        <el-form-item :label="t('roles.description')"><el-input v-model="createForm.description" type="textarea" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="createVisible=false">{{t('common.cancel')}}</el-button><el-button type="primary" @click="doCreate" :loading="creating">{{t('common.save')}}</el-button></template>
    </el-dialog>
    <el-dialog v-model="permVisible" :title="t('roles.editPermissionsTitle')" width="500px">
      <div v-if="allPermissions.length === 0 && !loadingPerms">{{ t('roles.noPermissions') }}</div>
      <el-checkbox-group v-model="selectedPermIds" v-loading="loadingPerms" v-else>
        <div v-for="perm in allPermissions" :key="perm.id" style="margin: 6px 0">
          <el-checkbox :label="perm.id">{{ perm.code }}<span style="color: var(--el-text-color-secondary); margin-left: 8px; font-size: 12px">{{ perm.description }}</span></el-checkbox>
        </div>
      </el-checkbox-group>
      <template #footer><el-button @click="permVisible=false">{{t('common.cancel')}}</el-button><el-button type="primary" @click="doUpdatePermissions" :loading="savingPerms">{{t('common.save')}}</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'; import { useI18n } from 'vue-i18n'; import { RefreshRight } from '@element-plus/icons-vue'; import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchRoles, createRole, deleteRole, fetchPermissions, fetchRolePermissions, updateRolePermissions, type Role, type Permission } from '@/api/roles'; import { useAuthStore } from '@/stores/auth'
const { t } = useI18n(); const auth = useAuthStore()
const isPlatformAdmin = auth.user?.is_platform_admin || false
const items = ref<Role[]>([]); const loading = ref(false); const errorMessage = ref('')
async function refresh() { loading.value=true; errorMessage.value=''; try { items.value = await fetchRoles() } catch (e: any) { items.value=[]; errorMessage.value = e?.message || String(e) } finally { loading.value=false } }
refresh()

// Create
const createVisible = ref(false); const creating = ref(false)
const createForm = ref({ name: '', display_name: '', description: '' })
function openCreate() { createForm.value = { name: '', display_name: '', description: '' }; createVisible.value = true }
async function doCreate() {
  creating.value = true
  try {
    await createRole(createForm.value)
    ElMessage.success(t('roles.saveSuccess'))
    createVisible.value = false; refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { creating.value = false }
}

// Delete
async function confirmDelete(row: Role) {
  try {
    await ElMessageBox.confirm(t('roles.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await deleteRole(row.id)
    ElMessage.success(t('roles.deleteSuccess'))
    refresh()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || t('common.error'))
  }
}

// Permissions
const permVisible = ref(false); const savingPerms = ref(false); const loadingPerms = ref(false)
const allPermissions = ref<Permission[]>([]); const selectedPermIds = ref<string[]>([]); const permErrorMessage = ref('')
const editingRole = ref<Role | null>(null)
async function openPermissions(row: Role) {
  editingRole.value = row
  permVisible.value = true
  loadingPerms.value = true
  try {
    allPermissions.value = await fetchPermissions()
    const existing = await fetchRolePermissions(row.id)
    selectedPermIds.value = existing.map((p: any) => p.id)
  } catch (e: any) { allPermissions.value = []; permErrorMessage.value = e?.message || String(e) }
  finally { loadingPerms.value = false }
}
async function doUpdatePermissions() {
  if (!editingRole.value) return
  savingPerms.value = true
  try {
    await updateRolePermissions(editingRole.value.id, selectedPermIds.value)
    ElMessage.success(t('roles.saveSuccess'))
    permVisible.value = false
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { savingPerms.value = false }
}
</script>
