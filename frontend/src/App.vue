<script setup lang="ts">
import { provideWorkspace } from './composables/useWorkspaceContext';
import {
} from 'lucide-vue-next';
import { usePicbedWorkspace } from './composables/usePicbedWorkspace';
import AuthView from './components/AuthView.vue';
import BootScreen from './components/BootScreen.vue';
import DeleteConfigDialog from './components/dialogs/DeleteConfigDialog.vue';
import GithubProxyDialog from './components/dialogs/GithubProxyDialog.vue';
import PasswordDialog from './components/dialogs/PasswordDialog.vue';
import OverviewMetrics from './components/workspace/OverviewMetrics.vue';
import ConvertPanel from './components/workspace/ConvertPanel.vue';
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
} = usePicbedWorkspace();

provideWorkspace({
  user, activeTab, authMode, message, error, errorNoticeKey, loading, booting, authForm, authErrors, authPasswordVisible,
  passwordDialogOpen, passwordForm, passwordErrors, passwordFieldVisible, togglePasswordFieldVisible,
  configForm, configErrors, convertForm, pasteForm, configs, records, batchFiles, deleteTarget,
  targetDropdownOpen, configTypeDropdownOpen, uploadDragActive, githubProxyDialogOpen, githubProxyEnabled, githubProxyURL,
  secretVisibility,
  isAuthed, supportedTypes, selectedFields, targetConfigs, selectedTargetConfig, totalImages, convertedCount,
  canConvertBatch, hasGithubImages, successRecords, typeLabel, fieldLabel, fieldPlaceholder, secretFieldVisible, toggleSecretField,
  statusLabel, targetConfigLabel, selectTargetConfig, selectConfigType, handleConfigTypeChange, switchAuthMode,
  clearAuthField, toggleAuthPasswordVisible, submitAuth, openPasswordDialog, closePasswordDialog, submitPasswordChange,
  logout, resetConfigForm, resetConvertForm, setActiveTab, editConfig, saveConfig, requestDeleteConfig, cancelDeleteConfig,
  confirmDeleteConfig, setDefault, handleFiles, handleFileDrop, addPastedDocument, removeBatchFile, analyzeBatch,
  convertBatch, closeGithubProxyDialog, confirmGithubProxyConvert, downloadFile, downloadAll, loadRecords,
});

</script>
<template>
  <main class="app-shell">
    <BootScreen v-if="booting" />
    <AuthView v-if="!booting && !isAuthed" />

    <section v-else-if="!booting" class="workspace">
      <WorkspaceHeader />
      <OverviewMetrics />
      <WorkspaceTabs />
      <p v-if="message" class="notice success">{{ message }}</p>
      <p v-if="error" class="notice error">{{ error }}</p>

      <ConvertPanel v-if="activeTab === 'convert'" />
      <ConfigsPanel v-if="activeTab === 'configs'" />
      <RecordsPanel v-if="activeTab === 'records'" />
    </section>
    <PasswordDialog />
    <GithubProxyDialog />
    <DeleteConfigDialog />
  </main>
</template>
