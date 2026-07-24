import { CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
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
import ActionCluster from '../../components/ActionCluster';
import { InsightHero, MetricCard, SummaryPanel } from '../../components/InsightHeader';
import type { EnforcementDeployment, EnforcementDeploymentStatus, EnforcementPolicy, EnforcementPolicyPayload } from '../../types/enforcement';

type PolicyTemplate = 'dangerous_command' | 'sensitive_file' | 'permission_change' | 'delete_behavior' | 'suspicious_process';
type PolicyMode = 'audit' | 'enforce' | 'disabled';
type UserMatchMode = 'all' | 'include' | 'exclude_root';

interface PolicyFormValues {
  template: PolicyTemplate;
  mode: PolicyMode;
  name: string;
  commands?: string[];
  filePaths?: string[];
  processNames?: string[];
  userMatchMode?: UserMatchMode;
  userIds?: string[];
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
  targetHosts: [],
};

// TetragonPolicyPage 渲染 Tetragon Policy Page 组件。
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
    targetHosts,
  }), [template, mode, name, description, enabled, commands, filePaths, processNames, userMatchMode, userIds, targetHosts]);
  const yaml = useMemo(() => generatePolicy(policy), [policy]);
  const allDeployments = Object.values(deployments).flat();
  const enabledPolicyCount = policies.filter((item) => item.enabled && item.mode !== 'disabled').length;
  const enforcePolicyCount = policies.filter((item) => item.enabled && item.mode === 'enforce').length;
  const failedDeploymentCount = allDeployments.filter((item) => item.status === 'failed').length;

  useEffect(() => {
    void loadPolicies();
  }, []);

  useEffect(() => {
    if (template === 'delete_behavior' && mode === 'enforce') {
      form.setFieldValue('mode', 'audit');
    }
  }, [form, mode, template]);

  // loadPolicies 加载页面所需数据。
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

  // loadDeploymentSummaries 加载页面所需数据。
  async function loadDeploymentSummaries(nextPolicies: EnforcementPolicy[]) {
    const entries = await Promise.all(
      nextPolicies.map(async (item) => [item.id, await listEnforcementDeployments(item.id)] as const),
    );
    setDeployments(Object.fromEntries(entries));
  }

  // copyYaml 复制 copy Yaml 到剪贴板。
  async function copyYaml() {
    await navigator.clipboard.writeText(yaml);
    message.success('策略 YAML 已复制');
  }

  // downloadYaml 导出或下载 download Yaml 数据。
  function downloadYaml() {
    downloadContent(name || 'diting-tetragon-policy', yaml);
  }

  // downloadContent 导出或下载 download Content 数据。
  function downloadContent(fileName: string, content: string) {
    const blob = new Blob([content], { type: 'text/yaml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${fileName || 'diting-tetragon-policy'}.yaml`;
    link.click();
    URL.revokeObjectURL(url);
  }

  // savePolicy 保存或更新 save Policy。
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

  // editPolicy 处理 edit Policy 相关逻辑。
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

  // removePolicy 删除指定的 remove Policy。
  async function removePolicy(id: string) {
    await deleteEnforcementPolicy(id);
    message.success('拦截策略已删除');
    await loadPolicies();
  }

  // markDeployment 处理 mark Deployment 相关逻辑。
  async function markDeployment(id: string, status: EnforcementDeploymentStatus, deploymentMessage: string) {
    await updateEnforcementDeployment(id, status, deploymentMessage);
    message.success('部署状态已更新');
    await loadPolicies();
  }

  // loadDeployments 加载页面所需数据。
  async function loadDeployments(policyId: string) {
    const next = await listEnforcementDeployments(policyId);
    setDeployments((current) => ({ ...current, [policyId]: next }));
  }

  // saveHostDeployment 保存或更新 save Host Deployment。
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

  // emergencyDisable 处理 emergency Disable 相关逻辑。
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
        <div className="page-heading-actions">
          <ActionCluster
            maxVisible={2}
            actions={[
              { key: 'save', label: editing ? '保存修改' : '保存策略', type: 'primary', loading: saving, onClick: () => void savePolicy() },
              ...(editing ? [{ key: 'cancel', label: '取消编辑', onClick: () => { setEditing(null); form.setFieldsValue(defaultValues); } }] : []),
              { key: 'copy', label: '复制 YAML', icon: <CopyOutlined />, onClick: () => void copyYaml() },
              { key: 'download', label: '下载 YAML', icon: <DownloadOutlined />, onClick: downloadYaml },
            ]}
          />
        </div>
      </Space>
      <section className="system-hero">
        <InsightHero
          className="policy-summary"
          kicker="TETRAGON ENFORCEMENT"
          title="运行时拦截策略控制"
          description="将危险命令、敏感路径、权限变更和可疑进程链路沉淀为可审计、可同步、可紧急停用的 Tetragon 策略。"
          actions={(
            <>
            <Link to="/settings/collectors"><Button ghost>检查同步状态</Button></Link>
            <Button ghost icon={<CopyOutlined />} onClick={() => void copyYaml()}>复制当前 YAML</Button>
            </>
          )}
        />
        <SummaryPanel
          kicker="DRAFT PREVIEW"
          title={name || '未命名策略'}
          description={`${templateLabelText(template)} · ${mode === 'enforce' ? '拦截模式' : mode === 'disabled' ? '禁用' : '仅审计'} · ${enabled ? '启用' : '停用'}`}
        />
      </section>
      <div className="metric-grid">
        <MetricCard label="策略总数" value={policies.length} hint="Saved policies" tone="cyan" />
        <MetricCard label="启用策略" value={enabledPolicyCount} hint="Sync eligible" tone="success" />
        <MetricCard label="强拦截" value={enforcePolicyCount} hint="Enforce mode" tone="danger" />
        <MetricCard label="同步失败" value={failedDeploymentCount} hint="Deployment failures" tone="warning" />
      </div>
      <div className="config-grid">
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
                { value: 'enforce', label: '拦截', disabled: template === 'delete_behavior' },
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
                  message={mode === 'enforce' ? '危险命令模板会按进程名拦截' : '危险命令模板适合先审计观察'}
                  description={mode === 'enforce'
                    ? '当前按二进制进程名拦截，适合明确禁止某类命令启动。不要直接拦截 rm、chmod 这类常用命令，避免影响正常运维。'
                    : '该模板按进程执行命令匹配，建议先审计观察；参数级危险组合先进入风险告警，不在这里承诺强拦截。'}
                />
                <Form.Item name="commands" label="命令/关键进程">
                  <Select mode="tags" tokenSeparators={[',']} placeholder="例如 nc / ncat / socat，拦截会在命令启动时生效" />
                </Form.Item>
              </>
            )}
            {(template === 'sensitive_file' || template === 'permission_change' || template === 'delete_behavior') && (
              <>
                {template === 'delete_behavior' && (
                  <Alert
                    type="info"
                    showIcon
                    style={{ marginBottom: 16 }}
                    message="删除行为模板仅用于审计和告警"
                    description="Tetragon 当前不提供稳定的按路径删除强拦截能力。需要阻止 rm -rf / 这类危险操作时，请使用危险命令模板。"
                  />
                )}
                <Form.Item
                  name="filePaths"
                  label={template === 'sensitive_file' ? '敏感路径' : '监控路径'}
                  tooltip={template === 'delete_behavior' ? '仅用于记录删除行为涉及的路径范围和风险命中，不承诺阻止删除。' : undefined}
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
        <Card className="data-card yaml-card" title="TracingPolicy YAML">
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
            { title: '适用主机', dataIndex: 'targetHosts', ellipsis: true, render: (hosts: string[]) => hosts?.length ? hosts.join(', ') : '通用' },
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

// defaultPolicyName 处理 default Policy Name 相关逻辑。
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

// templateLabel 生成 template Label 的展示内容。
function templateLabel(value: string) {
  return templateLabelText(value);
}

// templateLabelText 生成 template Label Text 的展示内容。
function templateLabelText(value: string) {
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

// modeTag 生成 mode Tag 的展示内容。
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

// deploymentTag 生成 deployment Tag 的展示内容。
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

// deploymentSummary 生成 deployment Summary 的展示内容。
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

// formatTime 格式化 format Time 以便界面展示。
function formatTime(value: string) {
  return value ? new Date(value).toLocaleString() : '-';
}

// generatePolicy 处理 generate Policy 相关逻辑。
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

// policyTemplate 处理 policy Template 相关逻辑。
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
      return syscallBlock('dangerous-command', [{ syscall: 'execve', argIndex: 0 }], 'process_exec', values.commands ?? [], [], null, values.mode, 'Postfix', false);
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

// syscallBlock 处理 syscall Block 相关逻辑。
function syscallBlock(name: string, syscalls: SyscallProbe[], tag: string, values: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode, operator: 'Prefix' | 'Postfix', returnProbe: boolean) {
  // matchValues 处理 match Values 相关逻辑。
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

// returnArgBlock 处理 return Arg Block 相关逻辑。
function returnArgBlock(returnProbe: boolean) {
  if (!returnProbe) {
    return '';
  }
  return `    returnArg:
      index: 0
      type: "int"
`;
}

// kprobeBlock 处理 kprobe Block 相关逻辑。
function kprobeBlock(name: string, call: string, tag: string, argType: string, paths: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode) {
  // values 处理 values 相关逻辑。
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

// deleteBehaviorBlock 删除指定的 delete Behavior Block。
function deleteBehaviorBlock(paths: string[], processNames: string[], user: UserMatcher | null) {
  // values 处理 values 相关逻辑。
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

// matchBinaries 处理 match Binaries 相关逻辑。
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

// uidDataBlock 处理 uid Data Block 相关逻辑。
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

// matchUser 处理 match User 相关逻辑。
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

// userMatcher 封装 user Matcher 相关的状态和行为。
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

// isUserId 处理 is User Id 相关逻辑。
function isUserId(value: string) {
  return /^\d+$/.test(value.trim());
}

// matchActions 处理 match Actions 相关逻辑。
function matchActions(mode: PolicyMode) {
  if (mode !== 'enforce') {
    return '';
  }
  return `
      matchActions:
      - action: Sigkill`;
}

// sanitizeName 处理 sanitize Name 相关逻辑。
function sanitizeName(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '') || 'diting-tetragon-policy';
}

// escapeYaml 处理 escape Yaml 相关逻辑。
function escapeYaml(value: string) {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}
