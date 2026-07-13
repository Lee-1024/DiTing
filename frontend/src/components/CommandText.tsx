import { CopyOutlined, EyeOutlined } from '@ant-design/icons';
import { Button, Tooltip, Typography, message } from 'antd';

interface CommandTextProps {
  value?: string;
  width?: number;
  onView?: () => void;
}

export default function CommandText({ value, width = 420, onView }: CommandTextProps) {
  const text = value || '';

  async function copy() {
    if (!text) {
      return;
    }
    await navigator.clipboard.writeText(text);
    message.success('已复制命令');
  }

  return (
    <div className="command-cell">
      <Tooltip title={text}>
        <Typography.Text className="ellipsis-text command-text" style={{ maxWidth: width }}>
          {text}
        </Typography.Text>
      </Tooltip>
      <div className="command-actions">
      <Tooltip title="复制命令">
        <Button size="small" type="text" icon={<CopyOutlined />} onClick={(event) => { event.stopPropagation(); void copy(); }} />
      </Tooltip>
      {onView && (
        <Tooltip title="查看详情">
          <Button size="small" type="text" icon={<EyeOutlined />} onClick={(event) => { event.stopPropagation(); onView(); }} />
        </Tooltip>
      )}
      </div>
    </div>
  );
}
