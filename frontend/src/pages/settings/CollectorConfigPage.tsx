import { DeleteOutlined, PlusOutlined, SaveOutlined } from '@ant-design/icons';
import { Button, Card, Divider, Form, Input, Select, Space, Switch, Typography, message } from 'antd';
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
  const rules = Form.useWatch('rules', form) ?? [];
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
        ignoreProcessNames: [],
        ignoreCommandKeywords: [],
        ignoreUsers: [],
        keepSeverities: values.keepSeverities ?? ['high', 'critical'],
        rules: normalizeRules(values.rules ?? []),
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
          description={`当前保留 ${keepSeverities.length ? keepSeverities.join(' / ') : '未指定'} 等级；启用规则 ${compactNumber(rules.filter((rule) => rule?.enabled).length)} 条。`}
        />
      </section>
      <div className="metric-grid">
        <MetricCard label="过滤规则" value={rules.length} hint="Rules" tone="cyan" />
        <MetricCard label="启用规则" value={rules.filter((rule) => rule?.enabled).length} hint="Enabled rules" tone="blue" />
        <MetricCard label="条件数" value={rules.reduce((sum, rule) => sum + (rule?.conditions?.length ?? 0), 0)} hint="Rule conditions" tone="success" />
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
            rules: [],
          }}
        >
          <Form.Item name="enabled" label="启用采集过滤" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="keepSeverities" label="保留风险等级" rules={[{ required: true }]}>
            <Select mode="multiple" options={severityOptions} />
          </Form.Item>
          <Divider />
          <Form.List name="rules">
            {(fields, { add, remove }) => (
              <Space direction="vertical" size={16} style={{ width: '100%' }}>
                {fields.map((field, index) => (
                  <Card
                    key={field.key}
                    size="small"
                    title={`过滤规则 ${index + 1}`}
                    extra={<Button danger type="text" icon={<DeleteOutlined />} onClick={() => remove(field.name)} />}
                  >
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <Form.Item name={[field.name, 'id']} hidden>
                        <Input />
                      </Form.Item>
                      <Space align="start" wrap>
                        <Form.Item name={[field.name, 'enabled']} label="启用" valuePropName="checked" initialValue>
                          <Switch />
                        </Form.Item>
                        <Form.Item name={[field.name, 'name']} label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
                          <Input style={{ width: 260 }} placeholder="例如：忽略 vite node" />
                        </Form.Item>
                      </Space>
                      <Form.List name={[field.name, 'conditions']}>
                        {(conditionFields, conditionOps) => (
                          <Space direction="vertical" size={10} style={{ width: '100%' }}>
                            {conditionFields.map((conditionField) => (
                              <Space key={conditionField.key} align="start" wrap>
                                <Form.Item name={[conditionField.name, 'field']} label="字段" rules={[{ required: true, message: '请选择字段' }]}>
                                  <Select style={{ width: 180 }} options={collectorFilterFieldOptions} />
                                </Form.Item>
                                <Form.Item name={[conditionField.name, 'op']} label="条件" rules={[{ required: true, message: '请选择条件' }]}>
                                  <Select style={{ width: 120 }} options={collectorFilterOpOptions} />
                                </Form.Item>
                                <Form.Item noStyle shouldUpdate>
                                  {({ getFieldValue }) => {
                                    const op = getFieldValue(['rules', field.name, 'conditions', conditionField.name, 'op']);
                                    return op === 'in' ? (
                                      <Form.Item name={[conditionField.name, 'values']} label="取值" rules={[{ required: true, message: '请输入取值' }]}>
                                        <Select mode="tags" tokenSeparators={[',']} style={{ width: 320 }} options={[]} />
                                      </Form.Item>
                                    ) : (
                                      <Form.Item name={[conditionField.name, 'value']} label="取值" rules={[{ required: true, message: '请输入取值' }]}>
                                        <Input style={{ width: 320 }} placeholder="支持精确匹配或包含匹配" />
                                      </Form.Item>
                                    );
                                  }}
                                </Form.Item>
                                <Button danger type="text" icon={<DeleteOutlined />} onClick={() => conditionOps.remove(conditionField.name)} />
                              </Space>
                            ))}
                            <Button
                              icon={<PlusOutlined />}
                              onClick={() => conditionOps.add({ field: 'process_name', op: 'eq', value: '' })}
                            >
                              添加条件
                            </Button>
                          </Space>
                        )}
                      </Form.List>
                    </Space>
                  </Card>
                ))}
                <Button
                  icon={<PlusOutlined />}
                  onClick={() => add({ id: newRuleID(), name: '新过滤规则', enabled: true, conditions: [{ field: 'process_name', op: 'eq', value: '' }] })}
                >
                  添加过滤规则
                </Button>
              </Space>
            )}
          </Form.List>
        </Form>
      </Card>
    </>
  );
}

const collectorFilterFieldOptions = [
  { value: 'event_type', label: '事件类型' },
  { value: 'severity', label: '风险等级' },
  { value: 'process_name', label: '进程名' },
  { value: 'cmdline', label: '命令行' },
  { value: 'username', label: '执行用户' },
  { value: 'login_username', label: '登录用户' },
  { value: 'file_path', label: '文件路径' },
  { value: 'dst_ip', label: '目标 IP' },
  { value: 'dst_port', label: '目标端口' },
];

const collectorFilterOpOptions = [
  { value: 'eq', label: '等于' },
  { value: 'contains', label: '包含' },
  { value: 'in', label: '属于' },
];

function newRuleID() {
  return `filter-${Date.now()}`;
}

function normalizeRules(rules: CollectorFilterConfig['rules']) {
  return rules.map((rule) => ({
    id: rule.id || newRuleID(),
    name: rule.name,
    enabled: Boolean(rule.enabled),
    conditions: (rule.conditions ?? []).map((condition) => ({
      field: condition.field,
      op: condition.op,
      value: condition.op === 'in' ? '' : condition.value ?? '',
      values: condition.op === 'in' ? condition.values ?? [] : [],
    })),
  }));
}
