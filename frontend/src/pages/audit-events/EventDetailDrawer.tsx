import { Descriptions, Drawer, Space, Spin, Table, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { getAuditEvent } from '../../api/audit';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import { formatJSON } from '../../utils/format';
import { eventTypeLabel, ruleFieldLabel, ruleOperatorLabel } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';

interface Props {
  event?: AuditEvent;
  eventId?: string;
  open: boolean;
  onClose: () => void;
}

export default function EventDetailDrawer({ event, eventId, open, onClose }: Props) {
  const [detail, setDetail] = useState<AuditEvent>();
  const [loading, setLoading] = useState(false);
  const selectedEventId = eventId || event?.eventId;

  useEffect(() => {
    if (!open || !selectedEventId) {
      setDetail(undefined);
      return;
    }
    let ignore = false;
    setLoading(true);
    getAuditEvent(selectedEventId)
      .then((data) => {
        if (!ignore) {
          setDetail(data);
        }
      })
      .catch(() => {
        if (!ignore) {
          message.error('事件详情加载失败');
          setDetail(event);
        }
      })
      .finally(() => {
        if (!ignore) {
          setLoading(false);
        }
      });
    return () => {
      ignore = true;
    };
  }, [event, open, selectedEventId]);

  const current = detail || event;

  return (
    <Drawer title="事件详情" width={760} open={open} onClose={onClose}>
      {loading && !current && <Spin />}
      {current && (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Descriptions title="基础信息" column={1} bordered size="small">
            <Descriptions.Item label="事件 ID">{current.eventId}</Descriptions.Item>
            <Descriptions.Item label="事件类型">{eventTypeLabel(current.eventType)}</Descriptions.Item>
            <Descriptions.Item label="时间">{formatLocalDateTime(current.eventTime)}</Descriptions.Item>
            <Descriptions.Item label="等级"><SeverityTag value={current.severity} /></Descriptions.Item>
            <Descriptions.Item label="风险分数">{current.riskScore}</Descriptions.Item>
            <Descriptions.Item label="节点">{current.nodeName || current.hostName}</Descriptions.Item>
            <Descriptions.Item label="容器">{current.containerName || current.containerId}</Descriptions.Item>
            <Descriptions.Item label="镜像">{current.image}</Descriptions.Item>
          </Descriptions>
          <Descriptions title="用户身份" column={1} bordered size="small">
            <Descriptions.Item label="登录用户">{current.loginUsername || current.username}</Descriptions.Item>
            <Descriptions.Item label="执行用户">{current.username}</Descriptions.Item>
            <Descriptions.Item label="AUID / UID / EUID">{[current.auid, current.uid, current.euid].filter((value) => value !== undefined).join(' / ')}</Descriptions.Item>
            <Descriptions.Item label="GID / EGID">{[current.gid, current.egid].filter((value) => value !== undefined).join(' / ')}</Descriptions.Item>
          </Descriptions>
          <Descriptions title="进程信息" column={1} bordered size="small">
            <Descriptions.Item label="进程">{current.processName}</Descriptions.Item>
            <Descriptions.Item label="二进制">{current.binaryPath}</Descriptions.Item>
            <Descriptions.Item label="命令">{current.cmdline}</Descriptions.Item>
            <Descriptions.Item label="工作目录">{current.cwd}</Descriptions.Item>
            <Descriptions.Item label="父进程">{current.parentProcessName}</Descriptions.Item>
            <Descriptions.Item label="父进程命令">{current.parentCmdline}</Descriptions.Item>
            <Descriptions.Item label="Namespace">{current.namespace}</Descriptions.Item>
            <Descriptions.Item label="Pod">{current.podName}</Descriptions.Item>
          </Descriptions>
          <Descriptions title="文件与网络" column={1} bordered size="small">
            <Descriptions.Item label="文件路径">{current.filePath || '-'}</Descriptions.Item>
            <Descriptions.Item label="文件操作">{current.fileOperation || '-'}</Descriptions.Item>
            <Descriptions.Item label="源地址">{current.srcIp ? `${current.srcIp}:${current.srcPort || ''}` : '-'}</Descriptions.Item>
            <Descriptions.Item label="目标地址">{current.dstIp ? `${current.dstIp}:${current.dstPort || ''}` : '-'}</Descriptions.Item>
            <Descriptions.Item label="协议">{current.protocol || '-'}</Descriptions.Item>
            <Descriptions.Item label="域名">{current.domain || '-'}</Descriptions.Item>
          </Descriptions>
          <Descriptions title="规则命中" column={1} bordered size="small">
            <Descriptions.Item label="标签">
              {current.tags?.map((tag) => <Tag key={tag}>{tag}</Tag>)}
            </Descriptions.Item>
            <Descriptions.Item label="命中规则">
              {current.ruleNames?.map((name) => <Tag key={name}>{name}</Tag>)}
            </Descriptions.Item>
          </Descriptions>
          {current.ruleMatches?.length ? (
            <Table
              rowKey={(record, index) => `${record.ruleId}-${record.field}-${index}`}
              size="small"
              pagination={false}
              dataSource={current.ruleMatches}
              columns={[
                { title: '规则', dataIndex: 'ruleName', width: 180, render: (value) => value || '-' },
                { title: '字段', dataIndex: 'field', width: 120, render: (value) => ruleFieldLabel(value) },
                { title: '条件', dataIndex: 'operator', width: 100, render: (value) => ruleOperatorLabel(value) },
                { title: '期望值', dataIndex: 'value', width: 180 },
                { title: '实际值', dataIndex: 'actual' },
              ]}
            />
          ) : null}
          <Typography.Title level={5}>原始事件</Typography.Title>
          <pre className="detail-json">{formatJSON(current.rawEvent)}</pre>
        </Space>
      )}
    </Drawer>
  );
}
