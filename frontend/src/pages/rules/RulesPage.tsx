import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { createRule, deleteRule, getRule, listRules, updateRule } from '../../api/rules';
import type { AuditRule, RulePayload } from '../../types/rule';

const defaultMatchExpr = '{"operator":"and","conditions":[{"field":"cmdline","op":"contains","value":"bash -i"}]}';

export default function RulesPage() {
  const [rules, setRules] = useState<AuditRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<AuditRule>();
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

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
      matchExpr: defaultMatchExpr,
    });
    setOpen(true);
  }

  async function openEdit(rule: AuditRule) {
    const detail = await getRule(rule.id);
    setEditing(detail);
    form.setFieldsValue({
      ...detail,
      tags: detail.tags?.join(',') ?? '',
      matchExpr: JSON.stringify(detail.matchExpr, null, 2),
    });
    setOpen(true);
  }

  function toPayload(values: Record<string, unknown>): RulePayload {
    return {
      name: String(values.name ?? ''),
      description: String(values.description ?? ''),
      eventType: String(values.eventType ?? ''),
      enabled: Boolean(values.enabled),
      severity: String(values.severity ?? ''),
      riskScore: Number(values.riskScore ?? 0),
      matchExpr: JSON.parse(String(values.matchExpr ?? '{}')),
      tags: values.tags ? String(values.tags).split(',').map((item) => item.trim()).filter(Boolean) : [],
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
    } catch (error) {
      message.error(error instanceof SyntaxError ? '匹配表达式不是合法 JSON' : '保存失败');
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
          scroll={{ x: 1120 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '名称', dataIndex: 'name' },
            { title: '事件类型', dataIndex: 'eventType', width: 140 },
            { title: '等级', dataIndex: 'severity', width: 100, render: (value) => <Tag>{value}</Tag> },
            { title: '分数', dataIndex: 'riskScore', width: 80 },
            { title: '启用', dataIndex: 'enabled', width: 90, render: (value, record) => <Switch checked={value} onChange={(checked) => void toggleEnabled(record, checked)} /> },
            { title: '标签', dataIndex: 'tags', render: (tags: string[]) => tags?.map((tag) => <Tag key={tag}>{tag}</Tag>) },
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
      <Modal title={editing ? '编辑规则' : '新建规则'} open={open} onOk={submit} confirmLoading={saving} onCancel={() => setOpen(false)} width={720}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="eventType" label="事件类型" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="severity" label="风险等级" rules={[{ required: true }]}>
            <Select options={['info', 'low', 'medium', 'high', 'critical'].map((value) => ({ value }))} />
          </Form.Item>
          <Form.Item name="riskScore" label="风险分数" rules={[{ required: true }]}>
            <InputNumber min={0} max={100} />
          </Form.Item>
          <Form.Item name="tags" label="标签">
            <Input placeholder="reverse-shell,suspicious-command" />
          </Form.Item>
          <Form.Item name="matchExpr" label="匹配表达式 JSON" rules={[{ required: true }]}>
            <Input.TextArea rows={8} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
