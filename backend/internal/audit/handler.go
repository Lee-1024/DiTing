package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Repository interface {
	ListEvents(ctx context.Context, query Query) ([]Event, int, error)
	GetEvent(ctx context.Context, eventID string) (Event, error)
}

var ErrNotFound = errors.New("audit event not found")

type Handler struct {
	repository Repository
}

// NewHandler 创建并初始化 New Handler 实例。
func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

// ListEvents 查询并返回 List Events 列表。
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	query, err := ParseQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	items := []Event{}
	total := 0
	if h.repository != nil {
		var err error
		items, total, err = h.repository.ListEvents(r.Context(), query)
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

// GetEvent 查询并返回指定的 Get Event。
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(r.PathValue("event_id"))
	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	if h.repository == nil {
		http.Error(w, "audit event not found", http.StatusNotFound)
		return
	}
	event, err := h.repository.GetEvent(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "audit event not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(event)
}

// ExportEvents 处理 Export Events 相关逻辑。
func (h *Handler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	query, err := ParseQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	query.Page = 1
	query.PageSize = 5000

	items := []Event{}
	if h.repository != nil {
		items, _, err = h.repository.ListEvents(r.Context(), query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="audit-events.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"event_time", "severity", "login_username", "username", "host", "process", "cmdline", "rule_names", "event_id"})
	for _, event := range items {
		_ = writer.Write([]string{
			event.EventTime.Format("2006-01-02 15:04:05.000"),
			event.Severity,
			firstNonEmpty(event.LoginUsername, event.Username),
			event.Username,
			firstNonEmpty(event.NodeName, event.HostName),
			event.ProcessName,
			event.Cmdline,
			strings.Join(event.RuleNames, "|"),
			event.EventID,
		})
	}
	writer.Flush()
}

// firstNonEmpty 处理 first Non Empty 相关逻辑。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
