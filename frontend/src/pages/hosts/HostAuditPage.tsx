import { Card, DatePicker, Descriptions, Drawer, Empty, Form, Input, Row, Col, Space, Statistic, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { queryAuditEvents } from '../../api/audit';
import { getHostAudits, getHostUsers } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { HostAuditItem, HostAuditQuery, HostUserItem } from '../../types/stats';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

export default function HostAuditPage() {
  const [items, setItems] = useState<HostAuditItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selected, setSelected] = useState<HostAuditItem>();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [riskEvents, setRiskEvents] = useState<AuditEvent[]>([]);
  const [hostUsers, setHostUsers] = useState<HostUserItem[]>([]);
  const [selectedUser, setSelectedUser] = useState<string>();
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
      const [data, riskData, usersData] = await Promise.all([
        queryAuditEvents({
          ...baseQuery,
          page_size: 100,
        }),
        queryAuditEvents({
          ...baseQuery,
          severity_in: 'high,critical',
          page_size: 10,
        }),
        getHostUsers(hostUserQuery),
      ]);
      setEvents(data.items ?? []);
      setRiskEvents(riskData.items ?? []);
      setHostUsers(usersData ?? []);
      setSelectedUser(undefined);
    } finally {
      setDetailLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">主机审计</Typography.Title>
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={() => void load()} onReset={() => void resetAndLoad()}>
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
        onClose={() => { setSelected(undefined); setEvents([]); setRiskEvents([]); setHostUsers([]); setSelectedUser(undefined); }}
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
            <Typography.Title level={5}>用户分布</Typography.Title>
            <Table
              rowKey="username"
              size="small"
              loading={detailLoading}
              dataSource={hostUsers}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户分布" /> }}
              pagination={false}
              onRow={(record) => ({ onClick: () => setSelectedUser(record.username) })}
              columns={[
                { title: '用户', dataIndex: 'username', width: 160 },
                { title: '命令数', dataIndex: 'commandCount', width: 100 },
                { title: '高危事件', dataIndex: 'highRiskEvents', width: 100 },
                { title: '首次活动', dataIndex: 'firstSeen', width: 180, render: (value) => formatLocalDateTime(value) },
                { title: '最近活动', dataIndex: 'lastSeen', width: 180, render: (value) => formatLocalDateTime(value) },
              ]}
            />
            <Typography.Title level={5}>命令明细</Typography.Title>
          </Space>
        )}
        <Table
          rowKey="eventId"
          size="small"
          loading={detailLoading}
          dataSource={selectedUser ? events.filter((event) => (event.loginUsername || event.username) === selectedUser) : events}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令明细" /> }}
          pagination={{ pageSize: 10 }}
          columns={commandColumns()}
        />
      </Drawer>
    </>
  );
}

function commandColumns() {
  return [
    { title: '时间', dataIndex: 'eventTime', width: 190, render: (value: string) => formatLocalDateTime(value) },
    { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_: string, record: AuditEvent) => record.loginUsername || record.username },
    { title: '执行用户', dataIndex: 'username', width: 110 },
    { title: '进程', dataIndex: 'processName', width: 120 },
    { title: '命令', dataIndex: 'cmdline', render: (value: string) => <CommandText value={value} /> },
    { title: '等级', dataIndex: 'severity', width: 90, render: (value: string) => <SeverityTag value={value} /> },
  ];
}
