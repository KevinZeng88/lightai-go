<template>
  <div class="tenants-page">
    <div class="page-header"><h2>{{ t('tenants.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate" v-if="isPlatformAdmin">{{ t('tenants.create') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-alert v-if="errorMessage" type="error" :title="errorMessage" show-icon closable @close="errorMessage=''" style="margin-bottom:12px" />
    <el-table :data="items" v-loading="loading" size="small">
      <el-table-column prop="name" :label="t('tenants.name')" min-width="140" />
      <el-table-column prop="slug" :label="t('tenants.slug')" width="120" />
      <el-table-column :label="t('tenants.type')" width="120"><template #default="{row}"><el-tag size="small" :type="row.type==='infrastructure'?'warning':'info'">{{ row.type }}</el-tag></template></el-table-column>
      <el-table-column prop="status" :label="t('tenants.status')" width="90" />
      <el-table-column :label="t('tenants.createdAt')" width="160"><template #default="{row}">{{ formatDateTime(row.created_at) }}</template></el-table-column>
      <el-table-column label="" width="200" fixed="right" v-if="isPlatformAdmin">
        <template #default="{row}">
          <el-button size="small" type="primary" link @click="openEdit(row)">{{ t('tenants.edit') }}</el-button>
          <el-button v-if="row.status==='active'" size="small" type="danger" link @click="confirmToggleStatus(row, 'disable')">{{ t('tenants.disable') }}</el-button>
          <el-button v-if="row.status==='disabled'" size="small" type="success" link @click="confirmToggleStatus(row, 'enable')">{{ t('tenants.enable') }}</el-button>
        </template>
      </el-table-column>
      <template #empty><el-empty :description="t('tenants.noData')" /></template>
    </el-table>
    <el-dialog v-model="dialogVisible" :title="t('tenants.create')" width="400px"><el-form :model="form" label-width="80px" size="small">
      <el-form-item :label="t('tenants.name')" required><el-input v-model="form.name" /></el-form-item>
      <el-form-item :label="t('tenants.slug')"><el-input v-model="form.slug" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="dialogVisible=false">{{t('common.cancel')}}</el-button><el-button type="primary" @click="save" :loading="saving">{{t('common.save')}}</el-button></template></el-dialog>
    <el-dialog v-model="editVisible" :title="t('tenants.editTenant')" width="400px"><el-form :model="editForm" label-width="80px" size="small">
      <el-form-item :label="t('tenants.name')" required><el-input v-model="editForm.name" /></el-form-item>
      <el-form-item :label="t('tenants.slug')"><el-input v-model="editForm.slug" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="editVisible=false">{{t('common.cancel')}}</el-button><el-button type="primary" @click="doUpdate" :loading="updating">{{t('common.save')}}</el-button></template></el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'; import { useI18n } from 'vue-i18n'; import { RefreshRight } from '@element-plus/icons-vue'; import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchTenants, createTenant, updateTenant, disableTenant, type Tenant } from '@/api/tenants'; import { useAuthStore } from '@/stores/auth'; import { formatDateTime } from '@/utils/format'
const { t } = useI18n(); const auth = useAuthStore()
const isPlatformAdmin = auth.user?.is_platform_admin || false
const items = ref<Tenant[]>([]); const loading = ref(false); const dialogVisible = ref(false); const saving = ref(false); const errorMessage = ref('')
const form = ref({ name: '', slug: '' })
async function refresh() { loading.value=true; errorMessage.value=''; try { items.value = await fetchTenants() } catch (e: any) { items.value=[]; errorMessage.value = e?.message || String(e) } finally { loading.value=false } }
function openCreate() { form.value={name:'',slug:''}; dialogVisible.value=true }
async function save() { saving.value=true; try { await createTenant(form.value); ElMessage.success('Created'); dialogVisible.value=false; refresh() } catch(e:any) { ElMessage.error(e?.message||'Error') } finally { saving.value=false } }
refresh()

// Edit
const editVisible = ref(false); const updating = ref(false)
const editForm = ref({ name: '', slug: '' })
const editingId = ref('')
function openEdit(row: Tenant) { editingId.value = row.id; editForm.value = { name: row.name, slug: row.slug }; editVisible.value = true }
async function doUpdate() {
  updating.value = true
  try {
    await updateTenant(editingId.value, editForm.value)
    ElMessage.success(t('common.save'))
    editVisible.value = false; refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { updating.value = false }
}

// Toggle status
async function confirmToggleStatus(row: Tenant, action: 'disable' | 'enable') {
  const key = action === 'disable' ? 'tenants.disableConfirm' : 'tenants.enableConfirm'
  try {
    await ElMessageBox.confirm(t(key), t('common.confirm'), { type: 'warning' })
    if (action === 'disable') {
      await disableTenant(row.id)
      ElMessage.success(t('tenants.disableSuccess'))
    } else {
      await updateTenant(row.id, { status: 'active' })
      ElMessage.success(t('tenants.enableSuccess'))
    }
    refresh()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || t('common.error'))
  }
}
</script>
