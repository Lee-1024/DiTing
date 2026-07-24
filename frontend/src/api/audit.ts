import { apiClient } from './client';
import type { AuditEvent, AuditEventQuery, PagedAuditEvents } from '../types/audit';

// queryAuditEvents 处理 query Audit Events 相关逻辑。
export async function queryAuditEvents(params: AuditEventQuery): Promise<PagedAuditEvents> {
  const response = await apiClient.get<PagedAuditEvents>('/audit/events', { params });
  return response.data;
}

// getAuditEvent 获取 get Audit Event 数据。
export async function getAuditEvent(eventId: string): Promise<AuditEvent> {
  const response = await apiClient.get<AuditEvent>(`/audit/events/${eventId}`);
  return response.data;
}

// exportAuditEvents 导出或下载 export Audit Events 数据。
export async function exportAuditEvents(params: AuditEventQuery): Promise<Blob> {
  const response = await apiClient.get('/audit/events/export', { params, responseType: 'blob' });
  return response.data;
}
