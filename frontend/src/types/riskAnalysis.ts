export type AIRiskVerdict = 'true_positive' | 'suspicious' | 'false_positive' | 'needs_review';

export interface AIRiskAnalysis {
  eventId: string;
  aiSeverity: string;
  verdict: AIRiskVerdict;
  confidence: number;
  reason: string;
  evidence: string[];
  suggestion: string;
  model: string;
  rawResponse?: string;
  analyzedAt: string;
  createdAt: string;
  updatedAt: string;
}

export type AIRiskAnalysisMap = Record<string, AIRiskAnalysis>;
