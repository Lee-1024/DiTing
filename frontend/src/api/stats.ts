import { apiClient } from './client';
import type { CommandItem, CommandStatsQuery, HostAuditItem, HostAuditQuery, HostBehavior, HostUserItem, OverviewStats, RuleHitItem, RuleHitQuery, StatsQuery, TopItem, TrendPoint, UserAuditItem, UserAuditQuery } from '../types/stats';

// getOverview 获取 get Overview 数据。
export async function getOverview(params?: StatsQuery): Promise<OverviewStats> {
  const response = await apiClient.get<OverviewStats>('/stats/overview', { params });
  return response.data;
}

// getEventTrend 获取 get Event Trend 数据。
export async function getEventTrend(params?: StatsQuery): Promise<TrendPoint[]> {
  const response = await apiClient.get<TrendPoint[]>('/stats/event-trend', { params });
  return response.data;
}

// getTopCommands 获取 get Top Commands 数据。
export async function getTopCommands(limit = 10, params?: StatsQuery): Promise<TopItem[]> {
  const response = await apiClient.get<TopItem[]>('/stats/top-commands', { params: { ...params, limit } });
  return response.data;
}

// getTopHosts 获取 get Top Hosts 数据。
export async function getTopHosts(limit = 10, params?: StatsQuery): Promise<TopItem[]> {
  const response = await apiClient.get<TopItem[]>('/stats/top-hosts', { params: { ...params, limit } });
  return response.data;
}

// getTopNamespaces 获取 get Top Namespaces 数据。
export async function getTopNamespaces(limit = 10, params?: StatsQuery): Promise<TopItem[]> {
  const response = await apiClient.get<TopItem[]>('/stats/top-namespaces', { params: { ...params, limit } });
  return response.data;
}

// getCommandStats 获取 get Command Stats 数据。
export async function getCommandStats(params: CommandStatsQuery): Promise<CommandItem[]> {
  const response = await apiClient.get<CommandItem[]>('/stats/commands', { params });
  return response.data;
}

// exportCommandStats 导出或下载 export Command Stats 数据。
export async function exportCommandStats(params: CommandStatsQuery): Promise<Blob> {
  const response = await apiClient.get('/stats/commands/export', { params, responseType: 'blob' });
  return response.data;
}

// getUserAudits 获取 get User Audits 数据。
export async function getUserAudits(params: UserAuditQuery): Promise<UserAuditItem[]> {
  const response = await apiClient.get<UserAuditItem[]>('/stats/users', { params });
  return response.data;
}

// getHostAudits 获取 get Host Audits 数据。
export async function getHostAudits(params: HostAuditQuery): Promise<HostAuditItem[]> {
  const response = await apiClient.get<HostAuditItem[]>('/stats/hosts', { params });
  return response.data;
}

// exportHostAudits 导出或下载 export Host Audits 数据。
export async function exportHostAudits(params: HostAuditQuery): Promise<Blob> {
  const response = await apiClient.get('/stats/hosts/export', { params, responseType: 'blob' });
  return response.data;
}

// getHostUsers 获取 get Host Users 数据。
export async function getHostUsers(params: HostAuditQuery): Promise<HostUserItem[]> {
  const response = await apiClient.get<HostUserItem[]>('/stats/hosts/users', { params });
  return response.data;
}

// getHostBehavior 获取 get Host Behavior 数据。
export async function getHostBehavior(params: HostAuditQuery): Promise<HostBehavior> {
  const response = await apiClient.get<HostBehavior>('/stats/hosts/behavior', { params });
  return response.data;
}

// getRuleHits 获取 get Rule Hits 数据。
export async function getRuleHits(params: RuleHitQuery): Promise<RuleHitItem[]> {
  const response = await apiClient.get<RuleHitItem[]>('/stats/rules', { params });
  return response.data;
}
