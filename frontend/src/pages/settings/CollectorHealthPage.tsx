import { ReloadOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Space, Table, Tag, Typography } from 'antd';
import { useEffect, useState } from 'react';
import { listCollectorHealth } from '../../api/collectorHealth';
import type { CollectorHeartbeat } from '../../types/collectorHealth';
import { compactNumber } from '../../utils/format';
import { formatLocalDateTime } from '../../utils/time';

export default function CollectorHealthPage() {
  const [items, setItems] = useState<CollectorHeartbeat[]>([]);
  const [loading, setLoading] = useState(false);
  const [tablePageSize, setTablePageSize] = useState(10);

  async function load() {
    setLoading(true);
    try {
      setItems(await listCollectorHealth());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <>
      <Space className="page-heading">
        <Typography.Title level={3} className="page-title">采集状态</Typography.Title>
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>刷新</Button>
      </Space>
      <Card className="data-card">
        <Table
          rowKey="hostId"
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无采集状态" /> }}
          scroll={{ x: 1120 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '主机 ID', dataIndex: 'hostId', width: 180 },
            { title: '主机名', dataIndex: 'hostName', width: 180, render: (value) => value || '-' },
            {
              title: '状态',
              dataIndex: 'status',
              width: 100,
              render: (value) => <Tag color={value === 'online' ? 'green' : 'red'}>{value === 'online' ? '在线' : '离线'}</Tag>,
            },
            { title: '最近心跳', dataIndex: 'lastSeenAt', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近事件', dataIndex: 'lastEventTime', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近写入', dataIndex: 'lastWriteAt', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '累计写入', dataIndex: 'eventsWritten', width: 120, render: (value) => compactNumber(value) },
          ]}
        />
      </Card>
    </>
  );
}
