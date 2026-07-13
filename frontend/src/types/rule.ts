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
