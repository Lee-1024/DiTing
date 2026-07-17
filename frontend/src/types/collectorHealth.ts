export interface CollectorHeartbeat {
  hostId: string;
  hostName: string;
  status: 'online' | 'offline';
  healthLevel: 'healthy' | 'warning' | 'critical';
  message: string;
  lastSeenAt: string;
  lastEventTime?: string;
  lastWriteAt?: string;
  heartbeatLagSeconds: number;
  eventLagSeconds?: number;
  writeLagSeconds?: number;
  eventsWritten: number;
  updatedAt: string;
}
