export interface CollectorHeartbeat {
  hostId: string;
  hostName: string;
  status: 'online' | 'offline';
  lastSeenAt: string;
  lastEventTime?: string;
  lastWriteAt?: string;
  eventsWritten: number;
  updatedAt: string;
}
