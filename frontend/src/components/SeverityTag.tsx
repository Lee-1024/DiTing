import { Tag } from 'antd';
import { severityLabel } from '../utils/labels';

const severityColor: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'blue',
  info: 'default',
};

// SeverityTag 渲染 Severity Tag 组件。
export default function SeverityTag({ value }: { value?: string }) {
  const severity = value || 'info';
  return <Tag color={severityColor[severity] || 'default'}>{severityLabel(severity)}</Tag>;
}
