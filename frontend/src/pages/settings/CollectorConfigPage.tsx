import { SaveOutlined } from '@ant-design/icons';
import { Button, Card, Form, Select, Space, Switch, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getCollectorFilterConfig, saveCollectorFilterConfig } from '../../api/systemConfig';
import { InsightHero, MetricCard, SummaryPanel } from '../../components/InsightHeader';
import type { CollectorFilterConfig } from '../../types/systemConfig';
import { compactNumber } from '../../utils/format';
import { severityOptions } from '../../utils/labels';

// CollectorConfigPage 渲染 Collector Config Page 组件。
export default function CollectorConfigPage() {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm<CollectorFilterConfig>();
  const enabled = Form.useWatch('enabled', form);
  const ignoreProcessNames = Form.useWatch('ignoreProcessNames', form) ?? [];
  const ignoreCommandKeywords = Form.useWatch('ignoreCommandKeywords', form) ?? [];
  const ignoreUsers = Form.useWatch('ignoreUsers', form) ?? [];
  const keepSeverities = Form.useWatch('keepSeverities', form) ?? [];

  // load 加载页面所需数据。
  async function load() {
    setLoading(true);
    try {
      form.setFieldsValue(await getCollectorFilterConfig());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  // submit 提交当前表单或操作。
  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const saved = await saveCollectorFilterConfig({
        enabled: Boolean(values.enabled),
        ignoreProcessNames: values.ignoreProcessNames ?? [],
        ignoreCommandKeywords: values.ignoreCommandKeywords ?? [],
        ignoreUsers: values.ignoreUsers ?? [],
        keepSeverities: values.keepSeverities ?? ['high', 'critical'],
      });
      form.setFieldsValue(saved);
      message.success('采集配置已保存');
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">采集配置</Typography.Title>
        <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => void submit()}>保存</Button>
      </Space>
      <section className="system-hero">
        <InsightHero
          kicker="COLLECTOR FILTER CONTROL"
          title="采集过滤策略"
          description="控制 Collector 在进入审计链路前的降噪规则，保留高价值风险等级，减少无效进程、用户和命令噪声。"
          actions={(
            <>
            <Link to="/settings/collectors"><Button ghost>查看采集状态</Button></Link>
            <Link to="/settings/collector-debug"><Button ghost>调试事件流</Button></Link>
            </>
          )}
        />
        <SummaryPanel
          className="collector-summary"
          kicker="CURRENT PROFILE"
          title={enabled ? '过滤已启用' : '过滤未启用'}
          description={`当前保留 ${keepSeverities.length ? keepSeverities.join(' / ') : '未指定'} 等级；忽略项共 ${compactNumber(ignoreProcessNames.length + ignoreCommandKeywords.length + ignoreUsers.length)} 条。`}
        />
      </section>
      <div className="metric-grid">
        <MetricCard label="忽略进程" value={ignoreProcessNames.length} hint="Process names" tone="cyan" />
        <MetricCard label="忽略命令" value={ignoreCommandKeywords.length} hint="Command keywords" tone="blue" />
        <MetricCard label="忽略用户" value={ignoreUsers.length} hint="User filters" tone="success" />
        <MetricCard label="保留等级" value={keepSeverities.length} hint="Risk severities" tone="danger" />
      </div>
      <Card className="data-card config-card" loading={loading}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            enabled: false,
            ignoreProcessNames: [],
            ignoreCommandKeywords: [],
            ignoreUsers: [],
            keepSeverities: ['high', 'critical'],
          }}
        >
          <Form.Item name="enabled" label="启用采集过滤" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="ignoreProcessNames" label="忽略进程名">
            <Select mode="tags" tokenSeparators={[',']} options={[]} />
          </Form.Item>
          <Form.Item name="ignoreCommandKeywords" label="忽略命令关键词">
            <Select mode="tags" tokenSeparators={[',']} options={[]} />
          </Form.Item>
          <Form.Item name="ignoreUsers" label="忽略用户">
            <Select mode="tags" tokenSeparators={[',']} options={[]} />
          </Form.Item>
          <Form.Item name="keepSeverities" label="保留风险等级" rules={[{ required: true }]}>
            <Select mode="multiple" options={severityOptions} />
          </Form.Item>
        </Form>
      </Card>
    </>
  );
}
