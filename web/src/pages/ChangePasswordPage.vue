<template>
  <div class="login-page">
    <el-card class="login-card">
      <h2>{{ t('auth.changePassword') }}</h2>
      <el-alert type="warning" :title="t('auth.forceChangePasswordHint')" :closable="false" style="margin-bottom: 16px" />
      <el-form @submit.prevent="doChange">
        <el-form-item>
          <el-input v-model="currentPassword" type="password" :placeholder="t('auth.currentPassword')" show-password @keyup.enter="doChange" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="newPassword" type="password" :placeholder="t('auth.newPassword')" show-password @keyup.enter="doChange" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="confirmPassword" type="password" :placeholder="t('auth.confirmPassword')" show-password @keyup.enter="doChange" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="doChange" :loading="loading" style="width: 100%">
            {{ t('auth.changePassword') }}
          </el-button>
        </el-form-item>
      </el-form>
      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const errorMsg = ref('')

async function doChange() {
  errorMsg.value = ''
  if (newPassword.value !== confirmPassword.value) {
    errorMsg.value = t('auth.passwordMismatch')
    return
  }
  if (newPassword.value.length < 8) {
    errorMsg.value = t('auth.passwordTooShort')
    return
  }
  loading.value = true
  try {
    await auth.changePassword(currentPassword.value, newPassword.value)
    await auth.logout()
    router.replace('/login')
  } catch (e: any) {
    errorMsg.value = t('auth.changePasswordFailed')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f0f2f5;
}
.login-card {
  width: 420px;
}
.login-card h2 {
  text-align: center;
  margin-bottom: 16px;
}
.error-msg {
  color: var(--el-color-danger);
  text-align: center;
  margin-top: 8px;
}
</style>
