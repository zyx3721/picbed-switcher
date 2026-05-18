import { reactive, ref } from 'vue';

export type AuthMode = 'login' | 'register' | 'forgot' | 'reset';

type AuthFormDeps = {
  showError: (text: string) => void;
  clearNotice: () => void;
};

export function useWorkspaceAuthForm({ showError, clearNotice }: AuthFormDeps) {
  const authMode = ref<AuthMode>('login');
  const authForm = reactive({ username: '', password: '', email: '', token: '', new_password: '', confirm_password: '' });
  const authErrors = reactive({ username: false, password: false, email: false, new_password: false, confirm_password: false });
  const authPasswordVisible = ref(false);

  function clearAuthErrors() {
    authErrors.username = false;
    authErrors.password = false;
    authErrors.email = false;
    authErrors.new_password = false;
    authErrors.confirm_password = false;
  }
  function clearAuthForm() {
    Object.assign(authForm, { username: '', password: '', email: '', token: '', new_password: '', confirm_password: '' });
    authMode.value = 'login';
    authPasswordVisible.value = false;
    clearAuthErrors();
  }
  function clearAuthField(field: 'username' | 'password' | 'email' | 'new_password' | 'confirm_password') {
    authForm[field] = '';
    authErrors[field] = false;
  }
  function toggleAuthPasswordVisible() {
    authPasswordVisible.value = !authPasswordVisible.value;
  }
  function switchAuthMode(mode: AuthMode) {
    authMode.value = mode;
    clearAuthErrors();
    clearNotice();
  }
  function activatePasswordReset(token: string) {
    Object.assign(authForm, { username: '', password: '', email: '', token, new_password: '', confirm_password: '' });
    authMode.value = 'reset';
    authPasswordVisible.value = false;
    clearAuthErrors();
    clearNotice();
  }
  function emailValid(value: string) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
  }
  function validateAuthForm() {
    clearAuthErrors();
    const username = authForm.username.trim();
    const email = authForm.email.trim();
    const password = authForm.password;
    if (authMode.value === 'login') {
      if (!username || !password) {
        authErrors.username = !username;
        authErrors.password = !password;
        showError('用户名或密码不能为空');
        return false;
      }
      return true;
    }
    if (authMode.value === 'forgot') {
      if (!email) {
        authErrors.email = true;
        showError('邮箱不能为空');
        return false;
      }
      if (!emailValid(email)) {
        authErrors.email = true;
        showError('邮箱格式不正确');
        return false;
      }
      return true;
    }
    if (authMode.value === 'reset') {
      if (!authForm.token) {
        showError('重置链接无效，请重新申请');
        return false;
      }
      if (authForm.new_password.length < 6 || authForm.confirm_password.length < 6) {
        authErrors.new_password = authForm.new_password.length < 6;
        authErrors.confirm_password = authForm.confirm_password.length < 6;
        showError('密码至少 6 个字符');
        return false;
      }
      if (authForm.new_password !== authForm.confirm_password) {
        authErrors.confirm_password = true;
        showError('新密码与确认密码不一致');
        return false;
      }
      return true;
    }
    if (!username || !email || !password) {
      authErrors.username = !username;
      authErrors.email = !email;
      authErrors.password = !password;
      showError('用户名、邮箱或密码不能为空');
      return false;
    }
    if (username.length < 3) {
      authErrors.username = true;
      showError('用户名至少 3 个字符');
      return false;
    }
    if (!emailValid(email)) {
      authErrors.email = true;
      showError('邮箱格式不正确');
      return false;
    }
    if (password.length < 6) {
      authErrors.password = true;
      showError('密码至少 6 个字符');
      return false;
    }
    return true;
  }

  return {
    authMode,
    authForm,
    authErrors,
    authPasswordVisible,
    clearAuthForm,
    clearAuthField,
    toggleAuthPasswordVisible,
    switchAuthMode,
    activatePasswordReset,
    validateAuthForm,
  };
}
