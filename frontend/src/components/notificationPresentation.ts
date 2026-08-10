import type { AppNotification, NotificationDisposition } from '../types/notification';
import { formatLocalDateTime } from '../utils/time';

export function dispositionLabel(value?: NotificationDisposition): string {
  switch (value) {
    case 'confirmed':
      return '已确认';
    case 'false_positive':
      return '误报';
    case 'ignored':
      return '已忽略';
    default:
      return '';
  }
}

export function notificationStatusText(item: AppNotification): string {
  if (item.type === 'enforcement') {
    return item.status === 'open' ? '待处理' : dispositionLabel(item.disposition) || '已处理';
  }
  if (item.status === 'resolved') {
    return item.resolvedAt ? `已恢复：${formatLocalDateTime(item.resolvedAt)}` : '已恢复';
  }
  return '告警中';
}