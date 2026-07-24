import { DownloadOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { QueryFilter } from '@ant-design/pro-components';
import { Button, Card, Space } from 'antd';
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
      <QueryFilter
        form={form}
        className="filter-form"
        layout="vertical"
        initialValues={initialValues}
        onFinish={() => {
          onSearch();
        }}
        onReset={() => {
          onReset();
        }}
        defaultCollapsed={false}
        span={{ xs: 24, sm: 12, md: 8, lg: 6, xl: 6, xxl: 6 }}
        searchGutter={[16, 14]}
        showHiddenNum
        optionRender={(_, __, dom) => (
          [
            <Space key="filter-actions" className="filter-actions" size={8} wrap>
              <Button type="primary" icon={<SearchOutlined />} htmlType="submit">查询</Button>
              <Button icon={<ReloadOutlined />} onClick={onReset}>重置</Button>
              {onExport && <Button icon={<DownloadOutlined />} onClick={onExport}>{exportText}</Button>}
              {dom.slice(2)}
            </Space>,
          ]
        )}
      >
        {children}
      </QueryFilter>
    </Card>
  );
}
