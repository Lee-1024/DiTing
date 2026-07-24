import { ReloadOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Space, Table, Tag, Typography } from 'antd';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { listCollectorHealth } from '../../api/collectorHealth';
import { InsightHero, LatestPanel, MetricCard } from '../../components/InsightHeader';
import type { CollectorHeartbeat } from '../../types/collectorHealth';
import { compactNumber } from '../../utils/format';
import { formatLocalDateTime } from '../../utils/time';

// CollectorHealthPage 渲染 Collector Health Page 组件。
export default function CollectorHealthPage() {
  const [items, setItems] = useState<CollectorHeartbeat[]>([]);
  const [loading, setLoading] = useState(false);
  const [tablePageSize, setTablePageSize] = useState(10);

  // load 加载页面所需数据。
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

  const onlineCount = items.filter((item) => item.status === 'online').length;
  const criticalCount = items.filter((item) => item.healthLevel === 'critical').length;
  const warningCount = items.filter((item) => item.healthLevel === 'warning').length;
  const bufferedTotal = items.reduce((sum, item) => sum + (item.bufferedEvents || 0), 0);
  const droppedTotal = items.reduce((sum, item) => sum + (item.droppedEvents || 0), 0);
  const writtenTotal = items.reduce((sum, item) => sum + (item.eventsWritten || 0), 0);
  const latestProblem = items.find((item) => item.healthLevel === 'critical' || item.healthLevel === 'warning');

  return (
    <>
      <Space className="page-heading" align="center">
        <div>
          <span className="page-kicker">COLLECTOR HEALTH</span>
          <Typography.Title level={3} className="page-title">采集状态工作台</Typography.Title>
        </div>
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>刷新</Button>
      </Space>
      <div className="collector-hero">
        <InsightHero
          className="collector-summary"
          kicker="Collector Pipeline"
          title="监控采集在线率、延迟、缓冲和丢弃风险"
          description="Collector 是审计数据入口；这里优先暴露离线、异常、写入延迟和缓冲堆积。"
          actions={(
            <>
            <Link to="/settings/collector-debug"><Button type="primary">采集调试</Button></Link>
            <Link to="/audit/events"><Button ghost>查看操作日志</Button></Link>
            </>
          )}
        />
        <LatestPanel
          label="当前异常"
          title={latestProblem?.hostName || latestProblem?.hostId || (criticalCount || warningCount ? '采集异常' : '暂无异常')}
          description={latestProblem?.lastError || latestProblem?.message || 'Collector 状态平稳'}
        />
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="在线 Collector" value={onlineCount} hint={`共 ${items.length} 个采集端`} tone="success" />
        <MetricCard label="异常 / 预警" value={criticalCount + warningCount} hint={`${criticalCount} 异常，${warningCount} 预警`} tone={criticalCount ? 'danger' : 'warning'} />
        <MetricCard label="缓冲事件" value={bufferedTotal} hint="待写入队列" tone={bufferedTotal ? 'warning' : 'blue'} />
        <MetricCard label="累计丢弃" value={droppedTotal} hint={`累计写入 ${compactNumber(writtenTotal)}`} tone={droppedTotal ? 'danger' : 'blue'} />
      </div>
      <Card className="data-card">
        <Table
          rowKey="hostId"
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无采集状态" /> }}
          scroll={{ x: 1360 }}
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
            { title: '采集模式', dataIndex: 'inputMode', width: 110, render: (value) => inputModeLabel(value) },
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
            { title: '提示', dataIndex: 'message', width: 220, ellipsis: true, render: (value) => value || '-' },
            { title: '最近错误', dataIndex: 'lastError', width: 280, ellipsis: true, render: (value) => value || '-' },
            { title: '最近心跳', dataIndex: 'lastSeenAt', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '心跳延迟', dataIndex: 'heartbeatLagSeconds', width: 124, align: 'right', className: 'number-cell', render: (value) => formatLag(value) },
            { title: '最近事件', dataIndex: 'lastEventTime', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '事件延迟', dataIndex: 'eventLagSeconds', width: 124, align: 'right', className: 'number-cell', render: (value) => formatLag(value) },
            { title: '最近写入', dataIndex: 'lastWriteAt', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '写入延迟', dataIndex: 'writeLagSeconds', width: 124, align: 'right', className: 'number-cell', render: (value) => formatLag(value) },
            { title: '累计写入', dataIndex: 'eventsWritten', width: 136, align: 'right', className: 'number-cell', render: (value) => compactNumber(value) },
            {
              title: '缓冲中',
              dataIndex: 'bufferedEvents',
              width: 124,
              align: 'right',
              className: 'number-cell',
              render: (value) => value ? <Tag color="gold">{compactNumber(value)}</Tag> : compactNumber(value || 0),
            },
            {
              title: '累计丢弃',
              dataIndex: 'droppedEvents',
              width: 136,
              align: 'right',
              className: 'number-cell',
              render: (value) => value ? <Tag color="red">{compactNumber(value)}</Tag> : compactNumber(value || 0),
            },
            { title: '更新时间', dataIndex: 'updatedAt', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
    </>
  );
}

// healthLabel 生成 health Label 的展示内容。
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

// healthColor 生成 health Color 的展示内容。
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

// inputModeLabel 生成 input Mode Label 的展示内容。
function inputModeLabel(value?: string) {
  if (value === 'grpc') {
    return <Tag color="blue">gRPC</Tag>;
  }
  return <Tag>文件</Tag>;
}

// formatLag 格式化 format Lag 以便界面展示。
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
