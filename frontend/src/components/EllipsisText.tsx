import { Tooltip, Typography } from 'antd';

// EllipsisText 渲染 Ellipsis Text 组件。
export default function EllipsisText({ value, width = 360 }: { value?: string; width?: number }) {
  const text = value || '';
  return (
    <Tooltip title={text}>
      <Typography.Text className="ellipsis-text" style={{ maxWidth: width }}>
        {text}
      </Typography.Text>
    </Tooltip>
  );
}
