import { request } from '../../shared/api/client';
import type { Account, TeamKind } from '../auth/api';

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

export const getAdministration = (signal?: AbortSignal) => request<Administration>('/admin/overview', { signal });

export function saveRecord(path: 'accounts' | 'groups' | 'teams', id: number | undefined, revision: number, body: AccountInput | Omit<PermissionGroup, 'id'> | Omit<Team, 'id'>) {
  return request(`/admin/${path}${id ? `/${id}` : ''}`, { method: id ? 'PUT' : 'POST', body, revision });
}

export const deleteRecord = (path: 'groups' | 'teams', id: number, revision: number) => request(`/admin/${path}/${id}`, { method: 'DELETE', revision });
export const saveModuleAdmins = (key: string, revision: number, adminIds: number[]) => request(`/admin/modules/${key}/admins`, { method: 'PUT', body: { adminIds }, revision });
