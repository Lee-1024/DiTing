import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, Modal, Popconfirm, Space, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { createHostAsset, deleteHostAsset, listHostAssets, updateHostAsset } from '../../api/hostAssets';
import type { HostAsset, HostAssetPayload } from '../../types/hostAsset';

// HostAssetsPage 渲染 Host Assets Page 组件。
export default function HostAssetsPage() {
  const [assets, setAssets] = useState<HostAsset[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<HostAsset>();
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

  // load 加载页面所需数据。
  async function load() {
    setLoading(true);
    try {
      setAssets(await listHostAssets());
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
      hostId: '',
      hostName: '',
      hostIp: '',
      environment: '',
      owner: '',
      department: '',
      description: '',
    });
    setOpen(true);
  }

  // openEdit 打开对应的弹窗或详情视图。
  function openEdit(asset: HostAsset) {
    setEditing(asset);
    form.setFieldsValue(asset);
    setOpen(true);
  }

  // toPayload 转换 to Payload 的数据结构。
  function toPayload(values: Record<string, unknown>): HostAssetPayload {
    return {
      hostId: String(values.hostId ?? ''),
      hostName: String(values.hostName ?? ''),
      nodeName: String(values.hostId ?? ''),
      displayName: String(values.hostName ?? ''),
      hostIp: String(values.hostIp ?? ''),
      environment: String(values.environment ?? ''),
      owner: String(values.owner ?? ''),
      department: String(values.department ?? ''),
      description: String(values.description ?? ''),
    };
  }

  // submit 提交当前表单或操作。
  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const payload = toPayload(values);
      if (editing) {
        await updateHostAsset(editing.id, payload);
        message.success('主机资产已更新');
      } else {
        await createHostAsset(payload);
        message.success('主机资产已创建');
      }
      setOpen(false);
      await load();
    } finally {
      setSaving(false);
    }
  }

  // remove 删除指定的 remove。
  async function remove(asset: HostAsset) {
    await deleteHostAsset(asset.id);
    message.success('主机资产已删除');
    await load();
  }

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">主机资产</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增主机</Button>
      </Space>
      <Card className="data-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={assets}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无主机资产" /> }}
          scroll={{ x: 1120 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: 'Host ID', dataIndex: 'hostId' },
            { title: '主机名', dataIndex: 'hostName' },
            { title: 'IP', dataIndex: 'hostIp', width: 150 },
            { title: '环境', dataIndex: 'environment', width: 110, render: (value) => value ? <Tag>{value}</Tag> : null },
            { title: '负责人', dataIndex: 'owner', width: 130 },
            { title: '部门', dataIndex: 'department', width: 130 },
            { title: '备注', dataIndex: 'description' },
            {
              title: '操作',
              width: 120,
              render: (_, record) => (
                <Space>
                  <Button aria-label="编辑主机资产" icon={<EditOutlined />} onClick={() => openEdit(record)} />
                  <Popconfirm title="删除主机资产" onConfirm={() => void remove(record)}>
                    <Button aria-label="删除主机资产" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Modal title={editing ? '编辑主机资产' : '新增主机资产'} open={open} onOk={submit} confirmLoading={saving} onCancel={() => setOpen(false)} width={640}>
        <Form form={form} layout="vertical">
          <Form.Item name="hostId" label="Host ID" rules={[{ required: true }]}>
            <Input placeholder="server-001 / machine-id" />
          </Form.Item>
          <Form.Item name="hostName" label="主机名" rules={[{ required: true }]}>
            <Input placeholder="prod-web-01" />
          </Form.Item>
          <Form.Item name="hostIp" label="IP">
            <Input placeholder="10.0.0.1" />
          </Form.Item>
          <Form.Item name="environment" label="环境">
            <Input placeholder="prod / test / dev" />
          </Form.Item>
          <Form.Item name="owner" label="负责人">
            <Input placeholder="运维组" />
          </Form.Item>
          <Form.Item name="department" label="部门">
            <Input placeholder="平台部" />
          </Form.Item>
          <Form.Item name="description" label="备注">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
