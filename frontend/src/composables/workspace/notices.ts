import { ref, watch } from 'vue';

export function useWorkspaceNotices() {
  const message = ref('');
  const error = ref('');
  const errorNoticeKey = ref(0);
  let errorTimer: ReturnType<typeof window.setTimeout> | undefined;

  function clearErrorTimer() {
    if (!errorTimer) return;
    window.clearTimeout(errorTimer);
    errorTimer = undefined;
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
    message.value = '';
    error.value = '';
  }

  watch(error, value => {
    if (!value) clearErrorTimer();
  });

  return {
    message,
    error,
    errorNoticeKey,
    showMessage,
    showError,
    clearNotice,
    clearErrorTimer,
  };
}
