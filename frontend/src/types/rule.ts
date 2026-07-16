export interface RuleCondition {
  field: string;
  op: string;
  value?: string;
  values?: string[];
}

export interface RuleExpression {
  operator: 'and' | 'or';
  conditions: RuleCondition[];
}

export interface AuditRule {
  id: string;
  name: string;
  description: string;
  eventType: string;
  enabled: boolean;
  severity: string;
  riskScore: number;
  matchExpr: RuleExpression;
  tags: string[];
  updatedAt: string;
}

export type RulePayload = Omit<AuditRule, 'id' | 'updatedAt'>;

export interface RuleTestEvent {
  eventType?: string;
  severity?: string;
  hostName?: string;
  nodeName?: string;
  namespace?: string;
  podName?: string;
  containerId?: string;
  processName?: string;
  binaryPath?: string;
  cmdline?: string;
  username?: string;
  loginUsername?: string;
  filePath?: string;
  dstIp?: string;
  domain?: string;
}

export interface RuleTestResponse {
  matched: boolean;
  message: string;
  matches: Array<{
    field: string;
    operator: string;
    value: string;
    actual: string;
  }>;
}
