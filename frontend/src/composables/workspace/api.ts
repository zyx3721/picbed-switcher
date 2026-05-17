export const apiBase = normalizeApiBase(import.meta.env.VITE_API_BASE_URL);

export function normalizeApiBase(value: string | undefined) {
  return (value || '/api').replace(/\/+$/, '').replace(/\/api$/, '');
}

export function apiURL(path: string) {
  return `${apiBase}${path.startsWith('/') ? path : `/${path}`}`;
}

export function createClientId() {
  if (globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID();
  }
  const randomPart = Math.random().toString(36).slice(2, 10);
  return `${Date.now().toString(36)}-${randomPart}`;
}
