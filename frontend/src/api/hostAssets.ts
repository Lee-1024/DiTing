import { apiClient } from './client';
import type { HostAsset, HostAssetPayload } from '../types/hostAsset';

// listHostAssets 查询并返回 list Host Assets 列表。
export async function listHostAssets(): Promise<HostAsset[]> {
  const response = await apiClient.get<HostAsset[]>('/host-assets');
  return response.data;
}

// createHostAsset 创建新的 create Host Asset。
export async function createHostAsset(asset: HostAssetPayload): Promise<HostAsset> {
  const response = await apiClient.post<HostAsset>('/host-assets', asset);
  return response.data;
}

// updateHostAsset 保存或更新 update Host Asset。
export async function updateHostAsset(id: string, asset: HostAssetPayload): Promise<HostAsset> {
  const response = await apiClient.put<HostAsset>(`/host-assets/${id}`, asset);
  return response.data;
}

// deleteHostAsset 删除指定的 delete Host Asset。
export async function deleteHostAsset(id: string): Promise<void> {
  await apiClient.delete(`/host-assets/${id}`);
}
