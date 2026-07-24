import { Alert, Card, Col, Empty, Row, Statistic, Typography } from 'antd';
import dayjs from 'dayjs';
import type { EChartsOption } from 'echarts';
import { useEffect, useState } from 'react';
import { getEventTrend, getOverview, getTopCommands, getTopHosts, getTopNamespaces } from '../../api/stats';
import EChart from '../../components/EChart';
import type { OverviewStats, TopItem, TrendPoint } from '../../types/stats';
import { compactNumber } from '../../utils/format';

const emptyOverview: OverviewStats = {
  totalEvents: 0,
  highRiskEvents: 0,
  activeHosts: 0,
  activeRules: 0,
};

// DashboardPage 渲染 Dashboard Page 组件。
export default function DashboardPage() {
  const [overview, setOverview] = useState<OverviewStats>(emptyOverview);
  const [trend, setTrend] = useState<TrendPoint[]>([]);
  const [topCommands, setTopCommands] = useState<TopItem[]>([]);
  const [topHosts, setTopHosts] = useState<TopItem[]>([]);
  const [topNamespaces, setTopNamespaces] = useState<TopItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // load 加载页面所需数据。
  async function load() {
    setLoading(true);
    setError('');
    try {
      const todayQuery = {
        start_time: dayjs().startOf('day').toISOString(),
        end_time: dayjs().endOf('day').toISOString(),
      };
      const [overviewData, trendData, commandData, hostData, namespaceData] = await Promise.all([
        getOverview(todayQuery),
        getEventTrend(todayQuery),
        getTopCommands(50, todayQuery),
        getTopHosts(12, todayQuery),
        getTopNamespaces(12, todayQuery),
      ]);
      setOverview(overviewData);
      setTrend(trendData ?? []);
      setTopCommands(commandData ?? []);
      setTopHosts(hostData ?? []);
      setTopNamespaces(namespaceData ?? []);
    } catch {
      setError('统计数据加载失败');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const trendOption: EChartsOption = {
    grid: { left: 36, right: 16, top: 24, bottom: 36 },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: trend.map((item) => item.time),
      axisLabel: { color: '#6b7280' },
      axisLine: { lineStyle: { color: '#e5e7eb' } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#6b7280' },
      splitLine: { lineStyle: { color: '#eef2f7' } },
    },
    series: [{
      type: 'line',
      smooth: true,
      symbolSize: 6,
      areaStyle: { color: 'rgba(37, 99, 235, 0.12)' },
      lineStyle: { color: '#2563eb', width: 2 },
      itemStyle: { color: '#2563eb' },
      data: trend.map((item) => item.count),
    }],
  };

  const commandOption: EChartsOption = {
    grid: { left: 92, right: 24, top: 16, bottom: 20 },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'value',
      axisLabel: { color: '#6b7280' },
      splitLine: { lineStyle: { color: '#eef2f7' } },
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: topCommands.slice(0, 12).map((item) => item.name),
      axisLabel: {
        color: '#374151',
        width: 80,
        overflow: 'truncate',
      },
      axisLine: { lineStyle: { color: '#e5e7eb' } },
    },
    series: [{
      type: 'bar',
      barWidth: 12,
      itemStyle: { color: '#0f766e', borderRadius: [0, 4, 4, 0] },
      data: topCommands.slice(0, 12).map((item) => item.count),
    }],
  };

  const hostOption = topBarOption(topHosts, '#2563eb');
  const namespaceOption = topBarOption(topNamespaces, '#7c3aed');

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">审计概览</Typography.Title>
      </div>
      {error && <Alert className="toolbar" type="error" showIcon message={error} />}
      <Row gutter={[16, 16]}>
        <Col xs={24} md={6}>
          <Card className="stat-card" loading={loading}><Statistic title="今日事件" value={compactNumber(overview.totalEvents)} /></Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="stat-card stat-card-danger" loading={loading}><Statistic title="高危事件" value={compactNumber(overview.highRiskEvents)} /></Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="stat-card" loading={loading}><Statistic title="活跃主机" value={compactNumber(overview.activeHosts)} /></Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="stat-card" loading={loading}><Statistic title="启用规则" value={compactNumber(overview.activeRules)} /></Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="今日事件趋势" loading={loading}>
            {trend.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无趋势数据" />
            ) : (
              <EChart option={trendOption} />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="TOP 命令" loading={loading}>
            {topCommands.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令数据" />
            ) : (
              <EChart option={commandOption} />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="TOP 主机" loading={loading}>
            {topHosts.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无主机数据" />
            ) : (
              <EChart option={hostOption} />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="TOP Namespace" loading={loading}>
            {topNamespaces.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无 Namespace 数据" />
            ) : (
              <EChart option={namespaceOption} />
            )}
          </Card>
        </Col>
      </Row>
    </>
  );
}

// topBarOption 转换 top Bar Option 的数据结构。
function topBarOption(items: TopItem[], color: string): EChartsOption {
  return {
    grid: { left: 112, right: 24, top: 16, bottom: 20 },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'value',
      axisLabel: { color: '#6b7280' },
      splitLine: { lineStyle: { color: '#eef2f7' } },
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: items.map((item) => item.name),
      axisLabel: {
        color: '#374151',
        width: 100,
        overflow: 'truncate',
      },
      axisLine: { lineStyle: { color: '#e5e7eb' } },
    },
    series: [{
      type: 'bar',
      barWidth: 12,
      itemStyle: { color, borderRadius: [0, 4, 4, 0] },
      data: items.map((item) => item.count),
    }],
  };
}
