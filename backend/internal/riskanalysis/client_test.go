package riskanalysis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"diting/backend/internal/audit"
	"diting/backend/internal/config"
)

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

func TestParseAnalysisJSONReturnsChineseErrorForThinkOnlyOutput(t *testing.T) {
	_, err := parseAnalysisJSON(`<think>模型内部推理，没有输出最终 JSON`)
	if err == nil {
		t.Fatal("expected parseAnalysisJSON to reject non-json output")
	}
	if got := err.Error(); !strings.HasPrefix(got, "模型输出不是合法") {
		t.Fatalf("expected chinese invalid json error, got %q", got)
	}
}

func TestFallbackAnalysisFromInvalidOutputKeepsReviewableResult(t *testing.T) {
	analysis := fallbackAnalysisFromInvalidOutput(`<think>模型内部推理，没有输出最终 JSON`)
	if analysis.Verdict != VerdictNeedsReview {
		t.Fatalf("expected needs_review verdict, got %q", analysis.Verdict)
	}
	if analysis.Confidence != 0 || len(analysis.Evidence) != 1 {
		t.Fatalf("unexpected fallback analysis: %#v", analysis)
	}
}

func TestAnalyzeRepairsInvalidModelOutput(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		content := `<think>只输出了思考，没有最终 JSON`
		if requests == 2 {
			content = `{"ai_severity":"low","verdict":"false_positive","confidence":88,"reason":"监控巡检命令","evidence":["docker ps"],"suggestion":"可加入可信动作过滤"}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": content}},
			},
		})
	}))
	defer server.Close()

	analyzer := NewOpenAICompatibleAnalyzer(config.AIConfig{
		Enabled:        true,
		BaseURL:        server.URL,
		Model:          "test-model",
		TimeoutSeconds: 5,
		MaxTokens:      800,
	})
	analysis, err := analyzer.Analyze(context.Background(), audit.Event{EventID: "event-1", Cmdline: "/usr/bin/docker ps"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected repair request, got %d requests", requests)
	}
	if analysis.Verdict != VerdictFalsePositive || analysis.AISeverity != "low" || analysis.Confidence != 88 {
		t.Fatalf("unexpected repaired analysis: %#v", analysis)
	}
}

func TestAnalyzeUsesMiniMaxResponsesAPIWithReasoningNone(t *testing.T) {
	var path string
	var reasoningEffort string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		var request struct {
			Reasoning struct {
				Effort string `json:"effort"`
			} `json:"reasoning"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		reasoningEffort = request.Reasoning.Effort
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "completed",
			"output_text": `{"ai_severity":"low","verdict":"false_positive","confidence":91,"reason":"监控巡检命令","evidence":["docker ps"],"suggestion":"可加入可信动作过滤"}`,
		})
	}))
	defer server.Close()

	analyzer := NewOpenAICompatibleAnalyzer(config.AIConfig{
		Enabled:        true,
		BaseURL:        server.URL,
		Model:          "MiniMax-M3",
		TimeoutSeconds: 5,
		MaxTokens:      800,
	})
	analysis, err := analyzer.Analyze(context.Background(), audit.Event{EventID: "event-1", Cmdline: "/usr/bin/docker ps"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if path != "/v1/responses" {
		t.Fatalf("expected MiniMax responses path, got %q", path)
	}
	if reasoningEffort != "none" {
		t.Fatalf("expected reasoning effort none, got %q", reasoningEffort)
	}
	if analysis.Verdict != VerdictFalsePositive || analysis.AISeverity != "low" || analysis.Confidence != 91 {
		t.Fatalf("unexpected MiniMax analysis: %#v", analysis)
	}
}
