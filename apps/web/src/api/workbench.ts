import { request } from './client';

export interface WorkbenchOverview {
  name: string;
  version: string;
}

export async function getWorkbenchOverview(signal: AbortSignal): Promise<WorkbenchOverview> {
  const data = await request<WorkbenchOverview>('/workbench/overview', { signal });
  if (typeof data?.name !== 'string' || typeof data?.version !== 'string') {
    throw new Error('工作台接口返回了无法识别的数据');
  }
  return data;
}
