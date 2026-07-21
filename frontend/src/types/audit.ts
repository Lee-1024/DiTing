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
  filePath?: string;
  fileOperation?: string;
  srcIp?: string;
  srcPort?: number;
  dstIp?: string;
  dstPort?: number;
  protocol?: string;
  domain?: string;
  ruleIds?: string[];
  ruleNames?: string[];
  ruleMatches?: RuleMatch[];
  rawEvent?: string;
}

export interface RuleMatch {
  ruleId: string;
  ruleName: string;
  field: string;
  operator: string;
  value: string;
  actual: string;
}

export interface AuditEventQuery {
  start_time?: string;
  end_time?: string;
  event_type?: string;
  severity?: string;
  severity_in?: string;
  host_name?: string;
  namespace?: string;
  pod_name?: string;
  username?: string;
  login_username?: string;
  exec_username?: string;
  keyword?: string;
  cmdline?: string;
  file_path?: string;
  dst_ip?: string;
  dst_port?: number;
  page?: number;
  page_size?: number;
}

export interface PagedAuditEvents {
  items: AuditEvent[];
  page: number;
  pageSize: number;
  total: number;
}
