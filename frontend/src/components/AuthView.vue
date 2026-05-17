<script setup lang="ts">
import { AlertCircle, CheckCircle2, Eye, EyeOff, ImageUp, X } from 'lucide-vue-next';
import { useWorkspaceContext } from '../composables/useWorkspaceContext';

const {
  authMode,
  message,
  error,
  errorNoticeKey,
  loading,
  authForm,
  authErrors,
  authPasswordVisible,
  clearAuthField,
  toggleAuthPasswordVisible,
  submitAuth,
  switchAuthMode,
} = useWorkspaceContext();
</script>

<template>
  <div v-if="error" :key="errorNoticeKey" class="auth-toast" role="alert">
    <X :size="18" />
    <span>{{ error }}</span>
    <button class="toast-close" type="button" aria-label="关闭提示" @click="error = ''">
      <X :size="18" />
    </button>
  </div>
  <section class="auth-layout" aria-labelledby="auth-title">
    <div class="auth-brand">
      <div class="brand-mark auth-logo"><ImageUp :size="32" /></div>
      <p class="brand-title">PicBed Switcher</p>
      <p class="brand-subtitle">Markdown 图床批量转换平台</p>
    </div>
    <form class="panel auth-panel" novalidate @submit.prevent="submitAuth">
      <div class="auth-panel-heading">
        <h1 id="auth-title">{{ authMode === 'login' ? '欢迎回来' : '创建账号' }}</h1>
        <p>{{ authMode === 'login' ? '登录您的账户以继续' : '创建账户后开始管理图床配置' }}</p>
      </div>
      <label :class="{ invalid: authErrors.username }">
        <span class="field-label-text">{{ authMode === 'login' ? '用户名或邮箱' : '用户名' }}</span>
        <div class="auth-input-wrap">
          <input
            v-model.trim="authForm.username"
            :class="{ invalid: authErrors.username }"
            autocomplete="username"
            :placeholder="authMode === 'login' ? '请输入用户名或邮箱' : '请输入用户名'"
            @input="authErrors.username = false"
          />
          <button v-if="authForm.username" class="field-action" type="button" aria-label="清空用户名" @click="clearAuthField('username')">
            <X :size="18" />
          </button>
        </div>
      </label>
      <label v-if="authMode === 'register'" :class="{ invalid: authErrors.email }">
        <span class="field-label-text">邮箱</span>
        <div class="auth-input-wrap">
          <input v-model.trim="authForm.email" :class="{ invalid: authErrors.email }" autocomplete="email" inputmode="email" placeholder="请输入邮箱" type="text" @input="authErrors.email = false" />
          <button v-if="authForm.email" class="field-action" type="button" aria-label="清空邮箱" @click="clearAuthField('email')">
            <X :size="18" />
          </button>
        </div>
      </label>
      <label :class="{ invalid: authErrors.password }">
        <span class="field-label-text">密码</span>
        <div class="auth-input-wrap">
          <input v-model="authForm.password" :class="{ invalid: authErrors.password }" autocomplete="current-password" placeholder="请输入密码" :type="authPasswordVisible ? 'text' : 'password'" @input="authErrors.password = false" />
          <button class="field-action" type="button" :aria-label="authPasswordVisible ? '隐藏密码' : '显示密码'" @click="toggleAuthPasswordVisible">
            <EyeOff v-if="authPasswordVisible" :size="18" />
            <Eye v-else :size="18" />
          </button>
        </div>
      </label>
      <div v-if="error" class="auth-alert" role="alert">
        <AlertCircle :size="19" />
        <span>{{ error }}</span>
      </div>
      <button class="primary auth-submit" :disabled="loading" type="submit">
        <CheckCircle2 :size="18" />{{ loading ? '处理中...' : authMode === 'login' ? '登录' : '创建账号' }}
      </button>
      <div class="auth-mode-inline">
        <span>{{ authMode === 'login' ? '还没有账户？' : '已有账户？' }}</span>
        <button class="link-button" type="button" @click="switchAuthMode(authMode === 'login' ? 'register' : 'login')">
          {{ authMode === 'login' ? '注册' : '登录' }}
        </button>
      </div>
      <p v-if="message" class="notice success">{{ message }}</p>
    </form>
    <p class="auth-footer">© 2026 PicBed Switcher. All rights reserved.</p>
  </section>
</template>
