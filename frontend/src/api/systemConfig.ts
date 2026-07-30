import { apiClient } from './client';
import type { AIProviderConfig, CollectorFilterConfig } from '../types/systemConfig';

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

export async function getAIProviderConfig(): Promise<AIProviderConfig> {
  const response = await apiClient.get<AIProviderConfig>('/system-configs/ai');
  return response.data;
}

export async function saveAIProviderConfig(config: AIProviderConfig): Promise<AIProviderConfig> {
  const response = await apiClient.put<AIProviderConfig>('/system-configs/ai', config);
  return response.data;
}

export async function testAIProviderConfig(config: AIProviderConfig): Promise<{ ok: boolean; latencyMs: number; message: string }> {
  const response = await apiClient.post<{ ok: boolean; latencyMs: number; message: string }>('/system-configs/ai/test', config);
  return response.data;
}
