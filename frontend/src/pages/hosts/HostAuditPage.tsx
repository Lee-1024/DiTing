import { Button, Card, DatePicker, Descriptions, Drawer, Empty, Form, Input, Row, Col, Select, Space, Statistic, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import { exportHostAudits, getHostAudits, getHostBehavior, getHostUsers } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import ProcessChain from '../../components/ProcessChain';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { BehaviorItem, HostAuditItem, HostAuditQuery, HostBehavior, HostUserItem } from '../../types/stats';
import { downloadBlob } from '../../utils/download';
import { eventTypeLabel, severityOptions } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;
const emptyBehavior: HostBehavior = { filePaths: [], network: [], eventTypes: [], ruleHits: [] };

interface DetailFilters {
  username?: string;
  keyword?: string;
  severity?: string;
}

export default function HostAuditPage() {
  const [items, setItems] = useState<HostAuditItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selected, setSelected] = useState<HostAuditItem>();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [riskEvents, setRiskEvents] = useState<AuditEvent[]>([]);
  const [riskTimeline, setRiskTimeline] = useState<AuditEvent[]>([]);
  const [hostUsers, setHostUsers] = useState<HostUserItem[]>([]);
  const [hostBehavior, setHostBehavior] = useState<HostBehavior>(emptyBehavior);
  const [selectedFileTarget, setSelectedFileTarget] = useState<BehaviorItem>();
  const [fileEvents, setFileEvents] = useState<AuditEvent[]>([]);
  const [fileLoading, setFileLoading] = useState(false);
  const [selectedNetworkTarget, setSelectedNetworkTarget] = useState<BehaviorItem>();
  const [networkEvents, setNetworkEvents] = useState<AuditEvent[]>([]);
  const [networkLoading, setNetworkLoading] = useState(false);
  const [detailFilters, setDetailFilters] = useState<DetailFilters>({});
  const [detailPage, setDetailPage] = useState(1);
  const [detailPageSize, setDetailPageSize] = useState(10);
  const [detailTotal, setDetailTotal] = useState(0);
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

  function buildQuery(): HostAuditQuery {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      keyword: values.keyword,
      limit: 50,
    };
  }

  async function load() {
    setLoading(true);
    try {
      const hostItems = await getHostAudits(buildQuery());
      setItems(hostItems);
    } finally {
      setLoading(false);
    }
  }

  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load();
  }

  async function exportHosts() {
    const blob = await exportHostAudits(buildQuery());
    downloadBlob(blob, 'host-audits.csv');
  }

  async function openDetails(item: HostAuditItem) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setSelected(item);
    setDetailLoading(true);
    try {
      const hostName = item.hostId || item.nodeName || item.hostName;
      const baseQuery = {
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        host_name: hostName,
        page: 1,
      };
      const hostUserQuery = {
        start_time: baseQuery.start_time,
        end_time: baseQuery.end_time,
        host_name: hostName,
        limit: 20,
      };
      const [data, riskData, timelineData, usersData, behaviorData] = await Promise.all([
        queryAuditEvents({
          ...baseQuery,
          page_size: detailPageSize,
        }),
        queryAuditEvents({
          ...baseQuery,
          severity_in: 'high,critical',
          page_size: 10,
        }),
        queryAuditEvents({
          start_time: baseQuery.start_time,
          end_time: baseQuery.end_time,
          host_name: hostName,
          severity_in: 'medium,high,critical',
          page: 1,
          page_size: 12,
        }),
        getHostUsers(hostUserQuery),
        getHostBehavior(hostUserQuery),
      ]);
      setEvents(data.items ?? []);
      setDetailPage(data.page);
      setDetailTotal(data.total);
      setRiskEvents(riskData.items ?? []);
      setRiskTimeline(timelineData.items ?? []);
      setHostUsers(usersData ?? []);
      setHostBehavior(behaviorData ?? emptyBehavior);
      setDetailFilters({});
    } finally {
      setDetailLoading(false);
    }
  }

  async function loadDetailEvents(item: HostAuditItem, filters: DetailFilters, nextPage = detailPage, nextPageSize = detailPageSize) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setDetailLoading(true);
    try {
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        host_name: item.hostId || item.nodeName || item.hostName,
        username: filters.username,
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

  async function loadNetworkEvents(item: HostAuditItem, target: BehaviorItem) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    const parsed = parseNetworkTarget(target.name);
    setSelectedNetworkTarget(target);
    setNetworkLoading(true);
    try {
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'network_connect',
        host_name: item.hostId || item.nodeName || item.hostName,
        dst_ip: parsed.ip,
        dst_port: parsed.port,
        page: 1,
        page_size: 20,
      });
      setNetworkEvents(data.items ?? []);
    } finally {
      setNetworkLoading(false);
    }
  }

  async function loadFileEvents(item: HostAuditItem, target: BehaviorItem) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setSelectedFileTarget(target);
    setFileLoading(true);
    try {
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'file_access',
        host_name: item.hostId || item.nodeName || item.hostName,
        file_path: target.name,
        page: 1,
        page_size: 20,
      });
      setFileEvents(data.items ?? []);
    } finally {
      setFileLoading(false);
    }
  }

  async function exportDetailEvents() {
    if (!selected) {
      return;
    }
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    const blob = await exportAuditEvents({
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      event_type: 'process_exec',
      host_name: selected.hostId || selected.nodeName || selected.hostName,
      username: detailFilters.username,
      keyword: detailFilters.keyword,
      severity: detailFilters.severity,
    });
    downloadBlob(blob, `${selected.hostName || 'host'}-commands.csv`);
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">主机审计</Typography.Title>
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={() => void load()} onReset={() => void resetAndLoad()} onExport={() => void exportHosts()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="keyword" label="主机">
          <Input className="filter-control-compact" placeholder="节点 / 主机名" allowClear />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey={(record) => record.hostId || record.nodeName || record.hostName}
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无主机审计数据" /> }}
          onRow={(record) => ({ onClick: () => void openDetails(record) })}
          scroll={{ x: 1120 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            {
              title: '主机',
              dataIndex: 'hostName',
              render: (value: string, record) => (
                <Space direction="vertical" size={0}>
                  <Typography.Text>{value}</Typography.Text>
                  {record.hostId && record.hostId !== value && <Typography.Text type="secondary">{record.hostId}</Typography.Text>}
                  {record.nodeName && record.nodeName !== value && record.nodeName !== record.hostId && <Typography.Text type="secondary">{record.nodeName}</Typography.Text>}
                </Space>
              ),
            },
            { title: '命令数', dataIndex: 'commandCount', width: 110 },
            { title: '活跃用户', dataIndex: 'activeUsers', width: 110 },
            { title: '高危事件', dataIndex: 'highRiskEvents', width: 110 },
            { title: '首次活动', dataIndex: 'firstSeen', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近活动', dataIndex: 'lastSeen', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
      <Drawer
        title={selected?.hostName ? `${selected.hostName} 主机审计详情` : '主机审计详情'}
        width={1080}
        open={Boolean(selected)}
        onClose={() => {
          setSelected(undefined);
          setEvents([]);
          setRiskEvents([]);
          setRiskTimeline([]);
          setHostUsers([]);
          setHostBehavior(emptyBehavior);
          setSelectedFileTarget(undefined);
          setFileEvents([]);
          setSelectedNetworkTarget(undefined);
          setNetworkEvents([]);
          setDetailFilters({});
          setDetailPage(1);
          setDetailTotal(0);
        }}
      >
        {selected && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="主机名">{selected.hostName || '-'}</Descriptions.Item>
              <Descriptions.Item label="Host ID">{selected.hostId || '-'}</Descriptions.Item>
              <Descriptions.Item label="节点名">{selected.nodeName || '-'}</Descriptions.Item>
              <Descriptions.Item label="首次活动">{formatLocalDateTime(selected.firstSeen)}</Descriptions.Item>
              <Descriptions.Item label="最近活动">{formatLocalDateTime(selected.lastSeen)}</Descriptions.Item>
            </Descriptions>
            <Row gutter={[12, 12]}>
              <Col xs={24} md={8}>
                <Card className="stat-card" size="small"><Statistic title="命令数" value={selected.commandCount} /></Card>
              </Col>
              <Col xs={24} md={8}>
                <Card className="stat-card" size="small"><Statistic title="活跃用户" value={selected.activeUsers} /></Card>
              </Col>
              <Col xs={24} md={8}>
                <Card className="stat-card stat-card-danger" size="small"><Statistic title="高危事件" value={selected.highRiskEvents} /></Card>
              </Col>
            </Row>
            <Typography.Title level={5}>高危命令</Typography.Title>
            <Table
              rowKey="eventId"
              size="small"
              loading={detailLoading}
              dataSource={riskEvents}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无高危命令" /> }}
              pagination={false}
              columns={commandColumns()}
            />
            <Typography.Title level={5}>最近风险时间线</Typography.Title>
            <Table
              rowKey="eventId"
              size="small"
              loading={detailLoading}
              dataSource={riskTimeline}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无风险事件" /> }}
              pagination={false}
              scroll={{ x: 1280 }}
              columns={riskTimelineColumns()}
            />
            <Typography.Title level={5}>用户分布</Typography.Title>
            <Table
              rowKey="username"
              size="small"
              loading={detailLoading}
              dataSource={hostUsers}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户分布" /> }}
              pagination={false}
              onRow={(record) => ({
                onClick: () => {
                  const nextFilters = { ...detailFilters, username: record.username };
                  setDetailFilters(nextFilters);
                  if (selected) {
                    void loadDetailEvents(selected, nextFilters, 1, detailPageSize);
                  }
                },
              })}
              columns={[
                { title: '用户', dataIndex: 'username', width: 160 },
                { title: '命令数', dataIndex: 'commandCount', width: 100 },
                { title: '高危事件', dataIndex: 'highRiskEvents', width: 100 },
                { title: '首次活动', dataIndex: 'firstSeen', width: 180, render: (value) => formatLocalDateTime(value) },
                { title: '最近活动', dataIndex: 'lastSeen', width: 180, render: (value) => formatLocalDateTime(value) },
              ]}
            />
            <Typography.Title level={5}>主机行为画像</Typography.Title>
            <Row gutter={[12, 12]}>
              <Col xs={24} lg={8}>
                <Typography.Text strong>敏感文件访问 Top</Typography.Text>
                <Table
                  rowKey={(record) => record.name}
                  size="small"
                  loading={detailLoading}
                  dataSource={hostBehavior.filePaths}
                  locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无文件访问" /> }}
                  pagination={false}
                  onRow={(record) => ({
                    onClick: () => {
                      if (selected) {
                        void loadFileEvents(selected, record);
                      }
                    },
                  })}
                  columns={behaviorColumns('文件路径')}
                />
              </Col>
              <Col xs={24} lg={8}>
                <Typography.Text strong>有效网络外联 Top</Typography.Text>
                <Table
                  rowKey={(record) => record.name}
                  size="small"
                  loading={detailLoading}
                  dataSource={hostBehavior.network}
                  locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无网络外联" /> }}
                  pagination={false}
                  onRow={(record) => ({
                    onClick: () => {
                      if (selected) {
                        void loadNetworkEvents(selected, record);
                      }
                    },
                  })}
                  columns={behaviorColumns('目标')}
                />
              </Col>
              <Col xs={24} lg={8}>
                <Typography.Text strong>事件类型</Typography.Text>
                <Table
                  rowKey={(record) => record.name}
                  size="small"
                  loading={detailLoading}
                  dataSource={hostBehavior.eventTypes}
                  locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无事件类型" /> }}
                  pagination={false}
                  columns={behaviorColumns('类型', (value) => eventTypeLabel(value))}
                />
              </Col>
            </Row>
            <Typography.Title level={5}>{selectedFileTarget ? `${selectedFileTarget.name} 文件访问明细` : '文件访问明细'}</Typography.Title>
            <Table
              rowKey="eventId"
              size="small"
              loading={fileLoading}
              dataSource={fileEvents}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="点击敏感文件查看访问明细" /> }}
              pagination={false}
              scroll={{ x: 1200 }}
              columns={fileColumns()}
            />
            <Typography.Title level={5}>{selectedNetworkTarget ? `${selectedNetworkTarget.name} 连接明细` : '网络连接明细'}</Typography.Title>
            <Table
              rowKey="eventId"
              size="small"
              loading={networkLoading}
              dataSource={networkEvents}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="点击网络目标查看连接明细" /> }}
              pagination={false}
              scroll={{ x: 1140 }}
              columns={networkColumns()}
            />
            <Typography.Title level={5}>规则命中分布</Typography.Title>
            <Table
              rowKey={(record) => record.name}
              size="small"
              loading={detailLoading}
              dataSource={hostBehavior.ruleHits}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无规则命中" /> }}
              pagination={false}
              columns={behaviorColumns('规则')}
            />
            <Typography.Title level={5}>命令明细</Typography.Title>
            <Space wrap>
              <Select
                allowClear
                placeholder="用户"
                style={{ width: 180 }}
                value={detailFilters.username}
                onChange={(value) => setDetailFilters((current) => ({ ...current, username: value }))}
                options={hostUsers.map((item) => ({ value: item.username, label: item.username }))}
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
                setDetailFilters({});
                if (selected) {
                  void loadDetailEvents(selected, {}, 1, detailPageSize);
                }
              }}>清空筛选</Button>
              <Button onClick={() => void exportDetailEvents()}>导出明细</Button>
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
          columns={commandColumns()}
        />
      </Drawer>
    </>
  );
}

function parseNetworkTarget(value: string) {
  if (value.startsWith('[')) {
    const end = value.indexOf(']');
    const ip = end > 0 ? value.slice(1, end) : value;
    const portText = end > 0 && value[end + 1] === ':' ? value.slice(end + 2) : '';
    const port = Number(portText);
    return { ip, port: Number.isFinite(port) && port > 0 ? port : undefined };
  }
  const separator = value.lastIndexOf(':');
  if (separator > -1) {
    const port = Number(value.slice(separator + 1));
    if (Number.isFinite(port) && port > 0) {
      return { ip: value.slice(0, separator), port };
    }
  }
  return { ip: value };
}

function behaviorColumns(title: string, renderName?: (value: string) => string) {
  return [
    { title, dataIndex: 'name', ellipsis: true, render: (value: string) => renderName ? renderName(value) : value },
    { title: '次数', dataIndex: 'count', width: 76 },
    { title: '最近', dataIndex: 'lastSeen', width: 150, render: (value: string) => formatLocalDateTime(value) },
  ] as Array<{
    title: string;
    dataIndex: keyof BehaviorItem;
    width?: number;
    ellipsis?: boolean;
    render?: (value: string) => string;
  }>;
}

function networkColumns() {
  return [
    { title: '时间', dataIndex: 'eventTime', width: 170, render: (value: string) => formatLocalDateTime(value) },
    { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_: string, record: AuditEvent) => record.loginUsername || record.username },
    { title: '执行用户', dataIndex: 'username', width: 110 },
    { title: '进程', dataIndex: 'processName', width: 120 },
    { title: '进程链路', width: 220, render: (_: string, record: AuditEvent) => <ProcessChain event={record} compact /> },
    { title: '命令', dataIndex: 'cmdline', render: (value: string) => <CommandText value={value} /> },
    { title: '目标', dataIndex: 'dstIp', width: 190, render: (_: string, record: AuditEvent) => formatNetworkTarget(record) },
    { title: '协议', dataIndex: 'protocol', width: 80, render: (value: string) => value || '-' },
    { title: '等级', dataIndex: 'severity', width: 90, render: (value: string) => <SeverityTag value={value} /> },
  ];
}

function fileColumns() {
  return [
    { title: '时间', dataIndex: 'eventTime', width: 170, render: (value: string) => formatLocalDateTime(value) },
    { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_: string, record: AuditEvent) => record.loginUsername || record.username },
    { title: '执行用户', dataIndex: 'username', width: 110 },
    { title: '进程', dataIndex: 'processName', width: 120 },
    { title: '进程链路', width: 220, render: (_: string, record: AuditEvent) => <ProcessChain event={record} compact /> },
    { title: '操作', dataIndex: 'fileOperation', width: 150, render: (value: string) => value || '-' },
    { title: '文件路径', dataIndex: 'filePath', render: (value: string) => value || '-' },
    { title: '命中规则', dataIndex: 'ruleNames', width: 220, render: (value: string[]) => value?.join('、') || '-' },
    { title: '等级', dataIndex: 'severity', width: 90, render: (value: string) => <SeverityTag value={value} /> },
  ];
}

function riskTimelineColumns() {
  return [
    { title: '时间', dataIndex: 'eventTime', width: 170, render: (value: string) => formatLocalDateTime(value) },
    { title: '类型', dataIndex: 'eventType', width: 110, render: (value: string) => eventTypeLabel(value) },
    { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_: string, record: AuditEvent) => record.loginUsername || record.username },
    { title: '执行用户', dataIndex: 'username', width: 110 },
    { title: '进程', dataIndex: 'processName', width: 120 },
    { title: '进程链路', width: 220, render: (_: string, record: AuditEvent) => <ProcessChain event={record} compact /> },
    { title: '风险目标', dataIndex: 'cmdline', render: (_: string, record: AuditEvent) => riskTarget(record) },
    { title: '命中规则', dataIndex: 'ruleNames', width: 220, render: (value: string[]) => value?.join('、') || '-' },
    { title: '等级', dataIndex: 'severity', width: 90, render: (value: string) => <SeverityTag value={value} /> },
  ];
}

function riskTarget(record: AuditEvent) {
  if (record.eventType === 'file_access' && record.filePath) {
    return `${record.fileOperation || '访问'} ${record.filePath}`;
  }
  if (record.eventType === 'network_connect' && record.dstIp) {
    return formatNetworkTarget(record);
  }
  return <CommandText value={record.cmdline || record.processName || '-'} />;
}

function formatNetworkTarget(record: AuditEvent) {
  if (!record.dstIp) {
    return '-';
  }
  const ip = record.dstIp.includes(':') ? `[${record.dstIp}]` : record.dstIp;
  return record.dstPort ? `${ip}:${record.dstPort}` : ip;
}

function commandColumns() {
  return [
    { title: '时间', dataIndex: 'eventTime', width: 190, render: (value: string) => formatLocalDateTime(value) },
    { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_: string, record: AuditEvent) => record.loginUsername || record.username },
    { title: '执行用户', dataIndex: 'username', width: 110 },
    { title: '进程', dataIndex: 'processName', width: 120 },
    { title: '进程链路', width: 220, render: (_: string, record: AuditEvent) => <ProcessChain event={record} compact /> },
    { title: '命令', dataIndex: 'cmdline', render: (value: string) => <CommandText value={value} /> },
    { title: '等级', dataIndex: 'severity', width: 90, render: (value: string) => <SeverityTag value={value} /> },
  ];
}
