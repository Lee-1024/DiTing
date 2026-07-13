export interface HostAsset {
  id: string;
  nodeName: string;
  displayName: string;
  hostIp: string;
  environment: string;
  owner: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export type HostAssetPayload = Omit<HostAsset, 'id' | 'createdAt' | 'updatedAt'>;
