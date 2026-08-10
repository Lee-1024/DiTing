import { AuditOutlined, CodeOutlined, DashboardOutlined, DownOutlined, FileSearchOutlined, HddOutlined, MonitorOutlined, RobotOutlined, SafetyCertificateOutlined, SettingOutlined, TeamOutlined, ThunderboltOutlined, UserOutlined } from '@ant-design/icons';
import { Button, Dropdown, Form, Input, Layout, Menu, Modal, Space, Typography, message } from 'antd';
import { useState } from 'react';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { changePassword } from '../api/auth';
import NotificationCenter from '../components/NotificationCenter';
import { clearSession, getUser } from '../stores/auth';

const { Header, Sider, Content } = Layout;


// MainLayout 渲染 Main Layout 组件。
export default function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = getUser();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [form] = Form.useForm();

  // logout 处理 logout 相关逻辑。
  function logout() {
    clearSession();
    navigate('/login');
  }

  // submitPassword 提交当前表单或操作。
  async function submitPassword() {
    const values = await form.validateFields();
    setPasswordLoading(true);
    try {
      await changePassword(values.oldPassword, values.newPassword);
      message.success('密码已修改，请重新登录');
      setPasswordOpen(false);
      clearSession();
      navigate('/login');
    } finally {
      setPasswordLoading(false);
    }
  }


  const userMenu = {
    items: [
      { key: 'password', label: '修改密码' },
      { key: 'logout', label: '退出登录' },
    ],
    onClick: ({ key }: { key: string }) => {
      if (key === 'password') {
        setPasswordOpen(true);
      }
      if (key === 'logout') {
        logout();
      }
    },
  };

  return (
    <Layout className="app-shell">
      <Sider width={232} className="app-sidebar">
        <div className="brand">
          <img className="brand-logo" src="/logo-mark.png" alt="DiTing" />
          <span>DiTing</span>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          onClick={({ key }) => navigate(key)}
          items={[
            {
              key: 'audit-analysis',
              type: 'group',
              label: '审计分析',
              children: [
                { key: '/', icon: <DashboardOutlined />, label: '审计概览' },
                { key: '/audit/risks', icon: <ThunderboltOutlined />, label: '风险事件' },
                { key: '/audit/events', icon: <FileSearchOutlined />, label: '操作日志' },
                { key: '/audit/commands', icon: <CodeOutlined />, label: '命令审计' },
                { key: '/audit/users', icon: <UserOutlined />, label: '用户审计' },
                { key: '/audit/hosts', icon: <HddOutlined />, label: '主机审计' },
                { key: '/audit/rules', icon: <SafetyCertificateOutlined />, label: '规则分析' },
              ],
            },
            {
              key: 'config',
              type: 'group',
              label: '配置管理',
              children: [
                { key: '/rules', icon: <SafetyCertificateOutlined />, label: '审计规则' },
                { key: '/settings/users', icon: <TeamOutlined />, label: '用户管理' },
                { key: '/settings/operation-logs', icon: <AuditOutlined />, label: '操作审计' },
                { key: '/settings/collector-health', icon: <MonitorOutlined />, label: '采集状态' },
                { key: '/settings/collector-debug', icon: <FileSearchOutlined />, label: '采集调试' },
                { key: '/settings/collector', icon: <SettingOutlined />, label: '采集配置' },
                { key: '/settings/ai', icon: <RobotOutlined />, label: 'AI 配置' },
                { key: '/settings/tetragon-policies', icon: <SafetyCertificateOutlined />, label: '拦截策略' },
              ],
            },
          ]}
        />
      </Sider>
      <Layout>
        <Header className="app-header">
          <div className="app-header-title">
            <Typography.Text strong>安全运营中心</Typography.Text>
            <Typography.Text type="secondary">DiTing Audit Platform</Typography.Text>
          </div>
          <Space className="header-actions">
            <NotificationCenter onNavigate={navigate} />
            <Dropdown menu={userMenu} trigger={['click']} placement="bottomRight">
              <Button type="text">
                <Space size={6}>
                  <UserOutlined />
                  <span>{user?.displayName || user?.username}</span>
                  <DownOutlined />
                </Space>
              </Button>
            </Dropdown>
          </Space>
        </Header>
        <Content className="app-content">
          <Outlet />
        </Content>
      </Layout>
      <Modal title="修改密码" open={passwordOpen} confirmLoading={passwordLoading} onOk={submitPassword} onCancel={() => setPasswordOpen(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="oldPassword" label="原密码" rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="newPassword" label="新密码" rules={[{ required: true }, { min: 6 }]}>
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
}
