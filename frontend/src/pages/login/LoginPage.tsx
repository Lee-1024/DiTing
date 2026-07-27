import { LockOutlined, SafetyCertificateOutlined, UserOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, Space, Typography, message } from 'antd';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { login } from '../../api/auth';
import { saveSession } from '../../stores/auth';

// LoginPage 渲染 Login Page 组件。
export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  // submit 提交当前表单或操作。
  async function submit(values: { username: string; password: string }) {
    setLoading(true);
    try {
      const result = await login(values.username, values.password);
      saveSession(result.token, result.user);
      navigate('/');
    } catch {
      message.error('用户名或密码错误');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-page">
      <Card className="login-card">
        <div className="login-brand">
          <div className="login-logo-frame">
            <img src="/logo.png" alt="DiTing" />
          </div>
          <div>
            <Typography.Title level={3} className="login-title">DiTing</Typography.Title>
            <Typography.Text className="login-subtitle">Runtime Security Operations Console</Typography.Text>
          </div>
        </div>
        <div className="login-signal">
          <SafetyCertificateOutlined />
          <span>审计追踪 · 风险调查 · 主机画像</span>
        </div>
        <Form layout="vertical" onFinish={submit} initialValues={{ username: 'admin' }}>
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
            <Input prefix={<UserOutlined />} placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="请输入密码" />
          </Form.Item>
          <Button block type="primary" htmlType="submit" loading={loading}>登录</Button>
        </Form>
        <Space className="login-footer" size={8}>
          <span>Collector</span>
          <span>Policy</span>
          <span>Audit</span>
        </Space>
      </Card>
    </div>
  );
}
