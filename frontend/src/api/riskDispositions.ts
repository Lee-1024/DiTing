import { apiClient } from './client';
import type { AuditEvent } from '../types/audit';
import type { RiskDisposition, RiskDispositionMap, RiskDispositionStatus } from '../types/riskDisposition';

export async function getRiskDispositions(events: AuditEvent[]): Promise<RiskDispositionMap> {
  if (events.length === 0) {
    return {};
  }
  const response = await apiClient.post<RiskDispositionMap>('/risk-dispositions/batch', {
    eventIds: events.map((event) => event.eventId),
    events: events.map((event) => ({ eventId: event.eventId, fingerprint: riskFingerprint(event) })),
  });
  return response.data;
}

export async function listRiskDispositions(status: RiskDispositionStatus, limit = 500): Promise<RiskDisposition[]> {
  const response = await apiClient.get<{ items: RiskDisposition[] }>('/risk-dispositions', { params: { status, limit } });
  return response.data.items ?? [];
}

export async function updateRiskDisposition(event: AuditEvent, status: RiskDispositionStatus, note: string): Promise<RiskDisposition> {
  const response = await apiClient.put<RiskDisposition>(`/risk-dispositions/${encodeURIComponent(event.eventId)}`, {
    status,
    note,
    scope: status === 'ignore_similar' ? 'similar' : 'event',
    fingerprint: riskFingerprint(event),
  });
  return response.data;
}

export function riskFingerprint(event: AuditEvent): string {
  return [
    event.eventType,
    [...(event.ruleIds ?? [])].sort().join(','),
    event.processName,
    event.cmdline,
    event.filePath,
    event.fileOperation,
    event.dstIp,
    event.dstPort,
    event.protocol,
    event.username,
  ].map((value) => String(value ?? '').trim().toLowerCase()).join('|');
}
