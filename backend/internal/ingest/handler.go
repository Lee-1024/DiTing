package ingest

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

type EventWriter interface {
	WriteEvents(ctx context.Context, events []audit.Event) error
}

type Handler struct {
	writer EventWriter
	token  string
}

// NewHandler 创建并初始化 New Handler 实例。
func NewHandler(writer EventWriter, token string) *Handler {
	return &Handler{writer: writer, token: strings.TrimSpace(token)}
}

// IngestEvents 处理 Ingest Events 相关逻辑。
func (h *Handler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.writer == nil || h.token == "" {
		http.Error(w, "ingest is not configured", http.StatusNotFound)
		return
	}
	if bearerToken(r.Header.Get("Authorization")) != h.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		Events []audit.Event `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	events := normalizeEvents(request.Events, time.Now().UTC())
	if err := h.writer.WriteEvents(r.Context(), events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]int{"accepted": len(events)})
}

// bearerToken 处理 bearer Token 相关逻辑。
func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

// normalizeEvents 规范化 normalize Events 的默认值和边界值。
func normalizeEvents(events []audit.Event, now time.Time) []audit.Event {
	normalized := make([]audit.Event, 0, len(events))
	for _, event := range events {
		if event.IngestTime.IsZero() {
			event.IngestTime = now
		}
		if event.EventTime.IsZero() {
			event.EventTime = now
		}
		if event.EventDate.IsZero() {
			event.EventDate = time.Date(event.EventTime.Year(), event.EventTime.Month(), event.EventTime.Day(), 0, 0, 0, 0, time.UTC)
		}
		normalized = append(normalized, event)
	}
	return normalized
}
