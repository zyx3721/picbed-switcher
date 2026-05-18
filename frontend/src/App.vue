<script setup lang="ts">
import { provideWorkspace } from './composables/useWorkspaceContext';
import { X } from 'lucide-vue-next';
import { usePicbedWorkspace } from './composables/usePicbedWorkspace';
import AuthView from './components/AuthView.vue';
import BootScreen from './components/BootScreen.vue';
import DeleteConfigDialog from './components/dialogs/DeleteConfigDialog.vue';
import GithubProxyDialog from './components/dialogs/GithubProxyDialog.vue';
import PasswordDialog from './components/dialogs/PasswordDialog.vue';
import TaskProgressDialog from './components/dialogs/TaskProgressDialog.vue';
import OverviewMetrics from './components/workspace/OverviewMetrics.vue';
import ConvertPanel from './components/workspace/ConvertPanel.vue';
import LocalUploadPanel from './components/workspace/LocalUploadPanel.vue';
import ConfigsPanel from './components/workspace/ConfigsPanel.vue';
import RecordsPanel from './components/workspace/RecordsPanel.vue';
import WorkspaceHeader from './components/workspace/WorkspaceHeader.vue';
import WorkspaceTabs from './components/workspace/WorkspaceTabs.vue';

const {
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
  openProfileDialog,
  closeProfileDialog,
  setProfileMode,
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
  resendEmailVerification,
} = usePicbedWorkspace();

provideWorkspace({
  user, activeTab, authMode, message, error, errorNoticeKey, toast, toastNoticeKey, loading, booting, authForm, authErrors, authPasswordVisible,
  taskProgress,
  profileDialogOpen, profileMode, profileError, passwordForm, emailForm, passwordErrors, emailErrors, passwordFieldVisible, togglePasswordFieldVisible,
  closeTaskProgress,
  configForm, configErrors, convertForm, pasteForm, configs, records, recordDetail, recordDetailOpen, batchFiles, deleteTarget,
  targetDropdownOpen, configTypeDropdownOpen, uploadDragActive, githubProxyDialogOpen, githubProxyEnabled, githubProxyURL,
  localTargetConfigId, localTargetDropdownOpen, localDocumentDragActive, localImageDragActive, localDocuments, localImages,
  secretVisibility,
  isAuthed, supportedTypes, selectedFields, targetConfigs, selectedTargetConfig, totalImages, convertedCount,
  canConvertBatch, hasGithubImages, localTargetConfigs, selectedLocalTargetConfig, localMatchedCount, localMissingCount,
  localConvertedCount, canUploadLocalBatch, successRecords, typeLabel, fieldLabel, fieldPlaceholder, secretFieldVisible, toggleSecretField,
  statusLabel, targetConfigLabel, localTargetConfigLabel, selectTargetConfig, selectLocalTargetConfig, selectConfigType, handleConfigTypeChange, switchAuthMode, activatePasswordReset,
  clearAuthField, toggleAuthPasswordVisible, submitAuth, openProfileDialog, closeProfileDialog, setProfileMode, submitPasswordChange, submitEmailChange,
  logout, resetConfigForm, resetConvertForm, setActiveTab, editConfig, saveConfig, testConfig, requestDeleteConfig, cancelDeleteConfig,
  confirmDeleteConfig, setDefault, handleFiles, handleFileDrop, addPastedDocument, removeBatchFile, localStatusLabel,
  handleLocalDocumentFiles, handleLocalDocumentDrop, handleLocalImageFiles, handleLocalImageDrop, removeLocalDocument, removeLocalImage,
  analyzeBatch, convertBatch, analyzeLocalBatch, uploadLocalBatch, closeGithubProxyDialog, confirmGithubProxyConvert,
  togglePreview, changedLines, downloadFile, downloadAll, downloadLocalFile, downloadAllLocalFiles, loadRecords,
  openRecordDetail, closeRecordDetail, resendEmailVerification,
});

</script>
<template>
  <main class="app-shell">
    <BootScreen v-if="booting" />
    <AuthView v-if="!booting && !isAuthed" />
    <div v-if="toast && isAuthed" :key="toastNoticeKey" class="auth-toast workspace-toast" role="alert">
      <X :size="18" />
      <span>{{ toast }}</span>
      <button class="toast-close" type="button" aria-label="关闭提示" @click="toast = ''">
        <X :size="18" />
      </button>
    </div>

    <section v-if="!booting && isAuthed" class="workspace">
      <WorkspaceHeader />
      <OverviewMetrics />
      <WorkspaceTabs />
      <p v-if="message" class="notice success">{{ message }}</p>
      <p v-if="error && !profileDialogOpen" class="notice error">{{ error }}</p>

      <ConvertPanel v-if="activeTab === 'convert'" />
      <LocalUploadPanel v-if="activeTab === 'localUpload'" />
      <ConfigsPanel v-if="activeTab === 'configs'" />
      <RecordsPanel v-if="activeTab === 'records'" />
    </section>
    <PasswordDialog />
    <GithubProxyDialog />
    <TaskProgressDialog />
    <DeleteConfigDialog />
  </main>
</template>
