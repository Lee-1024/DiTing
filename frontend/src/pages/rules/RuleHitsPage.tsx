import { Button, Card, DatePicker, Empty, Form, Input, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getRuleHits } from '../../api/stats';
import FilterToolbar from '../../components/FilterToolbar';
import { InsightHero, LatestPanel, MetricCard } from '../../components/InsightHeader';
import type { RuleHitItem, RuleHitQuery } from '../../types/stats';
import { compactNumber } from '../../utils/format';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

// RuleHitsPage 渲染 Rule Hits Page 组件。
export default function RuleHitsPage() {
  const [items, setItems] = useState<RuleHitItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [tablePageSize, setTablePageSize] = useState(10);
  const [form] = Form.useForm();

  // buildQuery 构建 build Query 所需的数据结构。
  function buildQuery(): RuleHitQuery {
    const values = form.getFieldsValue();
    const range = values.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      keyword: values.keyword,
      limit: 100,
    };
  }

  // load 加载页面所需数据。
  async function load() {
    setLoading(true);
    try {
      setItems(await getRuleHits(buildQuery()));
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

  useEffect(() => {
    void load();
  }, []);

  const totalHits = items.reduce((sum, item) => sum + item.hitCount, 0);
  const activeHosts = items.reduce((sum, item) => sum + item.activeHosts, 0);
  const activeUsers = items.reduce((sum, item) => sum + item.activeUsers, 0);
  const topRule = items[0];

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="page-kicker">RULE HIT ANALYTICS</span>
          <Typography.Title level={3} className="page-title">规则命中分析</Typography.Title>
        </div>
      </div>
      <div className="policy-hero">
        <InsightHero
          className="policy-summary"
          kicker="Rule Effectiveness"
          title="观察规则命中活跃度，定位噪声与关键检测点"
          description="命中分析帮助评估策略有效性：高频规则可能是关键威胁，也可能需要调优降噪。"
          actions={(
            <>
            <Link to="/rules"><Button type="primary">管理审计规则</Button></Link>
            <Link to="/audit/risks"><Button ghost>风险事件</Button></Link>
            </>
          )}
        />
        <LatestPanel
          label="最高频规则"
          title={topRule?.ruleName || '-'}
          description={topRule ? `${compactNumber(topRule.hitCount)} 次命中 / ${compactNumber(topRule.activeHosts)} 台主机 / ${compactNumber(topRule.activeUsers)} 个用户` : '暂无规则命中数据'}
        />
      </div>
      <div className="metric-grid risk-metric-grid">
        <MetricCard label="命中规则" value={items.length} hint="当前筛选结果" tone="blue" />
        <MetricCard label="命中次数" value={totalHits} hint="规则触发总量" tone="warning" />
        <MetricCard label="涉及主机" value={activeHosts} hint="规则覆盖主机足迹" tone="success" />
        <MetricCard label="涉及用户" value={activeUsers} hint="规则覆盖用户足迹" tone="cyan" />
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={() => void load()} onReset={() => void resetAndLoad()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="keyword" label="规则">
          <Input className="filter-control-compact" placeholder="规则名称" allowClear />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey="ruleName"
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无规则命中数据" /> }}
          scroll={{ x: 980 }}
          pagination={{
            pageSize: tablePageSize,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onShowSizeChange: (_, size) => setTablePageSize(size),
          }}
          columns={[
            { title: '规则', dataIndex: 'ruleName', ellipsis: true },
            { title: '命中次数', dataIndex: 'hitCount', width: 128, align: 'right', className: 'number-cell' },
            { title: '涉及主机', dataIndex: 'activeHosts', width: 128, align: 'right', className: 'number-cell' },
            { title: '涉及用户', dataIndex: 'activeUsers', width: 128, align: 'right', className: 'number-cell' },
            { title: '首次命中', dataIndex: 'firstSeen', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近命中', dataIndex: 'lastSeen', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
    </>
  );
}
