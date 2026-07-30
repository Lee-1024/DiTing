import { apiClient } from './client';
import type { AuditEvent } from '../types/audit';
import type { AIRiskAnalysis, AIRiskAnalysisMap } from '../types/riskAnalysis';

export async function getRiskAnalyses(events: AuditEvent[]): Promise<AIRiskAnalysisMap> {
  if (events.length === 0) {
    return {};
  }
  const response = await apiClient.post<AIRiskAnalysisMap>('/risk-analyses/batch', {
    eventIds: events.map((event) => event.eventId),
  });
  return response.data;
}

export async function analyzeRiskEvent(eventId: string): Promise<AIRiskAnalysis> {
  const response = await apiClient.post<AIRiskAnalysis>(`/risk-analyses/${encodeURIComponent(eventId)}/analyze`);
  return response.data;
}
