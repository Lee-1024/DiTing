import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { createRule, deleteRule, getRule, listRules, updateRule } from '../../api/rules';
import type { AuditRule, RuleCondition, RuleExpression, RulePayload } from '../../types/rule';

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

const fieldOptions = [
  { value: 'cmdline', label: '命令行' },
  { value: 'process_name', label: '进程名' },
  { value: 'username', label: '执行用户' },
  { value: 'login_username', label: '登录用户' },
  { value: 'host_name', label: '主机名' },
  { value: 'node_name', label: '节点名' },
  { value: 'namespace', label: 'Namespace' },
  { value: 'pod_name', label: 'Pod' },
  { value: 'container_id', label: '容器 ID' },
  { value: 'binary_path', label: '二进制路径' },
  { value: 'file_path', label: '文件路径' },
  { value: 'dst_ip', label: '目标 IP' },
  { value: 'domain', label: '域名' },
  { value: 'event_type', label: '事件类型' },
];

const opOptions = [
  { value: 'contains', label: '包含' },
  { value: 'eq', label: '等于' },
  { value: 'neq', label: '不等于' },
  { value: 'prefix', label: '前缀匹配' },
  { value: 'suffix', label: '后缀匹配' },
  { value: 'in', label: '属于列表' },
  { value: 'regex', label: '正则匹配' },
];

const severityOptions = [
  { value: 'info', label: '提示' },
  { value: 'low', label: '低危' },
  { value: 'medium', label: '中危' },
  { value: 'high', label: '高危' },
  { value: 'critical', label: '严重' },
];

const eventTypeOptions = [
  { value: 'process_exec', label: '进程执行' },
  { value: 'process_exit', label: '进程退出' },
];

const defaultConditions: FormCondition[] = [{ field: 'cmdline', op: 'contains', conditionValue: 'bash -i' }];

export default function RulesPage() {
  const [rules, setRules] = useState<AuditRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<AuditRule>();
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm<RuleFormValues>();

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
    setOpen(true);
  }

  async function openEdit(rule: AuditRule) {
    const detail = await getRule(rule.id);
    setEditing(detail);
    form.setFieldsValue({
      ...detail,
      tags: detail.tags?.join(',') ?? '',
      operator: detail.matchExpr?.operator ?? 'and',
      conditions: expressionToForm(detail.matchExpr),
    });
    setOpen(true);
  }

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

  async function toggleEnabled(rule: AuditRule, enabled: boolean) {
    await updateRule(rule.id, { ...rule, enabled });
    await load();
  }

  async function removeRule(rule: AuditRule) {
    await deleteRule(rule.id);
    message.success('规则已删除');
    await load();
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
      <Modal title={editing ? '编辑规则' : '新建规则'} open={open} onOk={submit} confirmLoading={saving} onCancel={() => setOpen(false)} width={880}>
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
                      <Select style={{ width: 170 }} options={fieldOptions} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'op']} rules={[{ required: true, message: '请选择匹配方式' }]}>
                      <Select style={{ width: 140 }} options={opOptions} />
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
    </>
  );
}

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

function formToExpression(values: RuleFormValues): RuleExpression {
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

function expressionSummary(expression?: RuleExpression) {
  if (!expression?.conditions?.length) {
    return '-';
  }
  const separator = expression.operator === 'or' ? ' 或 ' : ' 且 ';
  return expression.conditions.map((condition) => {
    const value = condition.op === 'in' ? (condition.values ?? []).join(', ') : condition.value;
    return `${optionLabel(fieldOptions, condition.field)} ${optionLabel(opOptions, condition.op)} ${value ?? ''}`;
  }).join(separator);
}

function optionLabel(options: Array<{ value: string; label: string }>, value: string) {
  return options.find((option) => option.value === value)?.label ?? value;
}
