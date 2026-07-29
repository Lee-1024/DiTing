export interface CollectorFilterConfig {
  enabled: boolean;
  ignoreProcessNames: string[];
  ignoreCommandKeywords: string[];
  ignoreUsers: string[];
  keepSeverities: string[];
  rules: CollectorFilterRule[];
}

export interface CollectorFilterRule {
  id: string;
  name: string;
  enabled: boolean;
  conditions: CollectorFilterCondition[];
}

export interface CollectorFilterCondition {
  field: string;
  op: string;
  value?: string;
  values?: string[];
}
