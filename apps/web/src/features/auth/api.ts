import { desktop, request, setDesktopToken } from '../../shared/api/client';

export interface Account {
  id: number;
  username: string;
  name: string;
  superAdmin: boolean;
  active: boolean;
  mustChangePassword: boolean;
}

export type TeamKind = 'team' | 'project';

export interface Identity extends Account {
  avatarVersion: string;
  permissions: string[];
  teams: { id: number; name: string; kind: TeamKind; admin: boolean }[];
  adminModules: string[];
}

export const getIdentity = (signal?: AbortSignal) => request<Identity>('/auth/session', { signal });

export async function login(body: { username: string; password: string }) {
  if (desktop) {
    const { token } = await request<{ token: string }>('/auth/desktop/login', { method: 'POST', body });
    await setDesktopToken(token);
  } else {
    await request('/auth/login', { method: 'POST', body });
  }
}

export const logout = () => request('/auth/logout', { method: 'POST' });
export const changePassword = (body: { current: string; next: string }) => request('/auth/password', { method: 'PUT', body });
