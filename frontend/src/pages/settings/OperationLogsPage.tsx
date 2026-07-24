import { Card, DatePicker, Empty, Form, Input, InputNumber, Select, Table, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { queryOperationLogs } from '../../api/operationLogs';
import FilterToolbar from '../../components/FilterToolbar';
import type { OperationLog, OperationLogQuery } from '../../types/operationLog';
import { formatLocalDateTime } from '../../utils/time';

const defaultRange = [dayjs().subtract(7, 'day'), dayjs()] as const;

// OperationLogsPage 渲染 Operation Logs Page 组件。
export default function OperationLogsPage() {
  const [items, setItems] = useState<OperationLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [form] = Form.useForm();
  const requestSeq = useRef(0);

  // buildQuery 构建 build Query 所需的数据结构。
  function buildQuery(nextPage = page, nextPageSize = pageSize, formValues = form.getFieldsValue()): OperationLogQuery {
    const range = formValues.timeRange ?? defaultRange;
    return {
      start_time: range?.[0]?.startOf('day').toISOString(),
      end_time: range?.[1]?.endOf('day').toISOString(),
      username: formValues.username,
      method: formValues.method,
      keyword: formValues.keyword,
      status: formValues.status,
      page: nextPage,
      page_size: nextPageSize,
    };
  }

  // load 加载页面所需数据。
  async function load(nextPage = page, nextPageSize = pageSize, formValues = form.getFieldsValue()) {
    const seq = requestSeq.current + 1;
    requestSeq.current = seq;
    setLoading(true);
    try {
      const data = await queryOperationLogs(buildQuery(nextPage, nextPageSize, formValues));
      if (seq !== requestSeq.current) {
        return;
      }
      setItems(data.items ?? []);
      setTotal(data.total);
      setPage(data.page);
      setPageSize(nextPageSize);
    } finally {
      if (seq === requestSeq.current) {
        setLoading(false);
      }
    }
  }

  // submit 提交当前表单或操作。
  function submit() {
    void load(1, pageSize, form.getFieldsValue());
  }

  // resetAndLoad 重置 reset And Load 状态。
  async function resetAndLoad() {
    form.resetFields();
    await Promise.resolve();
    await load(1, 10, form.getFieldsValue());
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <>
      <div className="page-heading">
        <Typography.Title level={3} className="page-title">操作审计</Typography.Title>
      </div>
      <FilterToolbar form={form} initialValues={{ timeRange: defaultRange }} onSearch={submit} onReset={() => void resetAndLoad()}>
        <Form.Item name="timeRange" label="时间" className="filter-field-time">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item name="username" label="用户">
          <Input className="filter-control-compact" placeholder="admin" allowClear />
        </Form.Item>
        <Form.Item name="method" label="方法">
          <Select className="filter-control-compact" allowClear options={['GET', 'POST', 'PUT', 'DELETE'].map((value) => ({ value }))} />
        </Form.Item>
        <Form.Item name="keyword" label="路径">
          <Input className="filter-control-compact" placeholder="/api/v1/rules" allowClear />
        </Form.Item>
        <Form.Item name="status" label="状态">
          <InputNumber className="filter-control-compact" min={100} max={599} controls={false} />
        </Form.Item>
      </FilterToolbar>
      <Card className="data-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={items}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无操作记录" /> }}
          scroll={{ x: 1280 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showTotal: (value) => `共 ${value} 条`,
            onChange: (nextPage, nextPageSize) => {
              const sizeChanged = nextPageSize !== pageSize;
              void load(sizeChanged ? 1 : nextPage, nextPageSize, form.getFieldsValue());
            },
          }}
          columns={[
            { title: '时间', dataIndex: 'createdAt', width: 190, fixed: 'left', render: (value) => formatLocalDateTime(value) },
            { title: '用户', dataIndex: 'username', width: 120 },
            { title: '方法', dataIndex: 'method', width: 90, render: (value) => <Tag>{value}</Tag> },
            { title: '路径', dataIndex: 'path', width: 360 },
            { title: '状态码', dataIndex: 'status', width: 100, render: (value) => <Tag color={value >= 400 ? 'red' : 'green'}>{value}</Tag> },
            { title: 'IP', dataIndex: 'ip', width: 180 },
            { title: 'User-Agent', dataIndex: 'userAgent', ellipsis: true },
          ]}
        />
      </Card>
    </>
  );
}
