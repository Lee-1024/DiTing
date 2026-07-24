import { Button, Card, DatePicker, Descriptions, Drawer, Empty, Form, Input, Select, Space, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import { getUserAudits } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import { MetricCard } from '../../components/InsightHeader';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { UserAuditItem, UserAuditQuery } from '../../types/stats';
import { downloadBlob } from '../../utils/download';
import { compactNumber } from '../../utils/format';
import { severityOptions } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

interface DetailFilters {
  hostName?: string;
  keyword?: string;
  severity?: string;
}

interface DistributionItem {
  name: string;
  count: number;
  highRiskEvents: number;
}

// UserAuditPage 封装 User Audit Page 相关的状态和行为。
export default function UserAuditPage() {
  const [items, setItems] = useState<UserAuditItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selected, setSelected] = useState<UserAuditItem>();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [riskEvents, setRiskEvents] = useState<AuditEvent[]>([]);
  const [hostDistribution, setHostDistribution] = useState<DistributionItem[]>([]);
  const [commandDistribution, setCommandDistribution] = useState<DistributionItem[]>([]);
  const [detailFilters, setDetailFilters] = useState<DetailFilters>({});
  const [detailPage, setDetailPage] = useState(1);
  const [detailPageSize, setDetailPageSize] = useState(10);
  const [detailTotal, setDetailTotal] = useState(0);
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

  // buildQuery 构建 build Query 所需的数据结构。
  function buildQuery(): UserAuditQuery {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      keyword: values.keyword,
      host_name: values.hostName,
      limit: 50,
    };
  }

  // load 加载页面所需数据。
  async function load() {
    setLoading(true);
    try {
      setItems(await getUserAudits(buildQuery()));
    } finally {
      setLoading(false);
    }
  }

  // resetAndLoad 重置 reset And Load 状态。
  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load();
  }

  // openDetails 打开对应的弹窗或详情视图。
  async function openDetails(item: UserAuditItem) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setSelected(item);
    setDetailLoading(true);
    try {
      const baseQuery = {
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        username: item.username,
        host_name: values.hostName,
      };
      const [data, riskData, sampleData] = await Promise.all([
        queryAuditEvents({
          ...baseQuery,
          page: 1,
          page_size: detailPageSize,
        }),
        queryAuditEvents({
          ...baseQuery,
          severity_in: 'high,critical',
          page: 1,
          page_size: 10,
        }),
        queryAuditEvents({
          ...baseQuery,
          page: 1,
          page_size: 500,
        }),
      ]);
      setEvents(data.items ?? []);
      setDetailPage(data.page);
      setDetailTotal(data.total);
      setRiskEvents(riskData.items ?? []);
      setHostDistribution(buildHostDistribution(sampleData.items ?? []));
      setCommandDistribution(buildCommandDistribution(sampleData.items ?? []));
      setDetailFilters({ hostName: values.hostName });
    } finally {
      setDetailLoading(false);
    }
  }

  // loadDetailEvents 加载页面所需数据。
  async function loadDetailEvents(item: UserAuditItem, filters: DetailFilters, nextPage = detailPage, nextPageSize = detailPageSize) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setDetailLoading(true);
    try {
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        username: item.username,
        host_name: filters.hostName,
        keyword: filters.keyword,
        severity: filters.severity,
        page: nextPage,
        page_size: nextPageSize,
      });
      setEvents(data.items ?? []);
      setDetailPage(data.page);
      setDetailPageSize(nextPageSize);
      setDetailTotal(data.total);
    } finally {
      setDetailLoading(false);
    }
  }

  // exportDetails 导出或下载 export Details 数据。
  async function exportDetails() {
    if (!selected) {
      return;
    }
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    const blob = await exportAuditEvents({
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      event_type: 'process_exec',
      username: selected.username,
      host_name: detailFilters.hostName,
      keyword: detailFilters.keyword,
      severity: detailFilters.severity,
    });
    downloadBlob(blob, `${selected.username || 'user'}-commands.csv`);
  }

  // closeDetails 关闭当前弹窗或详情视图。
  function closeDetails() {
    setSelected(undefined);
    setEvents([]);
    setRiskEvents([]);
    setHostDistribution([]);
    setCommandDistribution([]);
    setDetailFilters({});
    setDetailPage(1);
    setDetailTotal(0);
  }

  // detailColumns 处理 detail Columns 相关逻辑。
  function detailColumns() {
    return [
      { title: '时间', dataIndex: 'eventTime', width: 190, render: (value: string) => formatLocalDateTime(value) },
      { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_: string, record: AuditEvent) => record.loginUsername || record.username },
      { title: '执行用户', dataIndex: 'username', width: 110 },
      { title: '主机', dataIndex: 'hostName', width: 170, render: (_: string, record: AuditEvent) => record.hostName || record.nodeName || record.hostId || '-' },
      { title: '进程', dataIndex: 'processName', width: 120 },
      { title: '命令', dataIndex: 'cmdline', render: (value: string) => <CommandText value={value} /> },
      { title: '等级', dataIndex: 'severity', width: 90, render: (value: string) => <SeverityTag value={value} /> },
    ];
  }

  useEffect(() => {
    void load();
  }, []);

  const totalCommands = items.reduce((sum, item) => sum + item.commandCount, 0);
  const totalHighRisk = items.reduce((sum, item) => sum + item.highRiskEvents, 0);
  const activeHostFootprint = items.reduce((sum, item) => sum + item.activeHosts, 0);
  const latestUser = [...items].sort((left, right) => new Date(right.lastSeen).getTime() - new Date(left.lastSeen).getTime())[0];

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="page-kicker">USER BEHAVIOR PROFILE</span>
          <Typography.Title level={3} className="page-title">用户审计画像</Typography.Title>
        </div>
      </div>
      <div className="profile-hero">
        <section className="user-summary">
          <div className="ops-kicker">Identity Behavior</div>
          <Typography.Title level={2} className="investigation-title">从用户维度追踪命令、主机和高危行为</Typography.Title>
          <Typography.Text className="investigation-desc">
            用户画像聚合执行命令、涉及主机和高危命中；点击用户进入行为分布和命令明细。
          </Typography.Text>
          <div className="ops-hero-actions">
            <Link to="/audit/commands"><Button type="primary">命令审计</Button></Link>
            <Link to="/audit/hosts"><Button ghost>主机画像</Button></Link>
          </div>
        </section>
        <aside className="investigation-latest">
          <Typography.Text type="secondary">最近活跃用户</Typography.Text>
          <div className="latest-risk-title">{latestUser?.username || '-'}</div>
          <div className="latest-risk-desc">{latestUser ? `${compactNumber(latestUser.commandCount)} 条命令 / ${compactNumber(latestUser.highRiskEvents)} 条高危` : '暂无用户审计数据'}</div>
        </aside>
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="用户数" value={items.length} hint="当前筛选结果" tone="blue" />
        <MetricCard label="命令数" value={totalCommands} hint="聚合命令总量" tone="cyan" />
        <MetricCard label="主机覆盖" value={activeHostFootprint} hint="用户涉及主机足迹" tone="success" />
        <MetricCard label="高危事件" value={totalHighRisk} hint="需重点回溯" tone="danger" />
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={() => void load()} onReset={() => void resetAndLoad()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="keyword" label="用户">
          <Input className="filter-control-compact" placeholder="root / ubuntu" allowClear />
        </Form.Item>
        <Form.Item name="hostName" label="主机">
          <Input className="filter-control-compact" placeholder="主机名 / Host ID" allowClear />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey="username"
          loading={loading}
          dataSource={items}
          className="clickable-table"
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户审计数据" /> }}
          onRow={(record) => ({ onClick: () => void openDetails(record), title: '点击查看用户审计详情' })}
          scroll={{ x: 960 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '用户', dataIndex: 'username', width: 160 },
            { title: '命令数', dataIndex: 'commandCount', width: 128, align: 'right', className: 'number-cell' },
            { title: '活跃主机', dataIndex: 'activeHosts', width: 128, align: 'right', className: 'number-cell' },
            { title: '高危事件', dataIndex: 'highRiskEvents', width: 128, align: 'right', className: 'number-cell danger-number' },
            { title: '首次活动', dataIndex: 'firstSeen', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近活动', dataIndex: 'lastSeen', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
      <Drawer
        title={selected?.username ? `${selected.username} 用户审计详情` : '用户审计详情'}
        width={1120}
        open={Boolean(selected)}
        onClose={closeDetails}
        className="investigation-drawer"
      >
        {selected && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <div className="event-brief">
              <div>
                <div className="ops-kicker">User Profile</div>
                <Typography.Title level={4} className="event-brief-title">{selected.username}</Typography.Title>
                <Typography.Text className="event-brief-desc">
                  {compactNumber(selected.commandCount)} 条命令，覆盖 {compactNumber(selected.activeHosts)} 台主机，命中 {compactNumber(selected.highRiskEvents)} 条高危事件。
                </Typography.Text>
              </div>
              <div className="event-brief-meta">
                <span className="metric-label">高危事件</span>
                <span className="ops-status-value">{compactNumber(selected.highRiskEvents)}</span>
              </div>
            </div>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="用户">{selected.username}</Descriptions.Item>
              <Descriptions.Item label="涉及主机">{selected.activeHosts}</Descriptions.Item>
              <Descriptions.Item label="命令数">{selected.commandCount}</Descriptions.Item>
              <Descriptions.Item label="高危事件">{selected.highRiskEvents}</Descriptions.Item>
              <Descriptions.Item label="首次活动">{formatLocalDateTime(selected.firstSeen)}</Descriptions.Item>
              <Descriptions.Item label="最近活动">{formatLocalDateTime(selected.lastSeen)}</Descriptions.Item>
            </Descriptions>
            <Typography.Title level={5}>涉及主机</Typography.Title>
            <Table
              rowKey="name"
              size="small"
              loading={detailLoading}
              dataSource={hostDistribution}
              className="clickable-table"
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无主机分布" /> }}
              pagination={false}
              onRow={(record) => ({
                title: '点击按该主机筛选命令明细',
                onClick: () => {
                  const nextFilters = { ...detailFilters, hostName: record.name };
                  setDetailFilters(nextFilters);
                  if (selected) {
                    void loadDetailEvents(selected, nextFilters, 1, detailPageSize);
                  }
                },
              })}
              columns={[
                { title: '主机', dataIndex: 'name' },
                { title: '命令数', dataIndex: 'count', width: 112, align: 'right', className: 'number-cell' },
                { title: '高危事件', dataIndex: 'highRiskEvents', width: 112, align: 'right', className: 'number-cell danger-number' },
              ]}
            />
            <Typography.Title level={5}>TOP 命令</Typography.Title>
            <Table
              rowKey="name"
              size="small"
              loading={detailLoading}
              dataSource={commandDistribution}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令分布" /> }}
              pagination={false}
              columns={[
                { title: '命令', dataIndex: 'name', render: (value: string) => <CommandText value={value} /> },
                { title: '次数', dataIndex: 'count', width: 112, align: 'right', className: 'number-cell' },
                { title: '高危事件', dataIndex: 'highRiskEvents', width: 112, align: 'right', className: 'number-cell danger-number' },
              ]}
            />
            <Typography.Title level={5}>高危命令</Typography.Title>
            <Table
              rowKey="eventId"
              size="small"
              loading={detailLoading}
              dataSource={riskEvents}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无高危命令" /> }}
              pagination={false}
              columns={detailColumns()}
            />
            <Typography.Title level={5}>命令明细</Typography.Title>
            <Space wrap>
              <Input
                allowClear
                placeholder="主机名 / Host ID"
                style={{ width: 180 }}
                value={detailFilters.hostName}
                onChange={(event) => setDetailFilters((current) => ({ ...current, hostName: event.target.value }))}
              />
              <Input
                allowClear
                placeholder="命令 / 进程"
                style={{ width: 220 }}
                value={detailFilters.keyword}
                onChange={(event) => setDetailFilters((current) => ({ ...current, keyword: event.target.value }))}
              />
              <Select
                allowClear
                placeholder="风险等级"
                style={{ width: 140 }}
                value={detailFilters.severity}
                onChange={(value) => setDetailFilters((current) => ({ ...current, severity: value }))}
                options={severityOptions}
              />
              <Button type="primary" onClick={() => selected && void loadDetailEvents(selected, detailFilters, 1, detailPageSize)}>查询</Button>
              <Button onClick={() => {
                const nextFilters: DetailFilters = {};
                setDetailFilters(nextFilters);
                if (selected) {
                  void loadDetailEvents(selected, nextFilters, 1, detailPageSize);
                }
              }}>清空筛选</Button>
              <Button onClick={() => void exportDetails()}>导出明细</Button>
            </Space>
          </Space>
        )}
        <Table
          rowKey="eventId"
          size="small"
          loading={detailLoading}
          dataSource={events}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令明细" /> }}
          pagination={{
            current: detailPage,
            pageSize: detailPageSize,
            total: detailTotal,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onChange: (nextPage, nextPageSize) => {
              if (selected) {
                void loadDetailEvents(selected, detailFilters, nextPage, nextPageSize);
              }
            },
          }}
          columns={detailColumns()}
        />
      </Drawer>
    </>
  );
}

// buildHostDistribution 构建 build Host Distribution 所需的数据结构。
function buildHostDistribution(events: AuditEvent[]) {
  return topDistribution(events, (event) => event.hostName || event.nodeName || event.hostId || '-');
}

// buildCommandDistribution 构建 build Command Distribution 所需的数据结构。
function buildCommandDistribution(events: AuditEvent[]) {
  return topDistribution(events, (event) => event.cmdline || event.processName || '-');
}

// topDistribution 转换 top Distribution 的数据结构。
function topDistribution(events: AuditEvent[], keyOf: (event: AuditEvent) => string) {
  const byKey = new Map<string, DistributionItem>();
  for (const event of events) {
    const key = keyOf(event);
    const item = byKey.get(key) ?? { name: key, count: 0, highRiskEvents: 0 };
    item.count += 1;
    if (event.severity === 'high' || event.severity === 'critical') {
      item.highRiskEvents += 1;
    }
    byKey.set(key, item);
  }
  return Array.from(byKey.values())
    .sort((left, right) => right.count - left.count || right.highRiskEvents - left.highRiskEvents)
    .slice(0, 10);
}
