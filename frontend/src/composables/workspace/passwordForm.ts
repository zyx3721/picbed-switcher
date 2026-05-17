import { reactive, ref } from 'vue';

type PasswordFormDeps = {
  showError: (text: string) => void;
  clearNotice: () => void;
};

export function useWorkspacePasswordForm({ showError, clearNotice }: PasswordFormDeps) {
  const passwordDialogOpen = ref(false);
  const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' });
  const passwordErrors = reactive({ old_password: false, new_password: false, confirm_password: false });
  const passwordVisibility = reactive({ old_password: false, new_password: false, confirm_password: false });

  function clearPasswordErrors() {
    passwordErrors.old_password = false;
    passwordErrors.new_password = false;
    passwordErrors.confirm_password = false;
  }
  function resetPasswordForm() {
    Object.assign(passwordForm, { old_password: '', new_password: '', confirm_password: '' });
    Object.assign(passwordVisibility, { old_password: false, new_password: false, confirm_password: false });
    clearPasswordErrors();
  }
  function passwordFieldVisible(field: keyof typeof passwordVisibility) {
    return passwordVisibility[field];
  }
  function togglePasswordFieldVisible(field: keyof typeof passwordVisibility) {
    passwordVisibility[field] = !passwordVisibility[field];
  }
  function openPasswordDialog() {
    resetPasswordForm();
    clearNotice();
    passwordDialogOpen.value = true;
  }
  function closePasswordDialog() {
    passwordDialogOpen.value = false;
    resetPasswordForm();
  }
  function validatePasswordForm() {
    clearPasswordErrors();
    if (passwordForm.old_password.length < 6) passwordErrors.old_password = true;
    if (passwordForm.new_password.length < 6) passwordErrors.new_password = true;
    if (passwordForm.confirm_password.length < 6) passwordErrors.confirm_password = true;
    if (passwordErrors.old_password || passwordErrors.new_password || passwordErrors.confirm_password) {
      showError('密码至少 6 个字符');
      return false;
    }
    if (passwordForm.new_password === passwordForm.old_password) {
      passwordErrors.new_password = true;
      showError('新密码不能与旧密码相同');
      return false;
    }
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      passwordErrors.confirm_password = true;
      showError('新密码与确认密码不一致');
      return false;
    }
    return true;
  }

  return {
    passwordDialogOpen,
    passwordForm,
    passwordErrors,
    passwordFieldVisible,
    togglePasswordFieldVisible,
    openPasswordDialog,
    closePasswordDialog,
    validatePasswordForm,
  };
}
