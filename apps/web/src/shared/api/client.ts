import { invoke, isTauri } from '@tauri-apps/api/core';

declare const __ZWOA_DEV_API_TARGET__: string;

export const desktop = isTauri();
export const sessionChangedEvent = 'zwoa-session-changed';
const apiBase = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/+$/, '');
const sessionBase = apiBase || (import.meta.env.DEV ? __ZWOA_DEV_API_TARGET__ : '.');
export const desktopSessionScope = desktop
  ? `${import.meta.env.DEV ? 'dev' : 'release'}:${new URL(sessionBase, location.href).href}`
  : '';

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

export async function setDesktopToken(token: string) {
  try {
    await invoke<void>('set_desktop_session', { scope: desktopSessionScope, token });
  } catch {
    throw new ApiError(0, '无法保存桌面登录状态，请检查 Windows 凭据管理器。');
  }
}

async function readDesktopToken() {
  try {
    return await invoke<string | null>('desktop_session', { scope: desktopSessionScope });
  } catch {
    throw new ApiError(0, '无法读取桌面登录状态，请检查 Windows 凭据管理器。');
  }
}

async function clearDesktopToken(token: string) {
  try {
    return await invoke<boolean>('clear_desktop_session', { scope: desktopSessionScope, token });
  } catch {
    // 服务端已撤销或拒绝旧令牌，允许重新登录。
    return undefined;
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
  const desktopToken = desktop ? await readDesktopToken() : null;
  if (desktopToken) headers.set('Authorization', `Bearer ${desktopToken}`);
  const timeout = AbortSignal.timeout(10000);
  const response = await fetch(`${apiBase}/api/v1${path}`, {
    method, headers, credentials: desktop ? 'omit' : 'include',
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
  });
  const result = await response.json().catch(() => null);
  const revoked = response.ok && result?.code === 0 && (path === '/auth/logout' || path === '/auth/password');
  const unauthorized = response.status === 401 && !path.endsWith('/login');
  if (revoked || (unauthorized && !signal?.aborted)) {
    const cleared = desktopToken ? await clearDesktopToken(desktopToken) : undefined;
    if (cleared === false && unauthorized) {
      window.dispatchEvent(new Event(sessionChangedEvent));
      throw new DOMException('会话已更新', 'AbortError');
    }
    if (cleared !== false && (revoked || cleared === true || path !== '/auth/session')) {
      window.dispatchEvent(new Event(sessionChangedEvent));
    }
  }
  if (!response.ok || result?.code !== 0) {
    throw new ApiError(response.status, result?.message || '服务返回了无法识别的响应');
  }
  return result.data as T;
}

export function errorMessage(error: unknown) {
  return error instanceof ApiError ? error.message : '暂时无法连接服务，请稍后重试。';
}
