import { apiClient } from './client';
import type { CollectorHeartbeat } from '../types/collectorHealth';

// listCollectorHealth 查询并返回 list Collector Health 列表。
export async function listCollectorHealth(): Promise<CollectorHeartbeat[]> {
  const response = await apiClient.get<CollectorHeartbeat[]>('/collectors/health');
  return response.data;
}
