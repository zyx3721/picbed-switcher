import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useWorkspaceAuthForm } from './workspace/authForm';
import { useWorkspaceConfigActions } from './workspace/configActions';
import { useWorkspaceConfigForm } from './workspace/configForm';
import { useWorkspaceConvert } from './workspace/convertWorkspace';
import { useWorkspaceNotices } from './workspace/notices';
import { useWorkspacePasswordForm } from './workspace/passwordForm';
import { createWorkspaceRequest } from './workspace/request';
import { useWorkspaceData } from './workspace/workspaceData';
import type { ConversionRecord, PicbedConfig, RequestError, User, WorkspaceTab } from './workspace/types';

export function usePicbedWorkspace() {
  const token = ref(localStorage.getItem('picbed_token') || '');
  const user = ref<User | null>(null);
  const activeTab = ref<WorkspaceTab>('convert');
  const { message, error, errorNoticeKey, showMessage, showError, clearNotice, clearErrorTimer } = useWorkspaceNotices();
  const loading = ref(false);
  const booting = ref(true);
  let bootTimer: ReturnType<typeof window.setTimeout> | undefined;

  const {
    authMode,
    authForm,
    authErrors,
    authPasswordVisible,
    clearAuthForm,
    clearAuthField,
    toggleAuthPasswordVisible,
    switchAuthMode,
    validateAuthForm,
  } = useWorkspaceAuthForm({ showError, clearNotice });
  const {
    passwordDialogOpen,
    passwordForm,
    passwordErrors,
    passwordFieldVisible,
    togglePasswordFieldVisible,
    openPasswordDialog,
    closePasswordDialog,
    validatePasswordForm,
  } = useWorkspacePasswordForm({ showError, clearNotice });
  const configs = ref<PicbedConfig[]>([]);
  const {
    typeDefs,
    configForm,
    configErrors,
    secretVisibility,
    supportedTypes,
    selectedFields,
    typeLabel,
    fieldLabel,
    fieldPlaceholder,
    secretFieldVisible,
    toggleSecretField,
    handleConfigTypeChange,
    resetConfigForm,
    editConfig,
    validateConfigForm,
    mergeTypeDefs,
  } = useWorkspaceConfigForm({ activeTab, configs, showError, clearNotice });
  const records = ref<ConversionRecord[]>([]);
  const deleteTarget = ref<PicbedConfig | null>(null);
  const configTypeDropdownOpen = ref(false);

  const isAuthed = computed(() => token.value.length > 0 && user.value !== null);
  const successRecords = computed(() => records.value.filter(item => item.status === 'success').length);

  function closeDropdowns() {
    targetDropdownOpen.value = false;
    configTypeDropdownOpen.value = false;
  }
  function handleGlobalPointerDown(event: PointerEvent) {
    const target = event.target;
    if (!(target instanceof Element)) {
      closeDropdowns();
      return;
    }
    if (!target.closest('.custom-select')) closeDropdowns();
  }
  function selectConfigType(value: string) {
    configForm.picbed_type = value;
    configTypeDropdownOpen.value = false;
    handleConfigTypeChange();
  }

  const request = createWorkspaceRequest(() => token.value);
  let reloadRecords: () => Promise<void> = async () => {};
  const {
    convertForm,
    pasteForm,
    batchFiles,
    targetDropdownOpen,
    uploadDragActive,
    githubProxyDialogOpen,
    githubProxyEnabled,
    githubProxyURL,
    targetConfigs,
    defaultTarget,
    selectedTargetConfig,
    totalImages,
    convertedCount,
    canConvertBatch,
    hasGithubImages,
    statusLabel,
    targetConfigLabel,
    selectTargetConfig,
    resetConvertForm,
    handleFiles,
    handleFileDrop,
    addPastedDocument,
    removeBatchFile,
    analyzeBatch,
    convertBatch,
    closeGithubProxyDialog,
    confirmGithubProxyConvert,
    downloadFile,
    downloadAll,
  } = useWorkspaceConvert({
    configs,
    request,
    showMessage,
    showError,
    clearNotice,
    loadRecords: () => reloadRecords(),
    typeLabel,
    loading,
  });
  const { loadWorkspaceData, loadConfigs, loadRecords } = useWorkspaceData({
    request,
    typeDefs,
    configs,
    records,
    getTargetConfigId: () => convertForm.target_config_id,
    setTargetConfigId: id => {
      convertForm.target_config_id = id;
    },
    getDefaultTarget: () => defaultTarget.value,
    mergeTypeDefs,
  });
  reloadRecords = loadRecords;
  const { saveConfig, requestDeleteConfig, cancelDeleteConfig, confirmDeleteConfig, setDefault } =
    useWorkspaceConfigActions({
      request,
      configForm,
      selectedFields,
      deleteTarget,
      loading,
      validateConfigForm,
      resetConfigForm,
      loadConfigs,
      showMessage,
      showError,
    });
  async function submitAuth() {
    if (!validateAuthForm()) return;
    loading.value = true;
    try {
      const data = await request<{ token: string; user: User }>(`/api/auth/${authMode.value}`, {
        method: 'POST',
        body: JSON.stringify(authForm),
      });
      token.value = data.token;
      user.value = data.user;
      localStorage.setItem('picbed_token', data.token);
      showMessage(authMode.value === 'login' ? '登录成功' : '注册成功');
      clearAuthForm();
      await loadWorkspaceData();
    } catch (err) {
      const text = err instanceof Error ? err.message : '认证失败';
      if (text.includes('用户名或密码')) {
        authErrors.username = true;
        authErrors.password = true;
      } else if (text.includes('用户名')) authErrors.username = true;
      if (text.includes('邮箱')) authErrors.email = true;
      showError(text);
    } finally {
      loading.value = false;
    }
  }
  function clearWorkspaceDrafts() {
    resetConfigForm();
    resetConvertForm();
    configTypeDropdownOpen.value = false;
    deleteTarget.value = null;
  }
  async function submitPasswordChange() {
    if (!validatePasswordForm()) return;
    loading.value = true;
    try {
      await request('/api/auth/password', { method: 'PUT', body: JSON.stringify(passwordForm) });
      closePasswordDialog();
      token.value = '';
      user.value = null;
      configs.value = [];
      records.value = [];
      clearWorkspaceDrafts();
      clearAuthForm();
      localStorage.removeItem('picbed_token');
      showMessage('密码已修改，请重新登录');
    } catch (err) {
      const requestError = err as RequestError;
      if (requestError.status === 401) passwordErrors.old_password = true;
      showError(requestError.status === 401 ? '旧密码不正确' : requestError.message || '密码修改失败');
    } finally {
      loading.value = false;
    }
  }
  function logout() {
    token.value = '';
    user.value = null;
    configs.value = [];
    records.value = [];
    clearWorkspaceDrafts();
    closePasswordDialog();
    clearAuthForm();
    localStorage.removeItem('picbed_token');
    showMessage('已退出登录');
  }
  async function loadProfile() {
    if (!token.value) return;
    try {
      const data = await request<{ user: User }>('/api/auth/profile');
      user.value = data.user;
      await loadWorkspaceData();
    } catch {
      logout();
    }
  }
  function setActiveTab(tab: WorkspaceTab) {
    if (activeTab.value === 'configs' || tab === 'configs') resetConfigForm();
    if (activeTab.value === 'convert' || tab === 'convert') resetConvertForm();
    activeTab.value = tab;
  }
  onMounted(() => {
    document.addEventListener('pointerdown', handleGlobalPointerDown);
    bootTimer = window.setTimeout(() => {
      booting.value = false;
      bootTimer = undefined;
    }, 1000);
    void loadProfile();
  });
  onBeforeUnmount(() => {
    clearErrorTimer();
    if (bootTimer) window.clearTimeout(bootTimer);
    document.removeEventListener('pointerdown', handleGlobalPointerDown);
  });

  return {
    user,
    activeTab,
    authMode,
    message,
    error,
    errorNoticeKey,
    loading,
    booting,
    authForm,
    authErrors,
    authPasswordVisible,
    passwordDialogOpen,
    passwordForm,
    passwordErrors,
    passwordFieldVisible,
    togglePasswordFieldVisible,
    configForm,
    configErrors,
    convertForm,
    pasteForm,
    configs,
    records,
    batchFiles,
    deleteTarget,
    targetDropdownOpen,
    configTypeDropdownOpen,
    uploadDragActive,
    githubProxyDialogOpen,
    githubProxyEnabled,
    githubProxyURL,
    secretVisibility,
    isAuthed,
    supportedTypes,
    selectedFields,
    targetConfigs,
    selectedTargetConfig,
    totalImages,
    convertedCount,
    canConvertBatch,
    hasGithubImages,
    successRecords,
    typeLabel,
    fieldLabel,
    fieldPlaceholder,
    secretFieldVisible,
    toggleSecretField,
    statusLabel,
    targetConfigLabel,
    selectTargetConfig,
    selectConfigType,
    handleConfigTypeChange,
    switchAuthMode,
    clearAuthField,
    toggleAuthPasswordVisible,
    submitAuth,
    openPasswordDialog,
    closePasswordDialog,
    submitPasswordChange,
    logout,
    resetConfigForm,
    resetConvertForm,
    setActiveTab,
    editConfig,
    saveConfig,
    requestDeleteConfig,
    cancelDeleteConfig,
    confirmDeleteConfig,
    setDefault,
    handleFiles,
    handleFileDrop,
    addPastedDocument,
    removeBatchFile,
    analyzeBatch,
    convertBatch,
    closeGithubProxyDialog,
    confirmGithubProxyConvert,
    downloadFile,
    downloadAll,
    loadRecords,
  };
}
