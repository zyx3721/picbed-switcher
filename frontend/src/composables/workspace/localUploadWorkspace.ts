import { computed, ref, type Ref } from 'vue';
import { createClientId } from './api';
import type { LocalDocument, LocalImageFile, MarkdownImage, PicbedConfig } from './types';

type WorkspaceRequest = <T>(path: string, options?: RequestInit) => Promise<T>;

type LocalUploadWorkspaceDeps = {
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

type LocalUploadResult = {
  results: Array<{ filename: string; content?: string; changed?: number; status: string; error?: string }>;
};

const imageExtensions = new Set(['.png', '.jpg', '.jpeg', '.gif', '.webp', '.bmp', '.svg', '.avif']);

export function useWorkspaceLocalUpload({
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
}: LocalUploadWorkspaceDeps) {
  const localTargetConfigId = ref(0);
  const localTargetDropdownOpen = ref(false);
  const localDocumentDragActive = ref(false);
  const localImageDragActive = ref(false);
  const localDocuments = ref<LocalDocument[]>([]);
  const localImages = ref<LocalImageFile[]>([]);

  const localTargetConfigs = computed(() => configs.value);
  const localDefaultTarget = computed(() => localTargetConfigs.value.find(item => item.is_default) || localTargetConfigs.value[0]);
  const selectedLocalTargetConfig = computed(() =>
    localTargetConfigs.value.find(item => item.id === localTargetConfigId.value)
  );
  const localMatchedCount = computed(() => localDocuments.value.reduce((sum, item) => sum + item.matched, 0));
  const localMissingCount = computed(() => localDocuments.value.reduce((sum, item) => sum + item.missing.length, 0));
  const localConvertedCount = computed(() => localDocuments.value.filter(item => item.status === 'success').length);
  const canUploadLocalBatch = computed(
    () => localDocuments.value.length > 0 && localMatchedCount.value > 0 && localMissingCount.value === 0
  );

  function localTargetConfigLabel(config?: PicbedConfig) {
    return config ? `${config.config_name} · ${typeLabel(config.picbed_type)}` : '请选择';
  }
  function selectLocalTargetConfig(id: number) {
    localTargetConfigId.value = id;
    localTargetDropdownOpen.value = false;
  }
  function localStatusLabel(status: LocalDocument['status']) {
    return { ready: '待匹配', analyzed: '已匹配', success: '已上传', failed: '失败' }[status];
  }
  function resetLocalUploadForm() {
    clearNotice();
    localTargetConfigId.value = localDefaultTarget.value?.id || 0;
    localTargetDropdownOpen.value = false;
    localDocumentDragActive.value = false;
    localImageDragActive.value = false;
    localDocuments.value = [];
    localImages.value = [];
  }
  async function addLocalDocuments(fileList: FileList | File[]) {
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
        references: [] as MarkdownImage[],
        matched: 0,
        missing: [],
        convertedContent: '',
        changed: 0,
        status: 'ready' as const,
        error: '',
      }))
    );
    localDocuments.value = [...localDocuments.value, ...loaded];
    analyzeLocalMatches();
    showMessage(`已加入 ${loaded.length} 个 Markdown 文档`);
  }
  async function handleLocalDocumentFiles(event: Event) {
    const input = event.target as HTMLInputElement;
    await addLocalDocuments(input.files || []);
    input.value = '';
  }
  async function handleLocalDocumentDrop(event: DragEvent) {
    localDocumentDragActive.value = false;
    await addLocalDocuments(event.dataTransfer?.files || []);
  }
  function addLocalImages(fileList: FileList | File[]) {
    const files = Array.from(fileList).filter(isImageFile);
    if (!files.length) {
      showError('请上传本地图片文件或图片目录');
      return;
    }
    const loaded = files.map(file => ({
      key: `image_${createClientId()}`,
      name: file.name,
      path: filePath(file),
      file,
    }));
    localImages.value = [...localImages.value, ...loaded];
    analyzeLocalMatches();
    showMessage(`已加入 ${loaded.length} 张本地图片`);
  }
  function handleLocalImageFiles(event: Event) {
    const input = event.target as HTMLInputElement;
    addLocalImages(input.files || []);
    input.value = '';
  }
  function handleLocalImageDrop(event: DragEvent) {
    localImageDragActive.value = false;
    addLocalImages(event.dataTransfer?.files || []);
  }
  function removeLocalDocument(id: string) {
    localDocuments.value = localDocuments.value.filter(item => item.id !== id);
  }
  function removeLocalImage(key: string) {
    localImages.value = localImages.value.filter(item => item.key !== key);
    analyzeLocalMatches();
  }
  async function analyzeLocalBatch() {
    if (!localDocuments.value.length) {
      showError('请先添加 Markdown 文档');
      return;
    }
    analyzeLocalMatches();
    showMessage(`已匹配 ${localMatchedCount.value} 张本地图片`);
  }
  async function uploadLocalBatch() {
    if (!localTargetConfigId.value) {
      showError('请先选择目标图床配置');
      return;
    }
    if (!localDocuments.value.length) {
      showError('请先添加 Markdown 文档');
      return;
    }
    analyzeLocalMatches();
    if (localMissingCount.value > 0) {
      showError('仍有本地图片未匹配，请补充图片文件或目录');
      return;
    }
    if (!localMatchedCount.value) {
      showError('未匹配到本地图片，无需上传');
      return;
    }
    loading.value = true;
    startTaskProgress({
      title: '本地上传替换中',
      message: '正在准备上传并替换',
      detail: `共 ${localDocuments.value.length} 个文档，${localMatchedCount.value} 张本地图片。`,
      total: localDocuments.value.length,
    });
    try {
      let success = 0;
      let failed = 0;
      for (const [index, document] of localDocuments.value.entries()) {
        updateTaskProgress({
          current: index + 1,
          success,
          failed,
          message: `正在上传替换第 ${index + 1} / ${localDocuments.value.length} 个文档`,
          detail: `${document.filename} · 已匹配 ${document.matched} 张本地图片。`,
        });
        const manifest = buildSingleLocalManifest(document);
        const formData = new FormData();
        formData.append('manifest', JSON.stringify({ target_config_id: localTargetConfigId.value, documents: [manifest.document] }));
        for (const image of manifest.images) formData.append(image.key, image.file, image.name);
        try {
          const data = await request<LocalUploadResult>('/api/convert/local-batch', { method: 'POST', body: formData });
          const result = data.results[0];
          document.status = result?.status === 'success' ? 'success' : 'failed';
          document.convertedContent = result?.content || '';
          document.changed = result?.changed || 0;
          document.error = result?.error || '';
        } catch (err) {
          document.status = 'failed';
          document.convertedContent = '';
          document.changed = 0;
          document.error = err instanceof Error ? err.message : '本地图片上传失败';
        }
        if (document.status === 'success') success += 1;
        else failed += 1;
        updateTaskProgress({ current: index + 1, success, failed });
      }
      const finalStatus = failed > 0 ? 'failed' : 'success';
      finishTaskProgress({
        status: finalStatus,
        message: `本地图片上传完成，成功 ${success} 个，失败 ${failed} 个`,
        detail: '可以关闭此窗口并下载替换后的文档。',
      });
      showMessage(`本地图片上传完成，成功 ${localConvertedCount.value} 个文档`);
      await loadRecords();
    } catch (err) {
      finishTaskProgress({
        status: 'failed',
        message: '本地图片上传失败',
        detail: err instanceof Error ? err.message : '本地图片上传失败',
      });
      showError(err instanceof Error ? err.message : '本地图片上传失败');
      await loadRecords();
    } finally {
      loading.value = false;
    }
  }
  function downloadLocalFile(item: LocalDocument) {
    if (!item.convertedContent) return;
    const blob = new Blob([item.convertedContent], { type: 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = item.filename || 'converted.md';
    link.click();
    URL.revokeObjectURL(url);
  }
  function downloadAllLocalFiles() {
    localDocuments.value.filter(document => document.convertedContent).forEach(downloadLocalFile);
    showMessage('已开始下载本地上传后的文档');
  }
  function analyzeLocalMatches() {
    for (const document of localDocuments.value) {
      const references = extractMarkdownImages(document.content).filter(image => isLocalImageReference(image.url));
      const missing: string[] = [];
      let matched = 0;
      for (const reference of references) {
        if (findLocalImage(reference.url)) matched += 1;
        else missing.push(reference.url);
      }
      document.references = references;
      document.matched = matched;
      document.missing = [...new Set(missing)];
      document.status = document.status === 'success' || document.status === 'failed' ? document.status : 'analyzed';
      document.error = '';
    }
  }
  function buildSingleLocalManifest(document: LocalDocument) {
    const usedImages = new Map<string, LocalImageFile>();
    const item = {
      filename: document.filename,
      content: document.content,
      images: document.references.flatMap(reference => {
        const image = findLocalImage(reference.url);
        if (!image) return [];
        usedImages.set(image.key, image);
        return [{ source: reference.url, file_key: image.key }];
      }),
    };
    return { document: item, images: Array.from(usedImages.values()) };
  }
  function findLocalImage(source: string) {
    const normalizedSource = normalizeLocalPath(source);
    const sourceBase = baseName(normalizedSource);
    return (
      localImages.value.find(image => normalizeLocalPath(image.path) === normalizedSource) ||
      localImages.value.find(image => normalizeLocalPath(image.path).endsWith('/' + normalizedSource)) ||
      localImages.value.find(image => baseName(image.path).toLowerCase() === sourceBase.toLowerCase())
    );
  }

  return {
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
  };
}

function extractMarkdownImages(content: string): MarkdownImage[] {
  const images: MarkdownImage[] = [];
  const pattern = /!\[([^\]]*)\]\(([^\s)]+)(?:\s+"[^"]*")?\)|<img[^>]+src=["']([^"']+)["'][^>]*>/gi;
  let match: RegExpExecArray | null;
  while ((match = pattern.exec(content))) {
    const url = match[2] || match[3] || '';
    images.push({ raw: match[0], url, alt: match[1] || '', picbed: isLocalImageReference(url) ? 'local' : 'other' });
  }
  return images;
}

function isImageFile(file: File) {
  if (file.type.startsWith('image/')) return true;
  const name = file.name.toLowerCase();
  return Array.from(imageExtensions).some(ext => name.endsWith(ext));
}

function isLocalImageReference(raw: string) {
  const value = raw.trim().toLowerCase();
  return Boolean(value) && !value.startsWith('http://') && !value.startsWith('https://') && !value.startsWith('data:');
}

function filePath(file: File) {
  const entryPath = 'webkitRelativePath' in file ? String(file.webkitRelativePath || '') : '';
  return entryPath || file.name;
}

function normalizeLocalPath(raw: string) {
  let value = decodeURIComponent(raw.trim());
  value = value.replace(/^file:\/\/\/?/i, '');
  value = value.replace(/\\/g, '/');
  while (value.startsWith('./')) value = value.slice(2);
  return value.replace(/^\/+|\/+$/g, '').toLowerCase();
}

function baseName(raw: string) {
  const normalized = normalizeLocalPath(raw);
  const parts = normalized.split('/').filter(Boolean);
  return parts[parts.length - 1] || normalized;
}
