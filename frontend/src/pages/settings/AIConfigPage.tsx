import { ExperimentOutlined, SaveOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, InputNumber, Space, Switch, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { getAIProviderConfig, saveAIProviderConfig, testAIProviderConfig } from '../../api/systemConfig';
import { InsightHero, MetricCard, SummaryPanel } from '../../components/InsightHeader';
import type { AIProviderConfig } from '../../types/systemConfig';

export default function AIConfigPage() {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ ok: boolean; latencyMs: number; message: string }>();
  const [config, setConfig] = useState<AIProviderConfig>();
  const [form] = Form.useForm<AIProviderConfig>();
  const enabled = Form.useWatch('enabled', form);

  async function load() {
    setLoading(true);
    try {
      const data = await getAIProviderConfig();
      setConfig(data);
      form.setFieldsValue({ ...data, apiKey: '' });
    } finally {
      setLoading(false);
    }
  }

  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const saved = await saveAIProviderConfig(values);
      setConfig(saved);
      form.setFieldsValue({ ...saved, apiKey: '' });
      message.success('AI 配置已保存');
    } finally {
      setSaving(false);
    }
  }

  async function testConnection() {
    const values = await form.validateFields(['baseUrl', 'model', 'apiKey', 'timeoutSeconds', 'maxTokens']);
    setTesting(true);
    setTestResult(undefined);
    try {
      const result = await testAIProviderConfig({ ...form.getFieldsValue(), ...values });
      setTestResult(result);
      message.success(`模型服务可用，耗时 ${result.latencyMs}ms`);
    } catch (error: any) {
      message.error(error?.response?.data || error?.message || '模型服务测试失败');
    } finally {
      setTesting(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">AI 配置</Typography.Title>
        <Button icon={<ExperimentOutlined />} loading={testing} onClick={() => void testConnection()}>测试连接</Button>
        <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => void submit()}>保存</Button>
      </Space>
      <section className="system-hero">
        <InsightHero
          kicker="AI RISK REVIEW"
          title="OpenAI-compatible 风险复核"
          description="配置兼容 OpenAI Chat Completions 的模型服务，用于风险事件人工处理前的辅助判断。"
        />
        <SummaryPanel
          className="collector-summary"
          kicker="CURRENT PROVIDER"
          title={enabled ? 'AI 复核已启用' : 'AI 复核未启用'}
          description={config?.apiKeySet ? `密钥已保存：${config.maskedApiKey || '********'}` : '尚未保存 API Key'}
        />
      </section>
      <div className="metric-grid">
        <MetricCard label="启用状态" value={enabled ? '启用' : '停用'} hint="AI review" tone={enabled ? 'success' : 'default'} />
        <MetricCard label="模型" value={form.getFieldValue('model') || '-'} hint="Model" tone="blue" />
        <MetricCard label="超时" value={form.getFieldValue('timeoutSeconds') || '-'} hint="Seconds" tone="cyan" />
      </div>
      <Card className="data-card config-card" loading={loading}>
        <Alert
          showIcon
          type={testResult?.ok ? 'success' : 'info'}
          style={{ marginBottom: 18 }}
          message={testResult?.ok ? `模型服务可用，耗时 ${testResult.latencyMs}ms` : 'API Key 会加密入库，前端不会回显明文'}
          description={testResult?.ok
            ? '测试使用当前表单配置临时调用模型，不会保存配置。'
            : '保存时填写新 Key 会覆盖旧 Key；留空则保留当前已保存 Key。测试连接时若 Key 留空且已有保存密钥，会使用后端已保存密钥。'}
        />
        <Form
          form={form}
          layout="vertical"
          initialValues={{ enabled: false, baseUrl: 'https://api.minimaxi.com/v1', model: 'MiniMax-M3', timeoutSeconds: 120, maxTokens: 800 }}
        >
          <Form.Item name="enabled" label="启用 AI 复核" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="baseUrl" label="Base URL" rules={[{ required: enabled, message: '请输入模型服务 Base URL' }]}>
            <Input placeholder="例如：https://api.minimaxi.com/v1" />
          </Form.Item>
          <Form.Item name="model" label="模型" rules={[{ required: enabled, message: '请输入模型名称' }]}>
            <Input placeholder="例如：MiniMax-M3 / deepseek-chat" />
          </Form.Item>
          <Form.Item name="apiKey" label={config?.apiKeySet ? `API Key（当前 ${config.maskedApiKey || '已保存'}）` : 'API Key'}>
            <Input.Password placeholder={config?.apiKeySet ? '留空保留当前密钥' : '请输入 API Key，可为空用于无鉴权内网模型'} />
          </Form.Item>
          <Space align="start" wrap>
            <Form.Item name="timeoutSeconds" label="超时秒数" rules={[{ required: true }]}>
              <InputNumber min={10} max={900} />
            </Form.Item>
            <Form.Item name="maxTokens" label="最大输出 Token" rules={[{ required: true }]}>
              <InputNumber min={200} max={4000} />
            </Form.Item>
          </Space>
        </Form>
      </Card>
    </>
  );
}
