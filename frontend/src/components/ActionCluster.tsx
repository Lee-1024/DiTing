import { MoreOutlined } from '@ant-design/icons';
import { Button, Dropdown, Space } from 'antd';
import type { ButtonProps, MenuProps } from 'antd';
import type { ReactNode } from 'react';

export interface ActionItem {
  key: string;
  label: ReactNode;
  icon?: ReactNode;
  onClick?: () => void;
  type?: ButtonProps['type'];
  danger?: boolean;
  htmlType?: ButtonProps['htmlType'];
  loading?: boolean;
  disabled?: boolean;
}

interface ActionClusterProps {
  actions: ActionItem[];
  maxVisible?: number;
  className?: string;
}

// ActionCluster 统一页面和筛选区操作的主次层级。
export default function ActionCluster({ actions, maxVisible = 3, className }: ActionClusterProps) {
  const visibleActions = actions.slice(0, maxVisible);
  const overflowActions = actions.slice(maxVisible);
  const menuItems: MenuProps['items'] = overflowActions.map((action) => ({
    key: action.key,
    icon: action.icon,
    label: action.label,
    danger: action.danger,
    disabled: action.disabled || action.loading,
    onClick: action.onClick,
  }));

  return (
    <Space className={className ? `action-cluster ${className}` : 'action-cluster'} size={8} wrap>
      {visibleActions.map((action) => (
        <Button
          key={action.key}
          type={action.type}
          danger={action.danger}
          icon={action.icon}
          htmlType={action.htmlType}
          loading={action.loading}
          disabled={action.disabled}
          onClick={action.onClick}
        >
          {action.label}
        </Button>
      ))}
      {overflowActions.length > 0 && (
        <Dropdown menu={{ items: menuItems }} trigger={['click']}>
          <Button icon={<MoreOutlined />}>更多</Button>
        </Dropdown>
      )}
    </Space>
  );
}
