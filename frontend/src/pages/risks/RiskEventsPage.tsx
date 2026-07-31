import { Button, Card, DatePicker, Empty, Form, Input, Modal, Select, Space, Table, Tag, Typography, message } from 'antd';
import dayjs from 'dayjs';
import type { Key } from 'react';
import { useEffect, useRef, useState } from 'react';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import { analyzeRiskEvent, getRiskAnalyses } from '../../api/riskAnalyses';
import { getRiskDispositions, listRiskDispositions, updateRiskDisposition } from '../../api/riskDispositions';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import { InsightHero, LatestPanel, MetricCard } from '../../components/InsightHeader';
import ProcessChain from '../../components/ProcessChain';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent, AuditEventQuery } from '../../types/audit';
import type { AIRiskAnalysis, AIRiskAnalysisMap, AIRiskVerdict } from '../../types/riskAnalysis';
import type { RiskDisposition, RiskDispositionMap, RiskDispositionStatus } from '../../types/riskDisposition';
import { downloadBlob } from '../../utils/download';
import { displayHostIdentity } from '../../utils/hostDisplay';
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
  const [totalKnown, setTotalKnown] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [dispositions, setDispositions] = useState<RiskDispositionMap>({});
  const [aiAnalyses, setAIAnalyses] = useState<AIRiskAnalysisMap>({});
  const [dispositionOpen, setDispositionOpen] = useState(false);
  const [dispositionEvent, setDispositionEvent] = useState<AuditEvent>();
  const [savingDisposition, setSavingDisposition] = useState(false);
  const [batchDispositionOpen, setBatchDispositionOpen] = useState(false);
  const [savingBatchDisposition, setSavingBatchDisposition] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const [selectedRows, setSelectedRows] = useState<AuditEvent[]>([]);
  const [analyzingEventId, setAnalyzingEventId] = useState<string>();
  const [form] = Form.useForm();
  const [dispositionForm] = Form.useForm();
  const [batchDispositionForm] = Form.useForm();
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
      host_name: values.hostName,
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
          clearSelection();
          setTotal(0);
          setTotalKnown(true);
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
        const analysisMap = await getRiskAnalyses(items);
        if (seq !== requestSeq.current) {
          return;
        }
        setEvents(items);
        setDispositions(dispositionMap);
        setAIAnalyses(analysisMap);
        setVisibleEvents(items);
        clearSelection();
        setTotal(items.length);
        setTotalKnown(true);
        setPage(1);
        setPageSize(nextPageSize);
        return;
      }
      if (dispositionStatus === 'open') {
        const openItems = await loadOpenRiskEvents(nextPage, nextPageSize, formValues, () => seq === requestSeq.current);
        if (!openItems) {
          return;
        }
        if (openItems.items.length === 0 && nextPage > 1) {
          const previousPage = Math.max(Math.ceil(openItems.total / nextPageSize), 1);
          void load(previousPage, nextPageSize, formValues);
          return;
        }
        const analysisMap = await getRiskAnalyses(openItems.items);
        if (seq !== requestSeq.current) {
          return;
        }
        setEvents(openItems.items);
        setDispositions(openItems.dispositions);
        setAIAnalyses(analysisMap);
        setVisibleEvents(openItems.items);
        clearSelection();
        setTotal(openItems.total);
        setTotalKnown(openItems.totalKnown);
        setPage(nextPage);
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
      const analysisMap = await getRiskAnalyses(items);
      if (seq !== requestSeq.current) {
        return;
      }
      setDispositions(statusMap);
      setAIAnalyses(analysisMap);
      setVisibleEvents(filterEventsByDisposition(items, statusMap, dispositionStatus));
      clearSelection();
      setTotal(effectivePagedTotal(data.total, data.hasMore, data.page, data.pageSize, items.length));
      setTotalKnown(Boolean(data.total && data.total > 0));
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
      message.success('处置状态已更新');
      setDispositionOpen(false);
      await load(page, pageSize, form.getFieldsValue());
    } finally {
      setSavingDisposition(false);
    }
  }

  function clearSelection() {
    setSelectedRowKeys([]);
    setSelectedRows([]);
  }

  function openBatchDisposition() {
    if (selectedRows.length === 0) {
      message.warning('请先选择要处理的风险事件');
      return;
    }
    batchDispositionForm.setFieldsValue({
      status: 'confirmed',
      note: '',
    });
    setBatchDispositionOpen(true);
  }

  async function submitBatchDisposition() {
    const rows = selectedRows;
    if (rows.length === 0) {
      message.warning('请先选择要处理的风险事件');
      return;
    }
    const values = await batchDispositionForm.validateFields();
    setSavingBatchDisposition(true);
    try {
      const results = await Promise.allSettled(
        rows.map((event) => updateRiskDisposition(event, values.status, values.note ?? '')),
      );
      const updatedMap = { ...dispositions };
      results.forEach((result) => {
        if (result.status === 'fulfilled') {
          updatedMap[result.value.eventId] = result.value;
        }
      });
      const successCount = results.filter((result) => result.status === 'fulfilled').length;
      const failedCount = rows.length - successCount;
      setDispositions(updatedMap);
      clearSelection();
      setBatchDispositionOpen(false);
      if (failedCount > 0) {
        message.warning(`批量处理完成：成功 ${successCount} 条，失败 ${failedCount} 条`);
      } else {
        message.success(`批量处理完成：成功 ${successCount} 条`);
      }
      await load(page, pageSize, form.getFieldsValue());
    } finally {
      setSavingBatchDisposition(false);
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

  async function analyzeEvent(record: AuditEvent) {
    setAnalyzingEventId(record.eventId);
    try {
      const analysis = await analyzeRiskEvent(record.eventId);
      setAIAnalyses((current) => ({ ...current, [record.eventId]: analysis }));
      message.success('AI 分析已完成');
    } catch (error: any) {
      message.error(error?.response?.data || error?.message || 'AI 分析失败');
    } finally {
      setAnalyzingEventId(undefined);
    }
  }

  async function loadOpenRiskEvents(nextPage: number, nextPageSize: number, formValues: any, isCurrent: () => boolean) {
    const targetCount = nextPage * nextPageSize + 1;
    const batchSize = 100;
    const maxBatches = 20;
    const openEvents: AuditEvent[] = [];
    const allDispositions: RiskDispositionMap = {};

    for (let batchPage = 1; batchPage <= maxBatches && openEvents.length < targetCount; batchPage += 1) {
      const data = await queryAuditEvents({
        ...buildQuery(batchPage, batchSize, formValues),
        include_total: false,
      });
      if (!isCurrent()) {
        return undefined;
      }
      const items = data.items ?? [];
      if (items.length === 0) {
        break;
      }
      const statusMap = await getRiskDispositions(items);
      if (!isCurrent()) {
        return undefined;
      }
      Object.assign(allDispositions, statusMap);
      openEvents.push(...filterEventsByDisposition(items, statusMap, 'open'));
      if (!data.hasMore) {
        break;
      }
    }

    const start = (nextPage - 1) * nextPageSize;
    const pageEnd = nextPage * nextPageSize;
    const hasNextOpenPage = openEvents.length > pageEnd;
    return {
      items: openEvents.slice(start, start + nextPageSize),
      dispositions: allDispositions,
      total: hasNextOpenPage ? pageEnd + 1 : openEvents.length,
      totalKnown: !hasNextOpenPage,
    };
  }

  useEffect(() => {
    void load();
  }, []);

  const currentDispositionStatus = form.getFieldValue('dispositionStatus') ?? 'open';
  const openCount = currentDispositionStatus === 'open' ? total : visibleEvents.filter((item) => dispositionFor(item).status === 'open').length;
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
        <InsightHero
          className="investigation-summary"
          kicker="Risk Operations"
          title="按处置状态、风险等级和上下文快速收敛事件"
          description="默认聚焦待处理风险；点击任意事件进入调查抽屉，按概览、进程、规则、关联事件和原始数据分层排查。"
        />
        <LatestPanel
          label="最近风险"
          title={latestEvent ? eventTypeLabel(latestEvent.eventType) : '-'}
          description={latestEvent ? latestEvent.cmdline || latestEvent.filePath || latestEvent.dstIp || '-' : '暂无风险事件'}
        />
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="当前页" value={visibleEvents.length} hint={totalKnown ? `共 ${total} 条匹配结果` : '未执行全量计数，按页加载'} tone="blue" />
        <MetricCard label="待处理" value={openCount} hint={currentDispositionStatus === 'open' && !totalKnown ? '还有更多，按页加载' : '需要确认或关闭'} tone="danger" />
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
        <Form.Item name="hostName" label="主机">
          <Input className="filter-control-compact" placeholder="主机名 / 节点 / Host ID" allowClear />
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
        <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }} wrap>
          <Typography.Text type="secondary">已选 {selectedRows.length} 条</Typography.Text>
          <Space>
            <Button disabled={selectedRows.length === 0} onClick={clearSelection}>
              清空选择
            </Button>
            <Button type="primary" disabled={selectedRows.length === 0} onClick={openBatchDisposition}>
              批量处理
            </Button>
          </Space>
        </Space>
        <Table
          rowKey="eventId"
          loading={loading}
          dataSource={visibleEvents}
          className="clickable-table"
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无风险事件" /> }}
          scroll={{ x: 1400 }}
          rowSelection={{
            selectedRowKeys,
            preserveSelectedRowKeys: true,
            onChange: (keys, rows) => {
              setSelectedRowKeys(keys);
              setSelectedRows(rows);
            },
          }}
          onRow={(record) => ({ onClick: () => setSelected(record), title: '点击查看风险事件详情' })}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value, range) => paginationTotalText(totalKnown, value, range, page, pageSize),
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
            { title: '节点', dataIndex: 'nodeName', width: 150, ellipsis: true, render: (_, record) => displayHostIdentity(record) },
            { title: '进程', dataIndex: 'processName', width: 130, ellipsis: true },
            { title: '进程链路', width: 220, render: (_, record) => <ProcessChain event={record} compact /> },
            { title: '风险对象', width: 260, render: (_, record) => riskTarget(record) },
            {
              title: '命中规则',
              dataIndex: 'ruleNames',
              width: 220,
              render: (rules?: string[]) => rules?.length ? <div className="rule-tags rule-tags-compact">{rules.map((rule) => <Tag color="orange" key={rule}>{rule}</Tag>)}</div> : <Typography.Text type="secondary">-</Typography.Text>,
            },
            { title: '命令', dataIndex: 'cmdline', width: 360, render: (value, record) => <CommandText value={value} width={280} onView={() => setSelected(record)} /> },
            {
              title: '处置状态',
              width: 128,
              render: (_, record) => <DispositionTag disposition={dispositionFor(record)} />,
            },
            {
              title: 'AI 判断',
              width: 180,
              render: (_, record) => <AIAnalysisCell analysis={aiAnalyses[record.eventId]} />,
            },
            {
              title: '处置',
              width: 176,
              render: (_, record) => (
                <Space size={8}>
                  <Button size="small" onClick={(event) => { event.stopPropagation(); openDisposition(record); }}>
                    处理
                  </Button>
                  <Button size="small" loading={analyzingEventId === record.eventId} onClick={(event) => { event.stopPropagation(); void analyzeEvent(record); }}>
                    {aiAnalyses[record.eventId] ? '重分析' : 'AI 分析'}
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <EventDetailDrawer
        event={selected}
        aiAnalysis={selected ? aiAnalyses[selected.eventId] : undefined}
        open={Boolean(selected)}
        onClose={() => setSelected(undefined)}
      />
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
      <Modal
        title={`批量处理 ${selectedRows.length} 条风险事件`}
        open={batchDispositionOpen}
        confirmLoading={savingBatchDisposition}
        onOk={() => void submitBatchDisposition()}
        onCancel={() => setBatchDispositionOpen(false)}
        width={560}
      >
        <Form form={batchDispositionForm} layout="vertical">
          <Form.Item label="已选事件">
            <Typography.Text type="secondary">将对当前选中的 {selectedRows.length} 条风险事件执行同一处置状态。</Typography.Text>
          </Form.Item>
          <Form.Item name="status" label="处置状态" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'confirmed', label: '已处理' },
                { value: 'false_positive', label: '误报' },
                { value: 'ignored', label: '忽略当前' },
                { value: 'ignore_similar', label: '忽略同类' },
                { value: 'closed', label: '已关闭' },
                { value: 'open', label: '未处理' },
              ]}
            />
          </Form.Item>
          <Form.Item name="note" label="处置备注">
            <Input.TextArea rows={4} placeholder="记录本次批量处理原因" />
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

function AIAnalysisCell({ analysis }: { analysis?: AIRiskAnalysis }) {
  if (!analysis) {
    return <Typography.Text type="secondary">未分析</Typography.Text>;
  }
  return (
    <Space direction="vertical" size={0}>
      <Space size={6} wrap>
        <Tag color={aiVerdictColor(analysis.verdict)}>{aiVerdictText(analysis.verdict)}</Tag>
        <SeverityTag value={analysis.aiSeverity} />
      </Space>
      <Typography.Text type="secondary">{analysis.confidence}% / {analysis.model || '-'}</Typography.Text>
      {analysis.reason && <Typography.Text ellipsis style={{ maxWidth: 150 }}>{analysis.reason}</Typography.Text>}
    </Space>
  );
}

function aiVerdictText(verdict: AIRiskVerdict) {
  const config: Record<AIRiskVerdict, string> = {
    true_positive: '真实风险',
    suspicious: '可疑',
    false_positive: '可能误报',
    needs_review: '需复核',
  };
  return config[verdict] ?? verdict;
}

function aiVerdictColor(verdict: AIRiskVerdict) {
  const config: Record<AIRiskVerdict, string> = {
    true_positive: 'red',
    suspicious: 'orange',
    false_positive: 'blue',
    needs_review: 'purple',
  };
  return config[verdict] ?? 'default';
}

// filterEventsByDisposition 按条件过滤 filter Events By Disposition。
function filterEventsByDisposition(events: AuditEvent[], dispositions: RiskDispositionMap, status: DispositionFilter) {
  if (status === 'all') {
    return events;
  }
  return events.filter((event) => (dispositions[event.eventId]?.status ?? 'open') === status);
}

function effectivePagedTotal(total: number | undefined, hasMore: boolean | undefined, page: number, pageSize: number, itemCount: number) {
  if (total && total > 0) {
    return total;
  }
  const previous = (Math.max(page, 1) - 1) * pageSize;
  return hasMore ? previous + itemCount + 1 : previous + itemCount;
}

function paginationTotalText(totalKnown: boolean, value: number, range: [number, number], page: number, pageSize: number) {
  if (totalKnown) {
    return `共 ${value} 条`;
  }
  const hasMore = value > page * pageSize;
  return hasMore ? `当前第 ${page} 页 ${range[1] - range[0] + 1} 条，还有更多` : `共 ${range[1]} 条`;
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
