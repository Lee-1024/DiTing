import { Button, Card, DatePicker, Descriptions, Drawer, Empty, Form, Input, Select, Space, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import { getUserAudits } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { UserAuditItem, UserAuditQuery } from '../../types/stats';
import { downloadBlob } from '../../utils/download';
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

  async function load() {
    setLoading(true);
    try {
      setItems(await getUserAudits(buildQuery()));
    } finally {
      setLoading(false);
    }
  }

  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load();
  }

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

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">用户审计</Typography.Title>
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
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户审计数据" /> }}
          onRow={(record) => ({ onClick: () => void openDetails(record) })}
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
            { title: '命令数', dataIndex: 'commandCount', width: 110 },
            { title: '活跃主机', dataIndex: 'activeHosts', width: 110 },
            { title: '高危事件', dataIndex: 'highRiskEvents', width: 110 },
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
      >
        {selected && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
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
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无主机分布" /> }}
              pagination={false}
              onRow={(record) => ({
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
                { title: '命令数', dataIndex: 'count', width: 100 },
                { title: '高危事件', dataIndex: 'highRiskEvents', width: 100 },
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
                { title: '次数', dataIndex: 'count', width: 100 },
                { title: '高危事件', dataIndex: 'highRiskEvents', width: 100 },
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

function buildHostDistribution(events: AuditEvent[]) {
  return topDistribution(events, (event) => event.hostName || event.nodeName || event.hostId || '-');
}

function buildCommandDistribution(events: AuditEvent[]) {
  return topDistribution(events, (event) => event.cmdline || event.processName || '-');
}

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
