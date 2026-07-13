import { Card, DatePicker, Drawer, Empty, Form, Input, Space, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { queryAuditEvents } from '../../api/audit';
import { listHostAssets } from '../../api/hostAssets';
import { getHostAudits } from '../../api/stats';
import CommandText from '../../components/CommandText';
import FilterToolbar from '../../components/FilterToolbar';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import type { HostAsset } from '../../types/hostAsset';
import type { HostAuditItem, HostAuditQuery } from '../../types/stats';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

export default function HostAuditPage() {
  const [items, setItems] = useState<HostAuditItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selected, setSelected] = useState<HostAuditItem>();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [assetMap, setAssetMap] = useState<Record<string, HostAsset>>({});
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
      const [hostItems, assets] = await Promise.all([getHostAudits(buildQuery()), listHostAssets()]);
      setItems(hostItems);
      setAssetMap(Object.fromEntries(assets.map((asset) => [asset.nodeName, asset])));
    } finally {
      setLoading(false);
    }
  }

  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load();
  }

  function hostLabel(hostName: string) {
    return assetMap[hostName]?.displayName || hostName;
  }

  async function openDetails(item: HostAuditItem) {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    setSelected(item);
    setDetailLoading(true);
    try {
      const data = await queryAuditEvents({
        start_time: range?.[0]?.startOf('day').toISOString(),
        end_time: range?.[1]?.endOf('day').toISOString(),
        event_type: 'process_exec',
        host_name: item.hostName,
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
          rowKey="hostName"
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无主机审计数据" /> }}
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
            {
              title: '主机',
              dataIndex: 'hostName',
              render: (value: string) => (
                <Space direction="vertical" size={0}>
                  <Typography.Text>{hostLabel(value)}</Typography.Text>
                  {assetMap[value]?.displayName && <Typography.Text type="secondary">{value}</Typography.Text>}
                </Space>
              ),
            },
            { title: '命令数', dataIndex: 'commandCount', width: 110 },
            { title: '活跃用户', dataIndex: 'activeUsers', width: 110 },
            { title: '高危事件', dataIndex: 'highRiskEvents', width: 110 },
            { title: '首次活动', dataIndex: 'firstSeen', width: 190 },
            { title: '最近活动', dataIndex: 'lastSeen', width: 190 },
          ]}
        />
      </Card>
      <Drawer
        title={selected?.hostName ? `${hostLabel(selected.hostName)} 命令明细` : '命令明细'}
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
            { title: '时间', dataIndex: 'eventTime', width: 190 },
            { title: '登录用户', dataIndex: 'loginUsername', width: 110, render: (_, record) => record.loginUsername || record.username },
            { title: '执行用户', dataIndex: 'username', width: 110 },
            { title: '进程', dataIndex: 'processName', width: 120 },
            { title: '命令', dataIndex: 'cmdline', render: (value) => <CommandText value={value} /> },
            { title: '等级', dataIndex: 'severity', width: 90, render: (value) => <SeverityTag value={value} /> },
          ]}
        />
      </Drawer>
    </>
  );
}
