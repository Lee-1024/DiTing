export type PolicyTemplate = 'dangerous_command' | 'sensitive_file' | 'permission_change' | 'delete_behavior' | 'suspicious_process';
export type PolicyMode = 'audit' | 'enforce' | 'disabled';
export type UserMatchMode = 'all' | 'include' | 'exclude_root';
export type SensitiveFileOperation = 'read' | 'write' | 'create' | 'delete' | 'rename' | 'chmod' | 'chown' | 'all';

export interface CommandArgRule {
  binary: string;
  args: string[];
}

export interface PolicyFormValues {
  template: PolicyTemplate;
  mode: PolicyMode;
  name: string;
  commands?: string[];
  commandRuleText?: string;
  commandRules?: CommandArgRule[];
  filePaths?: string[];
  operations?: SensitiveFileOperation[];
  processNames?: string[];
  userMatchMode?: UserMatchMode;
  userIds?: string[];
  enabled?: boolean;
  description?: string;
  targetHosts?: string[];
}

// generatePolicy renders an operator preview only. Collector generates the trusted profile.
export function generatePolicy(values: PolicyFormValues) {
  const enabled = values.enabled !== false && values.mode !== 'disabled';
  const paths = (values.filePaths ?? []).map((item) => item.trim()).filter(Boolean);
  const operations = values.operations?.length ? values.operations : ['write'];
  const operationPreview = values.template === 'sensitive_file'
    ? `operations:\n${operations.map((operation) => `  - ${operation}`).join('\n')}\n`
    : '';
  return `engine: apparmor
profile: diting-sudo
enabled: ${enabled}
template: ${values.template}
mode: ${values.mode}
scope: sudo-and-descendants
direct_root: allowed
protected_paths:
${paths.length > 0 ? paths.map((path) => `  - ${sanitizePreviewValue(path)}`).join('\n') : '  - <required>'}
${operationPreview}
`;
}

export function isUserId(value: string) {
  return /^\d+$/.test(value.trim());
}

function sanitizePreviewValue(value: string) {
  return value.replace(/[\r\n]/g, '');
}
