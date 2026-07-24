import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { Button, Card, DatePicker, Empty, Form, Input, Select, Space, Switch, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { queryAuditEvents } from '../../api/audit';
import CommandText from '../../components/CommandText';
import { InsightHero, LatestPanel, MetricCard } from '../../components/InsightHeader';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent, AuditEventQuery } from '../../types/audit';
import { eventTypeLabel, eventTypeOptions } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';
import EventDetailDrawer from '../audit-events/EventDetailDrawer';

const defaultRange = [dayjs().subtract(1, 'hour'), dayjs()] as const;

// CollectorDebugPage 渲染 Collector Debug Page 组件。
export default function CollectorDebugPage() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<AuditEvent>();
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [total, setTotal] = useState(0);
  const [form] = Form.useForm();
  const requestSeq = useRef(0);

  // buildQuery 构建 build Query 所需的数据结构。
  function buildQuery(values = form.getFieldsValue()): AuditEventQuery {
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.toISOString(),
      end_time: range?.[1]?.toISOString(),
      event_type: values.eventType,
      host_name: values.hostName,
      keyword: values.keyword,
      page: 1,
      page_size: 50,
    };
  }

  // load 加载页面所需数据。
  async function load(values = form.getFieldsValue()) {
    const seq = requestSeq.current + 1;
    requestSeq.current = seq;
    setLoading(true);
    try {
      const data = await queryAuditEvents(buildQuery(values));
      if (seq !== requestSeq.current) {
        return;
      }
      setEvents(data.items ?? []);
      setTotal(data.total);
    } finally {
      if (seq === requestSeq.current) {
        setLoading(false);
      }
    }
  }

  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    if (!autoRefresh) {
      return;
    }
    const timer = window.setInterval(() => {
      void load();
    }, 5000);
    return () => window.clearInterval(timer);
  }, [autoRefresh]);

  const riskyEvents = events.filter((item) => item.severity === 'high' || item.severity === 'critical').length;
  const hostCount = Array.from(new Set(events.map((item) => item.hostName || item.nodeName || item.hostId).filter(Boolean))).length;
  const latestEvent = events[0];

  return (
    <>
      <Space className="page-heading" align="center">
        <div>
          <span className="page-kicker">LIVE EVENT STREAM</span>
          <Typography.Title level={3} className="page-title">采集调试工作台</Typography.Title>
        </div>
        <div className="page-heading-actions">
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>刷新</Button>
          <Switch checked={autoRefresh} onChange={setAutoRefresh} checkedChildren="自动" unCheckedChildren="手动" />
          <Typography.Text type="secondary">最近 {total} 条</Typography.Text>
        </div>
      </Space>
      <div className="collector-hero">
        <InsightHero
          className="debug-summary"
          kicker="Live Collector Debug"
          title="观察最近采集事件，验证规则与链路是否正常"
          description="调试页保留短时间窗口和自动刷新，用于确认 Collector 是否持续产生日志、风险事件是否及时入库。"
          actions={(
            <>
            <Link to="/settings/collector-health"><Button type="primary">采集状态</Button></Link>
            <Link to="/audit/events"><Button ghost>操作日志</Button></Link>
            </>
          )}
        />
        <LatestPanel
          label="最新事件"
          title={latestEvent ? eventTypeLabel(latestEvent.eventType) : '-'}
          description={latestEvent ? latestEvent.cmdline || latestEvent.filePath || latestEvent.processName || '-' : '暂无采集事件'}
        />
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="匹配总量" value={total} hint="当前查询结果" tone="blue" />
        <MetricCard label="当前页事件" value={events.length} hint="最近采集样本" tone="cyan" />
        <MetricCard label="高危事件" value={riskyEvents} hint="需验证规则命中" tone="danger" />
        <MetricCard label="涉及主机" value={hostCount} hint={autoRefresh ? '自动刷新中' : '手动刷新'} tone="success" />
      </div>
      <Card className="data-card live-filter-card">
        <Form
          form={form}
          className="filter-form inline-filter-form"
          layout="horizontal"
          labelCol={{ flex: '78px' }}
          wrapperCol={{ flex: '1 1 0' }}
          colon
          initialValues={{ timeRange: defaultRange, eventType: undefined }}
          onFinish={() => void load()}
        >
          <div className="filter-fields">
            <Form.Item name="timeRange" label="时间" className="filter-field-time">
              <DatePicker.RangePicker showTime />
            </Form.Item>
            <Form.Item name="eventType" label="事件">
              <Select allowClear options={eventTypeOptions} />
            </Form.Item>
            <Form.Item name="hostName" label="主机">
              <Input allowClear placeholder="主机名 / Host ID" />
            </Form.Item>
            <Form.Item name="keyword" label="关键字">
              <Input allowClear placeholder="命令 / 文件 / IP" />
            </Form.Item>
          </div>
          <div className="filter-footer">
            <Space className="filter-actions" size={10} wrap={false}>
              <Button type="primary" icon={<SearchOutlined />} htmlType="submit">查询</Button>
            </Space>
          </div>
        </Form>
        <Table
          rowKey="eventId"
          size="small"
          loading={loading}
          dataSource={events}
          className="clickable-table"
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无采集事件" /> }}
          scroll={{ x: 1500 }}
          pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50] }}
          onRow={(record) => ({ onClick: () => setSelected(record), title: '点击查看采集事件详情' })}
          columns={[
            { title: '时间', dataIndex: 'eventTime', width: 190, fixed: 'left', render: (value) => formatLocalDateTime(value) },
            { title: '等级', dataIndex: 'severity', width: 90, render: (value) => <SeverityTag value={value} /> },
            { title: '事件', dataIndex: 'eventType', width: 120, render: (value) => eventTypeLabel(value) },
            { title: '主机', dataIndex: 'hostName', width: 190, ellipsis: true, render: (_, record) => record.hostName || record.nodeName || record.hostId || '-' },
            { title: '进程', dataIndex: 'processName', width: 150, ellipsis: true },
            { title: '命令', dataIndex: 'cmdline', width: 260, render: (value) => <CommandText value={value} /> },
            { title: '文件路径', dataIndex: 'filePath', width: 320, ellipsis: true, render: (value) => value || '-' },
            { title: '文件操作', dataIndex: 'fileOperation', width: 100, render: (value) => value || '-' },
            { title: '目标地址', width: 180, render: (_, record) => record.dstIp ? `${record.dstIp}:${record.dstPort || ''}` : '-' },
            { title: '协议', dataIndex: 'protocol', width: 80, render: (value) => value || '-' },
          ]}
        />
      </Card>
      <EventDetailDrawer event={selected} open={Boolean(selected)} onClose={() => setSelected(undefined)} />
    </>
  );
}
