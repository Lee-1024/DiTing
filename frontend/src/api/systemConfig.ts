import { apiClient } from './client';
import type { CollectorFilterConfig } from '../types/systemConfig';

export async function getCollectorFilterConfig(): Promise<CollectorFilterConfig> {
  const response = await apiClient.get<CollectorFilterConfig>('/system-configs/collector-filter');
  return response.data;
}

export async function saveCollectorFilterConfig(config: CollectorFilterConfig): Promise<CollectorFilterConfig> {
  const response = await apiClient.put<CollectorFilterConfig>('/system-configs/collector-filter', config);
  return response.data;
}
