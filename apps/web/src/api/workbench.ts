export interface WorkbenchOverview {
  name: string;
  version: string;
}

const apiBase = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/+$/, '');

export async function getWorkbenchOverview(signal: AbortSignal): Promise<WorkbenchOverview> {
  const response = await fetch(`${apiBase}/api/v1/workbench/overview`, {
    headers: { Accept: 'application/json' },
    signal: AbortSignal.any([signal, AbortSignal.timeout(5000)]),
  });
  if (!response.ok) {
    throw new Error(`API 返回 ${response.status}`);
  }

  const result = await response.json();
  if (result?.code !== 0 || typeof result.data?.name !== 'string' || typeof result.data?.version !== 'string') {
    throw new Error('工作台接口返回了无法识别的数据');
  }
  return result.data;
}
