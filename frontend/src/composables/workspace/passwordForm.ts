import { reactive, ref } from 'vue';

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function useWorkspacePasswordForm() {
  const profileDialogOpen = ref(false);
  const profileMode = ref<'password' | 'email'>('password');
  const profileError = ref('');
  const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' });
  const emailForm = reactive({ email: '' });
  const passwordErrors = reactive({ old_password: false, new_password: false, confirm_password: false });
  const emailErrors = reactive({ email: false });
  const passwordVisibility = reactive({ old_password: false, new_password: false, confirm_password: false });

  function clearPasswordErrors() {
    passwordErrors.old_password = false;
    passwordErrors.new_password = false;
    passwordErrors.confirm_password = false;
  }
  function clearEmailErrors() {
    emailErrors.email = false;
  }
  function setProfileError(text: string) {
    profileError.value = text;
  }
  function clearProfileError() {
    profileError.value = '';
  }
  function resetPasswordForm() {
    Object.assign(passwordForm, { old_password: '', new_password: '', confirm_password: '' });
    Object.assign(passwordVisibility, { old_password: false, new_password: false, confirm_password: false });
    clearPasswordErrors();
  }
  function resetEmailForm(email = '') {
    emailForm.email = email;
    clearEmailErrors();
  }
  function passwordFieldVisible(field: keyof typeof passwordVisibility) {
    return passwordVisibility[field];
  }
  function togglePasswordFieldVisible(field: keyof typeof passwordVisibility) {
    passwordVisibility[field] = !passwordVisibility[field];
  }
  function openProfileDialog() {
    profileMode.value = 'password';
    resetPasswordForm();
    resetEmailForm();
    clearProfileError();
    profileDialogOpen.value = true;
  }
  function closeProfileDialog() {
    profileDialogOpen.value = false;
    resetPasswordForm();
    resetEmailForm();
    clearProfileError();
  }
  function setProfileMode(mode: 'password' | 'email') {
    profileMode.value = mode;
    resetPasswordForm();
    resetEmailForm();
    clearProfileError();
  }
  function validatePasswordForm() {
    clearPasswordErrors();
    clearProfileError();
    if (passwordForm.old_password.length < 6) passwordErrors.old_password = true;
    if (passwordForm.new_password.length < 6) passwordErrors.new_password = true;
    if (passwordForm.confirm_password.length < 6) passwordErrors.confirm_password = true;
    if (passwordErrors.old_password || passwordErrors.new_password || passwordErrors.confirm_password) {
      setProfileError('密码至少 6 个字符');
      return false;
    }
    if (passwordForm.new_password === passwordForm.old_password) {
      passwordErrors.new_password = true;
      setProfileError('新密码不能与旧密码相同');
      return false;
    }
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      passwordErrors.confirm_password = true;
      setProfileError('新密码与确认密码不一致');
      return false;
    }
    return true;
  }
  function validateEmailForm(currentEmail: string) {
    clearEmailErrors();
    clearProfileError();
    const email = emailForm.email.trim().toLowerCase();
    if (!email) {
      emailErrors.email = true;
      setProfileError('邮箱不能为空');
      return false;
    }
    if (!emailPattern.test(email)) {
      emailErrors.email = true;
      setProfileError('邮箱格式不正确');
      return false;
    }
    if (email === currentEmail.trim().toLowerCase()) {
      emailErrors.email = true;
      setProfileError('新邮箱不能与当前邮箱相同');
      return false;
    }
    emailForm.email = email;
    return true;
  }

  return {
    profileDialogOpen,
    profileMode,
    profileError,
    passwordForm,
    emailForm,
    passwordErrors,
    emailErrors,
    passwordFieldVisible,
    togglePasswordFieldVisible,
    openProfileDialog,
    closeProfileDialog,
    setProfileMode,
    validatePasswordForm,
    validateEmailForm,
    setProfileError,
    clearProfileError,
  };
}