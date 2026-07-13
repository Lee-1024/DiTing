import { apiClient } from './client';
import type { RiskDisposition, RiskDispositionMap, RiskDispositionStatus } from '../types/riskDisposition';

export async function getRiskDispositions(eventIds: string[]): Promise<RiskDispositionMap> {
  if (eventIds.length === 0) {
    return {};
  }
  const response = await apiClient.post<RiskDispositionMap>('/risk-dispositions/batch', { eventIds });
  return response.data;
}

export async function updateRiskDisposition(eventId: string, status: RiskDispositionStatus, note: string): Promise<RiskDisposition> {
  const response = await apiClient.put<RiskDisposition>(`/risk-dispositions/${encodeURIComponent(eventId)}`, { status, note });
  return response.data;
}
