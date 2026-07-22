import { CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, Select, Space, Typography, message } from 'antd';
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
}

const defaultValues: PolicyFormValues = {
  template: 'dangerous_command',
  mode: 'audit',
  name: 'diting-dangerous-command',
  commands: ['rm', 'chmod', 'chown', 'curl', 'wget', 'bash'],
  filePaths: ['/etc/passwd', '/etc/shadow', '/etc/sudoers', '/root/.ssh'],
  processNames: ['bash', 'sh', 'python', 'perl', 'nc', 'ncat', 'socat'],
};

export default function TetragonPolicyPage() {
  const [form] = Form.useForm<PolicyFormValues>();
  const values = Form.useWatch([], form) ?? defaultValues;
  const yaml = useMemo(() => generatePolicy({ ...defaultValues, ...values }), [values]);

  async function copyYaml() {
    await navigator.clipboard.writeText(yaml);
    message.success('策略 YAML 已复制');
  }

  function downloadYaml() {
    const blob = new Blob([yaml], { type: 'text/yaml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${values.name || 'diting-tetragon-policy'}.yaml`;
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
              <Select options={[
                { value: 'dangerous_command', label: '危险命令' },
                { value: 'sensitive_file', label: '敏感文件读写' },
                { value: 'permission_change', label: '权限变更' },
                { value: 'delete_behavior', label: '删除行为' },
                { value: 'suspicious_process', label: '可疑进程链路' },
              ]} />
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
            <Form.Item name="commands" label="命令/关键进程">
              <Select mode="tags" tokenSeparators={[',']} />
            </Form.Item>
            <Form.Item name="filePaths" label="敏感路径">
              <Select mode="tags" tokenSeparators={[',']} />
            </Form.Item>
            <Form.Item name="processNames" label="可疑父子进程">
              <Select mode="tags" tokenSeparators={[',']} />
            </Form.Item>
          </Form>
        </Card>
        <Card className="data-card" title="TracingPolicy YAML">
          <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 13 }}>{yaml}</pre>
        </Card>
      </div>
    </>
  );
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
      return kprobeBlock('file-access', 'security_file_open', 'file_access', 'file', values.filePaths ?? [], values.mode);
    case 'permission_change':
      return syscallBlock('permission-change', ['chmod', 'fchmodat', 'chown', 'fchownat'], 'process_exec', values.commands ?? [], values.mode);
    case 'delete_behavior':
      return syscallBlock('delete-behavior', ['unlink', 'unlinkat', 'rmdir'], 'process_exec', values.commands ?? [], values.mode);
    case 'suspicious_process':
      return syscallBlock('suspicious-process', ['execve'], 'process_exec', values.processNames ?? [], values.mode);
    default:
      return syscallBlock('dangerous-command', ['execve'], 'process_exec', values.commands ?? [], values.mode);
  }
}

function syscallBlock(name: string, _syscalls: string[], returnArg: string, binaries: string[], mode: PolicyMode) {
  const binarySelectors = (binaries.length ? binaries : ['bash']).map((item) => `            - "${escapeYaml(item)}"`).join('\n');
  return `  kprobes:
  - call: "sys_execve"
    syscall: true
    return: true
    args:
    - index: 0
      type: "string"
    returnArg:
      index: 0
      type: "int"
    tags:
    - "${name}"
    - "${returnArg}"
    selectors:
    - matchArgs:
      - index: 0
        operator: Postfix
        values:
${binarySelectors}${matchActions(mode)}`;
}

function kprobeBlock(name: string, call: string, tag: string, argType: string, paths: string[], mode: PolicyMode) {
  const values = (paths.length ? paths : ['/etc/passwd']).map((path) => `            - "${escapeYaml(path)}"`).join('\n');
  return `  kprobes:
  - call: "${call}"
    syscall: false
    return: true
    args:
    - index: 0
      type: "${argType}"
    tags:
    - "${name}"
    - "${tag}"
    selectors:
    - matchArgs:
      - index: 0
        operator: Prefix
        values:
${values}${matchActions(mode)}`;
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
