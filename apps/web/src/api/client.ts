import { invoke, isTauri } from '@tauri-apps/api/core';

export const desktop = isTauri();
const apiBase = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/+$/, '');
export function setDesktopToken(token: string | null) {
  return desktop ? invoke<void>('set_desktop_session', { token }) : Promise.resolve();
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

export async function request<T>(path: string, options: {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'; body?: unknown; signal?: AbortSignal; revision?: number;
} = {}): Promise<T> {
  const { method = 'GET', body, signal, revision } = options;
  const headers = new Headers({ Accept: 'application/json' });
  if (method !== 'GET') headers.set('X-ZwOA-Request', '1');
  if (body !== undefined) headers.set('Content-Type', 'application/json');
  if (revision !== undefined) headers.set('X-ZwOA-Revision', String(revision));
  const desktopToken = desktop ? await invoke<string | null>('desktop_session') : null;
  if (desktopToken) headers.set('Authorization', `Bearer ${desktopToken}`);
  const timeout = AbortSignal.timeout(10000);
  const response = await fetch(`${apiBase}/api/v1${path}`, {
    method, headers, credentials: desktop ? 'omit' : 'include',
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
  });
  const result = await response.json().catch(() => null);
  if (response.status === 401 && !path.endsWith('/login') && path !== '/auth/session') {
    window.dispatchEvent(new Event('zwoa-session-expired'));
  }
  if (!response.ok || result?.code !== 0) {
    throw new ApiError(response.status, result?.message || '服务返回了无法识别的响应');
  }
  return result.data as T;
}

export function errorMessage(error: unknown) {
  return error instanceof ApiError ? error.message : '暂时无法连接服务，请稍后重试。';
}
