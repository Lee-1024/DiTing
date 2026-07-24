package operationlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

// List 查询并返回 List 列表。
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query, err := ParseQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	items := []Entry{}
	total := 0
	if h.repository != nil {
		items, total, err = h.repository.List(r.Context(), query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":    items,
		"page":     query.Page,
		"pageSize": query.PageSize,
		"total":    total,
	})
}

// ParseQuery 解析 Parse Query 并返回结构化结果。
func ParseQuery(r *http.Request) (Query, error) {
	values := r.URL.Query()
	now := time.Now().UTC()
	query := Query{
		StartTime: now.Add(-7 * 24 * time.Hour),
		EndTime:   now,
		Username:  strings.TrimSpace(values.Get("username")),
		Method:    strings.TrimSpace(values.Get("method")),
		Keyword:   strings.TrimSpace(values.Get("keyword")),
		Page:      parsePositiveInt(values.Get("page"), 1),
		PageSize:  parsePositiveInt(values.Get("page_size"), 10),
	}
	if query.PageSize > 500 {
		query.PageSize = 500
	}
	if raw := values.Get("start_time"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return Query{}, fmt.Errorf("invalid start_time: %w", err)
		}
		query.StartTime = parsed
	}
	if raw := values.Get("end_time"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return Query{}, fmt.Errorf("invalid end_time: %w", err)
		}
		query.EndTime = parsed
	}
	if raw := values.Get("status"); raw != "" {
		status, err := strconv.Atoi(raw)
		if err != nil {
			return Query{}, fmt.Errorf("invalid status: %w", err)
		}
		query.Status = status
	}
	return query, nil
}

// parsePositiveInt 解析 parse Positive Int 并返回结构化结果。
func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
