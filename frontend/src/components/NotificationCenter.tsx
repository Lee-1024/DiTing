import { BellOutlined } from '@ant-design/icons';
import { Badge, Button, Dropdown, List, Space, Tabs, Tag, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { handleNotification, listNotifications, markAllNotificationsRead, markNotificationRead } from '../api/notifications';
import type { AppNotification, NotificationDisposition, NotificationListResult, NotificationView } from '../types/notification';
import { formatLocalDateTime } from '../utils/time';
import { dispositionLabel, notificationStatusText } from './notificationPresentation';

interface Props {
  onNavigate: (target: string) => void;
}

const emptyResult: NotificationListResult = {
  items: [],
  counts: { unread: 0, pending: 0, all: 0 },
};

export default function NotificationCenter({ onNavigate }: Props) {
  const [view, setView] = useState<NotificationView>('unread');
  const [result, setResult] = useState<NotificationListResult>(emptyResult);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (nextView: NotificationView = view) => {
    setLoading(true);
    try {
      setResult(await listNotifications(nextView));
    } catch {
      message.error('加载通知失败');
    } finally {
      setLoading(false);
    }
  }, [view]);

  useEffect(() => {
    void load(view);
    const timer = window.setInterval(() => void load(view), 10000);
    return () => window.clearInterval(timer);
  }, [load, view]);

  async function openNotification(item: AppNotification) {
    try {
      if (!item.read) {
        await markNotificationRead(item.id);
      }
      await load(view);
      if (item.target) {
        onNavigate(item.target);
      }
    } catch {
      message.error('打开通知失败');
    }
  }

  async function markAllRead() {
    try {
      await markAllNotificationsRead();
      await load(view);
      message.success('已全部标记为已读');
    } catch {
      message.error('全部已读操作失败');
    }
  }

  async function dispose(item: AppNotification, disposition: NotificationDisposition) {
    try {
      await handleNotification(item.id, disposition);
      await load(view);
      message.success(`通知已标记为${dispositionLabel(disposition)}`);
    } catch {
      message.error('处置通知失败');
    }
  }

  const panel = (
    <div className="notification-center-panel">
      <div className="notification-center-header">
        <div>
          <Typography.Text strong>通知中心</Typography.Text>
          <Typography.Text type="secondary" className="notification-center-summary">
            未读 {result.counts.unread} · 待处理 {result.counts.pending}
          </Typography.Text>
        </div>
        <Button type="link" size="small" disabled={result.counts.unread === 0} onClick={() => void markAllRead()}>
          全部已读
        </Button>
      </div>
      <Tabs
        size="small"
        activeKey={view}
        onChange={(key) => setView(key as NotificationView)}
        items={[
          { key: 'unread', label: `未读 ${result.counts.unread}` },
          { key: 'pending', label: `待处理 ${result.counts.pending}` },
          { key: 'all', label: `全部 ${result.counts.all}` },
        ]}
      />
      <List
        className="notification-center-list"
        loading={loading}
        dataSource={result.items}
        locale={{ emptyText: '暂无通知' }}
        renderItem={(item) => (
          <List.Item
            className={item.read ? 'notification-item' : 'notification-item notification-item-unread'}
            onClick={() => void openNotification(item)}
            actions={item.type === 'enforcement' && item.status === 'open' ? [
              <Button key="confirmed" size="small" type="link" onClick={(event) => { event.stopPropagation(); void dispose(item, 'confirmed'); }}>已确认</Button>,
              <Button key="false-positive" size="small" type="link" onClick={(event) => { event.stopPropagation(); void dispose(item, 'false_positive'); }}>误报</Button>,
              <Button key="ignored" size="small" type="link" danger onClick={(event) => { event.stopPropagation(); void dispose(item, 'ignored'); }}>忽略</Button>,
            ] : undefined}
          >
            <List.Item.Meta
              title={(
                <Space size={6}>
                  {!item.read && <span className="notification-unread-dot" />}
                  <span>{item.title}</span>
                  <Tag color={item.status === 'open' ? (item.type === 'enforcement' ? 'red' : 'orange') : 'default'}>
                    {notificationStatusText(item)}
                  </Tag>
                </Space>
              )}
              description={(
                <Space direction="vertical" size={2} className="notification-item-description">
                  <Typography.Text type="secondary">{formatLocalDateTime(item.createdAt)}</Typography.Text>
                  <Typography.Text ellipsis={{ tooltip: item.description }}>{item.description}</Typography.Text>
                  {item.handledBy && <Typography.Text type="secondary">处置人：{item.handledBy}</Typography.Text>}
                </Space>
              )}
            />
          </List.Item>
        )}
      />
    </div>
  );

  return (
    <Dropdown dropdownRender={() => panel} trigger={['click']} placement="bottomRight" onOpenChange={(open) => { if (open) void load(view); }}>
      <Badge count={result.counts.unread} size="small" overflowCount={99}>
        <Button icon={<BellOutlined />} onClick={(event) => event.preventDefault()}>通知</Button>
      </Badge>
    </Dropdown>
  );
}