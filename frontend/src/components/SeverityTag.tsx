import { Tag } from 'antd';

const severityColor: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'blue',
  info: 'default',
};

export default function SeverityTag({ value }: { value?: string }) {
  const severity = value || 'info';
  return <Tag color={severityColor[severity] || 'default'}>{severity}</Tag>;
}
