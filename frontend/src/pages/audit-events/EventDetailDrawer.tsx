import { Card, Descriptions, Drawer, Empty, Space, Spin, Table, Tabs, Tag, Typography, message } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { getAuditEvent, queryAuditEvents } from '../../api/audit';
import { InvestigationBrief } from '../../components/InsightHeader';
import ProcessChain from '../../components/ProcessChain';
import SeverityTag from '../../components/SeverityTag';
import type { AuditEvent, AuditEventQuery } from '../../types/audit';
import type { AIRiskAnalysis } from '../../types/riskAnalysis';
import { formatJSON } from '../../utils/format';
import { displayHostIdentity } from '../../utils/hostDisplay';
import { eventTypeLabel, ruleFieldLabel, ruleOperatorLabel } from '../../utils/labels';
import { formatLocalDateTime } from '../../utils/time';

interface Props {
  event?: AuditEvent;
  eventId?: string;
  relatedEvents?: AuditEvent[];
  aiAnalysis?: AIRiskAnalysis;
  open: boolean;
  onClose: () => void;
}

// EventDetailDrawer 渲染调查式事件详情抽屉。
export default function EventDetailDrawer({ event, eventId, relatedEvents = [], aiAnalysis, open, onClose }: Props) {
  const [detail, setDetail] = useState<AuditEvent>();
  const [selectedInlineEvent, setSelectedInlineEvent] = useState<AuditEvent>();
  const [autoRelatedEvents, setAutoRelatedEvents] = useState<AuditEvent[]>([]);
  const [relatedLoading, setRelatedLoading] = useState(false);
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
  const mergedRelatedEvents = current ? mergeRelatedEvents(current, relatedEvents, autoRelatedEvents) : relatedEvents;

  useEffect(() => {
    if (!open || !current?.eventId) {
      setAutoRelatedEvents([]);
      return;
    }
    const query = buildRelatedQuery(current);
    if (!query) {
      setAutoRelatedEvents([current]);
      return;
    }
    let ignore = false;
    setRelatedLoading(true);
    queryAuditEvents(query)
      .then((data) => {
        if (!ignore) {
          setAutoRelatedEvents(data.items || []);
        }
      })
      .catch(() => {
        if (!ignore) {
          setAutoRelatedEvents([current]);
        }
      })
      .finally(() => {
        if (!ignore) {
          setRelatedLoading(false);
        }
      });
    return () => {
      ignore = true;
    };
  }, [current?.eventId, open]);

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
              ...(aiAnalysis ? [{ key: 'ai', label: 'AI 分析', children: <AIAnalysisTab analysis={aiAnalysis} /> }] : []),
              { key: 'process', label: '进程与身份', children: <ProcessTab event={current} /> },
              { key: 'rules', label: '规则命中', children: <RulesTab event={current} /> },
              { key: 'related', label: `关联事件 ${mergedRelatedEvents.length > 1 ? mergedRelatedEvents.length : ''}`, children: <RelatedTab current={current} relatedEvents={mergedRelatedEvents} loading={relatedLoading} onSelect={setSelectedInlineEvent} /> },
              { key: 'raw', label: '原始数据', children: <pre className="detail-json">{formatJSON(current.rawEvent)}</pre> },
            ]}
          />
        </Space>
      )}
    </Drawer>
  );
}

function AIAnalysisTab({ analysis }: { analysis: AIRiskAnalysis }) {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card className="panel-card" size="small">
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="AI 判断"><Tag color={aiVerdictColor(analysis.verdict)}>{aiVerdictText(analysis.verdict)}</Tag></Descriptions.Item>
          <Descriptions.Item label="AI 等级"><SeverityTag value={analysis.aiSeverity} /></Descriptions.Item>
          <Descriptions.Item label="置信度">{analysis.confidence}%</Descriptions.Item>
          <Descriptions.Item label="模型">{analysis.model || '-'}</Descriptions.Item>
          <Descriptions.Item label="分析时间">{formatLocalDateTime(analysis.analyzedAt)}</Descriptions.Item>
          <Descriptions.Item label="判断理由">
            <Typography.Paragraph style={{ marginBottom: 0 }} copyable>
              {analysis.reason || '-'}
            </Typography.Paragraph>
          </Descriptions.Item>
          <Descriptions.Item label="处置建议">
            <Typography.Paragraph style={{ marginBottom: 0 }} copyable>
              {analysis.suggestion || '-'}
            </Typography.Paragraph>
          </Descriptions.Item>
        </Descriptions>
      </Card>
      <Card className="panel-card" title="证据" size="small">
        {analysis.evidence?.length ? (
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            {analysis.evidence.map((item, index) => (
              <Typography.Paragraph key={`${index}-${item}`} style={{ marginBottom: 0 }} copyable>
                {item}
              </Typography.Paragraph>
            ))}
          </Space>
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无证据" />
        )}
      </Card>
      {analysis.rawResponse ? (
        <Card className="panel-card" title="模型原始输出" size="small">
          <pre className="detail-json">{analysis.rawResponse}</pre>
        </Card>
      ) : null}
    </Space>
  );
}

function aiVerdictText(verdict: string) {
  const config: Record<string, string> = {
    true_positive: '真实风险',
    suspicious: '可疑',
    false_positive: '可能误报',
    needs_review: '需复核',
  };
  return config[verdict] ?? verdict;
}

function aiVerdictColor(verdict: string) {
  const config: Record<string, string> = {
    true_positive: 'red',
    suspicious: 'orange',
    false_positive: 'blue',
    needs_review: 'purple',
  };
  return config[verdict] ?? 'default';
}

// OverviewTab 渲染事件概览。
function OverviewTab({ event }: { event: AuditEvent }) {
  return (
    <Card className="panel-card" size="small">
      <Descriptions column={1} bordered size="small">
        <Descriptions.Item label="事件 ID">{event.eventId}</Descriptions.Item>
        <Descriptions.Item label="时间">{formatLocalDateTime(event.eventTime)}</Descriptions.Item>
        <Descriptions.Item label="主机">{displayHostIdentity(event)}</Descriptions.Item>
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
function RelatedTab({ current, relatedEvents, loading, onSelect }: { current: AuditEvent; relatedEvents: AuditEvent[]; loading: boolean; onSelect: (event: AuditEvent) => void }) {
  if (relatedEvents.length <= 1) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={loading ? '正在查找关联事件' : '同主机、相邻时间内暂无关联事件'} />;
  }
  return (
    <Table
      rowKey="eventId"
      size="small"
      pagination={false}
      loading={loading}
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

function buildRelatedQuery(event: AuditEvent): AuditEventQuery | undefined {
  const eventAt = parseEventTime(event.eventTime);
  const hostName = event.hostId || event.nodeName || event.hostName;
  if (!eventAt.isValid() || !hostName) {
    return undefined;
  }
  const query: AuditEventQuery = {
    start_time: eventAt.subtract(3, 'second').toISOString(),
    end_time: eventAt.add(3, 'second').toISOString(),
    host_name: hostName,
    page: 1,
    page_size: 20,
    include_total: false,
  };
  return query;
}

function parseEventTime(value?: string) {
  if (!value) {
    return dayjs('');
  }
  const normalized = /^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?$/.test(value) ? `${value.replace(' ', 'T')}Z` : value;
  return dayjs(normalized);
}

function mergeRelatedEvents(current: AuditEvent, propRelated: AuditEvent[], autoRelated: AuditEvent[]) {
  const merged = new Map<string, AuditEvent>();
  [current, ...propRelated, ...autoRelated].forEach((item) => {
    if (item?.eventId) {
      merged.set(item.eventId, item);
    }
  });
  return Array.from(merged.values()).sort((left, right) => parseEventTime(left.eventTime).valueOf() - parseEventTime(right.eventTime).valueOf());
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
