package riskanalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
		timeout = 120 * time.Second
	}
	return &OpenAICompatibleAnalyzer{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (a *OpenAICompatibleAnalyzer) Analyze(ctx context.Context, event audit.Event) (Analysis, error) {
	if !a.cfg.Enabled {
		return Analysis{}, fmt.Errorf("AI 复核未启用，请先到“配置管理 -> AI 配置”开启并保存")
	}
	if strings.TrimSpace(a.cfg.BaseURL) == "" || strings.TrimSpace(a.cfg.Model) == "" {
		return Analysis{}, fmt.Errorf("AI 配置不完整，请填写模型服务 Base URL 和模型名称")
	}
	raw, err := a.completion(ctx, aiSystemPrompt(), eventPrompt(event))
	if err != nil {
		return Analysis{}, err
	}
	analysis, err := parseAnalysisJSON(raw)
	if err != nil {
		repairedRaw, repairErr := a.completion(ctx, aiRepairSystemPrompt(), repairPrompt(event, raw))
		if repairErr == nil {
			if repairedAnalysis, parseErr := parseAnalysisJSON(repairedRaw); parseErr == nil {
				analysis = repairedAnalysis
				raw = raw + "\n\n--- repair ---\n" + repairedRaw
				err = nil
			}
		}
	}
	if err != nil {
		analysis = fallbackAnalysisFromInvalidOutput(raw)
	}
	analysis.EventID = event.EventID
	analysis.Model = a.cfg.Model
	analysis.RawResponse = raw
	analysis.AnalyzedAt = time.Now().UTC()
	return analysis, nil
}

func (a *OpenAICompatibleAnalyzer) completion(ctx context.Context, instructions string, prompt string) (string, error) {
	if a.usesMiniMaxResponsesAPI() {
		return a.responsesCompletion(ctx, instructions, prompt)
	}
	return a.chatCompletion(ctx, a.newChatCompletionRequest([]chatMessage{
		{Role: "system", Content: instructions},
		{Role: "user", Content: prompt},
	}))
}

func (a *OpenAICompatibleAnalyzer) usesMiniMaxResponsesAPI() bool {
	baseURL := strings.ToLower(strings.TrimSpace(a.cfg.BaseURL))
	model := strings.ToLower(strings.TrimSpace(a.cfg.Model))
	return strings.Contains(baseURL, "minimaxi.com") || strings.HasPrefix(model, "minimax-")
}

func (a *OpenAICompatibleAnalyzer) responsesCompletion(ctx context.Context, instructions string, prompt string) (string, error) {
	request := miniMaxResponsesRequest{
		Model:           a.cfg.Model,
		Input:           prompt,
		Instructions:    instructions,
		MaxOutputTokens: a.cfg.MaxTokens,
		Temperature:     0,
		Reasoning: miniMaxReasoning{
			Effort: "none",
		},
		Stream: false,
	}
	if request.MaxOutputTokens <= 0 {
		request.MaxOutputTokens = 800
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(normalizeMiniMaxBaseURL(a.cfg.BaseURL), "/") + "/responses"
	responseBody, err := a.postJSON(ctx, url, body)
	if err != nil {
		return "", err
	}
	var response miniMaxResponsesResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("模型服务返回的内容不是合法 JSON：%s", trimProviderBody(responseBody))
	}
	if response.Status == "failed" {
		return "", fmt.Errorf("模型服务返回失败：%s", response.Error.Message)
	}
	if response.Status == "incomplete" {
		return "", fmt.Errorf("模型服务输出未完成：%s", response.IncompleteDetails.Reason)
	}
	if strings.TrimSpace(response.OutputText) == "" {
		return "", fmt.Errorf("模型服务没有返回可用结果")
	}
	return strings.TrimSpace(response.OutputText), nil
}

func normalizeMiniMaxBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
	}
	if strings.HasSuffix(baseURL, "/responses") {
		baseURL = strings.TrimSuffix(baseURL, "/responses")
	}
	if strings.HasSuffix(baseURL, "/anthropic") {
		baseURL = strings.TrimSuffix(baseURL, "/anthropic") + "/v1"
	}
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return baseURL
}

func (a *OpenAICompatibleAnalyzer) newChatCompletionRequest(messages []chatMessage) chatCompletionRequest {
	request := chatCompletionRequest{
		Model:       a.cfg.Model,
		Messages:    messages,
		Temperature: 0,
		MaxTokens:   a.cfg.MaxTokens,
		ResponseFormat: &chatResponseFormat{
			Type: "json_object",
		},
	}
	if request.MaxTokens <= 0 {
		request.MaxTokens = 800
	}
	return request
}

func (a *OpenAICompatibleAnalyzer) chatCompletion(ctx context.Context, request chatCompletionRequest) (string, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(a.cfg.BaseURL, "/") + "/chat/completions"
	responseBody, err := a.postJSON(ctx, url, body)
	if err != nil {
		if request.ResponseFormat != nil && strings.Contains(err.Error(), "状态码 400") {
			request.ResponseFormat = nil
			return a.chatCompletion(ctx, request)
		}
		return "", err
	}
	var response chatCompletionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("模型服务返回的内容不是合法 JSON：%s", trimProviderBody(responseBody))
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("模型服务没有返回可用结果")
	}
	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

func (a *OpenAICompatibleAnalyzer) postJSON(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		if os.IsTimeout(err) || strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			return nil, fmt.Errorf("AI 分析超时，请在 AI 配置中调大超时秒数或降低最大输出 Token：%w", err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("模型服务返回错误，状态码 %d：%s", resp.StatusCode, trimProviderBody(responseBody))
	}
	return responseBody, nil
}

func trimProviderBody(data []byte) string {
	value := strings.TrimSpace(string(data))
	if len(value) > 500 {
		return value[:500] + "..."
	}
	return value
}

func parseAnalysisJSON(raw string) (Analysis, error) {
	raw = normalizeAnalysisJSONContent(raw)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return Analysis{}, fmt.Errorf("模型输出不是合法 AI 风险分析 JSON：%s", trimTextForError(raw))
	}
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

func normalizeAnalysisJSONContent(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	raw = stripThinkBlocks(raw)
	if start := strings.Index(raw, "{"); start >= 0 {
		if end := strings.LastIndex(raw, "}"); end > start {
			return strings.TrimSpace(raw[start : end+1])
		}
	}
	return raw
}

func stripThinkBlocks(raw string) string {
	for {
		start := strings.Index(strings.ToLower(raw), "<think>")
		if start < 0 {
			return strings.TrimSpace(raw)
		}
		end := strings.Index(strings.ToLower(raw[start:]), "</think>")
		if end < 0 {
			return strings.TrimSpace(raw[:start])
		}
		raw = raw[:start] + raw[start+end+len("</think>"):]
	}
}

func fallbackAnalysisFromInvalidOutput(raw string) Analysis {
	evidence := []string{}
	if trimmed := trimTextForError(raw); trimmed != "" {
		evidence = append(evidence, "模型原始输出："+trimmed)
	}
	return Analysis{
		AISeverity: "medium",
		Verdict:    VerdictNeedsReview,
		Confidence: 0,
		Reason:     "模型没有按要求输出结构化 JSON，系统已保留原始输出，请人工复核。",
		Evidence:   evidence,
		Suggestion: "建议在 AI 配置中降低最大输出 Token 或更换支持 JSON 输出的模型；当前事件不要仅依据 AI 自动降级。",
	}
}

func trimTextForError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 300 {
		return value[:300] + "..."
	}
	return value
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
必须只输出 JSON 对象，字段为：ai_severity(info|low|medium|high|critical), verdict(true_positive|suspicious|false_positive|needs_review), confidence(0-100), reason, evidence(字符串数组), suggestion。
不要输出 Markdown，不要输出 <think>，不要输出解释性前后缀。
判断时区分巡检/监控/健康检查噪声和真实攻击行为；保守处理不确定事件，避免把真正风险误判为无害。`
}

func aiRepairSystemPrompt() string {
	return `你是主机安全审计分析助手。你的任务是把上一次模型输出修正为可入库的风险分析 JSON。
必须只输出 JSON 对象，字段为：ai_severity(info|low|medium|high|critical), verdict(true_positive|suspicious|false_positive|needs_review), confidence(0-100), reason, evidence(字符串数组), suggestion。
不要输出 Markdown，不要输出 <think>，不要复述原始文本。`
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

func repairPrompt(event audit.Event, previousOutput string) string {
	return eventPrompt(event) + "\n\n上一次模型输出未能解析为 JSON，请基于同一事件重新输出合规 JSON。上一次原始输出如下：\n" + trimTextForError(previousOutput)
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
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	Temperature    float64             `json:"temperature"`
	MaxTokens      int                 `json:"max_tokens"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
}

type chatResponseFormat struct {
	Type string `json:"type"`
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

type miniMaxResponsesRequest struct {
	Model           string           `json:"model"`
	Input           string           `json:"input"`
	Instructions    string           `json:"instructions"`
	MaxOutputTokens int              `json:"max_output_tokens"`
	Temperature     float64          `json:"temperature"`
	Reasoning       miniMaxReasoning `json:"reasoning"`
	Stream          bool             `json:"stream"`
}

type miniMaxReasoning struct {
	Effort string `json:"effort"`
}

type miniMaxResponsesResponse struct {
	Status     string `json:"status"`
	OutputText string `json:"output_text"`
	Error      struct {
		Message string `json:"message"`
	} `json:"error"`
	IncompleteDetails struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}
