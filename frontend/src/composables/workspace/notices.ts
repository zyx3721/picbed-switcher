import { ref, watch } from 'vue';

export function useWorkspaceNotices() {
  const message = ref('');
  const error = ref('');
  const errorNoticeKey = ref(0);
  const toast = ref('');
  const toastNoticeKey = ref(0);
  let errorTimer: ReturnType<typeof window.setTimeout> | undefined;
  let toastTimer: ReturnType<typeof window.setTimeout> | undefined;

  function clearErrorTimer() {
    if (!errorTimer) return;
    window.clearTimeout(errorTimer);
    errorTimer = undefined;
  }

  function clearToastTimer() {
    if (!toastTimer) return;
    window.clearTimeout(toastTimer);
    toastTimer = undefined;
  }

  function showToast(text: string) {
    clearToastTimer();
    toast.value = text;
    toastNoticeKey.value += 1;
    toastTimer = window.setTimeout(() => {
      toast.value = '';
      toastTimer = undefined;
    }, 5000);
  }

  function showMessage(text: string) {
    clearErrorTimer();
    message.value = text;
    error.value = '';
  }

  function scheduleErrorClear() {
    errorTimer = window.setTimeout(() => {
      error.value = '';
      errorTimer = undefined;
    }, 5000);
  }

  function showError(text: string) {
    clearErrorTimer();
    error.value = text;
    errorNoticeKey.value += 1;
    message.value = '';
    scheduleErrorClear();
  }

  function clearNotice() {
    clearErrorTimer();
    clearToastTimer();
    message.value = '';
    error.value = '';
    toast.value = '';
  }

  watch(error, value => {
    if (!value) clearErrorTimer();
  });
  watch(toast, value => {
    if (!value) clearToastTimer();
  });

  return {
    message,
    error,
    errorNoticeKey,
    toast,
    toastNoticeKey,
    showMessage,
    showError,
    showToast,
    clearNotice,
    clearErrorTimer,
    clearToastTimer,
  };
}
