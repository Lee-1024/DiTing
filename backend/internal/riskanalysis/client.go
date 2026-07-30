package riskanalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"diting/backend/internal/audit"
	"diting/backend/internal/config"
	"diting/backend/internal/systemconfig"
)

type Analyzer interface {
	Analyze(ctx context.Context, event audit.Event) (Analysis, error)
}

type DynamicAnalyzer struct {
	repository systemconfig.Repository
}

func NewDynamicAnalyzer(repository systemconfig.Repository) *DynamicAnalyzer {
	return &DynamicAnalyzer{repository: repository}
}

func (a *DynamicAnalyzer) Analyze(ctx context.Context, event audit.Event) (Analysis, error) {
	cfg, err := a.repository.GetAIConfig(ctx)
	if err != nil {
		return Analysis{}, err
	}
	return NewOpenAICompatibleAnalyzer(config.AIConfig{
		Enabled:        cfg.Enabled,
		BaseURL:        cfg.BaseURL,
		APIKey:         cfg.APIKey,
		Model:          cfg.Model,
		TimeoutSeconds: cfg.TimeoutSeconds,
		MaxTokens:      cfg.MaxTokens,
	}).Analyze(ctx, event)
}

type OpenAICompatibleAnalyzer struct {
	cfg    config.AIConfig
	client *http.Client
}

func NewOpenAICompatibleAnalyzer(cfg config.AIConfig) *OpenAICompatibleAnalyzer {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &OpenAICompatibleAnalyzer{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (a *OpenAICompatibleAnalyzer) Analyze(ctx context.Context, event audit.Event) (Analysis, error) {
	if !a.cfg.Enabled {
		return Analysis{}, fmt.Errorf("ai analysis is disabled")
	}
	if strings.TrimSpace(a.cfg.BaseURL) == "" || strings.TrimSpace(a.cfg.Model) == "" {
		return Analysis{}, fmt.Errorf("ai base_url and model are required")
	}
	request := chatCompletionRequest{
		Model: a.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: aiSystemPrompt()},
			{Role: "user", Content: eventPrompt(event)},
		},
		Temperature: 0.1,
		MaxTokens:   a.cfg.MaxTokens,
	}
	if request.MaxTokens <= 0 {
		request.MaxTokens = 800
	}
	body, err := json.Marshal(request)
	if err != nil {
		return Analysis{}, err
	}
	url := strings.TrimRight(a.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Analysis{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return Analysis{}, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Analysis{}, err
	}
	var response chatCompletionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return Analysis{}, fmt.Errorf("ai provider returned invalid json: %s", strings.TrimSpace(string(responseBody)))
	}
	if resp.StatusCode >= 300 {
		message := response.Error.Message
		if message == "" {
			message = strings.TrimSpace(string(responseBody))
		}
		return Analysis{}, fmt.Errorf("ai provider returned %d: %s", resp.StatusCode, message)
	}
	if len(response.Choices) == 0 {
		return Analysis{}, fmt.Errorf("ai provider returned no choices")
	}
	raw := strings.TrimSpace(response.Choices[0].Message.Content)
	analysis, err := parseAnalysisJSON(raw)
	if err != nil {
		return Analysis{}, err
	}
	analysis.EventID = event.EventID
	analysis.Model = a.cfg.Model
	analysis.RawResponse = raw
	analysis.AnalyzedAt = time.Now().UTC()
	return analysis, nil
}

func parseAnalysisJSON(raw string) (Analysis, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	var payload struct {
		AISeverity string   `json:"ai_severity"`
		Verdict    string   `json:"verdict"`
		Confidence int      `json:"confidence"`
		Reason     string   `json:"reason"`
		Evidence   []string `json:"evidence"`
		Suggestion string   `json:"suggestion"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return Analysis{}, err
	}
	return normalizeAnalysis(Analysis{
		AISeverity: payload.AISeverity,
		Verdict:    payload.Verdict,
		Confidence: payload.Confidence,
		Reason:     payload.Reason,
		Evidence:   payload.Evidence,
		Suggestion: payload.Suggestion,
	}), nil
}

func normalizeAnalysis(analysis Analysis) Analysis {
	switch analysis.AISeverity {
	case "info", "low", "medium", "high", "critical":
	default:
		analysis.AISeverity = "medium"
	}
	switch analysis.Verdict {
	case VerdictTruePositive, VerdictSuspicious, VerdictFalsePositive, VerdictNeedsReview:
	default:
		analysis.Verdict = VerdictNeedsReview
	}
	if analysis.Confidence < 0 {
		analysis.Confidence = 0
	}
	if analysis.Confidence > 100 {
		analysis.Confidence = 100
	}
	if analysis.Evidence == nil {
		analysis.Evidence = []string{}
	}
	return analysis
}

func aiSystemPrompt() string {
	return `你是主机安全审计分析助手。只判断输入事件是否真实构成风险，不执行处置，不建议直接拦截。
必须只输出 JSON，字段为：ai_severity(info|low|medium|high|critical), verdict(true_positive|suspicious|false_positive|needs_review), confidence(0-100), reason, evidence(字符串数组), suggestion。
判断时区分巡检/监控/健康检查噪声和真实攻击行为；保守处理不确定事件，避免把真正风险误判为无害。`
}

func eventPrompt(event audit.Event) string {
	data, _ := json.MarshalIndent(map[string]any{
		"event_id":            event.EventID,
		"event_type":          event.EventType,
		"severity":            event.Severity,
		"risk_score":          event.RiskScore,
		"rule_names":          event.RuleNames,
		"rule_matches":        event.RuleMatches,
		"host":                firstNonEmpty(event.NodeName, event.HostName, event.HostID),
		"login_username":      event.LoginUsername,
		"username":            event.Username,
		"process_name":        event.ProcessName,
		"binary_path":         event.BinaryPath,
		"cmdline":             event.Cmdline,
		"cwd":                 event.CWD,
		"parent_process_name": event.ParentProcessName,
		"parent_cmdline":      event.ParentCmdline,
		"file_path":           event.FilePath,
		"file_operation":      event.FileOperation,
		"dst_ip":              event.DstIP,
		"dst_port":            event.DstPort,
		"protocol":            event.Protocol,
		"domain":              event.Domain,
	}, "", "  ")
	return "请分析以下审计事件：\n" + string(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
