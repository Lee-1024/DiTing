package systemconfig

import (
	"encoding/json"
	"net/http"
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
