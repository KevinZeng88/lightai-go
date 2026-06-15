<template>
  <div class="tenants-page">
    <div class="page-header"><h2>{{ t('tenants.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate" v-if="isPlatformAdmin">{{ t('tenants.create') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" size="small">
      <el-table-column prop="name" :label="t('tenants.name')" min-width="140" />
      <el-table-column prop="slug" :label="t('tenants.slug')" width="120" />
      <el-table-column :label="t('tenants.type')" width="120"><template #default="{row}"><el-tag size="small" :type="row.type==='infrastructure'?'warning':'info'">{{ row.type }}</el-tag></template></el-table-column>
      <el-table-column prop="status" :label="t('tenants.status')" width="90" />
      <el-table-column :label="t('tenants.createdAt')" width="160"><template #default="{row}">{{ formatDateTime(row.created_at) }}</template></el-table-column>
      <template #empty><el-empty :description="t('tenants.noData')" /></template>
    </el-table>
    <el-dialog v-model="dialogVisible" :title="t('tenants.create')" width="400px"><el-form :model="form" label-width="80px" size="small">
      <el-form-item :label="t('tenants.name')" required><el-input v-model="form.name" /></el-form-item>
      <el-form-item :label="t('tenants.slug')"><el-input v-model="form.slug" /></el-form-item>
    </el-form>
    <template #footer><el-button @click="dialogVisible=false">{{t('common.cancel')}}</el-button><el-button type="primary" @click="save" :loading="saving">{{t('common.save')}}</el-button></template></el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'; import { useI18n } from 'vue-i18n'; import { RefreshRight } from '@element-plus/icons-vue'; import { ElMessage } from 'element-plus'
import { fetchTenants, createTenant, type Tenant } from '@/api/tenants'; import { useAuthStore } from '@/stores/auth'; import { formatDateTime } from '@/utils/format'
const { t } = useI18n(); const auth = useAuthStore()
const isPlatformAdmin = auth.user?.is_platform_admin || false
const items = ref<Tenant[]>([]); const loading = ref(false); const dialogVisible = ref(false); const saving = ref(false)
const form = ref({ name: '', slug: '' })
async function refresh() { loading.value=true; try { items.value = await fetchTenants() } catch { items.value=[] } finally { loading.value=false } }
function openCreate() { form.value={name:'',slug:''}; dialogVisible.value=true }
async function save() { saving.value=true; try { await createTenant(form.value); ElMessage.success('Created'); dialogVisible.value=false; refresh() } catch(e:any) { ElMessage.error(e?.message||'Error') } finally { saving.value=false } }
refresh()
</script>
