import { apiClient } from './client';
import type { CommandItem, CommandStatsQuery, HostAuditItem, HostAuditQuery, HostBehavior, HostUserItem, OverviewStats, RuleHitItem, RuleHitQuery, StatsQuery, TopItem, TrendPoint, UserAuditItem, UserAuditQuery } from '../types/stats';

export async function getOverview(params?: StatsQuery): Promise<OverviewStats> {
  const response = await apiClient.get<OverviewStats>('/stats/overview', { params });
  return response.data;
}

export async function getEventTrend(params?: StatsQuery): Promise<TrendPoint[]> {
  const response = await apiClient.get<TrendPoint[]>('/stats/event-trend', { params });
  return response.data;
}

export async function getTopCommands(limit = 10, params?: StatsQuery): Promise<TopItem[]> {
  const response = await apiClient.get<TopItem[]>('/stats/top-commands', { params: { ...params, limit } });
  return response.data;
}

export async function getTopHosts(limit = 10, params?: StatsQuery): Promise<TopItem[]> {
  const response = await apiClient.get<TopItem[]>('/stats/top-hosts', { params: { ...params, limit } });
  return response.data;
}

export async function getTopNamespaces(limit = 10, params?: StatsQuery): Promise<TopItem[]> {
  const response = await apiClient.get<TopItem[]>('/stats/top-namespaces', { params: { ...params, limit } });
  return response.data;
}

export async function getCommandStats(params: CommandStatsQuery): Promise<CommandItem[]> {
  const response = await apiClient.get<CommandItem[]>('/stats/commands', { params });
  return response.data;
}

export async function exportCommandStats(params: CommandStatsQuery): Promise<Blob> {
  const response = await apiClient.get('/stats/commands/export', { params, responseType: 'blob' });
  return response.data;
}

export async function getUserAudits(params: UserAuditQuery): Promise<UserAuditItem[]> {
  const response = await apiClient.get<UserAuditItem[]>('/stats/users', { params });
  return response.data;
}

export async function getHostAudits(params: HostAuditQuery): Promise<HostAuditItem[]> {
  const response = await apiClient.get<HostAuditItem[]>('/stats/hosts', { params });
  return response.data;
}

export async function exportHostAudits(params: HostAuditQuery): Promise<Blob> {
  const response = await apiClient.get('/stats/hosts/export', { params, responseType: 'blob' });
  return response.data;
}

export async function getHostUsers(params: HostAuditQuery): Promise<HostUserItem[]> {
  const response = await apiClient.get<HostUserItem[]>('/stats/hosts/users', { params });
  return response.data;
}

export async function getHostBehavior(params: HostAuditQuery): Promise<HostBehavior> {
  const response = await apiClient.get<HostBehavior>('/stats/hosts/behavior', { params });
  return response.data;
}

export async function getRuleHits(params: RuleHitQuery): Promise<RuleHitItem[]> {
  const response = await apiClient.get<RuleHitItem[]>('/stats/rules', { params });
  return response.data;
}
