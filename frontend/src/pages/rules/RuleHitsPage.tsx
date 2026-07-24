import { Card, DatePicker, Empty, Form, Input, Table, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { getRuleHits } from '../../api/stats';
import FilterToolbar from '../../components/FilterToolbar';
import type { RuleHitItem, RuleHitQuery } from '../../types/stats';
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

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">规则命中分析</Typography.Title>
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
            { title: '规则', dataIndex: 'ruleName' },
            { title: '命中次数', dataIndex: 'hitCount', width: 110 },
            { title: '涉及主机', dataIndex: 'activeHosts', width: 110 },
            { title: '涉及用户', dataIndex: 'activeUsers', width: 110 },
            { title: '首次命中', dataIndex: 'firstSeen', width: 190, render: (value) => formatLocalDateTime(value) },
            { title: '最近命中', dataIndex: 'lastSeen', width: 190, render: (value) => formatLocalDateTime(value) },
          ]}
        />
      </Card>
    </>
  );
}
