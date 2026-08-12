export type PolicyTemplate = 'dangerous_command' | 'sensitive_file' | 'permission_change' | 'delete_behavior' | 'suspicious_process';
export type PolicyMode = 'audit' | 'enforce' | 'disabled';
export type UserMatchMode = 'all' | 'include' | 'exclude_root';
export type SensitiveFileOperation = 'read' | 'change' | 'all';

export interface CollectorHostOptionSource {
  hostId: string;
  hostName?: string;
  status?: string;
}

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
  const operations = values.operations?.length ? values.operations : ['change'];
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

export function buildCollectorHostOptions(hosts: CollectorHostOptionSource[], selectedHostIds: string[] = []) {
  const options = new Map<string, { value: string; label: string }>();
  [...hosts]
    .filter((host) => host.hostId)
    .sort((left, right) => left.hostId.localeCompare(right.hostId))
    .forEach((host) => {
      const identity = host.hostName ? `${host.hostId} / ${host.hostName}` : host.hostId;
      options.set(host.hostId, {
        value: host.hostId,
        label: `${identity}${hostStatusLabel(host.status)}`,
      });
    });
  selectedHostIds
    .filter((hostId) => hostId && !options.has(hostId))
    .sort((left, right) => left.localeCompare(right))
    .forEach((hostId) => {
      options.set(hostId, { value: hostId, label: `${hostId}（历史主机）` });
    });
  return [...options.values()];
}

function sanitizePreviewValue(value: string) {
  return value.replace(/[\r\n]/g, '');
}

function hostStatusLabel(status?: string) {
  if (status === 'online') {
    return '（在线）';
  }
  if (status === 'offline') {
    return '（离线）';
  }
  return '';
}
