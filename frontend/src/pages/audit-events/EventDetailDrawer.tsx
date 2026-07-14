import { Descriptions, Drawer, Space, Tag, Typography } from 'antd';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import { formatJSON } from '../../utils/format';
import { formatLocalDateTime } from '../../utils/time';

interface Props {
  event?: AuditEvent;
  open: boolean;
  onClose: () => void;
}

export default function EventDetailDrawer({ event, open, onClose }: Props) {
  return (
    <Drawer title="事件详情" width={760} open={open} onClose={onClose}>
      {event && (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Descriptions title="基础信息" column={1} bordered size="small">
            <Descriptions.Item label="事件 ID">{event.eventId}</Descriptions.Item>
            <Descriptions.Item label="时间">{formatLocalDateTime(event.eventTime)}</Descriptions.Item>
            <Descriptions.Item label="等级"><SeverityTag value={event.severity} /></Descriptions.Item>
            <Descriptions.Item label="风险分数">{event.riskScore}</Descriptions.Item>
            <Descriptions.Item label="节点">{event.nodeName || event.hostName}</Descriptions.Item>
            <Descriptions.Item label="容器">{event.containerName || event.containerId}</Descriptions.Item>
            <Descriptions.Item label="镜像">{event.image}</Descriptions.Item>
          </Descriptions>
          <Descriptions title="用户身份" column={1} bordered size="small">
            <Descriptions.Item label="登录用户">{event.loginUsername || event.username}</Descriptions.Item>
            <Descriptions.Item label="执行用户">{event.username}</Descriptions.Item>
            <Descriptions.Item label="AUID / UID / EUID">{[event.auid, event.uid, event.euid].filter((value) => value !== undefined).join(' / ')}</Descriptions.Item>
            <Descriptions.Item label="GID / EGID">{[event.gid, event.egid].filter((value) => value !== undefined).join(' / ')}</Descriptions.Item>
          </Descriptions>
          <Descriptions title="进程信息" column={1} bordered size="small">
            <Descriptions.Item label="进程">{event.processName}</Descriptions.Item>
            <Descriptions.Item label="二进制">{event.binaryPath}</Descriptions.Item>
            <Descriptions.Item label="命令">{event.cmdline}</Descriptions.Item>
            <Descriptions.Item label="工作目录">{event.cwd}</Descriptions.Item>
            <Descriptions.Item label="父进程">{event.parentProcessName}</Descriptions.Item>
            <Descriptions.Item label="父进程命令">{event.parentCmdline}</Descriptions.Item>
            <Descriptions.Item label="Namespace">{event.namespace}</Descriptions.Item>
            <Descriptions.Item label="Pod">{event.podName}</Descriptions.Item>
          </Descriptions>
          <Descriptions title="规则命中" column={1} bordered size="small">
            <Descriptions.Item label="标签">
              {event.tags?.map((tag) => <Tag key={tag}>{tag}</Tag>)}
            </Descriptions.Item>
            <Descriptions.Item label="命中规则">
              {event.ruleNames?.map((name) => <Tag key={name}>{name}</Tag>)}
            </Descriptions.Item>
          </Descriptions>
          <Typography.Title level={5}>原始事件</Typography.Title>
          <pre className="detail-json">{formatJSON(event.rawEvent)}</pre>
        </Space>
      )}
    </Drawer>
  );
}
