import { Card, DatePicker, Drawer, Empty, Form, Input, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { queryAuditEvents } from '../../api/audit';
import { exportCommandStats, getCommandStats } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { CommandItem, CommandStatsQuery } from '../../types/stats';
import { downloadBlob } from '../../utils/download';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

export default function CommandStatsPage() {
  const [items, setItems] = useState<CommandItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selected, setSelected] = useState<CommandItem>();
  const [executions, setExecutions] = useState<AuditEvent[]>([]);
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

  function buildQuery(): CommandStatsQuery {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      keyword: values.keyword,
      username: values.username,
      limit: 50,
    };
  }

  async function load() {
    setLoading(true);
    try {
      setItems(await getCommandStats(buildQuery()));
    } finally {
      setLoading(false);
    }
  }

  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load();
  }

  async function exportCSV() {
    const blob = await exportCommandStats(buildQuery());
    downloadBlob(blob, 'command-stats.csv');
  }

  async function openDetails(item: CommandItem) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setSelected(item);
    setDetailLoading(true);
    try {
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        username: item.loginUsername || item.username,
        cmdline: item.cmdline,
        page: 1,
        page_size: 100,
      });
      setExecutions(data.items ?? []);
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
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey={(record) => `${record.processName}-${record.cmdline}-${record.username}`}
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令统计" /> }}
          onRow={(record) => ({ onClick: () => void openDetails(record) })}
          scroll={{ x: 1220 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '命令', dataIndex: 'processName', width: 150 },
            { title: '完整命令', dataIndex: 'cmdline', render: (value, record) => <CommandText value={value} onView={() => void openDetails(record)} /> },
            { title: '登录用户', dataIndex: 'loginUsername', width: 120, render: (_, record) => record.loginUsername || record.username },
            { title: '执行用户', dataIndex: 'username', width: 120 },
            { title: '次数', dataIndex: 'count', width: 90 },
            { title: '首次执行', dataIndex: 'firstSeen', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近执行', dataIndex: 'lastSeen', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
      <Drawer
        title={selected?.processName ? `${selected.processName} 执行明细` : '执行明细'}
        width={960}
        open={Boolean(selected)}
        onClose={() => { setSelected(undefined); setExecutions([]); }}
      >
        <Table
          rowKey="eventId"
          size="small"
          loading={detailLoading}
          dataSource={executions}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无执行明细" /> }}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: '时间', dataIndex: 'eventTime', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_, record) => record.loginUsername || record.username },
            { title: '执行用户', dataIndex: 'username', width: 110 },
            { title: '节点', dataIndex: 'nodeName', width: 150, render: (_, record) => record.nodeName || record.hostName },
            { title: '进程', dataIndex: 'processName', width: 120 },
            { title: '命令', dataIndex: 'cmdline', render: (value) => <CommandText value={value} /> },
            { title: '等级', dataIndex: 'severity', width: 90, render: (value) => <SeverityTag value={value} /> },
          ]}
        />
      </Drawer>
    </>
  );
}
