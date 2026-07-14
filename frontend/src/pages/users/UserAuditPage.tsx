import { Card, DatePicker, Drawer, Empty, Form, Input, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { queryAuditEvents } from '../../api/audit';
import { getUserAudits } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { UserAuditItem, UserAuditQuery } from '../../types/stats';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

export default function UserAuditPage() {
  const [items, setItems] = useState<UserAuditItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selected, setSelected] = useState<UserAuditItem>();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

  function buildQuery(): UserAuditQuery {
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
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        username: item.username,
        page: 1,
        page_size: 100,
      });
      setEvents(data.items ?? []);
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
        <Typography.Title level={3} className="page-title">用户审计</Typography.Title>
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={() => void load()} onReset={() => void resetAndLoad()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="keyword" label="用户">
          <Input className="filter-control-compact" placeholder="root / ubuntu" allowClear />
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
        title={selected?.username ? `${selected.username} 命令明细` : '命令明细'}
        width={960}
        open={Boolean(selected)}
        onClose={() => { setSelected(undefined); setEvents([]); }}
      >
        <Table
          rowKey="eventId"
          size="small"
          loading={detailLoading}
          dataSource={events}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令明细" /> }}
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
