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
