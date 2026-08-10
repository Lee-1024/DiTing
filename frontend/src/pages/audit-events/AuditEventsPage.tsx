import { Button, Card, DatePicker, Empty, Form, Input, Select, Space, Table, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { exportAuditEvents, getAuditEvent, queryAuditEvents, queryAuditOperations } from '../../api/audit';
import { getRiskDispositions } from '../../api/riskDispositions';
import { AuditHostSelect, AuditUserSelect } from '../../components/AuditEntitySelect';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import { InsightHero, LatestPanel, MetricCard } from '../../components/InsightHeader';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent, AuditEventQuery, AuditOperationGroup } from '../../types/audit';
import type { RiskDispositionMap, RiskDispositionStatus } from '../../types/riskDisposition';
import { downloadBlob } from '../../utils/download';
import { compactNumber } from '../../utils/format';
import { displayHostIdentity } from '../../utils/hostDisplay';
import { eventTypeLabel, eventTypeOptions, severityOptions } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';
import EventDetailDrawer from './EventDetailDrawer';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;
// AuditEventsPage 渲染 Audit Events Page 组件。
export default function AuditEventsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [groups, setGroups] = useState<AuditOperationGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<AuditEvent>();
  const [relatedEvents, setRelatedEvents] = useState<AuditEvent[]>([]);
  const [riskDispositions, setRiskDispositions] = useState<RiskDispositionMap>({});
  const [total, setTotal] = useState(0);
  const [totalKnown, setTotalKnown] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [form] = Form.useForm();
  const requestSeq = useRef(0);
  const filterRange = Form.useWatch('timeRange', form) ?? defaultRange;
  const optionRange = {
    startTime: filterRange?.[0]?.startOf('day').toISOString(),
    endTime: filterRange?.[1]?.endOf('day').toISOString(),
  };

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
      const data = await queryAuditOperations(buildQuery(nextPage, nextPageSize, formValues));
      if (seq !== requestSeq.current) {
        return;
      }
      const items = data.items ?? [];
      const dispositionMap = await getRiskDispositions(items.map((item) => item.representative).filter(Boolean));
      if (seq !== requestSeq.current) {
        return;
      }
      setGroups(items);
      setRiskDispositions(dispositionMap);
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
    downloadBlob(blob, 'audit-events.csv');
  }

  useEffect(() => {
    void load();
  }, []);
  useEffect(() => {
    const eventID = searchParams.get('eventId');
    if (!eventID) {
      return;
    }
    let cancelled = false;
    void getAuditEvent(eventID).then((event) => {
      if (!cancelled) {
        setSelected(event);
        setRelatedEvents([event]);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [searchParams]);

  const riskyEvents = groups.filter((item) => isOpenRiskOperation(item, riskDispositions)).length;
  const criticalEvents = groups.filter((item) => item.maxSeverity === 'critical' && isOpenRiskOperation(item, riskDispositions)).length;
  const activeHosts = uniqueValues(groups.map((item) => displayHostIdentity(item.representative, '')).filter(Boolean)).length;
  const latestEvent = groups[0]?.representative;

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="page-kicker">AUDIT EVENT STREAM</span>
          <Typography.Title level={3} className="page-title">操作日志调查</Typography.Title>
        </div>
      </div>
      <div className="audit-hero">
        <InsightHero
          className="audit-summary"
          kicker="Audit Event Correlation"
          title="按同次操作聚合事件，快速还原执行上下文"
          description="文件、进程、网络事件会被折叠成同一次操作队列；展开分组或点击行可进入统一调查抽屉。"
          actions={(
            <>
            <Link to="/audit/risks"><Button type="primary">查看风险队列</Button></Link>
            <Link to="/audit/commands"><Button ghost>进入命令审计</Button></Link>
            </>
          )}
        />
        <LatestPanel
          label="最近操作"
          title={latestEvent ? eventTypeLabel(latestEvent.eventType) : '-'}
          description={latestEvent ? latestEvent.cmdline || latestEvent.filePath || latestEvent.processName || '-' : '暂无审计事件'}
        />
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="操作分组" value={groups.length} hint={totalKnown ? `共 ${compactNumber(total)} 条匹配结果` : '未执行全量计数，按页加载'} tone="blue" />
        <MetricCard label="原始事件" value={groups.reduce((sum, item) => sum + item.eventCount, 0)} hint="当前页聚合事件数" tone="cyan" />
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
          <AuditHostSelect className="filter-control-compact" {...optionRange} />
        </Form.Item>
        <Form.Item name="namespace" label="Namespace">
          <Input className="filter-control-compact" allowClear />
        </Form.Item>
        <Form.Item name="podName" label="Pod">
          <Input className="filter-control-compact" allowClear />
        </Form.Item>
        <Form.Item name="loginUsername" label="登录用户">
          <AuditUserSelect className="filter-control-compact" {...optionRange} />
        </Form.Item>
        <Form.Item name="execUsername" label="执行用户">
          <AuditUserSelect className="filter-control-compact" {...optionRange} />
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
          dataSource={groups}
          className="clickable-table"
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无审计事件" /> }}
          scroll={{ x: 1710 }}
          onRow={(record) => ({ onClick: () => {
            setSelected(record.representative);
            setRelatedEvents([record.representative]);
          }, title: '点击查看操作详情' })}
          expandable={{
            expandedRowRender: (group) => renderAuditGroupDetails(group, setSelected),
            expandedRowClassName: () => 'audit-expanded-row',
            rowExpandable: (group) => group.eventCount > 1,
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value, range) => paginationTotalText(totalKnown, value, range, page, pageSize, groups.length),
            onChange: (nextPage, nextPageSize) => {
              const sizeChanged = nextPageSize !== pageSize;
              void load(sizeChanged ? 1 : nextPage, nextPageSize, form.getFieldsValue());
            },
          }}
          columns={[
            { title: '时间', dataIndex: ['representative', 'eventTime'], width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '等级', dataIndex: 'maxSeverity', width: 150, render: (value, record) => renderOperationSeverity(value, record, riskDispositions) },
            { title: '事件', dataIndex: 'eventTypes', width: 160, render: (values: string[]) => values.map((value) => <Tag key={value}>{eventTypeLabel(value)}</Tag>) },
            { title: '明细数', dataIndex: 'eventCount', width: 104, align: 'right', className: 'number-cell' },
            { title: '主机/节点', dataIndex: ['representative', 'hostName'], width: 170, ellipsis: true, render: (_, record) => displayHostIdentity(record.representative) },
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
      <EventDetailDrawer event={selected} relatedEvents={relatedEvents} open={Boolean(selected)} onClose={() => {
        setSelected(undefined);
        setRelatedEvents([]);
        if (searchParams.has('eventId')) {
          const next = new URLSearchParams(searchParams);
          next.delete('eventId');
          setSearchParams(next, { replace: true });
        }
      }} />
    </>
  );
}

function renderAuditGroupDetails(group: AuditOperationGroup, onSelect: (event: AuditEvent) => void) {
  const event = group.representative;
  return (
    <div className="audit-detail-panel" onClick={(event) => event.stopPropagation()}>
      <div className="audit-detail-grid audit-detail-grid-head">
        <span>时间</span>
        <span>事件</span>
        <span>主机/节点</span>
        <span>文件路径</span>
        <span>文件操作</span>
        <span>标签</span>
      </div>
      {[
        event,
      ].map((event) => (
        <button
          key={event.eventId}
          type="button"
          className="audit-detail-grid audit-detail-row"
          onClick={() => onSelect(event)}
          title="点击查看明细事件"
        >
          <span>{formatLocalDateTime(event.eventTime)}</span>
          <span>{eventTypeLabel(event.eventType)}</span>
          <span>{displayHostIdentity(event)}</span>
          <span className="ellipsis-text">{event.filePath || '-'}</span>
          <span>{event.fileOperation || '-'}</span>
          <span className="audit-detail-tags">
            {event.tags?.length ? event.tags.map((tag) => <Tag key={tag}>{tag}</Tag>) : '-'}
          </span>
        </button>
      ))}
    </div>
  );
}

function renderOperationSeverity(value: string, record: AuditOperationGroup, dispositions: RiskDispositionMap) {
  const disposition = dispositions[record.representative.eventId];
  const status = disposition?.status ?? 'open';
  return (
    <Space size={6} wrap>
      <SeverityTag value={value} />
      {isRiskSeverity(value) && status !== 'open' && <Tag color={riskDispositionColor(status)}>{riskDispositionText(status)}</Tag>}
    </Space>
  );
}

function isOpenRiskOperation(item: AuditOperationGroup, dispositions: RiskDispositionMap) {
  if (!isRiskSeverity(item.maxSeverity)) {
    return false;
  }
  return (dispositions[item.representative.eventId]?.status ?? 'open') === 'open';
}

function isRiskSeverity(value: string) {
  return value === 'high' || value === 'critical';
}

function riskDispositionText(status: RiskDispositionStatus) {
  const config: Record<RiskDispositionStatus, string> = {
    open: '未处理',
    confirmed: '已处理',
    false_positive: '误报',
    ignored: '已忽略',
    ignore_similar: '忽略同类',
    closed: '已关闭',
  };
  return config[status] ?? status;
}

function riskDispositionColor(status: RiskDispositionStatus) {
  const config: Record<RiskDispositionStatus, string> = {
    open: 'red',
    confirmed: 'green',
    false_positive: 'blue',
    ignored: 'default',
    ignore_similar: 'purple',
    closed: 'cyan',
  };
  return config[status] ?? 'default';
}

// uniqueValues 处理 unique Values 相关逻辑。
function uniqueValues(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

function effectivePagedTotal(total: number | undefined, hasMore: boolean | undefined, page: number, pageSize: number, itemCount: number) {
  if (total && total > 0) {
    return total;
  }
  const previous = (Math.max(page, 1) - 1) * pageSize;
  return hasMore ? previous + itemCount + 1 : previous + itemCount;
}

function paginationTotalText(totalKnown: boolean, value: number, range: [number, number], page: number, pageSize: number, itemCount: number) {
  if (totalKnown) {
    return value > 0 ? `共 ${value} 个操作，当前 ${itemCount} 个操作` : `当前 ${itemCount} 个操作`;
  }
  const hasMore = value > page * pageSize;
  return hasMore ? `当前第 ${page} 页 ${range[1] - range[0] + 1} 个操作，还有更多` : `共 ${range[1]} 个操作`;
}
