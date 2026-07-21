import { AuditOutlined, BellOutlined, CodeOutlined, DashboardOutlined, FileSearchOutlined, HddOutlined, MonitorOutlined, SafetyCertificateOutlined, SettingOutlined, TeamOutlined, ThunderboltOutlined, UserOutlined } from '@ant-design/icons';
import { Badge, Button, Dropdown, Form, Input, Layout, List, Menu, Modal, Space, Typography, message } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
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

export default function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = getUser();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [alerts, setAlerts] = useState<HeaderAlert[]>([]);
  const knownAlertIds = useRef<Set<string>>(new Set());
  const alertInitialized = useRef(false);
  const [form] = Form.useForm();

  function logout() {
    clearSession();
    navigate('/login');
  }

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
    const nextIds = new Set(nextAlerts.map((item) => item.id));
    if (alertInitialized.current) {
      const fresh = nextAlerts.filter((item) => !knownAlertIds.current.has(item.id));
      if (fresh.length > 0) {
        const latest = fresh[0];
        message.warning(latest.type === 'collector' ? latest.title : `发现新的高危事件：${latest.title}`);
      }
    }
    knownAlertIds.current = nextIds;
    alertInitialized.current = true;
    setAlerts(nextAlerts);
  }

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
    <div style={{ width: 360, maxHeight: 420, overflow: 'auto', padding: 8 }}>
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
              <Badge count={alerts.length} size="small">
                <Button icon={<BellOutlined />} onClick={(event) => event.preventDefault()} />
              </Badge>
            </Dropdown>
            <Typography.Text>{user?.displayName || user?.username}</Typography.Text>
            <Button onClick={() => setPasswordOpen(true)}>修改密码</Button>
            <Button onClick={logout}>退出</Button>
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
