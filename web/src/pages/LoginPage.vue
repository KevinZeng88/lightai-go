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
        <el-form-item>
          <el-button type="primary" @click="doLogin" :loading="loading" style="width: 100%">
            {{ t('auth.login') }}
          </el-button>
        </el-form-item>
      </el-form>
      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
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
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMsg = ref('')

async function doLogin() {
  loading.value = true
  errorMsg.value = ''
  try {
    await auth.login(username.value, password.value)
    if (auth.mustChangePassword) {
      router.replace('/change-password')
    } else {
      router.replace('/')
    }
  } catch (e: any) {
    errorMsg.value = t('auth.invalidCredentials')
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
.error-msg {
  color: var(--el-color-danger);
  text-align: center;
  margin-top: 8px;
}
.lang-bar {
  margin-top: 16px;
}
</style>
