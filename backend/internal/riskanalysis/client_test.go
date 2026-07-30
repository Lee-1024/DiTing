package riskanalysis

import "testing"

func TestParseAnalysisJSONNormalizesModelVerdict(t *testing.T) {
	analysis, err := parseAnalysisJSON(`{"ai_severity":"critical","verdict":"true_positive","confidence":120,"reason":"敏感文件写入","evidence":["/etc/shadow"],"suggestion":"人工确认"}`)
	if err != nil {
		t.Fatalf("parseAnalysisJSON returned error: %v", err)
	}
	if analysis.AISeverity != "critical" || analysis.Verdict != VerdictTruePositive {
		t.Fatalf("unexpected analysis: %#v", analysis)
	}
	if analysis.Confidence != 100 {
		t.Fatalf("expected confidence capped to 100, got %d", analysis.Confidence)
	}
	if len(analysis.Evidence) != 1 || analysis.Evidence[0] != "/etc/shadow" {
		t.Fatalf("unexpected evidence: %#v", analysis.Evidence)
	}
}

func TestParseAnalysisJSONExtractsObjectAfterThinkBlock(t *testing.T) {
	analysis, err := parseAnalysisJSON(`<think>模型内部推理</think>
{"ai_severity":"low","verdict":"needs_review","confidence":80,"reason":"连通性测试","evidence":["test"],"suggestion":"测试通过"}`)
	if err != nil {
		t.Fatalf("parseAnalysisJSON returned error: %v", err)
	}
	if analysis.AISeverity != "low" || analysis.Verdict != VerdictNeedsReview || analysis.Reason != "连通性测试" {
		t.Fatalf("unexpected analysis: %#v", analysis)
	}
}
