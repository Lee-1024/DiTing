import { apiClient } from './client';
import type { AuditEvent, AuditEventQuery, PagedAuditEvents } from '../types/audit';

export async function queryAuditEvents(params: AuditEventQuery): Promise<PagedAuditEvents> {
  const response = await apiClient.get<PagedAuditEvents>('/audit/events', { params });
  return response.data;
}

export async function getAuditEvent(eventId: string): Promise<AuditEvent> {
  const response = await apiClient.get<AuditEvent>(`/audit/events/${eventId}`);
  return response.data;
}

export async function exportAuditEvents(params: AuditEventQuery): Promise<Blob> {
  const response = await apiClient.get('/audit/events/export', { params, responseType: 'blob' });
  return response.data;
}
