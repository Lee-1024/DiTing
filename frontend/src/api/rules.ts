import { apiClient } from './client';
import type { AuditRule, RulePayload, RuleTestEvent, RuleTestResponse } from '../types/rule';

// listRules 查询并返回 list Rules 列表。
export async function listRules(): Promise<AuditRule[]> {
  const response = await apiClient.get<AuditRule[]>('/rules');
  return response.data;
}

// createRule 创建新的 create Rule。
export async function createRule(rule: RulePayload): Promise<AuditRule> {
  const response = await apiClient.post<AuditRule>('/rules', rule);
  return response.data;
}

// getRule 获取 get Rule 数据。
export async function getRule(id: string): Promise<AuditRule> {
  const response = await apiClient.get<AuditRule>(`/rules/${id}`);
  return response.data;
}

// updateRule 保存或更新 update Rule。
export async function updateRule(id: string, rule: RulePayload): Promise<AuditRule> {
  const response = await apiClient.put<AuditRule>(`/rules/${id}`, rule);
  return response.data;
}

// deleteRule 删除指定的 delete Rule。
export async function deleteRule(id: string): Promise<void> {
  await apiClient.delete(`/rules/${id}`);
}

// testRule 处理 test Rule 相关逻辑。
export async function testRule(rule: RulePayload, event: RuleTestEvent): Promise<RuleTestResponse> {
  const response = await apiClient.post<RuleTestResponse>('/rules/test', { rule, event });
  return response.data;
}
