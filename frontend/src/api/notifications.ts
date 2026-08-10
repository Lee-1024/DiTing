import { apiClient } from './client';
import type { NotificationDisposition, NotificationListResult, NotificationView } from '../types/notification';

export async function listNotifications(view: NotificationView, limit = 20): Promise<NotificationListResult> {
  const response = await apiClient.get<NotificationListResult>('/notifications', { params: { view, limit } });
  return response.data;
}

export async function markNotificationRead(id: string): Promise<void> {
  await apiClient.post(`/notifications/${encodeURIComponent(id)}/read`);
}

export async function markAllNotificationsRead(): Promise<void> {
  await apiClient.post('/notifications/read-all');
}

export async function handleNotification(id: string, disposition: NotificationDisposition): Promise<void> {
  await apiClient.post(`/notifications/${encodeURIComponent(id)}/handle`, { disposition });
}