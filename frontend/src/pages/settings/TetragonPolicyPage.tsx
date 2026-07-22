import { CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, Select, Space, Typography, message } from 'antd';
import { useMemo } from 'react';

type PolicyTemplate = 'dangerous_command' | 'sensitive_file' | 'permission_change' | 'delete_behavior' | 'suspicious_process';
type PolicyMode = 'audit' | 'enforce' | 'disabled';

interface PolicyFormValues {
  template: PolicyTemplate;
  mode: PolicyMode;
  name: string;
  commands?: string[];
  filePaths?: string[];
  processNames?: string[];
  userIds?: string[];
}

const defaultValues: PolicyFormValues = {
  template: 'dangerous_command',
  mode: 'audit',
  name: 'diting-dangerous-command',
  commands: ['curl', 'wget', 'bash'],
  filePaths: ['/etc/passwd', '/etc/shadow', '/etc/sudoers', '/root/.ssh'],
  processNames: [],
  userIds: [],
};

export default function TetragonPolicyPage() {
  const [form] = Form.useForm<PolicyFormValues>();
  const template = Form.useWatch('template', form) ?? defaultValues.template;
  const mode = Form.useWatch('mode', form) ?? defaultValues.mode;
  const name = Form.useWatch('name', form) ?? defaultValues.name;
  const commands = Form.useWatch('commands', form) ?? defaultValues.commands;
  const filePaths = Form.useWatch('filePaths', form) ?? defaultValues.filePaths;
  const processNames = Form.useWatch('processNames', form) ?? defaultValues.processNames;
  const userIds = Form.useWatch('userIds', form) ?? defaultValues.userIds;
  const policy = useMemo<PolicyFormValues>(() => ({
    template,
    mode,
    name,
    commands,
    filePaths,
    processNames,
    userIds,
  }), [template, mode, name, commands, filePaths, processNames, userIds]);
  const yaml = useMemo(() => generatePolicy(policy), [policy]);

  async function copyYaml() {
    await navigator.clipboard.writeText(yaml);
    message.success('策略 YAML 已复制');
  }

  function downloadYaml() {
    const blob = new Blob([yaml], { type: 'text/yaml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${name || 'diting-tetragon-policy'}.yaml`;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">拦截策略</Typography.Title>
        <Button icon={<CopyOutlined />} onClick={() => void copyYaml()}>复制 YAML</Button>
        <Button icon={<DownloadOutlined />} onClick={downloadYaml}>下载 YAML</Button>
      </Space>
      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(360px, 520px) 1fr', gap: 16, alignItems: 'start' }}>
        <Card className="data-card">
          <Form form={form} layout="vertical" initialValues={defaultValues}>
            <Form.Item name="template" label="策略模板" rules={[{ required: true }]}>
              <Select
                onChange={(nextTemplate: PolicyTemplate) => form.setFieldsValue({ name: defaultPolicyName(nextTemplate) })}
                options={[
                  { value: 'dangerous_command', label: '危险命令' },
                  { value: 'sensitive_file', label: '敏感文件读写' },
                  { value: 'permission_change', label: '权限变更' },
                  { value: 'delete_behavior', label: '删除行为' },
                  { value: 'suspicious_process', label: '可疑进程链路' },
                ]}
              />
            </Form.Item>
            <Form.Item name="mode" label="策略模式" rules={[{ required: true }]}>
              <Select options={[
                { value: 'audit', label: '仅审计' },
                { value: 'enforce', label: '拦截' },
                { value: 'disabled', label: '禁用' },
              ]} />
            </Form.Item>
            <Form.Item name="name" label="策略名称" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            {template === 'dangerous_command' && (
              <>
                <Alert
                  type={mode === 'enforce' ? 'warning' : 'info'}
                  showIcon
                  style={{ marginBottom: 16 }}
                  message={mode === 'enforce' ? '危险命令模板不建议直接拦截' : '危险命令模板适合先审计观察'}
                  description={mode === 'enforce'
                    ? 'rm、chmod、vim 等命令本身不是风险，直接拦截会影响普通用户操作。需要拦截时请优先使用敏感文件、权限变更或删除行为模板，并配置目标路径。'
                    : '该模板按进程执行命令匹配，建议用于审计和发现行为。精确拦截请使用文件路径类模板。'}
                />
                <Form.Item name="commands" label="命令/关键进程">
                  <Select mode="tags" tokenSeparators={[',']} />
                </Form.Item>
              </>
            )}
            {(template === 'sensitive_file' || template === 'permission_change' || template === 'delete_behavior') && (
              <>
                <Form.Item name="filePaths" label={template === 'sensitive_file' ? '敏感路径' : '监控路径'}>
                  <Select mode="tags" tokenSeparators={[',']} />
                </Form.Item>
                <Form.Item name="processNames" label="限定进程（可选）" tooltip="留空表示不限制进程；填写 vim、rm、chmod 等可只拦截指定进程访问这些路径。">
                  <Select mode="tags" tokenSeparators={[',']} placeholder="例如 vim / rm / chmod，留空为不限进程" />
                </Form.Item>
                <Form.Item name="userIds" label="限定执行用户 UID（可选）" tooltip="Tetragon 策略按 UID 匹配用户；如 ubuntu 通常为 1000，可在主机上用 id -u ubuntu 查询。">
                  <Select mode="tags" tokenSeparators={[',']} placeholder="例如 1000 / 1001，留空为不限用户" />
                </Form.Item>
                {userIds?.some((item) => item && !isUserId(item)) && (
                  <Alert
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                    message="限定执行用户需要填写 UID"
                    description="Tetragon 策略无法直接按用户名匹配，请在目标主机执行 id -u 用户名 后填写数字 UID。非数字项不会写入 YAML。"
                  />
                )}
              </>
            )}
            {template === 'suspicious_process' && (
              <Form.Item name="processNames" label="可疑进程">
                <Select mode="tags" tokenSeparators={[',']} />
              </Form.Item>
            )}
          </Form>
        </Card>
        <Card className="data-card" title="TracingPolicy YAML">
          <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 13 }}>{yaml}</pre>
        </Card>
      </div>
    </>
  );
}

function defaultPolicyName(template: PolicyTemplate) {
  switch (template) {
    case 'sensitive_file':
      return 'diting-sensitive-file';
    case 'permission_change':
      return 'diting-permission-change';
    case 'delete_behavior':
      return 'diting-delete-behavior';
    case 'suspicious_process':
      return 'diting-suspicious-process';
    default:
      return 'diting-dangerous-command';
  }
}

function generatePolicy(values: PolicyFormValues) {
  const name = sanitizeName(values.name || 'diting-tetragon-policy');
  if (values.mode === 'disabled') {
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
      return kprobeBlock('file-access', 'security_file_open', 'file_access', 'file', values.filePaths ?? [], values.processNames ?? [], values.userIds ?? [], values.mode);
    case 'permission_change':
      return syscallBlock('permission-change', [
        { syscall: 'chmod', argIndex: 0 },
        { syscall: 'fchmodat', argIndex: 1 },
        { syscall: 'chown', argIndex: 0 },
        { syscall: 'fchownat', argIndex: 1 },
      ], 'file_access', values.filePaths ?? ['/'], values.processNames ?? [], values.userIds ?? [], values.mode, 'Prefix');
    case 'delete_behavior':
      return syscallBlock('delete-behavior', [
        { syscall: 'unlink', argIndex: 0 },
        { syscall: 'unlinkat', argIndex: 1 },
        { syscall: 'rmdir', argIndex: 0 },
      ], 'file_access', values.filePaths ?? ['/'], values.processNames ?? [], values.userIds ?? [], values.mode, 'Prefix');
    case 'suspicious_process':
      return syscallBlock('suspicious-process', [{ syscall: 'execve', argIndex: 0 }], 'process_exec', values.processNames ?? [], [], [], values.mode, 'Postfix');
    default:
      return syscallBlock('dangerous-command', [{ syscall: 'execve', argIndex: 0 }], 'process_exec', values.commands ?? [], [], [], 'audit', 'Postfix');
  }
}

interface SyscallProbe {
  syscall: string;
  argIndex: number;
}

function syscallBlock(name: string, syscalls: SyscallProbe[], returnArg: string, values: string[], processNames: string[], userIds: string[], mode: PolicyMode, operator: 'Prefix' | 'Postfix') {
  const matchValues = (values.filter(Boolean).length ? values.filter(Boolean) : ['']).map((item) => `            - "${escapeYaml(item)}"`).join('\n');
  return `  kprobes:
${syscalls.map(({ syscall, argIndex }) => `  - call: "sys_${syscall}"
    syscall: true
    return: true
    args:
    - index: ${argIndex}
      type: "string"
${uidDataBlock(userIds)}
    returnArg:
      index: 0
      type: "int"
    tags:
    - "${name}"
    - "${returnArg}"
    selectors:
    - matchArgs:
      - index: ${argIndex}
        operator: ${operator}
        values:
${matchValues}${matchBinaries(processNames)}${matchUserIds(userIds)}${matchActions(mode)}`).join('\n')}`;
}

function kprobeBlock(name: string, call: string, tag: string, argType: string, paths: string[], processNames: string[], userIds: string[], mode: PolicyMode) {
  const values = (paths.length ? paths : ['/etc/passwd']).map((path) => `            - "${escapeYaml(path)}"`).join('\n');
  return `  kprobes:
  - call: "${call}"
    syscall: false
    return: true
    args:
    - index: 0
      type: "${argType}"
${uidDataBlock(userIds)}
    returnArg:
      index: 0
      type: "int"
    tags:
    - "${name}"
    - "${tag}"
    selectors:
    - matchArgs:
      - index: 0
        operator: Prefix
        values:
${values}${matchBinaries(processNames)}${matchUserIds(userIds)}${matchActions(mode)}`;
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

function uidDataBlock(userIds: string[]) {
  if (!hasUserIds(userIds)) {
    return '';
  }
  return `    data:
    - index: 0
      type: "int"
      source: "current_task"
      resolve: "cred.uid.val"`;
}

function matchUserIds(userIds: string[]) {
  const values = userIds.filter(isUserId);
  if (values.length === 0) {
    return '';
  }
  return `
      matchData:
      - index: 0
        operator: Equal
        values:
${values.map((item) => `        - "${escapeYaml(item)}"`).join('\n')}`;
}

function hasUserIds(userIds: string[]) {
  return userIds.some(isUserId);
}

function isUserId(value: string) {
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

function sanitizeName(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '') || 'diting-tetragon-policy';
}

function escapeYaml(value: string) {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}
