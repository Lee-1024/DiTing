export type NotificationType = 'enforcement' | 'collector' | 'tetragon';
export type NotificationStatus = 'open' | 'resolved';
export type NotificationDisposition = 'confirmed' | 'false_positive' | 'ignored';
export type NotificationView = 'unread' | 'pending' | 'all';

export interface AppNotification {
  id: string;
  type: NotificationType;
  sourceId?: string;
  title: string;
  description: string;
  severity: string;
  target: string;
  status: NotificationStatus;
  disposition?: NotificationDisposition;
  handledBy?: string;
  handledAt?: string;
  read: boolean;
  createdAt: string;
  updatedAt: string;
  resolvedAt?: string;
}

export interface NotificationCounts {
  unread: number;
  pending: number;
  all: number;
}

export interface NotificationListResult {
  items: AppNotification[];
  counts: NotificationCounts;
}