<script setup lang="ts">
import { CheckCircle2, Eye, EyeOff, KeyRound } from 'lucide-vue-next';
import { useWorkspaceContext } from '../../composables/useWorkspaceContext';

const {
  passwordDialogOpen,
  passwordForm,
  passwordErrors,
  passwordFieldVisible,
  togglePasswordFieldVisible,
  error,
  loading,
  closePasswordDialog,
  submitPasswordChange,
} = useWorkspaceContext();
</script>

<template>
  <div v-if="passwordDialogOpen" class="modal-backdrop" role="presentation" @click.self="closePasswordDialog">
    <form
      class="confirm-dialog password-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="change-password-title"
      @submit.prevent="submitPasswordChange"
    >
      <div class="dialog-icon password-dialog-icon"><KeyRound :size="22" /></div>
      <div class="dialog-copy">
        <h2 id="change-password-title">修改密码</h2>
        <p>输入旧密码后设置新的登录密码。</p>
      </div>
      <div class="password-fields">
        <label :class="{ invalid: passwordErrors.old_password }">
          <span class="field-label-text">旧密码</span>
          <div class="auth-input-wrap">
            <input
              v-model="passwordForm.old_password"
              :class="{ invalid: passwordErrors.old_password }"
              autocomplete="current-password"
              placeholder="请输入旧密码"
              :type="passwordFieldVisible('old_password') ? 'text' : 'password'"
              @input="passwordErrors.old_password = false"
            />
            <button
              class="field-action"
              type="button"
              :aria-label="passwordFieldVisible('old_password') ? '隐藏旧密码' : '显示旧密码'"
              @click="togglePasswordFieldVisible('old_password')"
            >
              <EyeOff v-if="passwordFieldVisible('old_password')" :size="18" />
              <Eye v-else :size="18" />
            </button>
          </div>
        </label>
        <label :class="{ invalid: passwordErrors.new_password }">
          <span class="field-label-text">新密码</span>
          <div class="auth-input-wrap">
            <input
              v-model="passwordForm.new_password"
              :class="{ invalid: passwordErrors.new_password }"
              autocomplete="new-password"
              placeholder="请输入新密码"
              :type="passwordFieldVisible('new_password') ? 'text' : 'password'"
              @input="passwordErrors.new_password = false"
            />
            <button
              class="field-action"
              type="button"
              :aria-label="passwordFieldVisible('new_password') ? '隐藏新密码' : '显示新密码'"
              @click="togglePasswordFieldVisible('new_password')"
            >
              <EyeOff v-if="passwordFieldVisible('new_password')" :size="18" />
              <Eye v-else :size="18" />
            </button>
          </div>
        </label>
        <label :class="{ invalid: passwordErrors.confirm_password }">
          <span class="field-label-text">确认密码</span>
          <div class="auth-input-wrap">
            <input
              v-model="passwordForm.confirm_password"
              :class="{ invalid: passwordErrors.confirm_password }"
              autocomplete="new-password"
              placeholder="请再次输入新密码"
              :type="passwordFieldVisible('confirm_password') ? 'text' : 'password'"
              @input="passwordErrors.confirm_password = false"
            />
            <button
              class="field-action"
              type="button"
              :aria-label="passwordFieldVisible('confirm_password') ? '隐藏确认密码' : '显示确认密码'"
              @click="togglePasswordFieldVisible('confirm_password')"
            >
              <EyeOff v-if="passwordFieldVisible('confirm_password')" :size="18" />
              <Eye v-else :size="18" />
            </button>
          </div>
        </label>
      </div>
      <p v-if="error" class="notice error password-dialog-notice">{{ error }}</p>
      <div class="dialog-actions">
        <button class="ghost" type="button" @click="closePasswordDialog">取消</button>
        <button class="primary" type="submit" :disabled="loading">
          <CheckCircle2 :size="17" />{{ loading ? '提交中...' : '确认修改' }}
        </button>
      </div>
    </form>
  </div>
</template>
