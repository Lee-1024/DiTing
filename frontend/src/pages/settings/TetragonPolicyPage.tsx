import { CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from 'antd';
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
import { copyText } from '../../utils/clipboard';
import { generatePolicy, isUserId, type PolicyFormValues, type PolicyTemplate } from './tetragonPolicy';

const defaultValues: PolicyFormValues = {
  template: 'dangerous_command',
  mode: 'audit',
  name: 'diting-dangerous-command',
  description: '',
  enabled: true,
  commands: ['reboot', 'shutdown', 'poweroff', 'halt'],
  commandRuleText: 'systemctl restart|stop docker|docker.service',
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
  const commandRuleText = Form.useWatch('commandRuleText', form) ?? defaultValues.commandRuleText;
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
    commandRuleText,
    filePaths,
    processNames,
    userMatchMode,
    userIds,
    targetHosts,
  }), [template, mode, name, description, enabled, commands, commandRuleText, filePaths, processNames, userMatchMode, userIds, targetHosts]);
  const yaml = useMemo(() => generatePolicy(policy), [policy]);
  const allDeployments = Object.values(deployments).flat();
  const enabledPolicyCount = policies.filter((item) => item.enabled && item.mode !== 'disabled').length;
  const enforcePolicyCount = policies.filter((item) => item.enabled && item.mode === 'enforce').length;
  const failedDeploymentCount = allDeployments.filter((item) => item.status === 'failed').length;

  useEffect(() => {
    void loadPolicies();
  }, []);

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
    await copyText(yaml);
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
            {mode === 'enforce' && (
              <Alert
                type="warning"
                showIcon
                style={{ marginBottom: 16 }}
                message="拦截模式依赖 Tetragon 运行时能力"
                description="排除 root 使用 Linux audit loginuid 匹配：直接 root 登录 loginuid 为 0，普通用户 sudo 后当前 eUID 可能为 0 但 loginuid 仍是原登录 UID。请确认目标主机登录会话能正确写入 audit loginuid。"
              />
            )}
            {template === 'dangerous_command' && (
              <>
                <Alert
                  type={mode === 'enforce' ? 'warning' : 'info'}
                  showIcon
                  style={{ marginBottom: 16 }}
                  message={mode === 'enforce' ? '危险命令模板支持进程名和参数组合拦截' : '危险命令模板适合先审计观察'}
                  description={mode === 'enforce'
                    ? '命令/关键进程用于粗拦截；精细命令规则会同时匹配二进制和参数，适合 systemctl restart docker 这类高危组合。'
                    : '该模板按进程执行命令匹配，也可以先审计参数级危险组合，确认无误后再切换拦截。'}
                />
                <Form.Item name="commands" label="命令/关键进程">
                  <Select mode="tags" tokenSeparators={[',']} placeholder="例如 reboot / shutdown / poweroff，拦截会在命令启动时生效" />
                </Form.Item>
                <Form.Item
                  name="commandRuleText"
                  label="精细命令规则"
                  tooltip="每行一条：第一列是命令，后面依次是 argv 参数；同一位置多个允许值用 | 分隔。"
                >
                  <Input.TextArea rows={4} placeholder="systemctl restart|stop docker|docker.service" />
                </Form.Item>
                <Form.Item name="userMatchMode" label="精细规则登录用户范围">
                  <Select options={[
                    { value: 'exclude_root', label: '除 root 登录会话外所有用户' },
                    { value: 'include', label: '仅指定登录 UID' },
                    { value: 'all', label: '所有用户' },
                  ]} />
                </Form.Item>
                {userMatchMode === 'include' && (
                  <Form.Item name="userIds" label="限定登录用户 UID" tooltip="按 Linux audit loginuid 匹配，可覆盖 sudo 后 uid 变为 0 的场景；如 ubuntu 通常为 1000，可在主机上用 id -u ubuntu 查询。">
                    <Select mode="tags" tokenSeparators={[',']} placeholder="例如 1000 / 1001" />
                  </Form.Item>
                )}
                {userMatchMode === 'include' && userIds?.some((item) => item && !isUserId(item)) && (
                  <Alert
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                    message="限定登录用户需要填写 UID"
                    description="Tetragon 策略按 audit loginuid 匹配，请在目标主机执行 id -u 用户名 后填写数字 UID。非数字项不会写入 YAML。"
                  />
                )}
              </>
            )}
            {(template === 'sensitive_file' || template === 'permission_change' || template === 'delete_behavior') && (
              <>
                {template === 'delete_behavior' && (
                  <Alert
                    type="info"
                    showIcon
                    style={{ marginBottom: 16 }}
                    message="删除行为模板支持按路径拦截"
                    description="拦截模式使用 security_path_unlink 与 security_path_rmdir，并对匹配路径执行 Override 与 Sigkill；部署前需在目标主机确认内核和 Tetragon 版本支持该 hook 与动作。"
                  />
                )}
                <Form.Item
                  name="filePaths"
                  label={template === 'sensitive_file' ? '敏感路径' : '监控路径'}
                  tooltip={template === 'delete_behavior' ? '删除保护按路径精确匹配；目录删除使用 security_path_rmdir，文件删除使用 security_path_unlink。' : undefined}
                >
                  <Select mode="tags" tokenSeparators={[',']} />
                </Form.Item>
                <Form.Item name="processNames" label="限定进程（可选）" tooltip="留空表示不限制进程；填写 vim、rm、chmod 等可只拦截指定进程访问这些路径。">
                  <Select mode="tags" tokenSeparators={[',']} placeholder="例如 vim / rm / chmod，留空为不限进程" />
                </Form.Item>
                <Form.Item name="userMatchMode" label="登录用户范围">
                  <Select options={[
                    { value: 'exclude_root', label: '除 root 登录会话外所有用户' },
                    { value: 'include', label: '仅指定登录 UID' },
                    { value: 'all', label: '所有用户' },
                  ]} />
                </Form.Item>
                {userMatchMode === 'include' && (
                  <Form.Item name="userIds" label="限定登录用户 UID" tooltip="按 Linux audit loginuid 匹配，可覆盖 sudo 后 uid 变为 0 的场景；如 ubuntu 通常为 1000，可在主机上用 id -u ubuntu 查询。">
                    <Select mode="tags" tokenSeparators={[',']} placeholder="例如 1000 / 1001" />
                  </Form.Item>
                )}
                {userMatchMode === 'include' && userIds?.some((item) => item && !isUserId(item)) && (
                  <Alert
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                    message="限定登录用户需要填写 UID"
                    description="Tetragon 策略按 audit loginuid 匹配，请在目标主机执行 id -u 用户名 后填写数字 UID。非数字项不会写入 YAML。"
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
          scroll={{ x: 1320 }}
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
                  scroll={{ x: 920 }}
                  columns={[
                    { title: '主机 ID', dataIndex: 'hostId', width: 180, ellipsis: true },
                    { title: '主机名', dataIndex: 'hostName', width: 160, ellipsis: true, render: (value: string) => value || '-' },
                    { title: '状态', dataIndex: 'status', width: 120, render: deploymentTag },
                    { title: '说明', dataIndex: 'message', width: 220, ellipsis: true, render: (value: string) => value || '-' },
                    { title: '部署时间', dataIndex: 'deployedAt', width: 180, render: (value: string) => formatTime(value) },
                    { title: '更新时间', dataIndex: 'updatedAt', width: 180, render: (value: string) => formatTime(value) },
                  ]}
                />
              </div>
            ),
          }}
          columns={[
            { title: '策略名称', dataIndex: 'name', width: 180, ellipsis: true },
            { title: '模板', dataIndex: 'template', width: 140, render: templateLabel },
            { title: '模式', dataIndex: 'mode', width: 120, render: modeTag },
            { title: '启用', dataIndex: 'enabled', width: 110, render: (value: boolean) => (value ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>) },
            { title: '适用主机', dataIndex: 'targetHosts', width: 180, ellipsis: true, render: (hosts: string[]) => hosts?.length ? hosts.join(', ') : '通用' },
            { title: '自动同步状态', width: 170, render: (_: unknown, record: EnforcementPolicy) => deploymentSummary(record, deployments[record.id] ?? []) },
            { title: '更新时间', dataIndex: 'updatedAt', width: 180, render: (value: string) => formatTime(value) },
            {
              title: '操作',
              width: 280,
              fixed: 'right',
              render: (_: unknown, record: EnforcementPolicy) => (
                <ActionCluster
                  className="policy-action-cluster"
                  maxVisible={2}
                  actions={[
                    { key: 'edit', label: '编辑', onClick: () => editPolicy(record) },
                    { key: 'download', label: '下载', icon: <DownloadOutlined />, onClick: () => downloadContent(record.name, record.yaml) },
                    { key: 'deployed', label: '校正已部署', onClick: () => void markDeployment(record.id, 'deployed', '人工校正为已部署') },
                    { key: 'failed', label: '校正失败', onClick: () => void markDeployment(record.id, 'failed', '人工校正为加载失败') },
                    {
                      key: 'delete',
                      label: '删除',
                      danger: true,
                      onClick: () => {
                        Modal.confirm({
                          title: '确认删除该拦截策略？',
                          okText: '删除',
                          okButtonProps: { danger: true },
                          cancelText: '取消',
                          onOk: () => removePolicy(record.id),
                        });
                      },
                    },
                  ]}
                />
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

