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
  preAudit?: boolean;
  conditions: CollectorFilterCondition[];
}

export interface CollectorFilterCondition {
  field: string;
  op: string;
  value?: string;
  values?: string[];
}

export interface AIProviderConfig {
  enabled: boolean;
  baseUrl: string;
  model: string;
  timeoutSeconds: number;
  maxTokens: number;
  apiKey?: string;
  apiKeySet?: boolean;
  maskedApiKey?: string;
}
