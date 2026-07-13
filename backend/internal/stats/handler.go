package stats

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	repository Repository
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	result, err := h.repository.Overview(r.Context(), parseQuery(r))
	writeJSON(w, result, err)
}

func (h *Handler) EventTrend(w http.ResponseWriter, r *http.Request) {
	result, err := h.repository.EventTrend(r.Context(), parseQuery(r))
	writeJSON(w, result, err)
}

func (h *Handler) TopCommands(w http.ResponseWriter, r *http.Request) {
	result, err := h.repository.TopCommands(r.Context(), parseQuery(r))
	writeJSON(w, result, err)
}

func (h *Handler) CommandStats(w http.ResponseWriter, r *http.Request) {
	result, err := h.repository.CommandStats(r.Context(), parseQuery(r))
	writeJSON(w, result, err)
}

func (h *Handler) ExportCommandStats(w http.ResponseWriter, r *http.Request) {
	query := parseQuery(r)
	query.Limit = 5000
	result, err := h.repository.CommandStats(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="command-stats.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"process_name", "cmdline", "login_username", "username", "count", "first_seen", "last_seen"})
	for _, item := range result {
		_ = writer.Write([]string{
			item.ProcessName,
			item.Cmdline,
			item.LoginUsername,
			item.Username,
			strconv.FormatUint(item.Count, 10),
			item.FirstSeen,
			item.LastSeen,
		})
	}
	writer.Flush()
}

func (h *Handler) UserAudits(w http.ResponseWriter, r *http.Request) {
	result, err := h.repository.UserAudits(r.Context(), parseQuery(r))
	writeJSON(w, result, err)
}

func (h *Handler) HostAudits(w http.ResponseWriter, r *http.Request) {
	result, err := h.repository.HostAudits(r.Context(), parseQuery(r))
	writeJSON(w, result, err)
}

func parseQuery(r *http.Request) Query {
	values := r.URL.Query()
	now := time.Now().UTC()
	query := Query{
		StartTime: now.Add(-7 * 24 * time.Hour),
		EndTime:   now,
		Limit:     10,
	}
	if raw := values.Get("start_time"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			query.StartTime = parsed
		}
	}
	if raw := values.Get("end_time"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			query.EndTime = parsed
		}
	}
	if raw := values.Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			query.Limit = parsed
		}
	}
	query.Keyword = values.Get("keyword")
	query.Username = values.Get("username")
	return query
}

func writeJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
