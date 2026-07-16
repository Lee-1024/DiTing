import { apiClient } from './client';
import type { CollectorHeartbeat } from '../types/collectorHealth';

export async function listCollectorHealth(): Promise<CollectorHeartbeat[]> {
  const response = await apiClient.get<CollectorHeartbeat[]>('/collectors/health');
  return response.data;
}
