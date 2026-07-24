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
	request = normalizeCollectorFilterConfig(request)
	if err := h.repository.SaveCollectorFilter(r.Context(), request); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(request)
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
