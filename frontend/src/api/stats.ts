import { apiClient } from './client';
import type { CommandItem, CommandStatsQuery, HostAuditItem, HostAuditQuery, OverviewStats, TopItem, TrendPoint, UserAuditItem, UserAuditQuery } from '../types/stats';

export async function getOverview(): Promise<OverviewStats> {
  const response = await apiClient.get<OverviewStats>('/stats/overview');
  return response.data;
}

export async function getEventTrend(): Promise<TrendPoint[]> {
  const response = await apiClient.get<TrendPoint[]>('/stats/event-trend');
  return response.data;
}

export async function getTopCommands(limit = 10): Promise<TopItem[]> {
  const response = await apiClient.get<TopItem[]>('/stats/top-commands', { params: { limit } });
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
