import { CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import {
  createEnforcementPolicy,
  deleteEnforcementPolicy,
  emergencyDisableEnforcementPolicies,
  listEnforcementDeployments,
  listEnforcementPolicies,
  updateEnforcementDeployment,
  updateEnforcementPolicy,
  upsertEnforcementDeployment,
} from '../../api/enforcement';
import type { EnforcementDeployment, EnforcementDeploymentStatus, EnforcementPolicy, EnforcementPolicyPayload } from '../../types/enforcement';

type PolicyTemplate = 'dangerous_command' | 'sensitive_file' | 'permission_change' | 'delete_behavior' | 'suspicious_process';
type PolicyMode = 'audit' | 'enforce' | 'disabled';
type UserMatchMode = 'all' | 'include' | 'exclude_root';
type DeleteMatchMode = 'debug' | 'directory';

interface PolicyFormValues {
  template: PolicyTemplate;
  mode: PolicyMode;
  name: string;
  commands?: string[];
  filePaths?: string[];
  processNames?: string[];
  userMatchMode?: UserMatchMode;
  userIds?: string[];
  deleteMatchMode?: DeleteMatchMode;
  enabled?: boolean;
  description?: string;
  targetHosts?: string[];
}

const defaultValues: PolicyFormValues = {
  template: 'dangerous_command',
  mode: 'audit',
  name: 'diting-dangerous-command',
  description: '',
  enabled: true,
  commands: ['curl', 'wget', 'bash'],
  filePaths: ['/etc/passwd', '/etc/shadow', '/etc/sudoers', '/root/.ssh'],
  processNames: [],
  userMatchMode: 'exclude_root',
  userIds: [],
  deleteMatchMode: 'debug',
  targetHosts: [],
};

export default function TetragonPolicyPage() {
  const [form] = Form.useForm<PolicyFormValues>();
  const [policies, setPolicies] = useState<EnforcementPolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<EnforcementPolicy | null>(null);
  const [deployments, setDeployments] = useState<Record<string, EnforcementDeployment[]>>({});
  const [deploymentForms, setDeploymentForms] = useState<Record<string, Partial<EnforcementDeployment>>>({});
  const template = Form.useWatch('template', form) ?? defaultValues.template;
  const mode = Form.useWatch('mode', form) ?? defaultValues.mode;
  const name = Form.useWatch('name', form) ?? defaultValues.name;
  const description = Form.useWatch('description', form) ?? defaultValues.description;
  const enabled = Form.useWatch('enabled', form) ?? defaultValues.enabled;
  const commands = Form.useWatch('commands', form) ?? defaultValues.commands;
  const filePaths = Form.useWatch('filePaths', form) ?? defaultValues.filePaths;
  const processNames = Form.useWatch('processNames', form) ?? defaultValues.processNames;
  const userMatchMode = Form.useWatch('userMatchMode', form) ?? defaultValues.userMatchMode;
  const userIds = Form.useWatch('userIds', form) ?? defaultValues.userIds;
  const deleteMatchMode = Form.useWatch('deleteMatchMode', form) ?? defaultValues.deleteMatchMode;
  const targetHosts = Form.useWatch('targetHosts', form) ?? defaultValues.targetHosts;
  const policy = useMemo<PolicyFormValues>(() => ({
    template,
    mode,
    name,
    description,
    enabled,
    commands,
    filePaths,
    processNames,
    userMatchMode,
    userIds,
    deleteMatchMode,
    targetHosts,
  }), [template, mode, name, description, enabled, commands, filePaths, processNames, userMatchMode, userIds, deleteMatchMode, targetHosts]);
  const yaml = useMemo(() => generatePolicy(policy), [policy]);

  useEffect(() => {
    void loadPolicies();
  }, []);

  async function loadPolicies() {
    setLoading(true);
    try {
      const nextPolicies = await listEnforcementPolicies();
      setPolicies(nextPolicies);
      await loadDeploymentSummaries(nextPolicies);
    } finally {
      setLoading(false);
    }
  }

  async function loadDeploymentSummaries(nextPolicies: EnforcementPolicy[]) {
    const entries = await Promise.all(
      nextPolicies.map(async (item) => [item.id, await listEnforcementDeployments(item.id)] as const),
    );
    setDeployments(Object.fromEntries(entries));
  }

  async function copyYaml() {
    await navigator.clipboard.writeText(yaml);
    message.success('策略 YAML 已复制');
  }

  function downloadYaml() {
    downloadContent(name || 'diting-tetragon-policy', yaml);
  }

  function downloadContent(fileName: string, content: string) {
    const blob = new Blob([content], { type: 'text/yaml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${fileName || 'diting-tetragon-policy'}.yaml`;
    link.click();
    URL.revokeObjectURL(url);
  }

  async function savePolicy() {
    const values = await form.validateFields();
    const payload: EnforcementPolicyPayload = {
      name: values.name,
      description: values.description ?? '',
      template: values.template,
      mode: values.mode,
      enabled: values.enabled ?? true,
      targetHosts: values.targetHosts ?? [],
      definition: values as unknown as Record<string, unknown>,
      yaml: generatePolicy(values),
      deploymentStatus: editing?.deploymentStatus ?? 'draft',
      deploymentMessage: editing?.deploymentMessage ?? '',
    };
    setSaving(true);
    try {
      if (editing) {
        await updateEnforcementPolicy(editing.id, payload);
        message.success('拦截策略已更新');
      } else {
        await createEnforcementPolicy(payload);
        message.success('拦截策略已保存');
      }
      setEditing(null);
      form.setFieldsValue(defaultValues);
      await loadPolicies();
    } finally {
      setSaving(false);
    }
  }

  function editPolicy(policy: EnforcementPolicy) {
    setEditing(policy);
    const definition = policy.definition as Partial<PolicyFormValues> | undefined;
    form.setFieldsValue({
      ...defaultValues,
      name: policy.name,
      description: policy.description,
      template: policy.template,
      mode: policy.mode,
      enabled: policy.enabled,
      targetHosts: policy.targetHosts,
      ...definition,
    });
  }

  async function removePolicy(id: string) {
    await deleteEnforcementPolicy(id);
    message.success('拦截策略已删除');
    await loadPolicies();
  }

  async function markDeployment(id: string, status: EnforcementDeploymentStatus, deploymentMessage: string) {
    await updateEnforcementDeployment(id, status, deploymentMessage);
    message.success('部署状态已更新');
    await loadPolicies();
  }

  async function loadDeployments(policyId: string) {
    const next = await listEnforcementDeployments(policyId);
    setDeployments((current) => ({ ...current, [policyId]: next }));
  }

  async function saveHostDeployment(policyId: string) {
    const formValue = deploymentForms[policyId] ?? {};
    if (!formValue.hostId || !formValue.status) {
      message.warning('请填写主机 ID 和部署状态');
      return;
    }
    await upsertEnforcementDeployment(policyId, {
      hostId: formValue.hostId,
      hostName: formValue.hostName ?? '',
      status: formValue.status,
      message: formValue.message ?? '',
    });
    message.success('主机部署记录已保存');
    setDeploymentForms((current) => ({ ...current, [policyId]: {} }));
    await loadDeployments(policyId);
  }

  async function emergencyDisable() {
    const result = await emergencyDisableEnforcementPolicies();
    message.success(`已紧急停用 ${result.disabledCount} 条策略`);
    setEditing(null);
    form.setFieldsValue(defaultValues);
    await loadPolicies();
  }

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">拦截策略</Typography.Title>
        <Button type="primary" loading={saving} onClick={() => void savePolicy()}>{editing ? '保存修改' : '保存策略'}</Button>
        {editing && <Button onClick={() => { setEditing(null); form.setFieldsValue(defaultValues); }}>取消编辑</Button>}
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
            <Form.Item name="description" label="策略说明">
              <Input.TextArea rows={2} placeholder="说明这个策略保护的文件、用户范围和部署注意事项" />
            </Form.Item>
            <Form.Item name="enabled" label="启用策略" valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="停用" />
            </Form.Item>
            <Form.Item name="targetHosts" label="适用主机（可选）" tooltip="用于记录这份 YAML 计划部署到哪些主机；当前版本仍需手动放到对应 Tetragon 策略目录。">
              <Select mode="tags" tokenSeparators={[',']} placeholder="例如 server-001 / 10.40.0.184，留空表示通用策略" />
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
                {template === 'delete_behavior' && (
                  <Form.Item name="deleteMatchMode" label="删除匹配模式">
                    <Select options={[
                      { value: 'debug', label: '调试：只采集删除事件' },
                      { value: 'directory', label: '目录保护：按目录范围拦截' },
                    ]} />
                  </Form.Item>
                )}
                <Form.Item
                  name="filePaths"
                  label={template === 'sensitive_file' ? '敏感路径' : '监控路径'}
                  tooltip={template === 'delete_behavior' ? '删除保护建议填写目录。系统会同时保护该目录内删除行为，并保护该目录本身不被删除；单文件精确保护受 Tetragon dentry 匹配限制，按父目录范围处理。' : undefined}
                >
                  <Select mode="tags" tokenSeparators={[',']} />
                </Form.Item>
                <Form.Item name="processNames" label="限定进程（可选）" tooltip="留空表示不限制进程；填写 vim、rm、chmod 等可只拦截指定进程访问这些路径。">
                  <Select mode="tags" tokenSeparators={[',']} placeholder="例如 vim / rm / chmod，留空为不限进程" />
                </Form.Item>
                <Form.Item name="userMatchMode" label="用户范围">
                  <Select options={[
                    { value: 'exclude_root', label: '除 root 外所有用户' },
                    { value: 'include', label: '仅指定 UID' },
                    { value: 'all', label: '所有用户' },
                  ]} />
                </Form.Item>
                {userMatchMode === 'include' && (
                  <Form.Item name="userIds" label="限定执行用户 UID" tooltip="Tetragon 策略按 UID 匹配用户；如 ubuntu 通常为 1000，可在主机上用 id -u ubuntu 查询。">
                    <Select mode="tags" tokenSeparators={[',']} placeholder="例如 1000 / 1001" />
                  </Form.Item>
                )}
                {userMatchMode === 'include' && userIds?.some((item) => item && !isUserId(item)) && (
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
      <Card className="data-card" title="已保存拦截策略" style={{ marginTop: 16 }}>
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message="自动下发依赖 Collector 开启拦截策略同步"
          description="保存并启用策略后，不需要手动点击部署。开启同步的 Collector 会在下个同步周期自动拉取适用于本机的策略，写入本机 Tetragon 策略目录、重启 Tetragon 并上报主机部署结果。"
          action={(
            <Space>
              <Button onClick={() => void loadPolicies()}>刷新同步状态</Button>
              <Popconfirm title="确认紧急停用所有拦截策略？" onConfirm={() => void emergencyDisable()}>
                <Button danger>紧急停用全部</Button>
              </Popconfirm>
            </Space>
          )}
        />
        <Table
          rowKey="id"
          loading={loading}
          dataSource={policies}
          pagination={{ pageSize: 10 }}
          expandable={{
            onExpand: (expanded, record) => {
              if (expanded && !deployments[record.id]) {
                void loadDeployments(record.id);
              }
            },
            expandedRowRender: (record) => (
              <div>
                <Space style={{ marginBottom: 12 }} wrap>
                  <Input
                    style={{ width: 180 }}
                    placeholder="主机 ID"
                    value={deploymentForms[record.id]?.hostId}
                    onChange={(event) => setDeploymentForms((current) => ({ ...current, [record.id]: { ...current[record.id], hostId: event.target.value } }))}
                  />
                  <Input
                    style={{ width: 180 }}
                    placeholder="主机名"
                    value={deploymentForms[record.id]?.hostName}
                    onChange={(event) => setDeploymentForms((current) => ({ ...current, [record.id]: { ...current[record.id], hostName: event.target.value } }))}
                  />
                  <Select
                    style={{ width: 140 }}
                    placeholder="部署状态"
                    value={deploymentForms[record.id]?.status}
                    options={[
                      { value: 'deployed', label: '已部署' },
                      { value: 'failed', label: '加载失败' },
                      { value: 'disabled', label: '已停用' },
                      { value: 'draft', label: '未部署' },
                    ]}
                    onChange={(value) => setDeploymentForms((current) => ({ ...current, [record.id]: { ...current[record.id], status: value } }))}
                  />
                  <Input
                    style={{ width: 260 }}
                    placeholder="部署说明 / 失败原因"
                    value={deploymentForms[record.id]?.message}
                    onChange={(event) => setDeploymentForms((current) => ({ ...current, [record.id]: { ...current[record.id], message: event.target.value } }))}
                  />
                  <Button type="primary" onClick={() => void saveHostDeployment(record.id)}>保存主机记录</Button>
                </Space>
                <Table
                  size="small"
                  rowKey="id"
                  dataSource={deployments[record.id] ?? []}
                  pagination={false}
                  columns={[
                    { title: '主机 ID', dataIndex: 'hostId' },
                    { title: '主机名', dataIndex: 'hostName', render: (value: string) => value || '-' },
                    { title: '状态', dataIndex: 'status', render: deploymentTag },
                    { title: '说明', dataIndex: 'message', render: (value: string) => value || '-' },
                    { title: '部署时间', dataIndex: 'deployedAt', render: (value: string) => formatTime(value) },
                    { title: '更新时间', dataIndex: 'updatedAt', render: (value: string) => formatTime(value) },
                  ]}
                />
              </div>
            ),
          }}
          columns={[
            { title: '策略名称', dataIndex: 'name' },
            { title: '模板', dataIndex: 'template', render: templateLabel },
            { title: '模式', dataIndex: 'mode', render: modeTag },
            { title: '启用', dataIndex: 'enabled', render: (value: boolean) => (value ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>) },
            { title: '适用主机', dataIndex: 'targetHosts', render: (hosts: string[]) => hosts?.length ? hosts.join(', ') : '通用' },
            { title: '自动同步状态', render: (_: unknown, record: EnforcementPolicy) => deploymentSummary(record, deployments[record.id] ?? []) },
            { title: '更新时间', dataIndex: 'updatedAt', render: (value: string) => formatTime(value) },
            {
              title: '操作',
              render: (_: unknown, record: EnforcementPolicy) => (
                <Space>
                  <Button size="small" onClick={() => editPolicy(record)}>编辑</Button>
                  <Button size="small" icon={<DownloadOutlined />} onClick={() => downloadContent(record.name, record.yaml)}>下载</Button>
                  <Button size="small" onClick={() => void markDeployment(record.id, 'deployed', '人工校正为已部署')}>校正已部署</Button>
                  <Button size="small" onClick={() => void markDeployment(record.id, 'failed', '人工校正为加载失败')}>校正失败</Button>
                  <Popconfirm title="确认删除该拦截策略？" onConfirm={() => void removePolicy(record.id)}>
                    <Button size="small" danger>删除</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
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

function templateLabel(value: string) {
  switch (value) {
    case 'sensitive_file':
      return '敏感文件读写';
    case 'permission_change':
      return '权限变更';
    case 'delete_behavior':
      return '删除行为';
    case 'suspicious_process':
      return '可疑进程链路';
    default:
      return '危险命令';
  }
}

function modeTag(value: string) {
  switch (value) {
    case 'enforce':
      return <Tag color="red">拦截</Tag>;
    case 'disabled':
      return <Tag>禁用</Tag>;
    default:
      return <Tag color="blue">仅审计</Tag>;
  }
}

function deploymentTag(value: string) {
  switch (value) {
    case 'deployed':
      return <Tag color="green">已部署</Tag>;
    case 'failed':
      return <Tag color="red">加载失败</Tag>;
    case 'disabled':
      return <Tag>已停用</Tag>;
    default:
      return <Tag color="blue">草稿</Tag>;
  }
}

function deploymentSummary(policy: EnforcementPolicy, deployments: EnforcementDeployment[]) {
  if (!policy.enabled || policy.mode === 'disabled') {
    return <Tag>策略停用</Tag>;
  }
  if (deployments.length === 0) {
    return <Tag color="blue">等待 Collector 同步</Tag>;
  }
  const deployed = deployments.filter((item) => item.status === 'deployed').length;
  const failed = deployments.filter((item) => item.status === 'failed').length;
  const disabled = deployments.filter((item) => item.status === 'disabled').length;
  return (
    <Space size={4} wrap>
      {deployed > 0 && <Tag color="green">{deployed} 已部署</Tag>}
      {failed > 0 && <Tag color="red">{failed} 失败</Tag>}
      {disabled > 0 && <Tag>{disabled} 已停用</Tag>}
      {deployed === 0 && failed === 0 && disabled === 0 && <Tag color="blue">等待同步</Tag>}
    </Space>
  );
}

function formatTime(value: string) {
  return value ? new Date(value).toLocaleString() : '-';
}

function generatePolicy(values: PolicyFormValues) {
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
      return deleteBehaviorBlock(values.filePaths ?? ['/'], values.processNames ?? [], userMatcher(values), values.mode, values.deleteMatchMode ?? 'debug');
    case 'suspicious_process':
      return syscallBlock('suspicious-process', [{ syscall: 'execve', argIndex: 0 }], 'process_exec', values.processNames ?? [], [], null, values.mode, 'Postfix', false);
    default:
      return syscallBlock('dangerous-command', [{ syscall: 'execve', argIndex: 0 }], 'process_exec', values.commands ?? [], [], null, 'audit', 'Postfix', false);
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
    selectors:
    - matchArgs:
      - index: ${argIndex}
        operator: ${operator}
        values:
${matchValues}${matchBinaries(processNames)}${matchUser(user)}${matchActions(mode)}`).join('\n')}`;
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
    selectors:
    - matchArgs:
      - index: 0
        operator: Prefix
        values:
${values}${matchBinaries(processNames)}${matchUser(user)}${matchActions(mode)}`;
}

function deleteBehaviorBlock(paths: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode, matchMode: DeleteMatchMode) {
  if (matchMode === 'directory') {
    return deleteSyscallBlock(paths, processNames, user, mode);
  }
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
    - "delete-debug"
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
    - "delete-debug"`;
}

function deleteSyscallBlock(paths: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode) {
  return `  kprobes:
  - call: "sys_unlink"
    syscall: true
    return: false
    args:
    - index: 0
      type: "string"
${uidDataBlock(user)}
    tags:
    - "delete-behavior"
    - "file_access"
    selectors:
${deleteSyscallSelectors(0, paths, processNames, user, mode)}
  - call: "sys_unlinkat"
    syscall: true
    return: false
    args:
    - index: 1
      type: "string"
${uidDataBlock(user)}
    tags:
    - "delete-behavior"
    - "file_access"
    selectors:
${deleteSyscallSelectors(1, paths, processNames, user, mode)}
  - call: "sys_rmdir"
    syscall: true
    return: false
    args:
    - index: 0
      type: "string"
${uidDataBlock(user)}
    tags:
    - "delete-behavior"
    - "file_access"
    selectors:
${deleteSyscallSelectors(0, paths, processNames, user, mode)}`;
}

function deleteSyscallSelectors(argIndex: number, paths: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode) {
  const selectors: string[] = [];
  const values = deleteSyscallValues(paths).map((item) => `            - "${escapeYaml(item)}"`).join('\n');
  selectors.push(`    - matchArgs:
      - index: ${argIndex}
        operator: Prefix
        values:
${values}${matchBinaries(processNames)}${matchUser(user)}${deleteMatchActions(mode)}`);
  return selectors.join('\n');
}

function deleteSyscallValues(paths: string[]) {
  const result: string[] = [];
  for (const path of paths.filter(Boolean)) {
    const normalized = normalizePath(path);
    const base = baseName(normalized);
    result.push(normalized);
    result.push(`${normalized}/`);
    if (base) {
      result.push(base);
      result.push(`${base}/`);
    }
  }
  return Array.from(new Set(result));
}

function deleteMatchActions(mode: PolicyMode) {
  if (mode !== 'enforce') {
    return '';
  }
  return `
      matchActions:
      - action: Sigkill`;
}

function normalizePath(value: string) {
  const trimmed = value.trim().replace(/\/+$/g, '');
  return trimmed || '/';
}

function parentPath(value: string) {
  const normalized = normalizePath(value);
  const index = normalized.lastIndexOf('/');
  if (index <= 0) {
    return '/';
  }
  return normalized.slice(0, index);
}

function baseName(value: string) {
  const normalized = normalizePath(value);
  const index = normalized.lastIndexOf('/');
  return index >= 0 ? normalized.slice(index + 1) : normalized;
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
      resolve: "cred.uid.val"`;
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
