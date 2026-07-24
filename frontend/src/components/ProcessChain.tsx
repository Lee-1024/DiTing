import { Space, Tag, Typography } from 'antd';
import type { AuditEvent } from '../types/audit';

interface Props {
  event: AuditEvent;
  compact?: boolean;
}

// ProcessChain 处理 Process Chain 相关逻辑。
export default function ProcessChain({ event, compact = false }: Props) {
  const nodes = chainNodes(event);
  if (nodes.length === 0) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }
  if (compact) {
    return (
      <Typography.Text ellipsis title={nodes.map((node) => node.name).join(' -> ')}>
        {nodes.map((node) => node.name).join(' -> ')}
      </Typography.Text>
    );
  }
  return (
    <Space direction="vertical" size={8} style={{ width: '100%' }}>
      {nodes.map((node, index) => (
        <Space key={`${node.role}-${node.name}-${index}`} align="start" size={8}>
          <Tag color={node.role === '当前进程' ? 'blue' : 'default'}>{node.role}</Tag>
          <Space direction="vertical" size={0}>
            <Typography.Text strong={node.role === '当前进程'}>{node.name}</Typography.Text>
            {node.command && <Typography.Text type="secondary">{node.command}</Typography.Text>}
          </Space>
        </Space>
      ))}
    </Space>
  );
}

// chainNodes 处理 chain Nodes 相关逻辑。
function chainNodes(event: AuditEvent) {
  const nodes: Array<{ role: string; name: string; command?: string }> = [];
  if (event.parentProcessName || event.parentCmdline) {
    nodes.push({
      role: '父进程',
      name: event.parentProcessName || commandName(event.parentCmdline) || '-',
      command: event.parentCmdline,
    });
  }
  if (event.processName || event.cmdline) {
    nodes.push({
      role: '当前进程',
      name: event.processName || commandName(event.cmdline) || '-',
      command: event.cmdline,
    });
  }
  return nodes;
}

// commandName 生成 command Name 的展示内容。
function commandName(command?: string) {
  return command?.trim().split(/\s+/)[0];
}
