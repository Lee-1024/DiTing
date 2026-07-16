import { apiClient } from './client';
import type { AuditRule, RulePayload, RuleTestEvent, RuleTestResponse } from '../types/rule';

export async function listRules(): Promise<AuditRule[]> {
  const response = await apiClient.get<AuditRule[]>('/rules');
  return response.data;
}

export async function createRule(rule: RulePayload): Promise<AuditRule> {
  const response = await apiClient.post<AuditRule>('/rules', rule);
  return response.data;
}

export async function getRule(id: string): Promise<AuditRule> {
  const response = await apiClient.get<AuditRule>(`/rules/${id}`);
  return response.data;
}

export async function updateRule(id: string, rule: RulePayload): Promise<AuditRule> {
  const response = await apiClient.put<AuditRule>(`/rules/${id}`, rule);
  return response.data;
}

export async function deleteRule(id: string): Promise<void> {
  await apiClient.delete(`/rules/${id}`);
}

export async function testRule(rule: RulePayload, event: RuleTestEvent): Promise<RuleTestResponse> {
  const response = await apiClient.post<RuleTestResponse>('/rules/test', { rule, event });
  return response.data;
}
