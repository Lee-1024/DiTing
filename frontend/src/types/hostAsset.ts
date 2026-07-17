export interface HostAsset {
  id: string;
  hostId: string;
  hostName: string;
  nodeName: string;
  displayName: string;
  hostIp: string;
  environment: string;
  owner: string;
  department: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export type HostAssetPayload = Omit<HostAsset, 'id' | 'createdAt' | 'updatedAt'>;
