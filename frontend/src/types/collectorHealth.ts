export interface CollectorHeartbeat {
  hostId: string;
  hostName: string;
  inputMode: string;
  status: 'online' | 'offline';
  healthLevel: 'healthy' | 'warning' | 'critical';
  message: string;
  lastError: string;
  lastSeenAt: string;
  lastEventTime?: string;
  lastWriteAt?: string;
  heartbeatLagSeconds: number;
  eventLagSeconds?: number;
  writeLagSeconds?: number;
  eventsWritten: number;
  bufferedEvents: number;
  droppedEvents: number;
  updatedAt: string;
}
