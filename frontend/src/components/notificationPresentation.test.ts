import { describe, expect, it } from 'vitest';
import { dispositionLabel, notificationStatusText } from './notificationPresentation';
import type { AppNotification } from '../types/notification';

function notification(overrides: Partial<AppNotification> = {}): AppNotification {
  return {
    id: 'notification-1',
    type: 'enforcement',
    sourceId: 'event-1',
    title: '拦截策略触发',
    description: 'blocked',
    severity: 'critical',
    target: '/audit/events?eventId=event-1',
    status: 'open',
    read: false,
    createdAt: '2026-08-10T10:00:00Z',
    updatedAt: '2026-08-10T10:00:00Z',
    ...overrides,
  };
}

describe('notification presentation', () => {
  it('maps operator dispositions to Chinese labels', () => {
    expect(dispositionLabel('confirmed')).toBe('已确认');
    expect(dispositionLabel('false_positive')).toBe('误报');
    expect(dispositionLabel('ignored')).toBe('已忽略');
  });

  it('shows pending for open enforcement notifications', () => {
    expect(notificationStatusText(notification())).toBe('待处理');
  });

  it('shows recovery time for resolved service alerts', () => {
    expect(notificationStatusText(notification({ type: 'collector', status: 'resolved', resolvedAt: '2026-08-10T10:05:00Z' }))).toContain('已恢复');
  });
});