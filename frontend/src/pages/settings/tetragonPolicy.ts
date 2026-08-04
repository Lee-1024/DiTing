export type PolicyTemplate = 'dangerous_command' | 'sensitive_file' | 'permission_change' | 'delete_behavior' | 'suspicious_process';
export type PolicyMode = 'audit' | 'enforce' | 'disabled';
export type UserMatchMode = 'all' | 'include' | 'exclude_root';

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
  processNames?: string[];
  userMatchMode?: UserMatchMode;
  userIds?: string[];
  enabled?: boolean;
  description?: string;
  targetHosts?: string[];
}

// generatePolicy builds the Tetragon TracingPolicy YAML shown and deployed by DiTing.
export function generatePolicy(values: PolicyFormValues) {
  const name = sanitizeName(values.name || 'diting-tetragon-policy');
  if (values.mode === 'disabled' || values.enabled === false) {
    return `# 当前策略已禁用，未生成可部署的 TracingPolicy。\n# 请选择“仅审计”或“拦截”后再复制/下载。`;
  }
  const template = policyTemplate(values);
  return `apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: ${name}
spec:
${template}`;
}

function policyTemplate(values: PolicyFormValues) {
  switch (values.template) {
    case 'sensitive_file':
      return kprobeBlock('file-access', 'security_file_open', 'file_access', 'file', values.filePaths ?? [], values.processNames ?? [], userMatcher(values), values.mode);
    case 'permission_change':
      return syscallBlock('permission-change', [
        { syscall: 'chmod', argIndex: 0 },
        { syscall: 'fchmodat', argIndex: 1 },
        { syscall: 'chown', argIndex: 0 },
        { syscall: 'fchownat', argIndex: 1 },
      ], 'file_access', values.filePaths ?? ['/'], values.processNames ?? [], userMatcher(values), values.mode, 'Prefix', false);
    case 'delete_behavior':
      return deleteBehaviorBlock(values.filePaths ?? ['/'], values.processNames ?? [], userMatcher(values));
    case 'suspicious_process':
      return syscallBlock('suspicious-process', [{ syscall: 'execve', argIndex: 0 }], 'process_exec', values.processNames ?? [], [], null, values.mode, 'Postfix', false);
    default:
      return dangerousCommandBlock(values);
  }
}

interface SyscallProbe {
  syscall: string;
  argIndex: number;
}

interface UserMatcher {
  operator: 'Equal' | 'NotEqual';
  values: string[];
}

function dangerousCommandBlock(values: PolicyFormValues) {
  const broadBlock = syscallBlock('dangerous-command', [{ syscall: 'execve', argIndex: 0 }], 'process_exec', values.commands ?? [], [], null, values.mode, 'Postfix', false);
  const preciseRules = parseCommandRules(values.commandRules, values.commandRuleText);
  if (preciseRules.length === 0) {
    return broadBlock;
  }
  return `${broadBlock}
${preciseCommandBlock(preciseRules, values.mode, userMatcher(values))}`;
}

function syscallBlock(name: string, syscalls: SyscallProbe[], tag: string, values: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode, operator: 'Prefix' | 'Postfix', returnProbe: boolean) {
  const matchValues = (values.filter(Boolean).length ? values.filter(Boolean) : ['']).map((item) => `            - "${escapeYaml(item)}"`).join('\n');
  return `  kprobes:
${syscalls.map(({ syscall, argIndex }) => `  - call: "sys_${syscall}"
    syscall: true
    return: ${returnProbe ? 'true' : 'false'}
    args:
    - index: ${argIndex}
      type: "string"
${uidDataBlock(user)}
${returnArgBlock(returnProbe)}
    tags:
    - "${name}"
    - "${tag}"
${enforcementTags(mode)}
    selectors:
    - matchArgs:
      - index: ${argIndex}
        operator: ${operator}
        values:
${matchValues}${matchBinaries(processNames)}${matchUser(user)}${matchActions(mode)}`).join('\n')}`;
}

function preciseCommandBlock(rules: CommandArgRule[], mode: PolicyMode, user: UserMatcher | null) {
  return `  - call: "sys_execve"
    syscall: true
    return: false
    args:
    - index: 0
      type: "string"
    - index: 1
      type: "string"
    - index: 2
      type: "string"
    - index: 3
      type: "string"
${uidDataBlock(user)}
    tags:
    - "dangerous-command-args"
    - "process_exec"
${enforcementTags(mode)}
    selectors:
${rules.map((rule) => preciseCommandSelector(rule, mode, user)).join('\n')}`;
}

function preciseCommandSelector(rule: CommandArgRule, mode: PolicyMode, user: UserMatcher | null) {
  const binary = rule.binary.trim();
  const args = rule.args.map(splitArgAlternatives);
  return `    - matchArgs:
      - index: 0
        operator: Postfix
        values:
            - "${escapeYaml(binary)}"
${args.map((values, index) => `      - index: ${index + 1}
        operator: Equal
        values:
${values.map((value) => `            - "${escapeYaml(value)}"`).join('\n')}`).join('\n')}${matchUser(user)}${matchActions(mode)}`;
}

function parseCommandRules(rules: CommandArgRule[] | undefined, text: string | undefined) {
  const fromObjects = (rules ?? []).filter((rule) => rule.binary.trim() && rule.args.some((arg) => arg.trim()));
  const fromText = (text ?? '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#'))
    .map((line) => {
      const parts = line.split(/\s+/).filter(Boolean);
      return { binary: parts[0] ?? '', args: parts.slice(1) };
    })
    .filter((rule) => rule.binary && rule.args.length > 0);
  return [...fromObjects, ...fromText];
}

function splitArgAlternatives(value: string) {
  return value.split('|').map((item) => item.trim()).filter(Boolean);
}

function returnArgBlock(returnProbe: boolean) {
  if (!returnProbe) {
    return '';
  }
  return `    returnArg:
      index: 0
      type: "int"
`;
}

function kprobeBlock(name: string, call: string, tag: string, argType: string, paths: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode) {
  const values = (paths.length ? paths : ['/etc/passwd']).map((path) => `            - "${escapeYaml(path)}"`).join('\n');
  return `  kprobes:
  - call: "${call}"
    syscall: false
    return: true
    args:
    - index: 0
      type: "${argType}"
${uidDataBlock(user)}
    returnArg:
      index: 0
      type: "int"
    tags:
    - "${name}"
    - "${tag}"
${enforcementTags(mode)}
    selectors:
    - matchArgs:
      - index: 0
        operator: Prefix
        values:
${values}${matchBinaries(processNames)}${matchUser(user)}${matchActions(mode)}`;
}

function deleteBehaviorBlock(paths: string[], processNames: string[], user: UserMatcher | null) {
  const values = (paths.filter(Boolean).length ? paths.filter(Boolean) : ['/']).map((path) => `            - "${escapeYaml(path)}"`).join('\n');
  return `  kprobes:
  - call: "security_path_unlink"
    syscall: false
    return: false
    args:
    - index: 0
      type: "path"
${uidDataBlock(user)}
    tags:
    - "delete-behavior"
    - "file_access"
    selectors:
    - matchArgs:
      - index: 0
        operator: Prefix
        values:
${values}${matchBinaries(processNames)}${matchUser(user)}
  - call: "security_path_rmdir"
    syscall: false
    return: false
    args:
    - index: 0
      type: "path"
${uidDataBlock(user)}
    tags:
    - "delete-behavior"
    - "file_access"
    selectors:
    - matchArgs:
      - index: 0
        operator: Prefix
        values:
${values}${matchBinaries(processNames)}${matchUser(user)}`;
}

function matchBinaries(processNames: string[]) {
  const values = processNames.filter(Boolean);
  if (values.length === 0) {
    return '';
  }
  return `
      matchBinaries:
      - operator: Postfix
        values:
${values.map((item) => `        - "${escapeYaml(item)}"`).join('\n')}`;
}

function uidDataBlock(user: UserMatcher | null) {
  if (!user) {
    return '';
  }
  return `    data:
    - index: 0
      type: "int"
      source: "current_task"
      resolve: "loginuid.val"`;
}

function matchUser(user: UserMatcher | null) {
  if (!user) {
    return '';
  }
  return `
      matchData:
      - index: 0
        operator: ${user.operator}
        values:
${user.values.map((item) => `        - "${escapeYaml(item)}"`).join('\n')}`;
}

function userMatcher(values: PolicyFormValues): UserMatcher | null {
  if (values.userMatchMode === 'exclude_root') {
    return { operator: 'NotEqual', values: ['0'] };
  }
  if (values.userMatchMode === 'include') {
    const ids = (values.userIds ?? []).filter(isUserId);
    return ids.length > 0 ? { operator: 'Equal', values: ids } : null;
  }
  return null;
}

export function isUserId(value: string) {
  return /^\d+$/.test(value.trim());
}

function matchActions(mode: PolicyMode) {
  if (mode !== 'enforce') {
    return '';
  }
  return `
      matchActions:
      - action: Sigkill`;
}

function enforcementTags(mode: PolicyMode) {
  if (mode !== 'enforce') {
    return '';
  }
  return `    - "diting-enforcement"
    - "diting-blocked-command"`;
}

function sanitizeName(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '') || 'diting-tetragon-policy';
}

function escapeYaml(value: string) {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}
