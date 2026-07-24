import { DownloadOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { Button, Card, Form, Space } from 'antd';
import type { FormInstance } from 'antd';
import type { ReactNode } from 'react';

interface FilterToolbarProps {
  form: FormInstance;
  initialValues?: Record<string, unknown>;
  children: ReactNode;
  onSearch: () => void;
  onReset: () => void;
  onExport?: () => void;
  exportText?: string;
}

// FilterToolbar 按条件过滤 Filter Toolbar。
export default function FilterToolbar({
  form,
  initialValues,
  children,
  onSearch,
  onReset,
  onExport,
  exportText = '导出 CSV',
}: FilterToolbarProps) {
  return (
    <Card className="filter-card">
      <Form
        form={form}
        className="filter-form"
        layout="horizontal"
        labelCol={{ flex: '78px' }}
        wrapperCol={{ flex: '1 1 0' }}
        colon
        initialValues={initialValues}
        onFinish={() => {
          onSearch();
        }}
      >
        <div className="filter-fields">
          {children}
        </div>
        <div className="filter-footer">
          <Space className="filter-actions" size={10} wrap={false}>
            <Button type="primary" icon={<SearchOutlined />} htmlType="submit">查询</Button>
            <Button icon={<ReloadOutlined />} onClick={onReset}>重置</Button>
            {onExport && <Button icon={<DownloadOutlined />} onClick={onExport}>{exportText}</Button>}
          </Space>
        </div>
      </Form>
    </Card>
  );
}
