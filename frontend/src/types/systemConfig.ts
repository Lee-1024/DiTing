export interface CollectorFilterConfig {
  enabled: boolean;
  ignoreProcessNames: string[];
  ignoreCommandKeywords: string[];
  ignoreUsers: string[];
  keepSeverities: string[];
}
