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
            {
              title: '健康',
              dataIndex: 'healthLevel',
              width: 110,
              render: (value) => <Tag color={healthColor(value)}>{healthLabel(value)}</Tag>,
            },
            { title: '提示', dataIndex: 'message', width: 180, render: (value) => value || '-' },
            { title: '最近心跳', dataIndex: 'lastSeenAt', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '心跳延迟', dataIndex: 'heartbeatLagSeconds', width: 110, render: (value) => formatLag(value) },
            { title: '最近事件', dataIndex: 'lastEventTime', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '事件延迟', dataIndex: 'eventLagSeconds', width: 110, render: (value) => formatLag(value) },
            { title: '最近写入', dataIndex: 'lastWriteAt', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '写入延迟', dataIndex: 'writeLagSeconds', width: 110, render: (value) => formatLag(value) },
            { title: '累计写入', dataIndex: 'eventsWritten', width: 120, render: (value) => compactNumber(value) },
            { title: '更新时间', dataIndex: 'updatedAt', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
    </>
  );
}

function healthLabel(value?: string) {
  if (value === 'healthy') {
    return '正常';
  }
  if (value === 'warning') {
    return '预警';
  }
  if (value === 'critical') {
    return '异常';
  }
  return value || '-';
}

function healthColor(value?: string) {
  if (value === 'healthy') {
    return 'green';
  }
  if (value === 'warning') {
    return 'gold';
  }
  if (value === 'critical') {
    return 'red';
  }
  return 'default';
}

function formatLag(value?: number) {
  if (value === undefined || value === null) {
    return '-';
  }
  if (value < 60) {
    return `${value}s`;
  }
  const minutes = Math.floor(value / 60);
  const seconds = value % 60;
  if (minutes < 60) {
    return seconds ? `${minutes}m ${seconds}s` : `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const restMinutes = minutes % 60;
  return restMinutes ? `${hours}h ${restMinutes}m` : `${hours}h`;
}
