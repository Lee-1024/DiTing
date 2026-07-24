import { ReloadOutlined } from '@ant-design/icons';
import { Button, Card, DatePicker, Empty, Form, Input, Select, Space, Switch, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { queryAuditEvents } from '../../api/audit';
import CommandText from '../../components/CommandText';
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

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">采集调试</Typography.Title>
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>刷新</Button>
        <Switch checked={autoRefresh} onChange={setAutoRefresh} checkedChildren="自动" unCheckedChildren="手动" />
        <Typography.Text type="secondary">最近 {total} 条</Typography.Text>
      </Space>
      <Card className="data-card">
        <Form form={form} layout="inline" initialValues={{ timeRange: defaultRange, eventType: undefined }} style={{ marginBottom: 16 }}>
          <Form.Item name="timeRange" label="时间">
            <DatePicker.RangePicker showTime />
          </Form.Item>
          <Form.Item name="eventType" label="事件">
            <Select allowClear style={{ width: 160 }} options={eventTypeOptions} />
          </Form.Item>
          <Form.Item name="hostName" label="主机">
            <Input allowClear style={{ width: 180 }} placeholder="主机名 / Host ID" />
          </Form.Item>
          <Form.Item name="keyword" label="关键字">
            <Input allowClear style={{ width: 220 }} placeholder="命令 / 文件 / IP" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={() => void load()}>查询</Button>
          </Form.Item>
        </Form>
        <Table
          rowKey="eventId"
          size="small"
          loading={loading}
          dataSource={events}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无采集事件" /> }}
          scroll={{ x: 1500 }}
          pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50] }}
          onRow={(record) => ({ onClick: () => setSelected(record) })}
          columns={[
            { title: '时间', dataIndex: 'eventTime', width: 190, fixed: 'left', render: (value) => formatLocalDateTime(value) },
            { title: '等级', dataIndex: 'severity', width: 90, render: (value) => <SeverityTag value={value} /> },
            { title: '事件', dataIndex: 'eventType', width: 120, render: (value) => eventTypeLabel(value) },
            { title: '主机', dataIndex: 'hostName', width: 170, render: (_, record) => record.hostName || record.nodeName || record.hostId || '-' },
            { title: '进程', dataIndex: 'processName', width: 130 },
            { title: '命令', dataIndex: 'cmdline', width: 260, render: (value) => <CommandText value={value} /> },
            { title: '文件路径', dataIndex: 'filePath', width: 260, render: (value) => value || '-' },
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
