export type EnforcementTemplate = 'dangerous_command' | 'sensitive_file' | 'permission_change' | 'delete_behavior' | 'suspicious_process';
export type EnforcementMode = 'audit' | 'enforce' | 'disabled';
export type EnforcementDeploymentStatus = 'draft' | 'deployed' | 'failed' | 'disabled';

export interface EnforcementPolicy {
  id: string;
  name: string;
  description: string;
  template: EnforcementTemplate;
  mode: EnforcementMode;
  enabled: boolean;
  targetHosts: string[];
  definition: Record<string, unknown>;
  yaml: string;
  deploymentStatus: EnforcementDeploymentStatus;
  deploymentMessage: string;
  deployedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export type EnforcementPolicyPayload = Omit<EnforcementPolicy, 'id' | 'createdAt' | 'updatedAt' | 'deployedAt'> & {
  deployedAt?: string;
};
