package systemconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	repository Repository
}

// NewHandler 创建并初始化 New Handler 实例。
func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

// GetCollectorFilter 查询并返回指定的 Get Collector Filter。
func (h *Handler) GetCollectorFilter(w http.ResponseWriter, r *http.Request) {
	config, err := h.repository.GetCollectorFilter(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

// SaveCollectorFilter 处理 Save Collector Filter 相关逻辑。
func (h *Handler) SaveCollectorFilter(w http.ResponseWriter, r *http.Request) {
	var request CollectorFilterConfig
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validCollectorFilterSeverities(request.KeepSeverities) {
		http.Error(w, "invalid keep severity", http.StatusBadRequest)
		return
	}
	if !validCollectorFilterRules(request.Rules) {
		http.Error(w, "invalid collector filter rule", http.StatusBadRequest)
		return
	}
	request = normalizeCollectorFilterConfig(request)
	if err := h.repository.SaveCollectorFilter(r.Context(), request); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(request)
}

func (h *Handler) GetAIConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.repository.GetAIConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	config.APIKey = ""
	config.EncryptedAPIKey = ""
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

func (h *Handler) SaveAIConfig(w http.ResponseWriter, r *http.Request) {
	var request AIProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.Enabled && (request.BaseURL == "" || request.Model == "") {
		http.Error(w, "请填写模型服务 Base URL 和模型名称", http.StatusBadRequest)
		return
	}
	if err := h.repository.SaveAIConfig(r.Context(), request); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	config, err := h.repository.GetAIConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	config.APIKey = ""
	config.EncryptedAPIKey = ""
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

func (h *Handler) TestAIConfig(w http.ResponseWriter, r *http.Request) {
	var request AIProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.BaseURL == "" || request.Model == "" {
		http.Error(w, "请填写模型服务 Base URL 和模型名称", http.StatusBadRequest)
		return
	}
	if request.APIKey == "" {
		existing, err := h.repository.GetAIConfig(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		request.APIKey = existing.APIKey
	}
	started := time.Now()
	if err := testOpenAICompatibleProvider(r, request); err != nil {
		slog.Error("test ai provider failed", "base_url", request.BaseURL, "model", request.Model, "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"latencyMs": time.Since(started).Milliseconds(),
		"message":   "模型服务可用",
	})
}

func testOpenAICompatibleProvider(r *http.Request, config AIProviderConfig) error {
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	maxTokens := config.MaxTokens
	if maxTokens <= 0 || maxTokens > 200 {
		maxTokens = 80
	}
	body, err := json.Marshal(map[string]any{
		"model": config.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是主机安全审计分析助手。必须只输出 JSON。"},
			{"role": "user", "content": `请按这个 JSON Schema 输出一次测试分析，不要输出 Markdown：{"ai_severity":"low","verdict":"needs_review","confidence":80,"reason":"连通性测试","evidence":["test"],"suggestion":"测试通过"}`},
		},
		"temperature": 0,
		"max_tokens":  maxTokens,
	})
	if err != nil {
		return err
	}
	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(config.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("模型服务返回的内容不是合法 JSON：%s", trimForError(responseBody))
	}
	if resp.StatusCode >= 300 {
		message := response.Error.Message
		if message == "" {
			message = trimForError(responseBody)
		}
		return fmt.Errorf("模型服务返回错误，状态码 %d：%s", resp.StatusCode, message)
	}
	if len(response.Choices) == 0 {
		return fmt.Errorf("模型服务没有返回可用结果")
	}
	if err := validateAIAnalysisContent(response.Choices[0].Message.Content); err != nil {
		return err
	}
	return nil
}

func validateAIAnalysisContent(content string) error {
	raw := strings.TrimSpace(content)
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
		return fmt.Errorf("模型服务可连通，但没有按 AI 风险分析 JSON 格式输出：%s", raw)
	}
	if payload.AISeverity == "" || payload.Verdict == "" || payload.Reason == "" {
		return fmt.Errorf("模型服务可连通，但 AI 风险分析 JSON 字段不完整：%s", raw)
	}
	return nil
}

func trimForError(data []byte) string {
	value := strings.TrimSpace(string(data))
	if len(value) > 500 {
		return value[:500] + "..."
	}
	return value
}

func validCollectorFilterRules(rules []CollectorFilterRule) bool {
	for _, rule := range rules {
		if rule.PreAudit && !validPreAuditCollectorFilterRule(rule) {
			return false
		}
		for _, condition := range rule.Conditions {
			if !validCollectorFilterField(condition.Field) || !validCollectorFilterOp(condition.Op) {
				return false
			}
			if condition.Op == "in" && len(condition.Values) == 0 {
				return false
			}
			if condition.Op != "in" && condition.Value == "" {
				return false
			}
		}
	}
	return true
}

func validPreAuditCollectorFilterRule(rule CollectorFilterRule) bool {
	hasSource := false
	hasAction := false
	for _, condition := range rule.Conditions {
		switch condition.Field {
		case "parent_process_name", "parent_cmdline", "username", "login_username", "cwd":
			hasSource = true
		case "process_name", "cmdline", "file_path", "file_operation", "dst_ip", "dst_port", "protocol", "domain":
			hasAction = true
		}
	}
	return hasSource && hasAction
}

func validCollectorFilterField(field string) bool {
	switch field {
	case "event_type", "severity", "process_name", "cmdline", "cwd", "parent_process_name", "parent_cmdline", "username", "login_username", "file_path", "file_operation", "dst_ip", "dst_port", "protocol", "domain":
		return true
	default:
		return false
	}
}

func validCollectorFilterOp(op string) bool {
	switch op {
	case "eq", "contains", "in", "prefix", "regex":
		return true
	default:
		return false
	}
}

// validCollectorFilterSeverities 校验 valid Collector Filter Severities 是否满足要求。
func validCollectorFilterSeverities(values []string) bool {
	for _, value := range values {
		switch value {
		case "info", "low", "medium", "high", "critical":
		default:
			return false
		}
	}
	return true
}
