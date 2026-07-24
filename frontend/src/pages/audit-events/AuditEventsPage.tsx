import { Button, Card, DatePicker, Empty, Form, Input, Select, Table, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import { MetricCard } from '../../components/InsightHeader';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent, AuditEventQuery } from '../../types/audit';
import { downloadBlob } from '../../utils/download';
import { compactNumber } from '../../utils/format';
import { eventTypeLabel, eventTypeOptions, severityOptions } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';
import EventDetailDrawer from './EventDetailDrawer';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

interface AuditEventGroup {
  groupId: string;
  representative: AuditEvent;
  events: AuditEvent[];
  eventTypes: string[];
  filePaths: string[];
  tags: string[];
  maxSeverity: string;
}

// AuditEventsPage 渲染 Audit Events Page 组件。
export default function AuditEventsPage() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<AuditEvent>();
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [form] = Form.useForm();
  const requestSeq = useRef(0);
  const groupedEvents = useMemo(() => groupAuditEvents(events), [events]);

  // buildQuery 构建 build Query 所需的数据结构。
  function buildQuery(nextPage = page, nextPageSize = pageSize, formValues = form.getFieldsValue()): AuditEventQuery {
    const values = formValues;
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      event_type: values.eventType,
      severity: values.severity,
      host_name: values.hostName,
      namespace: values.namespace,
      pod_name: values.podName,
      login_username: values.loginUsername,
      exec_username: values.execUsername,
      keyword: values.keyword,
      tag: values.tag,
      page: nextPage,
      page_size: nextPageSize,
    };
  }

  // load 加载页面所需数据。
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

  // submit 提交当前表单或操作。
  function submit() {
    void load(1, pageSize, form.getFieldsValue());
  }

  // resetAndLoad 重置 reset And Load 状态。
  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load(1, 10, form.getFieldsValue());
  }

  // exportCSV 导出或下载 export CSV 数据。
  async function exportCSV() {
    const blob = await exportAuditEvents(buildQuery(1, 5000));
    downloadBlob(blob, 'audit-events.csv');
  }

  useEffect(() => {
    void load();
  }, []);

  const riskyEvents = events.filter((item) => item.severity === 'high' || item.severity === 'critical').length;
  const criticalEvents = events.filter((item) => item.severity === 'critical').length;
  const activeHosts = uniqueValues(events.map((item) => item.hostName || item.nodeName || item.hostId || '').filter(Boolean)).length;
  const latestEvent = groupedEvents[0]?.representative;

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="page-kicker">AUDIT EVENT STREAM</span>
          <Typography.Title level={3} className="page-title">操作日志调查</Typography.Title>
        </div>
      </div>
      <div className="audit-hero">
        <section className="audit-summary">
          <div className="ops-kicker">Audit Event Correlation</div>
          <Typography.Title level={2} className="investigation-title">按同次操作聚合事件，快速还原执行上下文</Typography.Title>
          <Typography.Text className="investigation-desc">
            文件、进程、网络事件会被折叠成同一次操作队列；展开分组或点击行可进入统一调查抽屉。
          </Typography.Text>
          <div className="ops-hero-actions">
            <Link to="/audit/risks"><Button type="primary">查看风险队列</Button></Link>
            <Link to="/audit/commands"><Button ghost>进入命令审计</Button></Link>
          </div>
        </section>
        <aside className="investigation-latest">
          <Typography.Text type="secondary">最近操作</Typography.Text>
          <div className="latest-risk-title">{latestEvent ? eventTypeLabel(latestEvent.eventType) : '-'}</div>
          <div className="latest-risk-desc">{latestEvent ? latestEvent.cmdline || latestEvent.filePath || latestEvent.processName || '-' : '暂无审计事件'}</div>
        </aside>
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="操作分组" value={groupedEvents.length} hint={`共 ${compactNumber(total)} 条匹配结果`} tone="blue" />
        <MetricCard label="原始事件" value={events.length} hint="当前页事件数" tone="cyan" />
        <MetricCard label="高危/严重" value={riskyEvents} hint={`${criticalEvents} 条严重事件`} tone="danger" />
        <MetricCard label="活跃主机" value={activeHosts} hint="当前页涉及主机" tone="success" />
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={submit} onReset={() => void resetAndLoad()} onExport={() => void exportCSV()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="eventType" label="事件">
          <Select className="filter-control-compact" allowClear options={eventTypeOptions} />
        </Form.Item>
        <Form.Item name="severity" label="等级">
          <Select className="filter-control-compact" allowClear options={severityOptions} />
        </Form.Item>
        <Form.Item name="hostName" label="主机">
          <Input className="filter-control-compact" placeholder="主机名 / 节点" allowClear />
        </Form.Item>
        <Form.Item name="namespace" label="Namespace">
          <Input className="filter-control-compact" allowClear />
        </Form.Item>
        <Form.Item name="podName" label="Pod">
          <Input className="filter-control-compact" allowClear />
        </Form.Item>
        <Form.Item name="loginUsername" label="登录用户">
          <Input className="filter-control-compact" allowClear />
        </Form.Item>
        <Form.Item name="execUsername" label="执行用户">
          <Input className="filter-control-compact" allowClear />
        </Form.Item>
        <Form.Item name="keyword" label="关键字">
          <Input className="filter-control-compact" placeholder="命令 / 用户 / 进程" allowClear />
        </Form.Item>
        <Form.Item name="tag" label="标签">
          <Input className="filter-control-compact" placeholder="delete-syscall-debug" allowClear />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey="groupId"
          loading={loading}
          dataSource={groupedEvents}
          className="clickable-table"
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无审计事件" /> }}
          scroll={{ x: 1540 }}
          onRow={(record) => ({ onClick: () => setSelected(record.representative), title: '点击查看操作详情' })}
          expandable={{
            expandedRowRender: (group) => (
              <Table
                rowKey="eventId"
                size="small"
                pagination={false}
                dataSource={group.events}
                className="clickable-table"
                onRow={(record) => ({ onClick: (event) => {
                  event.stopPropagation();
                  setSelected(record);
                }, title: '点击查看明细事件' })}
                columns={[
                  { title: '时间', dataIndex: 'eventTime', width: 180, render: (value) => formatLocalDateTime(value) },
                  { title: '事件', dataIndex: 'eventType', width: 120, render: (value) => eventTypeLabel(value) },
                  { title: '文件路径', dataIndex: 'filePath', width: 420, ellipsis: true, render: (value) => value || '-' },
                  { title: '文件操作', dataIndex: 'fileOperation', width: 120, render: (value) => value || '-' },
                  { title: '标签', dataIndex: 'tags', render: (tags: string[]) => tags?.length ? tags.map((tag) => <Tag key={tag}>{tag}</Tag>) : '-' },
                ]}
              />
            ),
            rowExpandable: (group) => group.events.length > 1,
          }}
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
            { title: '时间', dataIndex: ['representative', 'eventTime'], width: 190, fixed: 'left', render: (value) => formatLocalDateTime(value) },
            { title: '等级', dataIndex: 'maxSeverity', width: 100, render: (value) => <SeverityTag value={value} /> },
            { title: '事件', dataIndex: 'eventTypes', width: 160, render: (values: string[]) => values.map((value) => <Tag key={value}>{eventTypeLabel(value)}</Tag>) },
            { title: '明细数', dataIndex: ['events', 'length'], width: 104, align: 'right', className: 'number-cell', render: (_, record) => record.events.length },
            { title: 'Namespace', dataIndex: ['representative', 'namespace'], width: 160, ellipsis: true },
            { title: 'Pod', dataIndex: ['representative', 'podName'], width: 200, ellipsis: true },
            { title: '登录用户', dataIndex: ['representative', 'loginUsername'], width: 120, render: (_, record) => record.representative.loginUsername || record.representative.username },
            { title: '执行用户', dataIndex: ['representative', 'username'], width: 120 },
            { title: '进程', dataIndex: ['representative', 'processName'], width: 140 },
            { title: '代表路径', dataIndex: 'filePaths', width: 320, render: (values: string[]) => values.length ? <span className="stacked-text">{values.slice(0, 2).join('\n')}</span> : '-' },
            { title: '标签', dataIndex: 'tags', width: 180, render: (tags: string[]) => tags?.length ? tags.map((tag) => <Tag key={tag}>{tag}</Tag>) : '-' },
            { title: '命令', dataIndex: ['representative', 'cmdline'], render: (value, record) => <CommandText value={value} onView={() => setSelected(record.representative)} /> },
          ]}
        />
      </Card>
      <EventDetailDrawer event={selected} relatedEvents={findRelatedEvents(groupedEvents, selected)} open={Boolean(selected)} onClose={() => setSelected(undefined)} />
    </>
  );
}

// groupAuditEvents 处理 group Audit Events 相关逻辑。
function groupAuditEvents(events: AuditEvent[]): AuditEventGroup[] {
  const groups = new Map<string, AuditEvent[]>();
  for (const event of events) {
    const key = operationGroupKey(event);
    groups.set(key, [...(groups.get(key) ?? []), event]);
  }
  return Array.from(groups.entries()).map(([groupId, groupEvents]) => {
    const sorted = [...groupEvents].sort((a, b) => new Date(b.eventTime).getTime() - new Date(a.eventTime).getTime());
    const representative = sorted[0];
    return {
      groupId,
      representative,
      events: sorted,
      eventTypes: uniqueValues(sorted.map((event) => event.eventType)),
      filePaths: uniqueValues(sorted.map((event) => event.filePath).filter(Boolean) as string[]),
      tags: uniqueValues(sorted.flatMap((event) => event.tags ?? [])),
      maxSeverity: maxSeverity(sorted.map((event) => event.severity)),
    };
  });
}

// operationGroupKey 处理 operation Group Key 相关逻辑。
function operationGroupKey(event: AuditEvent) {
  const second = dayjs(event.eventTime).format('YYYY-MM-DD HH:mm:ss');
  return [
    second,
    event.hostId || event.nodeName || event.hostName,
    event.namespace,
    event.podName,
    event.loginUsername || event.username,
    event.username,
    event.processName,
    event.cmdline,
  ].join('|');
}

// uniqueValues 处理 unique Values 相关逻辑。
function uniqueValues(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

// maxSeverity 处理 max Severity 相关逻辑。
function maxSeverity(values: string[]) {
  const order: Record<string, number> = { info: 1, low: 2, medium: 3, high: 4, critical: 5 };
  return values.reduce((max, value) => (order[value] ?? 0) > (order[max] ?? 0) ? value : max, values[0] || 'info');
}

// findRelatedEvents 处理 find Related Events 相关逻辑。
function findRelatedEvents(groups: AuditEventGroup[], selected?: AuditEvent) {
  if (!selected) {
    return [];
  }
  return groups.find((group) => group.events.some((event) => event.eventId === selected.eventId))?.events ?? [selected];
}
