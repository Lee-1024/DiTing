import { SaveOutlined } from '@ant-design/icons';
import { Button, Card, Form, Select, Space, Switch, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { getCollectorFilterConfig, saveCollectorFilterConfig } from '../../api/systemConfig';
import type { CollectorFilterConfig } from '../../types/systemConfig';
import { severityOptions } from '../../utils/labels';

export default function CollectorConfigPage() {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm<CollectorFilterConfig>();

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
      <Card className="data-card" loading={loading}>
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
