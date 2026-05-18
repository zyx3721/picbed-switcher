import { apiURL } from './api';
import type { RequestError } from './types';

export function createWorkspaceRequest(getToken: () => string) {
  return async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers = new Headers(options.headers);
    if (!(options.body instanceof FormData)) headers.set('Content-Type', 'application/json');
    const token = getToken();
    if (token) headers.set('Authorization', `Bearer ${token}`);
    const response = await fetch(apiURL(path), { ...options, headers });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(data.error || '\u8bf7\u6c42\u5931\u8d25') as RequestError;
      error.status = response.status;
      throw error;
    }
    return data as T;
  };
}
