import { apiClient } from './client';
import type { EnforcementDeployment, EnforcementDeploymentStatus, EnforcementPolicy, EnforcementPolicyPayload } from '../types/enforcement';

export async function listEnforcementPolicies(): Promise<EnforcementPolicy[]> {
  const response = await apiClient.get<EnforcementPolicy[]>('/enforcement-policies');
  return response.data;
}

export async function createEnforcementPolicy(policy: EnforcementPolicyPayload): Promise<EnforcementPolicy> {
  const response = await apiClient.post<EnforcementPolicy>('/enforcement-policies', policy);
  return response.data;
}

export async function updateEnforcementPolicy(id: string, policy: EnforcementPolicyPayload): Promise<EnforcementPolicy> {
  const response = await apiClient.put<EnforcementPolicy>(`/enforcement-policies/${id}`, policy);
  return response.data;
}

export async function deleteEnforcementPolicy(id: string): Promise<void> {
  await apiClient.delete(`/enforcement-policies/${id}`);
}

export async function updateEnforcementDeployment(
  id: string,
  status: EnforcementDeploymentStatus,
  message: string,
): Promise<EnforcementPolicy> {
  const response = await apiClient.post<EnforcementPolicy>(`/enforcement-policies/${id}/deployment`, { status, message });
  return response.data;
}

export async function emergencyDisableEnforcementPolicies(): Promise<{ disabledCount: number; message: string }> {
  const response = await apiClient.post<{ disabledCount: number; message: string }>('/enforcement-policies/emergency-disable');
  return response.data;
}

export async function listEnforcementDeployments(policyId: string): Promise<EnforcementDeployment[]> {
  const response = await apiClient.get<EnforcementDeployment[]>(`/enforcement-policies/${policyId}/deployments`);
  return response.data;
}

export async function upsertEnforcementDeployment(
  policyId: string,
  deployment: Pick<EnforcementDeployment, 'hostId' | 'hostName' | 'status' | 'message'>,
): Promise<EnforcementDeployment> {
  const response = await apiClient.post<EnforcementDeployment>(`/enforcement-policies/${policyId}/deployments`, deployment);
  return response.data;
}
