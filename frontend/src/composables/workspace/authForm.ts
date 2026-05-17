import { reactive, ref } from 'vue';

type AuthFormDeps = {
  showError: (text: string) => void;
  clearNotice: () => void;
};

export function useWorkspaceAuthForm({ showError, clearNotice }: AuthFormDeps) {
  const authMode = ref<'login' | 'register'>('login');
  const authForm = reactive({ username: '', password: '', email: '' });
  const authErrors = reactive({ username: false, password: false, email: false });
  const authPasswordVisible = ref(false);

  function clearAuthErrors() {
    authErrors.username = false;
    authErrors.password = false;
    authErrors.email = false;
  }
  function clearAuthForm() {
    Object.assign(authForm, { username: '', password: '', email: '' });
    authMode.value = 'login';
    authPasswordVisible.value = false;
    clearAuthErrors();
  }
  function clearAuthField(field: 'username' | 'password' | 'email') {
    authForm[field] = '';
    authErrors[field] = false;
  }
  function toggleAuthPasswordVisible() {
    authPasswordVisible.value = !authPasswordVisible.value;
  }
  function switchAuthMode(mode: 'login' | 'register') {
    authMode.value = mode;
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
    validateAuthForm,
  };
}
