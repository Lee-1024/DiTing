import { AuditOutlined, BellOutlined, CodeOutlined, DashboardOutlined, DownOutlined, FileSearchOutlined, HddOutlined, MonitorOutlined, SafetyCertificateOutlined, SettingOutlined, TeamOutlined, ThunderboltOutlined, UserOutlined } from '@ant-design/icons';
import { Badge, Button, Dropdown, Form, Input, Layout, List, Menu, Modal, Space, Typography, message } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { queryAuditEvents } from '../api/audit';
import { changePassword } from '../api/auth';
import { listCollectorHealth } from '../api/collectorHealth';
import { clearSession, getUser } from '../stores/auth';
import type { AuditEvent } from '../types/audit';
import type { CollectorHeartbeat } from '../types/collectorHealth';
import { eventTypeLabel } from '../utils/labels';
import { formatLocalDateTime } from '../utils/time';

const { Header, Sider, Content } = Layout;

interface HeaderAlert {
  id: string;
  type: 'risk' | 'collector';
  title: string;
  description: string;
  time?: string;
  target: string;
}

// MainLayout 渲染 Main Layout 组件。
export default function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = getUser();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [alerts, setAlerts] = useState<HeaderAlert[]>([]);
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

  // loadHeaderAlerts 加载页面所需数据。
  async function loadHeaderAlerts() {
    const end = dayjs();
    const [riskData, collectors] = await Promise.all([
      queryAuditEvents({
        start_time: end.subtract(10, 'minute').toISOString(),
        end_time: end.toISOString(),
        severity_in: 'high,critical',
        page: 1,
        page_size: 10,
      }),
      listCollectorHealth(),
    ]);
    const nextAlerts = [
      ...collectorAlerts(collectors),
      ...(riskData.items ?? []).map(riskAlert),
    ].slice(0, 20);
    setAlerts(nextAlerts);
  }

  // openAlertTarget 打开对应的弹窗或详情视图。
  function openAlertTarget(target: string) {
    navigate(target);
  }

  useEffect(() => {
    void loadHeaderAlerts();
    const timer = window.setInterval(() => {
      void loadHeaderAlerts();
    }, 10000);
    return () => window.clearInterval(timer);
  }, []);

  const alertDropdown = (
    <div className="header-alert-dropdown">
      <List
        size="small"
        dataSource={alerts}
        locale={{ emptyText: '暂无高危风险或采集异常' }}
        renderItem={(item) => (
          <List.Item onClick={() => openAlertTarget(item.target)} style={{ cursor: 'pointer' }}>
            <List.Item.Meta
              title={item.title}
              description={
                <Space direction="vertical" size={0}>
                  {item.time && <Typography.Text type="secondary">{formatLocalDateTime(item.time)}</Typography.Text>}
                  <Typography.Text ellipsis style={{ maxWidth: 300 }}>{item.description}</Typography.Text>
                </Space>
              }
            />
          </List.Item>
        )}
      />
    </div>
  );

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
                { key: '/settings/tetragon-policies', icon: <SafetyCertificateOutlined />, label: '拦截策略' },
              ],
            },
          ]}
        />
      </Sider>
      <Layout>
        <Header className="app-header">
          <Typography.Text strong>操作日志审计平台</Typography.Text>
          <Space className="header-actions">
            <Dropdown dropdownRender={() => alertDropdown} trigger={['click']} placement="bottomRight">
              <Badge count={alerts.length} size="small" showZero>
                <Button icon={<BellOutlined />} onClick={(event) => event.preventDefault()}>告警</Button>
              </Badge>
            </Dropdown>
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

// riskAlert 生成 risk Alert 的展示内容。
function riskAlert(event: AuditEvent): HeaderAlert {
  return {
    id: `risk:${event.eventId}`,
    type: 'risk',
    title: `${eventTypeLabel(event.eventType) || event.eventType} / ${event.processName || '-'}`,
    description: event.cmdline || event.filePath || event.dstIp || '-',
    time: event.eventTime,
    target: '/audit/risks',
  };
}

// collectorAlerts 处理 collector Alerts 相关逻辑。
function collectorAlerts(items: CollectorHeartbeat[]): HeaderAlert[] {
  return items
    .filter((item) => item.healthLevel === 'warning' || item.healthLevel === 'critical')
    .map((item) => ({
      id: `collector:${item.hostId || item.hostName}:${item.healthLevel}:${item.message}`,
      type: 'collector',
      title: `${item.healthLevel === 'critical' ? '采集异常' : '采集预警'}：${item.hostName || item.hostId || '-'}`,
      description: item.lastError || item.message || (item.status === 'offline' ? 'Collector 心跳超时' : '采集状态异常'),
      time: item.lastSeenAt,
      target: '/settings/collector-health',
    }));
}
