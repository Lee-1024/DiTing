import { DownloadOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { Card, Form, Typography } from 'antd';
import type { FormInstance } from 'antd';
import type { ReactNode } from 'react';
import ActionCluster from './ActionCluster';

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
      <Form form={form} className="filter-form" layout="vertical" initialValues={initialValues} onFinish={onSearch}>
        <div className="filter-fields">
          {children}
          <Form.Item className="filter-actions">
            <Typography.Text className="filter-actions-label">操作</Typography.Text>
            <ActionCluster
              actions={[
                { key: 'search', label: '查询', icon: <SearchOutlined />, type: 'primary', htmlType: 'submit' },
                { key: 'reset', label: '重置', icon: <ReloadOutlined />, onClick: onReset },
                ...(onExport ? [{ key: 'export', label: exportText, icon: <DownloadOutlined />, onClick: onExport }] : []),
              ]}
            />
          </Form.Item>
        </div>
      </Form>
    </Card>
  );
}
