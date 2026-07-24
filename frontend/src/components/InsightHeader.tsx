import type { CSSProperties, ReactNode } from 'react';
import { Typography } from 'antd';
import { compactNumber } from '../utils/format';

type MetricTone = 'default' | 'success' | 'warning' | 'danger' | 'blue' | 'cyan';

const toneColors: Record<MetricTone, string> = {
  default: '#2563eb',
  success: '#16a34a',
  warning: '#d97706',
  danger: '#dc2626',
  blue: '#2563eb',
  cyan: '#06b6d4',
};

interface InsightHeroProps {
  className?: string;
  kicker: string;
  title: string;
  description: ReactNode;
  actions?: ReactNode;
}

export function InsightHero({ className = 'system-summary', kicker, title, description, actions }: InsightHeroProps) {
  return (
    <div className={className}>
      <div className="ops-kicker">{kicker}</div>
      <Typography.Title level={2} className="investigation-title">{title}</Typography.Title>
      <Typography.Text className="investigation-desc">{description}</Typography.Text>
      {actions && <div className="ops-hero-actions">{actions}</div>}
    </div>
  );
}

interface SummaryPanelProps {
  className?: string;
  kicker: string;
  title: ReactNode;
  description: ReactNode;
}

export function SummaryPanel({ className = 'system-summary system-summary-alt', kicker, title, description }: SummaryPanelProps) {
  return (
    <div className={className}>
      <div className="ops-kicker">{kicker}</div>
      <Typography.Title level={4} className="investigation-title">{title}</Typography.Title>
      <Typography.Text className="investigation-desc">{description}</Typography.Text>
    </div>
  );
}

interface MetricCardProps {
  label: string;
  value: number | string;
  hint: string;
  tone?: MetricTone;
  color?: string;
}

export function MetricCard({ label, value, hint, tone = 'default', color }: MetricCardProps) {
  const metricColor = color ?? toneColors[tone];
  const displayValue = typeof value === 'number' ? compactNumber(value) : value;
  return (
    <div className="metric-card" style={{ '--metric-color': metricColor } as CSSProperties}>
      <div className="metric-label">{label}</div>
      <div className="metric-value">{displayValue}</div>
      <div className="metric-hint">{hint}</div>
    </div>
  );
}
