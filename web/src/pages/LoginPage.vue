<template>
  <div class="login-page">
    <el-card class="login-card">
      <h2>LightAI Go</h2>
      <el-form @submit.prevent="doLogin">
        <el-form-item>
          <el-input v-model="username" :placeholder="t('auth.username')" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="password" type="password" :placeholder="t('auth.password')" show-password />
        </el-form-item>
        <!-- P0-007: Tenant selection for multi-tenant users -->
        <el-form-item v-if="availableTenants.length > 1">
          <el-select v-model="selectedTenantId" :placeholder="t('auth.selectTenant')" style="width: 100%">
            <el-option
              v-for="t in availableTenants"
              :key="t"
              :label="t"
              :value="t"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="doLogin" :loading="loading" style="width: 100%">
            {{ t('auth.login') }}
          </el-button>
        </el-form-item>
      </el-form>
      <!-- P0-007: Show structured error messages -->
      <el-alert v-if="errorMsg" :title="errorMsg" type="error" show-icon :closable="false" />
    </el-card>
    <div class="lang-bar">
      <LanguageSwitcher />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMsg = ref('')
const availableTenants = ref<string[]>([])
const selectedTenantId = ref('')

async function doLogin() {
  loading.value = true
  errorMsg.value = ''
  availableTenants.value = []

  try {
    await auth.login(username.value, password.value, selectedTenantId.value || undefined)
    // P0-007: Check must_change_password from the response.
    if (auth.mustChangePassword) {
      router.replace('/change-password')
    } else {
      router.replace('/')
    }
  } catch (e: any) {
    // P0-007: Handle specific error cases.
    if (e instanceof ApiError) {
      if (e.status === 409 && e.data?.available_tenant_ids) {
        // Multi-tenant: user must select a tenant.
        availableTenants.value = e.data.available_tenant_ids
        errorMsg.value = t('auth.selectTenantRequired') || 'Please select a tenant.'
        return
      }
      if (e.status === 401 || e.status === 403) {
        errorMsg.value = t('auth.invalidCredentials')
        return
      }
      if (e.status === 429) {
        errorMsg.value = t('auth.tooManyRequests') || 'Too many requests. Please wait.'
        return
      }
      errorMsg.value = e.message || t('auth.invalidCredentials')
    } else {
      errorMsg.value = t('auth.networkError') || 'Network error. Please try again.'
    }
    // P0-007: Ensure not logged in on any error.
    auth.isLoggedIn = false
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f0f2f5;
}
.login-card {
  width: 380px;
}
.login-card h2 {
  text-align: center;
  margin-bottom: 24px;
}
.lang-bar {
  margin-top: 16px;
}
</style>
