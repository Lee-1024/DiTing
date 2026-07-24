import { Card, Descriptions, Drawer, Empty, Space, Spin, Table, Tabs, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { getAuditEvent } from '../../api/audit';
import { InvestigationBrief } from '../../components/InsightHeader';
import ProcessChain from '../../components/ProcessChain';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent } from '../../types/audit';
import { formatJSON } from '../../utils/format';
import { eventTypeLabel, ruleFieldLabel, ruleOperatorLabel } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';

interface Props {
  event?: AuditEvent;
  eventId?: string;
  relatedEvents?: AuditEvent[];
  open: boolean;
  onClose: () => void;
}

// EventDetailDrawer 渲染调查式事件详情抽屉。
export default function EventDetailDrawer({ event, eventId, relatedEvents = [], open, onClose }: Props) {
  const [detail, setDetail] = useState<AuditEvent>();
  const [selectedInlineEvent, setSelectedInlineEvent] = useState<AuditEvent>();
  const [loading, setLoading] = useState(false);
  const selectedEventId = eventId || selectedInlineEvent?.eventId || event?.eventId;

  useEffect(() => {
    if (!open || !selectedEventId) {
      setDetail(undefined);
      setSelectedInlineEvent(undefined);
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
          setDetail(selectedInlineEvent || event);
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
  }, [event, open, selectedEventId, selectedInlineEvent]);

  const current = detail || event;

  return (
    <Drawer title={drawerTitle(current)} width={920} open={open} onClose={onClose} className="investigation-drawer">
      {loading && !current && <Spin />}
      {current && (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <InvestigationBrief
            kicker="Event Investigation"
            title={eventTypeLabel(current.eventType)}
            description={current.cmdline || current.filePath || formatAddress(current.dstIp, current.dstPort) || current.processName || '-'}
            metaExtra={<SeverityTag value={current.severity} />}
            metaValue={current.riskScore}
          />
          <Tabs
            className="investigation-tabs"
            items={[
              { key: 'overview', label: '概览', children: <OverviewTab event={current} /> },
              { key: 'process', label: '进程与身份', children: <ProcessTab event={current} /> },
              { key: 'rules', label: '规则命中', children: <RulesTab event={current} /> },
              { key: 'related', label: `关联事件 ${relatedEvents.length > 1 ? relatedEvents.length : ''}`, children: <RelatedTab current={current} relatedEvents={relatedEvents} onSelect={setSelectedInlineEvent} /> },
              { key: 'raw', label: '原始数据', children: <pre className="detail-json">{formatJSON(current.rawEvent)}</pre> },
            ]}
          />
        </Space>
      )}
    </Drawer>
  );
}

// OverviewTab 渲染事件概览。
function OverviewTab({ event }: { event: AuditEvent }) {
  return (
    <Card className="panel-card" size="small">
      <Descriptions column={1} bordered size="small">
        <Descriptions.Item label="事件 ID">{event.eventId}</Descriptions.Item>
        <Descriptions.Item label="时间">{formatLocalDateTime(event.eventTime)}</Descriptions.Item>
        <Descriptions.Item label="主机">{event.nodeName || event.hostName || event.hostId || '-'}</Descriptions.Item>
        <Descriptions.Item label="容器">{event.containerName || event.containerId || '-'}</Descriptions.Item>
        <Descriptions.Item label="镜像">{event.image || '-'}</Descriptions.Item>
        <Descriptions.Item label="文件路径">{event.filePath || '-'}</Descriptions.Item>
        <Descriptions.Item label="文件操作">{event.fileOperation || '-'}</Descriptions.Item>
        <Descriptions.Item label="源地址">{formatAddress(event.srcIp, event.srcPort)}</Descriptions.Item>
        <Descriptions.Item label="目标地址">{formatAddress(event.dstIp, event.dstPort)}</Descriptions.Item>
        <Descriptions.Item label="协议">{event.protocol || '-'}</Descriptions.Item>
        <Descriptions.Item label="域名">{event.domain || '-'}</Descriptions.Item>
      </Descriptions>
    </Card>
  );
}

// ProcessTab 渲染身份和进程上下文。
function ProcessTab({ event }: { event: AuditEvent }) {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card className="panel-card" title="进程链路" size="small">
        <ProcessChain event={event} />
      </Card>
      <Card className="panel-card" size="small">
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="登录用户">{event.loginUsername || event.username}</Descriptions.Item>
          <Descriptions.Item label="执行用户">{event.username}</Descriptions.Item>
          <Descriptions.Item label="AUID / UID / EUID">{[event.auid, event.uid, event.euid].filter((value) => value !== undefined).join(' / ') || '-'}</Descriptions.Item>
          <Descriptions.Item label="GID / EGID">{[event.gid, event.egid].filter((value) => value !== undefined).join(' / ') || '-'}</Descriptions.Item>
          <Descriptions.Item label="进程">{event.processName || '-'}</Descriptions.Item>
          <Descriptions.Item label="二进制">{event.binaryPath || '-'}</Descriptions.Item>
          <Descriptions.Item label="命令">{event.cmdline || '-'}</Descriptions.Item>
          <Descriptions.Item label="工作目录">{event.cwd || '-'}</Descriptions.Item>
          <Descriptions.Item label="父进程">{event.parentProcessName || '-'}</Descriptions.Item>
          <Descriptions.Item label="父进程命令">{event.parentCmdline || '-'}</Descriptions.Item>
          <Descriptions.Item label="Namespace">{event.namespace || '-'}</Descriptions.Item>
          <Descriptions.Item label="Pod">{event.podName || '-'}</Descriptions.Item>
        </Descriptions>
      </Card>
    </Space>
  );
}

// RulesTab 渲染命中规则和条件。
function RulesTab({ event }: { event: AuditEvent }) {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card className="panel-card" title="命中标签与规则" size="small">
        <Space direction="vertical" size={10}>
          <div className="rule-tags">
            {event.tags?.length ? event.tags.map((tag) => <Tag key={tag}>{tag}</Tag>) : <Typography.Text type="secondary">暂无标签</Typography.Text>}
          </div>
          <div className="rule-tags">
            {event.ruleNames?.length ? event.ruleNames.map((name) => <Tag color="orange" key={name}>{name}</Tag>) : <Typography.Text type="secondary">暂无命中规则</Typography.Text>}
          </div>
        </Space>
      </Card>
      {event.ruleMatches?.length ? (
        <Table
          rowKey={(record, index) => `${record.ruleId}-${record.field}-${index}`}
          size="small"
          pagination={false}
          dataSource={event.ruleMatches}
          columns={[
            { title: '规则', dataIndex: 'ruleName', width: 180, render: (value) => value || '-' },
            { title: '字段', dataIndex: 'field', width: 120, render: (value) => ruleFieldLabel(value) },
            { title: '条件', dataIndex: 'operator', width: 110, render: (value) => ruleOperatorLabel(value) },
            { title: '期望值', dataIndex: 'value', width: 200 },
            { title: '实际值', dataIndex: 'actual' },
          ]}
        />
      ) : null}
    </Space>
  );
}

// RelatedTab 渲染同次操作事件。
function RelatedTab({ current, relatedEvents, onSelect }: { current: AuditEvent; relatedEvents: AuditEvent[]; onSelect: (event: AuditEvent) => void }) {
  if (relatedEvents.length <= 1) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无关联事件" />;
  }
  return (
    <Table
      rowKey="eventId"
      size="small"
      pagination={false}
      dataSource={relatedEvents}
      className="clickable-table"
      rowClassName={(record) => record.eventId === current.eventId ? 'ant-table-row-selected' : ''}
      onRow={(record) => ({ onClick: () => onSelect(record), title: '点击切换到该事件详情' })}
      columns={[
        { title: '时间', dataIndex: 'eventTime', width: 170, render: (value) => formatLocalDateTime(value) },
        { title: '事件', dataIndex: 'eventType', width: 120, render: (value) => eventTypeLabel(value) },
        { title: '文件路径', dataIndex: 'filePath', ellipsis: true, render: (value) => value || '-' },
        { title: '操作', dataIndex: 'fileOperation', width: 110, render: (value) => value || '-' },
      ]}
    />
  );
}

// drawerTitle 生成抽屉标题。
function drawerTitle(event?: AuditEvent) {
  if (!event) {
    return '事件详情';
  }
  return `${eventTypeLabel(event.eventType)} / ${event.processName || event.fileOperation || event.protocol || '-'}`;
}

// formatAddress 格式化 format Address 以便界面展示。
function formatAddress(ip?: string, port?: number) {
  if (!ip) {
    return '-';
  }
  const formattedIP = ip.includes(':') ? `[${ip}]` : ip;
  return port ? `${formattedIP}:${port}` : formattedIP;
}
