import { apiClient } from './client';
import type { HostAsset, HostAssetPayload } from '../types/hostAsset';

export async function listHostAssets(): Promise<HostAsset[]> {
  const response = await apiClient.get<HostAsset[]>('/host-assets');
  return response.data;
}

export async function createHostAsset(asset: HostAssetPayload): Promise<HostAsset> {
  const response = await apiClient.post<HostAsset>('/host-assets', asset);
  return response.data;
}

export async function updateHostAsset(id: string, asset: HostAssetPayload): Promise<HostAsset> {
  const response = await apiClient.put<HostAsset>(`/host-assets/${id}`, asset);
  return response.data;
}

export async function deleteHostAsset(id: string): Promise<void> {
  await apiClient.delete(`/host-assets/${id}`);
}
