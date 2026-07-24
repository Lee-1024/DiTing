import { DeleteOutlined, EditOutlined, KeyOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { createUser, deleteUser, listRoles, listUsers, resetUserPassword, updateUser } from '../../api/userAdmin';
import { InsightHero, MetricCard, SummaryPanel } from '../../components/InsightHeader';
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

// UserManagementPage 封装 User Management Page 相关的状态和行为。
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

  // load 加载页面所需数据。
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

  // openCreate 打开对应的弹窗或详情视图。
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

  // openEdit 打开对应的弹窗或详情视图。
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

  // submit 提交当前表单或操作。
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

  // openResetPassword 打开对应的弹窗或详情视图。
  function openResetPassword(user: ManagedUser) {
    setPasswordUser(user);
    passwordForm.resetFields();
    setPasswordOpen(true);
  }

  // submitPassword 提交当前表单或操作。
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

  // remove 删除指定的 remove。
  async function remove(user: ManagedUser) {
    await deleteUser(user.id);
    message.success('用户已删除');
    await load();
  }

  const roleOptions = roles.map((role) => ({ value: role.name, label: role.name }));
  const activeUsers = users.filter((user) => user.status === 'active').length;
  const disabledUsers = users.filter((user) => user.status === 'disabled').length;
  const latestUser = users[0];

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">用户管理</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建用户</Button>
      </Space>
      <section className="system-hero">
        <InsightHero
          kicker="IDENTITY CONTROL"
          title="账号与权限工作台"
          description="管理平台用户、角色授权与密码重置，快速识别停用账号和近期新增身份，减少权限漂移。"
          actions={(
            <>
            <Button ghost icon={<PlusOutlined />} onClick={openCreate}>新建用户</Button>
            <Link to="/settings/operation-logs"><Button ghost>查看操作审计</Button></Link>
            </>
          )}
        />
        <SummaryPanel
          className="user-summary"
          kicker="LATEST IDENTITY"
          title={latestUser ? latestUser.username : '暂无用户'}
          description={latestUser ? `${latestUser.displayName || '未设置显示名'} · ${latestUser.status === 'active' ? '启用' : '停用'}` : '等待账号创建后生成身份摘要'}
        />
      </section>
      <div className="metric-grid">
        <MetricCard label="用户总数" value={users.length} hint="Managed users" tone="cyan" />
        <MetricCard label="启用账号" value={activeUsers} hint="Active identities" tone="success" />
        <MetricCard label="停用账号" value={disabledUsers} hint="Disabled identities" tone="danger" />
        <MetricCard label="角色数量" value={roles.length} hint="Role catalog" tone="blue" />
      </div>
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
            { title: '用户名', dataIndex: 'username', width: 160, ellipsis: true },
            { title: '显示名', dataIndex: 'displayName', width: 170, ellipsis: true },
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
