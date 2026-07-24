import { Button, Card, DatePicker, Empty, Form, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import { getRiskDispositions, listRiskDispositions, updateRiskDisposition } from '../../api/riskDispositions';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import { MetricCard } from '../../components/InsightHeader';
import ProcessChain from '../../components/ProcessChain';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent, AuditEventQuery } from '../../types/audit';
import type { RiskDisposition, RiskDispositionMap, RiskDispositionStatus } from '../../types/riskDisposition';
import { downloadBlob } from '../../utils/download';
import { eventTypeLabel, eventTypeOptions, severityLabel } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';
import EventDetailDrawer from '../audit-events/EventDetailDrawer';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;
type DispositionFilter = 'all' | RiskDispositionStatus;

// RiskEventsPage 生成 Risk Events Page 的展示内容。
export default function RiskEventsPage() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [visibleEvents, setVisibleEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<AuditEvent>();
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [dispositions, setDispositions] = useState<RiskDispositionMap>({});
  const [dispositionOpen, setDispositionOpen] = useState(false);
  const [dispositionEvent, setDispositionEvent] = useState<AuditEvent>();
  const [savingDisposition, setSavingDisposition] = useState(false);
  const [form] = Form.useForm();
  const [dispositionForm] = Form.useForm();
  const requestSeq = useRef(0);

  // buildQuery 构建 build Query 所需的数据结构。
  function buildQuery(nextPage = page, nextPageSize = pageSize, formValues = form.getFieldsValue()): AuditEventQuery {
    const values = formValues;
    const range = values.timeRange ?? defaultRange;
    const severity = values.severity ?? 'medium,high,critical';
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      event_type: values.eventType,
      severity_in: severity,
      username: values.username,
      keyword: values.keyword,
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
      const dispositionStatus = formValues.dispositionStatus ?? 'open';
      if (dispositionStatus !== 'open' && dispositionStatus !== 'all') {
        const dispositionItems = await listRiskDispositions(dispositionStatus, 500);
        if (seq !== requestSeq.current) {
          return;
        }
        const eventIds = dispositionItems.map((item) => item.eventId).filter(Boolean);
        if (eventIds.length === 0) {
          setEvents([]);
          setDispositions({});
          setVisibleEvents([]);
          setTotal(0);
          setPage(1);
          setPageSize(nextPageSize);
          return;
        }
        const data = await queryAuditEvents({
          ...buildQuery(1, Math.min(eventIds.length, 500), formValues),
          event_ids: eventIds.join(','),
          page: 1,
          page_size: Math.min(eventIds.length, 500),
        });
        if (seq !== requestSeq.current) {
          return;
        }
        const dispositionMap = Object.fromEntries(dispositionItems.map((item) => [item.eventId, item]));
        const items = data.items ?? [];
        setEvents(items);
        setDispositions(dispositionMap);
        setVisibleEvents(items);
        setTotal(items.length);
        setPage(1);
        setPageSize(nextPageSize);
        return;
      }
      const data = await queryAuditEvents(buildQuery(nextPage, nextPageSize, formValues));
      if (seq !== requestSeq.current) {
        return;
      }
      const items = data.items ?? [];
      setEvents(items);
      const statusMap = await getRiskDispositions(items);
      if (seq !== requestSeq.current) {
        return;
      }
      setDispositions(statusMap);
      setVisibleEvents(filterEventsByDisposition(items, statusMap, formValues.dispositionStatus ?? 'open'));
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
    downloadBlob(blob, 'risk-events.csv');
  }

  // openDisposition 打开对应的弹窗或详情视图。
  function openDisposition(record: AuditEvent) {
    const existing = dispositions[record.eventId];
    setDispositionEvent(record);
    dispositionForm.setFieldsValue({
      status: existing?.status ?? 'open',
      note: existing?.note ?? '',
    });
    setDispositionOpen(true);
  }

  // submitDisposition 提交当前表单或操作。
  async function submitDisposition() {
    if (!dispositionEvent) {
      return;
    }
    const values = await dispositionForm.validateFields();
    setSavingDisposition(true);
    try {
      const updated = await updateRiskDisposition(dispositionEvent, values.status, values.note ?? '');
      setDispositions((current) => ({ ...current, [updated.eventId]: updated }));
      const nextDispositions = { ...dispositions, [updated.eventId]: updated };
      setVisibleEvents(filterEventsByDisposition(events, nextDispositions, form.getFieldValue('dispositionStatus') ?? 'open'));
      message.success('处置状态已更新');
      setDispositionOpen(false);
    } finally {
      setSavingDisposition(false);
    }
  }

  // dispositionFor 处理 disposition For 相关逻辑。
  function dispositionFor(record: AuditEvent): RiskDisposition {
    return dispositions[record.eventId] ?? {
      eventId: record.eventId,
      status: 'open',
      note: '',
      scope: '',
      fingerprint: '',
      handledBy: '',
      createdAt: '',
      updatedAt: '',
    };
  }

  useEffect(() => {
    void load();
  }, []);

  const openCount = visibleEvents.filter((item) => dispositionFor(item).status === 'open').length;
  const criticalCount = visibleEvents.filter((item) => item.severity === 'critical').length;
  const highCount = visibleEvents.filter((item) => item.severity === 'high').length;
  const latestEvent = visibleEvents[0];

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="page-kicker">INVESTIGATION QUEUE</span>
          <Typography.Title level={3} className="page-title">风险事件调查</Typography.Title>
        </div>
      </div>
      <div className="investigation-hero">
        <section className="investigation-summary">
          <div className="ops-kicker">Risk Operations</div>
          <Typography.Title level={2} className="investigation-title">按处置状态、风险等级和上下文快速收敛事件</Typography.Title>
          <Typography.Text className="investigation-desc">
            默认聚焦待处理风险；点击任意事件进入调查抽屉，按概览、进程、规则、关联事件和原始数据分层排查。
          </Typography.Text>
        </section>
        <aside className="investigation-latest">
          <Typography.Text type="secondary">最近风险</Typography.Text>
          <div className="latest-risk-title">{latestEvent ? eventTypeLabel(latestEvent.eventType) : '-'}</div>
          <div className="latest-risk-desc">{latestEvent ? latestEvent.cmdline || latestEvent.filePath || latestEvent.dstIp || '-' : '暂无风险事件'}</div>
        </aside>
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="当前队列" value={visibleEvents.length} hint={`共 ${total} 条匹配结果`} tone="blue" />
        <MetricCard label="待处理" value={openCount} hint="需要确认或关闭" tone="danger" />
        <MetricCard label="Critical" value={criticalCount} hint="最高优先级" tone="danger" />
        <MetricCard label="High" value={highCount} hint="高优先级" tone="warning" />
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange, severity: 'medium,high,critical', dispositionStatus: 'open' }} onSearch={submit} onReset={() => void resetAndLoad()} onExport={() => void exportCSV()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="eventType" label="类型">
          <Select
            allowClear
            className="filter-control-compact"
            placeholder="全部风险类型"
            options={eventTypeOptions}
          />
        </Form.Item>
        <Form.Item name="severity" label="等级">
          <Select
            className="filter-control-compact"
            options={[
              { value: 'medium,high,critical', label: `${severityLabel('medium')} + ${severityLabel('high')} + ${severityLabel('critical')}` },
              { value: 'high,critical', label: `${severityLabel('high')} + ${severityLabel('critical')}` },
              { value: 'medium', label: severityLabel('medium') },
              { value: 'high', label: severityLabel('high') },
              { value: 'critical', label: severityLabel('critical') },
            ]}
          />
        </Form.Item>
        <Form.Item name="username" label="用户">
          <Input className="filter-control-compact" placeholder="root / ubuntu" allowClear />
        </Form.Item>
        <Form.Item name="dispositionStatus" label="处置状态">
          <Select
            className="filter-control-compact"
            options={[
              { value: 'open', label: '待处理' },
              { value: 'all', label: '全部状态' },
              { value: 'confirmed', label: '已处理' },
              { value: 'false_positive', label: '误报' },
              { value: 'ignored', label: '忽略当前' },
              { value: 'ignore_similar', label: '忽略同类' },
              { value: 'closed', label: '已关闭' },
            ]}
          />
        </Form.Item>
        <Form.Item name="keyword" label="关键字">
          <Input className="filter-control-compact" placeholder="wget / docker" allowClear />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey="eventId"
          loading={loading}
          dataSource={visibleEvents}
          className="clickable-table"
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无风险事件" /> }}
          scroll={{ x: 1400 }}
          onRow={(record) => ({ onClick: () => setSelected(record), title: '点击查看风险事件详情' })}
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
            { title: '时间', dataIndex: 'eventTime', width: 170, fixed: 'left', render: (value) => formatLocalDateTime(value) },
            { title: '等级', dataIndex: 'severity', width: 96, render: (value) => <SeverityTag value={value} /> },
            { title: '类型', dataIndex: 'eventType', width: 124, render: (value) => eventTypeLabel(value) || '-' },
            { title: '登录用户', dataIndex: 'loginUsername', width: 112, render: (_, record) => record.loginUsername || record.username },
            { title: '执行用户', dataIndex: 'username', width: 112 },
            { title: '节点', dataIndex: 'nodeName', width: 150, ellipsis: true, render: (_, record) => record.nodeName || record.hostName },
            { title: '进程', dataIndex: 'processName', width: 130, ellipsis: true },
            { title: '进程链路', width: 220, render: (_, record) => <ProcessChain event={record} compact /> },
            { title: '风险对象', width: 260, render: (_, record) => riskTarget(record) },
            { title: '命令', dataIndex: 'cmdline', render: (value, record) => <CommandText value={value} onView={() => setSelected(record)} /> },
            {
              title: '命中规则',
              dataIndex: 'ruleNames',
              width: 180,
              render: (rules?: string[]) => rules?.length ? <div className="rule-tags">{rules.map((rule) => <Tag color="orange" key={rule}>{rule}</Tag>)}</div> : <Typography.Text type="secondary">-</Typography.Text>,
            },
            {
              title: '处置状态',
              width: 128,
              render: (_, record) => <DispositionTag disposition={dispositionFor(record)} />,
            },
            {
              title: '处置',
              width: 88,
              render: (_, record) => (
                <Button size="small" onClick={(event) => { event.stopPropagation(); openDisposition(record); }}>
                  处理
                </Button>
              ),
            },
          ]}
        />
      </Card>
      <EventDetailDrawer event={selected} open={Boolean(selected)} onClose={() => setSelected(undefined)} />
      <Modal
        title="处置风险事件"
        open={dispositionOpen}
        confirmLoading={savingDisposition}
        onOk={() => void submitDisposition()}
        onCancel={() => setDispositionOpen(false)}
        width={560}
      >
        <Form form={dispositionForm} layout="vertical">
          <Form.Item label="命令">
            <CommandText value={dispositionEvent?.cmdline} />
          </Form.Item>
          <Form.Item label="命中规则">
            {dispositionEvent?.ruleNames?.length ? (
              <div className="rule-tags">
                {dispositionEvent.ruleNames.map((rule) => <Tag color="orange" key={rule}>{rule}</Tag>)}
              </div>
            ) : (
              <Typography.Text type="secondary">-</Typography.Text>
            )}
          </Form.Item>
          <Form.Item name="status" label="处置状态" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'open', label: '未处理' },
                { value: 'confirmed', label: '已处理' },
                { value: 'false_positive', label: '误报' },
                { value: 'ignored', label: '忽略当前' },
                { value: 'ignore_similar', label: '忽略同类' },
                { value: 'closed', label: '已关闭' },
              ]}
            />
          </Form.Item>
          <Form.Item name="note" label="处置备注">
            <Input.TextArea rows={4} placeholder="记录确认原因、忽略理由或后续处理说明" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

// riskTarget 生成 risk Target 的展示内容。
function riskTarget(record: AuditEvent) {
  if (record.eventType === 'network_connect') {
    return record.dstIp ? (
      <Space direction="vertical" size={0}>
        <Typography.Text>{formatNetworkTarget(record)}</Typography.Text>
        <Typography.Text type="secondary">{record.protocol || '-'}</Typography.Text>
      </Space>
    ) : <Typography.Text type="secondary">-</Typography.Text>;
  }
  if (record.eventType === 'file_access') {
    return record.filePath ? (
      <Space direction="vertical" size={0}>
        <Typography.Text ellipsis style={{ maxWidth: 190 }}>{record.filePath}</Typography.Text>
        <Typography.Text type="secondary">{record.fileOperation || '-'}</Typography.Text>
      </Space>
    ) : <Typography.Text type="secondary">-</Typography.Text>;
  }
  return record.processName ? <Typography.Text>{record.processName}</Typography.Text> : <Typography.Text type="secondary">-</Typography.Text>;
}

// filterEventsByDisposition 按条件过滤 filter Events By Disposition。
function filterEventsByDisposition(events: AuditEvent[], dispositions: RiskDispositionMap, status: DispositionFilter) {
  if (status === 'all') {
    return events;
  }
  return events.filter((event) => (dispositions[event.eventId]?.status ?? 'open') === status);
}

// formatNetworkTarget 格式化 format Network Target 以便界面展示。
function formatNetworkTarget(record: AuditEvent) {
  if (!record.dstIp) {
    return '-';
  }
  const ip = record.dstIp.includes(':') ? `[${record.dstIp}]` : record.dstIp;
  return record.dstPort ? `${ip}:${record.dstPort}` : ip;
}

// DispositionTag 渲染 Disposition Tag 组件。
function DispositionTag({ disposition }: { disposition: RiskDisposition }) {
  const config: Record<RiskDispositionStatus, { color: string; text: string }> = {
    open: { color: 'red', text: '未处理' },
    confirmed: { color: 'green', text: '已处理' },
    false_positive: { color: 'blue', text: '误报' },
    ignored: { color: 'default', text: '忽略当前' },
    ignore_similar: { color: 'purple', text: '忽略同类' },
    closed: { color: 'cyan', text: '已关闭' },
  };
  const current = config[disposition.status] ?? config.open;
  return (
    <Space direction="vertical" size={0}>
      <Tag color={current.color}>{current.text}</Tag>
      {disposition.handledBy && <Typography.Text type="secondary">{disposition.handledBy}</Typography.Text>}
    </Space>
  );
}
