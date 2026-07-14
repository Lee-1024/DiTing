import { Card, DatePicker, Empty, Form, Input, Select, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent, AuditEventQuery } from '../../types/audit';
import { downloadBlob } from '../../utils/download';
import { formatLocalDateTime } from '../../utils/time';
import EventDetailDrawer from './EventDetailDrawer';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

export default function AuditEventsPage() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<AuditEvent>();
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [form] = Form.useForm();
  const requestSeq = useRef(0);

  function buildQuery(nextPage = page, nextPageSize = pageSize, formValues = form.getFieldsValue()): AuditEventQuery {
    const values = formValues;
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      event_type: values.eventType,
      severity: values.severity,
      keyword: values.keyword,
      page: nextPage,
      page_size: nextPageSize,
    };
  }

  async function load(nextPage = page, nextPageSize = pageSize, formValues = form.getFieldsValue()) {
    const seq = requestSeq.current + 1;
    requestSeq.current = seq;
    setLoading(true);
    try {
      const data = await queryAuditEvents(buildQuery(nextPage, nextPageSize, formValues));
      if (seq !== requestSeq.current) {
        return;
      }
      setEvents(data.items ?? []);
      setTotal(data.total);
      setPage(data.page);
      setPageSize(nextPageSize);
    } finally {
      if (seq === requestSeq.current) {
        setLoading(false);
      }
    }
  }

  function submit() {
    void load(1, pageSize, form.getFieldsValue());
  }

  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load(1, 10, form.getFieldsValue());
  }

  async function exportCSV() {
    const blob = await exportAuditEvents(buildQuery(1, 5000));
    downloadBlob(blob, 'audit-events.csv');
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">操作日志</Typography.Title>
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={submit} onReset={() => void resetAndLoad()} onExport={() => void exportCSV()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="eventType" label="事件">
          <Select className="filter-control-compact" allowClear options={['process_exec', 'process_exit'].map((value) => ({ value }))} />
        </Form.Item>
        <Form.Item name="severity" label="等级">
          <Select className="filter-control-compact" allowClear options={['info', 'low', 'medium', 'high', 'critical'].map((value) => ({ value }))} />
        </Form.Item>
        <Form.Item name="keyword" label="关键字">
          <Input className="filter-control-compact" placeholder="命令 / 用户 / 进程" allowClear />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey="eventId"
          loading={loading}
          dataSource={events}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无审计事件" /> }}
          scroll={{ x: 1320 }}
          onRow={(record) => ({ onClick: () => setSelected(record) })}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onChange: (nextPage, nextPageSize) => {
              const sizeChanged = nextPageSize !== pageSize;
              void load(sizeChanged ? 1 : nextPage, nextPageSize, form.getFieldsValue());
            },
          }}
          columns={[
            { title: '时间', dataIndex: 'eventTime', width: 190, fixed: 'left', render: (value) => formatLocalDateTime(value) },
            { title: '等级', dataIndex: 'severity', width: 100, render: (value) => <SeverityTag value={value} /> },
            { title: '事件', dataIndex: 'eventType', width: 140 },
            { title: 'Namespace', dataIndex: 'namespace', width: 140 },
            { title: 'Pod', dataIndex: 'podName', width: 180 },
            { title: '登录用户', dataIndex: 'loginUsername', width: 120, render: (_, record) => record.loginUsername || record.username },
            { title: '执行用户', dataIndex: 'username', width: 120 },
            { title: '进程', dataIndex: 'processName', width: 140 },
            { title: '命令', dataIndex: 'cmdline', render: (value, record) => <CommandText value={value} onView={() => setSelected(record)} /> },
          ]}
        />
      </Card>
      <EventDetailDrawer event={selected} open={Boolean(selected)} onClose={() => setSelected(undefined)} />
    </>
  );
}
