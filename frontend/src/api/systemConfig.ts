import { apiClient } from './client';
import type { CollectorFilterConfig } from '../types/systemConfig';

// getCollectorFilterConfig 获取 get Collector Filter Config 数据。
export async function getCollectorFilterConfig(): Promise<CollectorFilterConfig> {
  const response = await apiClient.get<CollectorFilterConfig>('/system-configs/collector-filter');
  return response.data;
}

// saveCollectorFilterConfig 保存或更新 save Collector Filter Config。
export async function saveCollectorFilterConfig(config: CollectorFilterConfig): Promise<CollectorFilterConfig> {
  const response = await apiClient.put<CollectorFilterConfig>('/system-configs/collector-filter', config);
  return response.data;
}
