import { Alert, Button, Card, Empty, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import type { EChartsOption } from 'echarts';
import type { CSSProperties } from 'react';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
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

// DashboardPage 渲染安全运营工作台。
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

  const riskRate = overview.totalEvents > 0 ? Math.round((overview.highRiskEvents / overview.totalEvents) * 1000) / 10 : 0;
  const posture = overview.highRiskEvents > 0 ? '需要关注' : '平稳运行';
  const peakPoint = [...trend].sort((left, right) => right.count - left.count)[0];
  const trendOption = trendChartOption(trend, peakPoint);
  const commandOption = topBarOption(topCommands.slice(0, 12), '#16a34a');

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="page-kicker">REAL-TIME OPERATIONS</span>
          <Typography.Title level={3} className="page-title">今日安全态势</Typography.Title>
        </div>
      </div>
      {error && <Alert className="toolbar" type="error" showIcon message={error} />}
      <div className="ops-hero">
        <section className="ops-command-center">
          <div className="ops-command-content">
            <div className="ops-kicker">DiTing Command Center</div>
            <Typography.Title level={2} className="ops-title">聚焦高危操作、异常峰值与采集健康</Typography.Title>
            <Typography.Text className="ops-subtitle">
              当前首页按安全值守场景组织：先判断风险态势，再进入事件调查、主机画像或采集状态。
            </Typography.Text>
            <div className="ops-hero-actions">
              <Link to="/audit/risks"><Button type="primary">查看风险事件</Button></Link>
              <Link to="/settings/collector-health"><Button ghost>检查采集状态</Button></Link>
            </div>
          </div>
        </section>
        <aside className="ops-status-panel">
          <div className="ops-status-row">
            <Typography.Text type="secondary">今日态势</Typography.Text>
            <Tag color={overview.highRiskEvents > 0 ? 'red' : 'green'}>{posture}</Tag>
          </div>
          <div className="ops-status-row">
            <Typography.Text type="secondary">高危占比</Typography.Text>
            <span className="ops-status-value">{riskRate}%</span>
          </div>
          <div className="ops-status-row">
            <Typography.Text type="secondary">峰值时段</Typography.Text>
            <span className="ops-status-value">{peakPoint?.time || '-'}</span>
          </div>
        </aside>
      </div>
      <div className="metric-grid">
        <MetricCard label="今日事件" value={compactNumber(overview.totalEvents)} hint="审计事件总量" loading={loading} />
        <MetricCard label="高危事件" value={compactNumber(overview.highRiskEvents)} hint="需优先处置" loading={loading} tone="danger" />
        <MetricCard label="活跃主机" value={compactNumber(overview.activeHosts)} hint="产生审计行为" loading={loading} />
        <MetricCard label="启用规则" value={compactNumber(overview.activeRules)} hint="参与风险判定" loading={loading} tone="success" />
      </div>
      <div className="workbench-grid">
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Card className="panel-card" title="事件趋势与异常峰值" loading={loading}>
            {trend.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无趋势数据" />
            ) : (
              <EChart option={trendOption} height={318} />
            )}
          </Card>
          <Card className="panel-card" title="TOP 命令" loading={loading}>
            {topCommands.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无命令数据" />
            ) : (
              <EChart option={commandOption} />
            )}
          </Card>
        </Space>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Card className="panel-card" title="调查入口" loading={loading}>
            <div className="signal-list">
              <Signal title="高危事件调查" desc="从风险事件进入命令、文件、网络上下文。" to="/audit/risks" />
              <Signal title="主机行为画像" desc="查看主机用户分布、敏感文件和网络外联。" to="/audit/hosts" />
              <Signal title="采集健康巡检" desc="确认 Collector 心跳、延迟和写入状态。" to="/settings/collector-health" />
            </div>
          </Card>
          <Card className="panel-card" title="活跃主机" loading={loading}>
            <RankList items={topHosts} empty="暂无主机数据" />
          </Card>
          <Card className="panel-card" title="Namespace 分布" loading={loading}>
            <RankList items={topNamespaces} empty="暂无 Namespace 数据" />
          </Card>
        </Space>
      </div>
    </>
  );
}

// MetricCard 渲染安全态势指标卡。
function MetricCard({ label, value, hint, loading, tone }: { label: string; value: string; hint: string; loading: boolean; tone?: 'danger' | 'success' }) {
  const color = tone === 'danger' ? '#dc2626' : tone === 'success' ? '#16a34a' : '#2563eb';
  return (
    <div className="metric-card" style={{ '--metric-color': color } as CSSProperties}>
      <div className="metric-label">{label}</div>
      <div className="metric-value">{loading ? '-' : value}</div>
      <div className="metric-hint">{hint}</div>
    </div>
  );
}

// RankList 渲染高密度排行列表。
function RankList({ items, empty }: { items: TopItem[]; empty: string }) {
  if (items.length === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={empty} />;
  }
  return (
    <div className="rank-list">
      {items.slice(0, 6).map((item, index) => (
        <div className="rank-item" key={`${item.name}-${index}`}>
          <span className="rank-index">{index + 1}</span>
          <span className="rank-name" title={item.name}>{item.name}</span>
          <span className="rank-value">{compactNumber(item.count)}</span>
        </div>
      ))}
    </div>
  );
}

// Signal 渲染首页调查入口。
function Signal({ title, desc, to }: { title: string; desc: string; to: string }) {
  return (
    <Link to={to} className="signal-item">
      <div className="signal-title">{title}</div>
      <div className="signal-desc">{desc}</div>
    </Link>
  );
}

// trendChartOption 构建趋势图配置。
function trendChartOption(trend: TrendPoint[], peakPoint?: TrendPoint): EChartsOption {
  return {
    grid: { left: 42, right: 20, top: 28, bottom: 34 },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: trend.map((item) => item.time),
      axisLabel: { color: '#64748b' },
      axisLine: { lineStyle: { color: '#cbd5e1' } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#64748b' },
      splitLine: { lineStyle: { color: '#e9eef5' } },
    },
    series: [{
      type: 'line',
      smooth: true,
      symbolSize: 7,
      areaStyle: { color: 'rgba(37, 99, 235, 0.14)' },
      lineStyle: { color: '#2563eb', width: 2 },
      itemStyle: { color: '#2563eb' },
      markPoint: peakPoint ? {
        symbolSize: 42,
        itemStyle: { color: '#dc2626' },
        label: { color: '#fff', fontSize: 10 },
        data: [{ name: '峰值', coord: [peakPoint.time, peakPoint.count], value: peakPoint.count }],
      } : undefined,
      data: trend.map((item) => item.count),
    }],
  };
}

// topBarOption 转换 top Bar Option 的数据结构。
function topBarOption(items: TopItem[], color: string): EChartsOption {
  return {
    grid: { left: 112, right: 24, top: 16, bottom: 20 },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'value',
      axisLabel: { color: '#64748b' },
      splitLine: { lineStyle: { color: '#e9eef5' } },
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: items.map((item) => item.name),
      axisLabel: {
        color: '#334155',
        width: 100,
        overflow: 'truncate',
      },
      axisLine: { lineStyle: { color: '#cbd5e1' } },
    },
    series: [{
      type: 'bar',
      barWidth: 12,
      itemStyle: { color, borderRadius: [0, 4, 4, 0] },
      data: items.map((item) => item.count),
    }],
  };
}
