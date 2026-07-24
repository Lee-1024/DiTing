import { DeleteOutlined, EditOutlined, ExperimentOutlined, PlusOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Empty, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { createRule, deleteRule, getRule, listRules, testRule, updateRule } from '../../api/rules';
import type { AuditRule, RuleCondition, RuleExpression, RulePayload, RuleTestEvent, RuleTestResponse } from '../../types/rule';
import { eventTypeOptions, optionLabel, ruleFieldLabel, ruleFieldOptions, ruleOperatorLabel, ruleOperatorOptions, severityOptions } from '../../utils/labels';

interface RuleFormValues {
  name: string;
  description?: string;
  eventType: string;
  enabled: boolean;
  severity: string;
  riskScore: number;
  tags?: string;
  operator: 'and' | 'or';
  conditions: FormCondition[];
}

interface FormCondition {
  field?: string;
  op?: string;
  conditionValue?: string;
}

const defaultConditions: FormCondition[] = [{ field: 'cmdline', op: 'contains', conditionValue: 'bash -i' }];

// RulesPage 渲染 Rules Page 组件。
export default function RulesPage() {
  const [rules, setRules] = useState<AuditRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [open, setOpen] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [testResult, setTestResult] = useState<RuleTestResponse>();
  const [editing, setEditing] = useState<AuditRule>();
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm<RuleFormValues>();
  const [testForm] = Form.useForm<RuleTestEvent>();

  // load 加载页面所需数据。
  async function load() {
    setLoading(true);
    try {
      setRules(await listRules());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  // openCreate 打开对应的弹窗或详情视图。
  function openCreate() {
    setEditing(undefined);
    form.setFieldsValue({
      name: '',
      description: '',
      enabled: true,
      severity: 'high',
      eventType: 'process_exec',
      riskScore: 80,
      tags: '',
      operator: 'and',
      conditions: defaultConditions,
    });
    setTestResult(undefined);
    setOpen(true);
  }

  // openEdit 打开对应的弹窗或详情视图。
  async function openEdit(rule: AuditRule) {
    const detail = await getRule(rule.id);
    setEditing(detail);
    form.setFieldsValue({
      ...detail,
      tags: detail.tags?.join(',') ?? '',
      operator: detail.matchExpr?.operator ?? 'and',
      conditions: expressionToForm(detail.matchExpr),
    });
    setTestResult(undefined);
    setOpen(true);
  }

  // toPayload 转换 to Payload 的数据结构。
  function toPayload(values: RuleFormValues): RulePayload {
    return {
      name: values.name,
      description: values.description ?? '',
      eventType: values.eventType,
      enabled: Boolean(values.enabled),
      severity: values.severity,
      riskScore: Number(values.riskScore ?? 0),
      matchExpr: formToExpression(values),
      tags: values.tags ? values.tags.split(',').map((item) => item.trim()).filter(Boolean) : [],
    };
  }

  // submit 提交当前表单或操作。
  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const payload = toPayload(values);
      if (editing) {
        await updateRule(editing.id, payload);
        message.success('规则已更新');
      } else {
        await createRule(payload);
        message.success('规则已创建');
      }
      setOpen(false);
      await load();
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  }

  // toggleEnabled 切换 toggle Enabled 状态。
  async function toggleEnabled(rule: AuditRule, enabled: boolean) {
    await updateRule(rule.id, { ...rule, enabled });
    await load();
  }

  // removeRule 删除指定的 remove Rule。
  async function removeRule(rule: AuditRule) {
    await deleteRule(rule.id);
    message.success('规则已删除');
    await load();
  }

  // openTest 打开对应的弹窗或详情视图。
  async function openTest() {
    await form.validateFields();
    const values = form.getFieldsValue();
    testForm.setFieldsValue({
      eventType: values.eventType,
      severity: values.severity,
      hostId: 'host-001',
      hostName: 'server-001',
      nodeName: 'node-1',
      username: 'ubuntu',
      loginUsername: 'ubuntu',
      processName: values.eventType === 'network_connect' ? 'curl' : 'bash',
      cmdline: values.eventType === 'network_connect' ? '/usr/bin/curl https://10.0.0.8' : '/bin/bash -i',
      dstIp: values.eventType === 'network_connect' ? '10.0.0.8' : undefined,
      dstPort: values.eventType === 'network_connect' ? 443 : undefined,
      protocol: values.eventType === 'network_connect' ? 'tcp' : undefined,
    });
    setTestResult(undefined);
    setTestOpen(true);
  }

  // submitTest 提交当前表单或操作。
  async function submitTest() {
    const ruleValues = await form.validateFields();
    const eventValues = await testForm.validateFields();
    setTesting(true);
    try {
      setTestResult(await testRule(toPayload(ruleValues), eventValues));
    } catch {
      message.error('规则测试失败');
    } finally {
      setTesting(false);
    }
  }

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">审计规则</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建规则</Button>
      </Space>
      <Card className="data-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={rules}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无审计规则" /> }}
          scroll={{ x: 1320 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '名称', dataIndex: 'name', width: 180 },
            { title: '事件类型', dataIndex: 'eventType', width: 120, render: (value) => optionLabel(eventTypeOptions, value) },
            { title: '等级', dataIndex: 'severity', width: 90, render: (value) => <Tag>{optionLabel(severityOptions, value)}</Tag> },
            { title: '分数', dataIndex: 'riskScore', width: 80 },
            { title: '匹配条件', dataIndex: 'matchExpr', render: (value: RuleExpression) => expressionSummary(value) },
            { title: '启用', dataIndex: 'enabled', width: 90, render: (value, record) => <Switch checked={value} onChange={(checked) => void toggleEnabled(record, checked)} /> },
            { title: '标签', dataIndex: 'tags', width: 180, render: (tags: string[]) => tags?.map((tag) => <Tag key={tag}>{tag}</Tag>) },
            {
              title: '操作',
              width: 120,
              render: (_, record) => (
                <Space>
                  <Button aria-label="编辑规则" icon={<EditOutlined />} onClick={() => void openEdit(record)} />
                  <Popconfirm title="删除规则" onConfirm={() => void removeRule(record)}>
                    <Button aria-label="删除规则" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Modal
        title={editing ? '编辑规则' : '新建规则'}
        open={open}
        confirmLoading={saving}
        onCancel={() => setOpen(false)}
        width={880}
        footer={[
          <Button key="test" icon={<ExperimentOutlined />} onClick={() => void openTest()}>测试规则</Button>,
          <Button key="cancel" onClick={() => setOpen(false)}>取消</Button>,
          <Button key="submit" type="primary" loading={saving} onClick={() => void submit()}>保存</Button>,
        ]}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Space size={16} align="start" wrap>
            <Form.Item name="eventType" label="事件类型" rules={[{ required: true, message: '请选择事件类型' }]}>
              <Select style={{ width: 160 }} options={eventTypeOptions} />
            </Form.Item>
            <Form.Item name="severity" label="风险等级" rules={[{ required: true, message: '请选择风险等级' }]}>
              <Select style={{ width: 140 }} options={severityOptions} />
            </Form.Item>
            <Form.Item name="riskScore" label="风险分数" rules={[{ required: true, message: '请输入风险分数' }]}>
              <InputNumber min={0} max={100} style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
          <Form.Item name="tags" label="标签">
            <Input placeholder="reverse-shell,suspicious-command" />
          </Form.Item>
          <Form.Item name="operator" label="条件关系" rules={[{ required: true }]}>
            <Select
              style={{ width: 160 }}
              options={[
                { value: 'and', label: '全部满足' },
                { value: 'or', label: '任一满足' },
              ]}
            />
          </Form.Item>
          <Form.List name="conditions" rules={[{ validator: async (_, value) => (value?.length ? undefined : Promise.reject(new Error('至少添加一个条件'))) }]}>
            {(fields, { add, remove }, { errors }) => (
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {fields.map((field) => (
                  <Space key={field.key} align="baseline" wrap>
                    <Form.Item name={[field.name, 'field']} rules={[{ required: true, message: '请选择字段' }]}>
                      <Select style={{ width: 170 }} options={ruleFieldOptions} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'op']} rules={[{ required: true, message: '请选择匹配方式' }]}>
                      <Select style={{ width: 140 }} options={ruleOperatorOptions} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'conditionValue']} rules={[{ required: true, message: '请输入匹配值' }]}>
                      <Input style={{ width: 360 }} />
                    </Form.Item>
                    <Button aria-label="删除条件" icon={<DeleteOutlined />} onClick={() => remove(field.name)} />
                  </Space>
                ))}
                <Form.ErrorList errors={errors} />
                <Button icon={<PlusOutlined />} onClick={() => add({ field: 'cmdline', op: 'contains', conditionValue: '' })}>添加条件</Button>
              </Space>
            )}
          </Form.List>
        </Form>
      </Modal>
      <Modal
        title="测试规则"
        open={testOpen}
        confirmLoading={testing}
        onOk={() => void submitTest()}
        onCancel={() => setTestOpen(false)}
        width={760}
      >
        <Form form={testForm} layout="vertical">
          <Space size={16} align="start" wrap>
            <Form.Item name="eventType" label="事件类型">
              <Select style={{ width: 160 }} options={eventTypeOptions} />
            </Form.Item>
            <Form.Item name="severity" label="风险等级">
              <Select style={{ width: 140 }} options={severityOptions} />
            </Form.Item>
            <Form.Item name="username" label="执行用户">
              <Input style={{ width: 160 }} />
            </Form.Item>
            <Form.Item name="loginUsername" label="登录用户">
              <Input style={{ width: 160 }} />
            </Form.Item>
          </Space>
          <Space size={16} align="start" wrap>
            <Form.Item name="processName" label="进程名">
              <Input style={{ width: 160 }} />
            </Form.Item>
            <Form.Item name="binaryPath" label="二进制路径">
              <Input style={{ width: 240 }} />
            </Form.Item>
            <Form.Item name="hostName" label="主机名">
              <Input style={{ width: 160 }} />
            </Form.Item>
            <Form.Item name="hostId" label="Host ID">
              <Input style={{ width: 160 }} />
            </Form.Item>
            <Form.Item name="namespace" label="Namespace">
              <Input style={{ width: 160 }} />
            </Form.Item>
          </Space>
          <Form.Item name="cmdline" label="命令行">
            <Input />
          </Form.Item>
          <Space size={16} align="start" wrap>
            <Form.Item name="podName" label="Pod">
              <Input style={{ width: 180 }} />
            </Form.Item>
            <Form.Item name="containerId" label="容器 ID">
              <Input style={{ width: 220 }} />
            </Form.Item>
            <Form.Item name="filePath" label="文件路径">
              <Input style={{ width: 220 }} />
            </Form.Item>
            <Form.Item name="fileOperation" label="文件操作">
              <Input style={{ width: 140 }} />
            </Form.Item>
          </Space>
          <Space size={16} align="start" wrap>
            <Form.Item name="dstIp" label="目标 IP">
              <Input style={{ width: 180 }} />
            </Form.Item>
            <Form.Item name="dstPort" label="目标端口">
              <InputNumber min={0} max={65535} style={{ width: 130 }} />
            </Form.Item>
            <Form.Item name="protocol" label="网络协议">
              <Select style={{ width: 120 }} options={[{ value: 'tcp', label: 'TCP' }, { value: 'udp', label: 'UDP' }]} allowClear />
            </Form.Item>
          </Space>
          {testResult && (
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Alert type={testResult.matched ? 'success' : 'warning'} showIcon message={testResult.matched ? '规则命中' : '规则未命中'} />
              {testResult.matches?.length ? (
                <Table
                  size="small"
                  pagination={false}
                  rowKey={(record, index) => `${record.field}-${record.operator}-${index}`}
                  dataSource={testResult.matches}
                  columns={[
                    { title: '字段', dataIndex: 'field', width: 130, render: (value) => ruleFieldLabel(value) },
                    { title: '条件', dataIndex: 'operator', width: 120, render: (value) => ruleOperatorLabel(value) },
                    { title: '期望值', dataIndex: 'value', width: 180 },
                    { title: '实际值', dataIndex: 'actual' },
                  ]}
                />
              ) : null}
            </Space>
          )}
        </Form>
      </Modal>
    </>
  );
}

// expressionToForm 处理 expression To Form 相关逻辑。
function expressionToForm(expression?: RuleExpression): FormCondition[] {
  if (!expression?.conditions?.length) {
    return defaultConditions;
  }
  return expression.conditions.map((condition) => ({
    field: condition.field,
    op: condition.op,
    conditionValue: condition.op === 'in' ? (condition.values ?? []).join(',') : String(condition.value ?? ''),
  }));
}

// formToExpression 处理 form To Expression 相关逻辑。
function formToExpression(values: RuleFormValues): RuleExpression {
  // conditions 处理 conditions 相关逻辑。
  const conditions = (values.conditions ?? []).map((condition) => {
    const value = String(condition.conditionValue ?? '').trim();
    const next: RuleCondition = {
      field: String(condition.field ?? ''),
      op: String(condition.op ?? ''),
      value,
    };
    if (next.op === 'in') {
      next.values = value.split(',').map((item) => item.trim()).filter(Boolean);
      delete next.value;
    }
    return next;
  });
  return { operator: values.operator ?? 'and', conditions };
}

// expressionSummary 处理 expression Summary 相关逻辑。
function expressionSummary(expression?: RuleExpression) {
  if (!expression?.conditions?.length) {
    return '-';
  }
  const separator = expression.operator === 'or' ? ' 或 ' : ' 且 ';
  return expression.conditions.map((condition) => {
    const value = condition.op === 'in' ? (condition.values ?? []).join(', ') : condition.value;
    return `${optionLabel(ruleFieldOptions, condition.field)} ${optionLabel(ruleOperatorOptions, condition.op)} ${value ?? ''}`;
  }).join(separator);
}
