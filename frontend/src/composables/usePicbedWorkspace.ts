import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useWorkspaceAuthForm } from './workspace/authForm';
import { useWorkspaceConfigActions } from './workspace/configActions';
import { useWorkspaceConfigForm } from './workspace/configForm';
import { useWorkspaceConvert } from './workspace/convertWorkspace';
import { useWorkspaceLocalUpload } from './workspace/localUploadWorkspace';
import { useWorkspaceNotices } from './workspace/notices';
import { useWorkspacePasswordForm } from './workspace/passwordForm';
import { createWorkspaceRequest } from './workspace/request';
import { useTaskProgress } from './workspace/taskProgress';
import { useWorkspaceData } from './workspace/workspaceData';
import type { ConversionRecord, PicbedConfig, RequestError, User, WorkspaceTab } from './workspace/types';

export function usePicbedWorkspace() {
  const token = ref(localStorage.getItem('picbed_token') || '');
  const user = ref<User | null>(null);
  const activeTab = ref<WorkspaceTab>('convert');
  const { message, error, errorNoticeKey, showMessage, showError, clearNotice, clearErrorTimer } = useWorkspaceNotices();
  const loading = ref(false);
  const booting = ref(true);
  const { taskProgress, startTaskProgress, updateTaskProgress, finishTaskProgress, closeTaskProgress } = useTaskProgress();
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
  } = useWorkspacePasswordForm();
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
    localTargetDropdownOpen.value = false;
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
    startTaskProgress,
    updateTaskProgress,
    finishTaskProgress,
  });
  const {
    localTargetConfigId,
    localTargetDropdownOpen,
    localDocumentDragActive,
    localImageDragActive,
    localDocuments,
    localImages,
    localTargetConfigs,
    selectedLocalTargetConfig,
    localMatchedCount,
    localMissingCount,
    localConvertedCount,
    canUploadLocalBatch,
    localTargetConfigLabel,
    selectLocalTargetConfig,
    localStatusLabel,
    resetLocalUploadForm,
    handleLocalDocumentFiles,
    handleLocalDocumentDrop,
    handleLocalImageFiles,
    handleLocalImageDrop,
    removeLocalDocument,
    removeLocalImage,
    analyzeLocalBatch,
    uploadLocalBatch,
    downloadLocalFile,
    downloadAllLocalFiles,
  } = useWorkspaceLocalUpload({
    configs,
    request,
    showMessage,
    showError,
    clearNotice,
    loadRecords: () => reloadRecords(),
    typeLabel,
    loading,
    startTaskProgress,
    updateTaskProgress,
    finishTaskProgress,
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
    resetLocalUploadForm();
    configTypeDropdownOpen.value = false;
    deleteTarget.value = null;
  }
  function openProfileDialogForUser() {
    clearNotice();
    openProfileDialog();
  }
  function setProfileModeForUser(mode: 'password' | 'email') {
    setProfileMode(mode);
  }
  async function submitPasswordChange() {
    if (!validatePasswordForm()) return;
    loading.value = true;
    try {
      await request('/api/auth/password', { method: 'PUT', body: JSON.stringify(passwordForm) });
      closeProfileDialog();
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
      setProfileError(requestError.status === 401 ? '旧密码不正确' : requestError.message || '密码修改失败');
    } finally {
      loading.value = false;
    }
  }
  async function submitEmailChange() {
    if (!user.value || !validateEmailForm(user.value.email || '')) return;
    loading.value = true;
    try {
      const data = await request<{ user: User; message: string }>('/api/auth/email', {
        method: 'PUT',
        body: JSON.stringify(emailForm),
      });
      user.value = data.user;
      closeProfileDialog();
      showMessage('邮箱已修改');
    } catch (err) {
      const requestError = err as RequestError;
      if (requestError.status === 400 || requestError.status === 409) emailErrors.email = true;
      setProfileError(requestError.message || '邮箱修改失败');
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
    closeProfileDialog();
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
    if (activeTab.value === 'localUpload' || tab === 'localUpload') resetLocalUploadForm();
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
    taskProgress,
    authForm,
    authErrors,
    authPasswordVisible,
    profileDialogOpen,
    profileMode,
    profileError,
    passwordForm,
    emailForm,
    passwordErrors,
    emailErrors,
    passwordFieldVisible,
    togglePasswordFieldVisible,
    closeTaskProgress,
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
    localTargetConfigId,
    localTargetDropdownOpen,
    localDocumentDragActive,
    localImageDragActive,
    localDocuments,
    localImages,
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
    localTargetConfigs,
    selectedLocalTargetConfig,
    localMatchedCount,
    localMissingCount,
    localConvertedCount,
    canUploadLocalBatch,
    successRecords,
    typeLabel,
    fieldLabel,
    fieldPlaceholder,
    secretFieldVisible,
    toggleSecretField,
    statusLabel,
    targetConfigLabel,
    localTargetConfigLabel,
    selectTargetConfig,
    selectLocalTargetConfig,
    selectConfigType,
    handleConfigTypeChange,
    switchAuthMode,
    clearAuthField,
    toggleAuthPasswordVisible,
    submitAuth,
    openProfileDialog: openProfileDialogForUser,
    closeProfileDialog,
    setProfileMode: setProfileModeForUser,
    submitPasswordChange,
    submitEmailChange,
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
    localStatusLabel,
    handleLocalDocumentFiles,
    handleLocalDocumentDrop,
    handleLocalImageFiles,
    handleLocalImageDrop,
    removeLocalDocument,
    removeLocalImage,
    analyzeBatch,
    convertBatch,
    analyzeLocalBatch,
    uploadLocalBatch,
    closeGithubProxyDialog,
    confirmGithubProxyConvert,
    downloadFile,
    downloadAll,
    downloadLocalFile,
    downloadAllLocalFiles,
    loadRecords,
  };
}
