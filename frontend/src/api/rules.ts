import { apiClient } from './client';
import type { AuditRule } from '../types/rule';

export async function listRules(): Promise<AuditRule[]> {
  const response = await apiClient.get<AuditRule[]>('/rules');
  return response.data;
}

export async function createRule(rule: Omit<AuditRule, 'id' | 'updatedAt'>): Promise<AuditRule> {
  const response = await apiClient.post<AuditRule>('/rules', rule);
  return response.data;
}

export async function getRule(id: string): Promise<AuditRule> {
  const response = await apiClient.get<AuditRule>(`/rules/${id}`);
  return response.data;
}

export async function updateRule(id: string, rule: Omit<AuditRule, 'id' | 'updatedAt'>): Promise<AuditRule> {
  const response = await apiClient.put<AuditRule>(`/rules/${id}`, rule);
  return response.data;
}

export async function deleteRule(id: string): Promise<void> {
  await apiClient.delete(`/rules/${id}`);
}
