import { apiClient } from './client';
import type { EnforcementDeployment, EnforcementDeploymentStatus, EnforcementPolicy, EnforcementPolicyPayload } from '../types/enforcement';

// listEnforcementPolicies 查询并返回 list Enforcement Policies 列表。
export async function listEnforcementPolicies(): Promise<EnforcementPolicy[]> {
  const response = await apiClient.get<EnforcementPolicy[]>('/enforcement-policies');
  return response.data;
}

// createEnforcementPolicy 创建新的 create Enforcement Policy。
export async function createEnforcementPolicy(policy: EnforcementPolicyPayload): Promise<EnforcementPolicy> {
  const response = await apiClient.post<EnforcementPolicy>('/enforcement-policies', policy);
  return response.data;
}

// updateEnforcementPolicy 保存或更新 update Enforcement Policy。
export async function updateEnforcementPolicy(id: string, policy: EnforcementPolicyPayload): Promise<EnforcementPolicy> {
  const response = await apiClient.put<EnforcementPolicy>(`/enforcement-policies/${id}`, policy);
  return response.data;
}

// deleteEnforcementPolicy 删除指定的 delete Enforcement Policy。
export async function deleteEnforcementPolicy(id: string): Promise<void> {
  await apiClient.delete(`/enforcement-policies/${id}`);
}

// updateEnforcementDeployment 保存或更新 update Enforcement Deployment。
export async function updateEnforcementDeployment(
  id: string,
  status: EnforcementDeploymentStatus,
  message: string,
): Promise<EnforcementPolicy> {
  const response = await apiClient.post<EnforcementPolicy>(`/enforcement-policies/${id}/deployment`, { status, message });
  return response.data;
}

// emergencyDisableEnforcementPolicies 处理 emergency Disable Enforcement Policies 相关逻辑。
export async function emergencyDisableEnforcementPolicies(): Promise<{ disabledCount: number; message: string }> {
  const response = await apiClient.post<{ disabledCount: number; message: string }>('/enforcement-policies/emergency-disable');
  return response.data;
}

// listEnforcementDeployments 查询并返回 list Enforcement Deployments 列表。
export async function listEnforcementDeployments(policyId: string): Promise<EnforcementDeployment[]> {
  const response = await apiClient.get<EnforcementDeployment[]>(`/enforcement-policies/${policyId}/deployments`);
  return response.data;
}

// upsertEnforcementDeployment 处理 upsert Enforcement Deployment 相关逻辑。
export async function upsertEnforcementDeployment(
  policyId: string,
  deployment: Pick<EnforcementDeployment, 'hostId' | 'hostName' | 'status' | 'message'>,
): Promise<EnforcementDeployment> {
  const response = await apiClient.post<EnforcementDeployment>(`/enforcement-policies/${policyId}/deployments`, deployment);
  return response.data;
}
