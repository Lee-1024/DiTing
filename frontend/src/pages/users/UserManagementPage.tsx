import { DeleteOutlined, EditOutlined, KeyOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { createUser, deleteUser, listRoles, listUsers, resetUserPassword, updateUser } from '../../api/userAdmin';
import type { ManagedUser, Role } from '../../types/userAdmin';
import { formatLocalDateTime } from '../../utils/time';

type UserFormValues = {
  username: string;
  password?: string;
  displayName: string;
  email?: string;
  status: 'active' | 'disabled';
  roles: string[];
};

export default function UserManagementPage() {
  const [users, setUsers] = useState<ManagedUser[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<ManagedUser>();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [passwordUser, setPasswordUser] = useState<ManagedUser>();
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm<UserFormValues>();
  const [passwordForm] = Form.useForm<{ password: string }>();

  async function load() {
    setLoading(true);
    try {
      const [nextUsers, nextRoles] = await Promise.all([listUsers(), listRoles()]);
      setUsers(nextUsers);
      setRoles(nextRoles);
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
      username: '',
      password: '',
      displayName: '',
      email: '',
      status: 'active',
      roles: ['admin'],
    });
    setOpen(true);
  }

  function openEdit(user: ManagedUser) {
    setEditing(user);
    form.setFieldsValue({
      username: user.username,
      displayName: user.displayName,
      email: user.email,
      status: user.status,
      roles: user.roles?.length ? user.roles : ['admin'],
    });
    setOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing) {
        await updateUser(editing.id, {
          displayName: values.displayName,
          email: values.email ?? '',
          status: values.status,
          roles: values.roles,
        });
        message.success('用户已更新');
      } else {
        await createUser({
          username: values.username,
          password: values.password ?? '',
          displayName: values.displayName,
          email: values.email ?? '',
          status: values.status,
          roles: values.roles,
        });
        message.success('用户已创建');
      }
      setOpen(false);
      await load();
    } finally {
      setSaving(false);
    }
  }

  function openResetPassword(user: ManagedUser) {
    setPasswordUser(user);
    passwordForm.resetFields();
    setPasswordOpen(true);
  }

  async function submitPassword() {
    if (!passwordUser) {
      return;
    }
    const values = await passwordForm.validateFields();
    setSaving(true);
    try {
      await resetUserPassword(passwordUser.id, values.password);
      message.success('密码已重置');
      setPasswordOpen(false);
    } finally {
      setSaving(false);
    }
  }

  async function remove(user: ManagedUser) {
    await deleteUser(user.id);
    message.success('用户已删除');
    await load();
  }

  const roleOptions = roles.map((role) => ({ value: role.name, label: role.name }));

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">用户管理</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建用户</Button>
      </Space>
      <Card className="data-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={users}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户" /> }}
          scroll={{ x: 1120 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '用户名', dataIndex: 'username', width: 150 },
            { title: '显示名', dataIndex: 'displayName', width: 160 },
            { title: '邮箱', dataIndex: 'email', width: 220, render: (value) => value || '-' },
            {
              title: '状态',
              dataIndex: 'status',
              width: 100,
              render: (value) => <Tag color={value === 'active' ? 'green' : 'default'}>{value === 'active' ? '启用' : '停用'}</Tag>,
            },
            {
              title: '角色',
              dataIndex: 'roles',
              render: (values: string[]) => values?.map((role) => <Tag key={role}>{role}</Tag>),
            },
            { title: '创建时间', dataIndex: 'createdAt', width: 190, render: (value) => formatLocalDateTime(value) },
            {
              title: '操作',
              width: 160,
              fixed: 'right',
              render: (_, record) => (
                <Space>
                  <Button aria-label="编辑用户" icon={<EditOutlined />} onClick={() => openEdit(record)} />
                  <Button aria-label="重置密码" icon={<KeyOutlined />} onClick={() => openResetPassword(record)} />
                  <Popconfirm title="删除用户" onConfirm={() => void remove(record)}>
                    <Button aria-label="删除用户" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Modal title={editing ? '编辑用户' : '新建用户'} open={open} onOk={submit} confirmLoading={saving} onCancel={() => setOpen(false)} width={640}>
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
            <Input disabled={Boolean(editing)} />
          </Form.Item>
          {!editing && (
            <Form.Item name="password" label="初始密码" rules={[{ required: true }, { min: 6 }]}>
              <Input.Password />
            </Form.Item>
          )}
          <Form.Item name="displayName" label="显示名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select options={[{ value: 'active', label: '启用' }, { value: 'disabled', label: '停用' }]} />
          </Form.Item>
          <Form.Item name="roles" label="角色" rules={[{ required: true }]}>
            <Select mode="multiple" options={roleOptions} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal title={passwordUser ? `重置 ${passwordUser.username} 的密码` : '重置密码'} open={passwordOpen} onOk={submitPassword} confirmLoading={saving} onCancel={() => setPasswordOpen(false)}>
        <Form form={passwordForm} layout="vertical">
          <Form.Item name="password" label="新密码" rules={[{ required: true }, { min: 6 }]}>
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
