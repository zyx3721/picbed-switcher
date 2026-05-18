import { computed, reactive, ref, type Ref } from 'vue';
import { createClientId } from './api';
import { isGithubImageURL, normalizeProxyURL, withGithubProxy } from './imageProxy';
import type { BatchFile, MarkdownImage, PicbedConfig } from './types';

type WorkspaceRequest = <T>(path: string, options?: RequestInit) => Promise<T>;

type ConvertWorkspaceDeps = {
  configs: Ref<PicbedConfig[]>;
  request: WorkspaceRequest;
  showMessage: (text: string) => void;
  showError: (text: string) => void;
  clearNotice: () => void;
  loadRecords: () => Promise<void>;
  typeLabel: (value: string) => string;
  loading: Ref<boolean>;
  startTaskProgress: (input: { title: string; message: string; total: number; detail?: string }) => void;
  updateTaskProgress: (input: { message?: string; detail?: string; current?: number; success?: number; failed?: number }) => void;
  finishTaskProgress: (input: { status: 'success' | 'failed'; message: string; detail?: string }) => void;
};

export function useWorkspaceConvert({
  configs,
  request,
  showMessage,
  showError,
  clearNotice,
  loadRecords,
  typeLabel,
  loading,
  startTaskProgress,
  updateTaskProgress,
  finishTaskProgress,
}: ConvertWorkspaceDeps) {
  const convertForm = reactive({ target_config_id: 0 });
  const pasteForm = reactive({ filename: 'pasted.md', content: '' });
  const batchFiles = ref<BatchFile[]>([]);
  const targetDropdownOpen = ref(false);
  const uploadDragActive = ref(false);
  const githubProxyDialogOpen = ref(false);
  const githubProxyEnabled = ref(true);
  const githubProxyURL = ref('https://gh-proxy.com/');

  const targetConfigs = computed(() => configs.value);
  const defaultTarget = computed(() => targetConfigs.value.find(item => item.is_default) || targetConfigs.value[0]);
  const selectedTargetConfig = computed(() =>
    targetConfigs.value.find(item => item.id === convertForm.target_config_id)
  );
  const totalImages = computed(() => batchFiles.value.reduce((sum, item) => sum + item.images.length, 0));
  const convertedCount = computed(() => batchFiles.value.filter(item => item.status === 'success').length);
  const isBatchAnalyzed = computed(
    () => batchFiles.value.length > 0 && batchFiles.value.every(item => item.status === 'analyzed')
  );
  const canConvertBatch = computed(() => isBatchAnalyzed.value && totalImages.value > 0);
  const hasGithubImages = computed(() =>
    batchFiles.value.some(file =>
      file.status === 'analyzed' && file.images.some(image => image.picbed === 'github' || isGithubImageURL(image.url))
    )
  );

  function statusLabel(status: BatchFile['status']) {
    return { ready: '待识别', analyzed: '已识别', success: '已转换', failed: '失败' }[status];
  }
  function targetConfigLabel(config?: PicbedConfig) {
    return config ? `${config.config_name} · ${typeLabel(config.picbed_type)}` : '请选择';
  }
  function selectTargetConfig(id: number) {
    convertForm.target_config_id = id;
    targetDropdownOpen.value = false;
  }

  function resetConvertForm() {
    clearNotice();
    convertForm.target_config_id = defaultTarget.value?.id || 0;
    targetDropdownOpen.value = false;
    pasteForm.filename = 'pasted.md';
    pasteForm.content = '';
    batchFiles.value = [];
    closeGithubProxyDialog();
  }

  async function addMarkdownFiles(fileList: FileList | File[]) {
    const files = Array.from(fileList).filter(file => file.name.toLowerCase().endsWith('.md'));
    if (!files.length) {
      showError('请上传或拖动 .md 文件');
      return;
    }
    const loaded = await Promise.all(
      files.map(async file => ({
        id: `${file.name}-${file.size}-${createClientId()}`,
        filename: file.name,
        content: await file.text(),
        images: [],
        convertedContent: '',
        changed: 0,
        status: 'ready' as const,
        error: '',
      }))
    );
    batchFiles.value = [...batchFiles.value, ...loaded];
    showMessage(`已加入 ${loaded.length} 个 Markdown 文件`);
  }
  async function handleFiles(event: Event) {
    const input = event.target as HTMLInputElement;
    await addMarkdownFiles(input.files || []);
    input.value = '';
  }
  async function handleFileDrop(event: DragEvent) {
    uploadDragActive.value = false;
    await addMarkdownFiles(event.dataTransfer?.files || []);
  }
  function addPastedDocument() {
    if (!pasteForm.content.trim()) {
      showError('请先粘贴 Markdown 内容');
      return;
    }
    batchFiles.value.push({
      id: `paste-${createClientId()}`,
      filename: pasteForm.filename || 'pasted.md',
      content: pasteForm.content,
      images: [],
      convertedContent: '',
      changed: 0,
      status: 'ready',
      error: '',
    });
    pasteForm.content = '';
    showMessage('已加入粘贴文档');
  }
  function removeBatchFile(id: string) {
    batchFiles.value = batchFiles.value.filter(item => item.id !== id);
  }
  function closeGithubProxyDialog() {
    githubProxyDialogOpen.value = false;
    githubProxyEnabled.value = true;
    githubProxyURL.value = 'https://gh-proxy.com/';
  }
  function confirmGithubProxyConvert() {
    void runConvertBatch(githubProxyEnabled.value ? normalizeProxyURL(githubProxyURL.value) : '');
  }
  async function analyzeBatch() {
    if (!batchFiles.value.length) {
      showError('请先添加 Markdown 文件');
      return;
    }
    loading.value = true;
    try {
      for (const file of batchFiles.value) {
        const data = await request<{ images: MarkdownImage[]; total: number }>('/api/convert/analyze', {
          method: 'POST',
          body: JSON.stringify({ content: file.content }),
        });
        file.images = data.images;
        file.status = 'analyzed';
      }
      showMessage(`已完成识别，共 ${totalImages.value} 个图片地址`);
    } catch (err) {
      showError(err instanceof Error ? err.message : '批量识别失败');
    } finally {
      loading.value = false;
    }
  }
  async function convertBatch() {
    if (!convertForm.target_config_id) {
      showError('请先选择目标图床配置');
      return;
    }
    if (!batchFiles.value.length) {
      showError('请先添加 Markdown 文件');
      return;
    }
    if (!isBatchAnalyzed.value) {
      showError('\u8bf7\u5148\u70b9\u51fb\u6279\u91cf\u8bc6\u522b');
      return;
    }
    if (!totalImages.value) {
      showError('\u672a\u8bc6\u522b\u5230\u56fe\u7247\uff0c\u65e0\u9700\u8f6c\u6362');
      return;
    }
    if (hasGithubImages.value) {
      githubProxyDialogOpen.value = true;
      githubProxyEnabled.value = true;
      githubProxyURL.value = githubProxyURL.value.trim() || 'https://gh-proxy.com/';
      return;
    }
    await runConvertBatch('');
  }
  async function runConvertBatch(githubProxyURLForConvert: string) {
    loading.value = true;
    githubProxyDialogOpen.value = false;
    startTaskProgress({
      title: '批量转换处理中',
      message: '正在准备批量转换',
      detail: `共 ${batchFiles.value.length} 个文档，${totalImages.value} 个图片地址。`,
      total: batchFiles.value.length,
    });
    try {
      let success = 0;
      let failed = 0;
      for (const [index, file] of batchFiles.value.entries()) {
        updateTaskProgress({
          current: index + 1,
          success,
          failed,
          message: `正在转换第 ${index + 1} / ${batchFiles.value.length} 个文档`,
          detail: `${file.filename} · ${file.images.length} 个图片地址。`,
        });
        try {
          const result = await request<{ filename: string; content?: string; changed?: number; status: string; error?: string }>(
            '/api/convert/process',
            {
              method: 'POST',
              body: JSON.stringify({
                target_config_id: convertForm.target_config_id,
                filename: file.filename,
                content: githubProxyURLForConvert ? withGithubProxy(file.content, githubProxyURLForConvert) : file.content,
              }),
            }
          );
          file.status = result.status === 'success' ? 'success' : 'failed';
          file.convertedContent = result.content || '';
          file.changed = result.changed || 0;
          file.error = result.error || '';
        } catch (err) {
          file.status = 'failed';
          file.convertedContent = '';
          file.changed = 0;
          file.error = err instanceof Error ? err.message : '转换失败';
        }
        if (file.status === 'success') success += 1;
        else failed += 1;
        updateTaskProgress({ current: index + 1, success, failed });
      }
      const finalStatus = failed > 0 ? 'failed' : 'success';
      finishTaskProgress({
        status: finalStatus,
        message: `批量转换完成，成功 ${success} 个，失败 ${failed} 个`,
        detail: '可以关闭此窗口并下载已转换的文档。',
      });
      showMessage(`批量转换完成，成功 ${convertedCount.value} 个文件`);
      await loadRecords();
    } catch (err) {
      finishTaskProgress({
        status: 'failed',
        message: '批量转换失败',
        detail: err instanceof Error ? err.message : '批量转换失败',
      });
      showError(err instanceof Error ? err.message : '批量转换失败');
      await loadRecords();
    } finally {
      loading.value = false;
    }
  }
  function downloadFile(file: BatchFile) {
    if (!file.convertedContent) return;
    const blob = new Blob([file.convertedContent], { type: 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = file.filename || 'converted.md';
    link.click();
    URL.revokeObjectURL(url);
  }
  function downloadAll() {
    batchFiles.value.filter(file => file.convertedContent).forEach(downloadFile);
    showMessage('已开始下载转换后的文件');
  }
  return {
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
  };
}
