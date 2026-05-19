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
  const { message, error, errorNoticeKey, toast, toastNoticeKey, showMessage, showError, showToast, clearNotice, clearErrorTimer, clearToastTimer } = useWorkspaceNotices();
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
    activatePasswordReset,
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
  const recordDetail = ref<ConversionRecord | null>(null);
  const recordDetailOpen = ref(false);
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
    restoreConvertWorkspace,
    stopConvertTaskPolling,
    togglePreview,
    changedLines,
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
    restoreLocalUploadWorkspace,
    stopLocalUploadTaskPolling,
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
  const { saveConfig, requestDeleteConfig, cancelDeleteConfig, confirmDeleteConfig, setDefault, testConfig } =
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
      if (authMode.value === 'forgot') {
        const data = await request<{ message: string }>('/api/auth/password/forgot', {
          method: 'POST',
          body: JSON.stringify({ email: authForm.email }),
        });
        switchAuthMode('login');
        Object.assign(authForm, { username: '', password: '', email: '' });
        showMessage(data.message || '密码重置邮件已发送，请检查邮箱');
        return;
      }
      if (authMode.value === 'reset') {
        const data = await request<{ message: string }>('/api/auth/password/reset', {
          method: 'POST',
          body: JSON.stringify({
            token: authForm.token,
            new_password: authForm.new_password,
            confirm_password: authForm.confirm_password,
          }),
        });
        showMessage(data.message || '密码已重置，请重新登录');
        clearAuthForm();
        window.history.replaceState({}, document.title, window.location.pathname);
        return;
      }
      const data = await request<{ token: string; user: User }>(`/api/auth/${authMode.value}`, {
        method: 'POST',
        body: JSON.stringify(authForm),
      });
      token.value = data.token;
      user.value = data.user;
      localStorage.setItem('picbed_token', data.token);
      const authMessage = authMode.value === 'login' ? '登录成功' : '注册成功';
      const authNotice = data.user.email_verified === false ? authMessage + '，请完成邮箱验证，启用密码找回' : authMessage;
      showMessage(authNotice);
      if (data.user.email_verified === false) showToast(authNotice);
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
  async function verifyEmailToken(tokenValue: string) {
    loading.value = true;
    try {
      const data = await request<{ message: string }>('/api/auth/email/verify', { method: 'POST', body: JSON.stringify({ token: tokenValue }) });
      showMessage(data.message || '邮箱已验证');
      window.history.replaceState({}, document.title, window.location.pathname);
      await loadProfile();
    } catch (err) {
      showError(err instanceof Error ? err.message : '邮箱验证失败');
    } finally {
      loading.value = false;
    }
  }
  async function resendEmailVerification() {
    loading.value = true;
    try {
      const data = await request<{ message: string }>('/api/auth/email/verification', { method: 'POST' });
      const text = data.message || '验证邮件已发送，请检查邮箱';
      showMessage(text);
      showToast(text);
    } catch (err) {
      const text = err instanceof Error ? err.message : '验证邮件发送失败';
      showError(text);
      showToast(text);
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
      showMessage('邮箱已修改，请查收验证邮件');
    } catch (err) {
      const requestError = err as RequestError;
      if (requestError.status === 400 || requestError.status === 409) emailErrors.email = true;
      setProfileError(requestError.message || '邮箱修改失败');
    } finally {
      loading.value = false;
    }
  }
  function logout() {
    stopConvertTaskPolling();
    stopLocalUploadTaskPolling();
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
      restoreConvertWorkspace();
      restoreLocalUploadWorkspace();
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
  async function openRecordDetail(record: ConversionRecord) {
    loading.value = true;
    try {
      const data = await request<{ record: ConversionRecord }>(`/api/convert/records/${record.id}`);
      recordDetail.value = data.record;
      recordDetailOpen.value = true;
    } catch (err) {
      showError(err instanceof Error ? err.message : '读取转换记录详情失败');
    } finally {
      loading.value = false;
    }
  }
  function closeRecordDetail() {
    recordDetailOpen.value = false;
    recordDetail.value = null;
  }
  async function deleteRecords(ids: number[]) {
    const uniqueIds = Array.from(new Set(ids.filter(id => id > 0)));
    if (uniqueIds.length === 0) return;
    loading.value = true;
    try {
      const data = await request<{ message: string }>('/api/convert/records', {
        method: 'DELETE',
        body: JSON.stringify({ ids: uniqueIds }),
      });
      if (recordDetail.value && uniqueIds.includes(recordDetail.value.id)) closeRecordDetail();
      records.value = records.value.filter(record => !uniqueIds.includes(record.id));
      showMessage(data.message || `已删除 ${uniqueIds.length} 条转换记录`);
      await loadRecords();
    } catch (err) {
      showError(err instanceof Error ? err.message : '删除转换记录失败');
    } finally {
      loading.value = false;
    }
  }
  onMounted(() => {
    document.addEventListener('pointerdown', handleGlobalPointerDown);
    const query = new URLSearchParams(window.location.search);
    const resetToken = query.get('reset_token');
    const verifyToken = query.get('verify_email_token');
    if (resetToken) {
      token.value = '';
      user.value = null;
      localStorage.removeItem('picbed_token');
      activatePasswordReset(resetToken);
    }
    bootTimer = window.setTimeout(() => {
      booting.value = false;
      bootTimer = undefined;
    }, 1000);
    if (verifyToken) void verifyEmailToken(verifyToken);
    else if (!resetToken) void loadProfile();
  });
  onBeforeUnmount(() => {
    clearErrorTimer();
    clearToastTimer();
    if (bootTimer) window.clearTimeout(bootTimer);
    stopConvertTaskPolling();
    stopLocalUploadTaskPolling();
    document.removeEventListener('pointerdown', handleGlobalPointerDown);
  });

  return {
    user,
    activeTab,
    authMode,
    message,
    error,
    errorNoticeKey,
    toast,
    toastNoticeKey,
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
    recordDetail,
    recordDetailOpen,
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
    activatePasswordReset,
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
    testConfig,
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
    togglePreview,
    changedLines,
    downloadFile,
    downloadAll,
    downloadLocalFile,
    downloadAllLocalFiles,
    loadRecords,
    openRecordDetail,
    closeRecordDetail,
    deleteRecords,
    resendEmailVerification,
  };
}
