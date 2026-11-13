import { desktop, request, setDesktopToken } from './client';

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
  permissions: string[];
  teams: { id: number; name: string; kind: TeamKind; admin: boolean }[];
  adminModules: string[];
}

export interface ManagedAccount extends Account { groupIds: number[] }
export interface AccountInput {
  username: string;
  name: string;
  password?: string;
  active: boolean;
  groupIds: number[];
}

export interface PermissionGroup {
  id: number;
  name: string;
  description: string;
  permissions: string[];
}

export interface Team {
  id: number;
  name: string;
  kind: TeamKind;
  description: string;
  memberIds: number[];
  adminIds: number[];
}

export interface ManagedModule {
  key: string;
  name: string;
  permission: string;
  adminIds: number[];
}

export interface Administration {
  revision: number;
  accounts: ManagedAccount[];
  groups: PermissionGroup[];
  teams: Team[];
  modules: ManagedModule[];
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
export const getAdministration = (signal?: AbortSignal) => request<Administration>('/admin/overview', { signal });

export function saveRecord(path: 'accounts' | 'groups' | 'teams', id: number | undefined, revision: number, body: AccountInput | Omit<PermissionGroup, 'id'> | Omit<Team, 'id'>) {
  return request(`/admin/${path}${id ? `/${id}` : ''}`, { method: id ? 'PUT' : 'POST', body, revision });
}

export const deleteRecord = (path: 'groups' | 'teams', id: number, revision: number) => request(`/admin/${path}/${id}`, { method: 'DELETE', revision });
export const saveModuleAdmins = (key: string, revision: number, adminIds: number[]) => request(`/admin/modules/${key}/admins`, { method: 'PUT', body: { adminIds }, revision });
