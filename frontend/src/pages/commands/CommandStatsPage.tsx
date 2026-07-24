import { Card, DatePicker, Descriptions, Drawer, Empty, Form, Input, Space, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { exportAuditEvents, queryAuditEvents } from '../../api/audit';
import { exportCommandStats, getCommandStats } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { CommandItem, CommandStatsQuery } from '../../types/stats';
import { downloadBlob } from '../../utils/download';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

// commandName 生成 command Name 的展示内容。
function commandName(item: CommandItem) {
  return item.processName || item.cmdline?.split(/\s+/)[0] || '-';
}

// CommandStatsPage 生成 Command Stats Page 的展示内容。
export default function CommandStatsPage() {
  const [items, setItems] = useState<CommandItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selected, setSelected] = useState<CommandItem>();
  const [executions, setExecutions] = useState<AuditEvent[]>([]);
  const [riskExecutions, setRiskExecutions] = useState<AuditEvent[]>([]);
  const [detailPage, setDetailPage] = useState(1);
  const [detailPageSize, setDetailPageSize] = useState(10);
  const [detailTotal, setDetailTotal] = useState(0);
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

  // buildQuery 构建 build Query 所需的数据结构。
  function buildQuery(): CommandStatsQuery {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      keyword: values.keyword,
      username: values.username,
      host_name: values.hostName,
      limit: 50,
    };
  }

  // load 加载页面所需数据。
  async function load() {
    setLoading(true);
    try {
      setItems(await getCommandStats(buildQuery()));
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

  // exportCSV 导出或下载 export CSV 数据。
  async function exportCSV() {
    const blob = await exportCommandStats(buildQuery());
    downloadBlob(blob, 'command-stats.csv');
  }

  // openDetails 打开对应的弹窗或详情视图。
  async function openDetails(item: CommandItem) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setSelected(item);
    setDetailLoading(true);
    try {
      const baseQuery = {
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        username: item.loginUsername || item.username,
        cmdline: item.cmdline,
      };
      const [data, riskData] = await Promise.all([
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
      ]);
      setExecutions(data.items ?? []);
      setDetailPage(data.page);
      setDetailTotal(data.total);
      setRiskExecutions(riskData.items ?? []);
    } finally {
      setDetailLoading(false);
    }
  }

  // loadDetailEvents 加载页面所需数据。
  async function loadDetailEvents(item: CommandItem, nextPage = detailPage, nextPageSize = detailPageSize) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setDetailLoading(true);
    try {
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        username: item.loginUsername || item.username,
        cmdline: item.cmdline,
        page: nextPage,
        page_size: nextPageSize,
      });
      setExecutions(data.items ?? []);
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
      username: selected.loginUsername || selected.username,
      cmdline: selected.cmdline,
    });
    downloadBlob(blob, `${selected.processName || 'command'}-details.csv`);
  }

  // closeDetails 关闭当前弹窗或详情视图。
  function closeDetails() {
    setSelected(undefined);
    setExecutions([]);
    setRiskExecutions([]);
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

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">命令审计</Typography.Title>
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={() => void load()} onReset={() => void resetAndLoad()} onExport={() => void exportCSV()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="keyword" label="命令">
          <Input className="filter-control-compact" placeholder="whoami / docker" allowClear />
        </Form.Item>
        <Form.Item name="username" label="用户">
          <Input className="filter-control-compact" placeholder="root / ubuntu" allowClear />
        </Form.Item>
        <Form.Item name="hostName" label="主机">
          <Input className="filter-control-compact" placeholder="主机名 / Host ID" allowClear />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey={(record) => `${record.processName}-${record.cmdline}-${record.username}`}
          loading={loading}
          dataSource={items}
          className="clickable-table"
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令统计" /> }}
          onRow={(record) => ({ onClick: () => void openDetails(record), title: '点击查看命令执行明细' })}
          scroll={{ x: 1220 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '命令', dataIndex: 'processName', width: 150, render: (_, record) => commandName(record) },
            { title: '完整命令', dataIndex: 'cmdline', render: (value, record) => <CommandText value={value} onView={() => void openDetails(record)} /> },
            { title: '登录用户', dataIndex: 'loginUsername', width: 120, render: (_, record) => record.loginUsername || record.username },
            { title: '执行用户', dataIndex: 'username', width: 120 },
            { title: '最近主机', dataIndex: 'hostName', width: 170, render: (_, record) => record.hostName || record.nodeName || record.hostId || '-' },
            { title: '涉及主机', dataIndex: 'hostCount', width: 116, align: 'right', className: 'number-cell' },
            { title: '次数', dataIndex: 'count', width: 104, align: 'right', className: 'number-cell' },
            { title: '首次执行', dataIndex: 'firstSeen', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近执行', dataIndex: 'lastSeen', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
      <Drawer
        title={selected?.processName ? `${selected.processName} 执行明细` : '执行明细'}
        width={1120}
        open={Boolean(selected)}
        onClose={closeDetails}
      >
        {selected && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="命令">{commandName(selected)}</Descriptions.Item>
              <Descriptions.Item label="涉及主机">{selected.hostCount}</Descriptions.Item>
              <Descriptions.Item label="登录用户">{selected.loginUsername || selected.username}</Descriptions.Item>
              <Descriptions.Item label="执行用户">{selected.username}</Descriptions.Item>
              <Descriptions.Item label="执行次数">{selected.count}</Descriptions.Item>
              <Descriptions.Item label="最近主机">{selected.hostName || selected.nodeName || selected.hostId || '-'}</Descriptions.Item>
              <Descriptions.Item label="首次执行">{formatLocalDateTime(selected.firstSeen)}</Descriptions.Item>
              <Descriptions.Item label="最近执行">{formatLocalDateTime(selected.lastSeen)}</Descriptions.Item>
            </Descriptions>
            <Typography.Title level={5}>高危命中</Typography.Title>
            <Table
              rowKey="eventId"
              size="small"
              loading={detailLoading}
              dataSource={riskExecutions}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无高危命中" /> }}
              pagination={false}
              columns={detailColumns()}
            />
            <Space>
              <Typography.Title level={5} style={{ margin: 0 }}>执行明细</Typography.Title>
              <Typography.Link onClick={() => void exportDetails()}>导出明细</Typography.Link>
            </Space>
          </Space>
        )}
        <Table
          rowKey="eventId"
          size="small"
          loading={detailLoading}
          dataSource={executions}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无执行明细" /> }}
          pagination={{
            current: detailPage,
            pageSize: detailPageSize,
            total: detailTotal,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onChange: (nextPage, nextPageSize) => {
              if (selected) {
                void loadDetailEvents(selected, nextPage, nextPageSize);
              }
            },
          }}
          columns={detailColumns()}
        />
      </Drawer>
    </>
  );
}
