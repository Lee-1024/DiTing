export interface AuditEvent {
  eventId: string;
  eventTime: string;
  eventType: string;
  severity: string;
  riskScore: number;
  hostName: string;
  hostId?: string;
  nodeName?: string;
  namespace: string;
  podName: string;
  containerId?: string;
  containerName?: string;
  image?: string;
  username: string;
  loginUsername?: string;
  processName: string;
  binaryPath?: string;
  cmdline: string;
  cwd?: string;
  parentProcessName?: string;
  parentCmdline?: string;
  uid?: number;
  gid?: number;
  auid?: number;
  euid?: number;
  egid?: number;
  tags: string[];
  ruleIds?: string[];
  ruleNames?: string[];
  rawEvent?: string;
}

export interface AuditEventQuery {
  start_time?: string;
  end_time?: string;
  event_type?: string;
  severity?: string;
  severity_in?: string;
  host_name?: string;
  username?: string;
  keyword?: string;
  cmdline?: string;
  page?: number;
  page_size?: number;
}

export interface PagedAuditEvents {
  items: AuditEvent[];
  page: number;
  pageSize: number;
  total: number;
}
