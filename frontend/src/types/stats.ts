export interface OverviewStats {
  totalEvents: number;
  highRiskEvents: number;
  activeHosts: number;
  activeRules: number;
}

export interface TrendPoint {
  time: string;
  count: number;
}

export interface TopItem {
  name: string;
  count: number;
}

export interface StatsQuery {
  start_time?: string;
  end_time?: string;
}

export interface CommandItem {
  processName: string;
  cmdline: string;
  username: string;
  loginUsername: string;
  hostId?: string;
  hostName?: string;
  nodeName?: string;
  hostCount: number;
  count: number;
  firstSeen: string;
  lastSeen: string;
}

export interface CommandStatsQuery {
  start_time?: string;
  end_time?: string;
  keyword?: string;
  username?: string;
  host_name?: string;
  limit?: number;
}

export interface UserAuditItem {
  username: string;
  commandCount: number;
  activeHosts: number;
  highRiskEvents: number;
  firstSeen: string;
  lastSeen: string;
}

export interface UserAuditQuery {
  start_time?: string;
  end_time?: string;
  keyword?: string;
  host_name?: string;
  limit?: number;
}

export interface HostAuditItem {
  hostId?: string;
  hostName: string;
  nodeName?: string;
  commandCount: number;
  activeUsers: number;
  highRiskEvents: number;
  firstSeen: string;
  lastSeen: string;
}

export interface HostUserItem {
  username: string;
  commandCount: number;
  highRiskEvents: number;
  firstSeen: string;
  lastSeen: string;
}

export interface BehaviorItem {
  name: string;
  count: number;
  firstSeen: string;
  lastSeen: string;
}

export interface HostBehavior {
  filePaths: BehaviorItem[];
  network: BehaviorItem[];
  eventTypes: BehaviorItem[];
  ruleHits: BehaviorItem[];
}

export interface RuleHitItem {
  ruleName: string;
  hitCount: number;
  activeHosts: number;
  activeUsers: number;
  firstSeen: string;
  lastSeen: string;
}

export interface HostAuditQuery {
  start_time?: string;
  end_time?: string;
  keyword?: string;
  host_name?: string;
  limit?: number;
}

export interface RuleHitQuery {
  start_time?: string;
  end_time?: string;
  keyword?: string;
  limit?: number;
}
