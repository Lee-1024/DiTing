import { apiClient } from './client';
import type { OperationLogQuery, PagedOperationLogs } from '../types/operationLog';

export async function queryOperationLogs(params: OperationLogQuery): Promise<PagedOperationLogs> {
  const response = await apiClient.get<PagedOperationLogs>('/operation-logs', { params });
  return response.data;
}
