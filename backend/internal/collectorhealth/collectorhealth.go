package collectorhealth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Heartbeat struct {
	HostID              string     `json:"hostId"`
	HostName            string     `json:"hostName"`
	InputMode           string     `json:"inputMode"`
	Status              string     `json:"status"`
	HealthLevel         string     `json:"healthLevel"`
	Message             string     `json:"message"`
	LastError           string     `json:"lastError"`
	LastSeenAt          time.Time  `json:"lastSeenAt"`
	LastEventTime       *time.Time `json:"lastEventTime,omitempty"`
	LastWriteAt         *time.Time `json:"lastWriteAt,omitempty"`
	HeartbeatLagSeconds int64      `json:"heartbeatLagSeconds"`
	EventLagSeconds     int64      `json:"eventLagSeconds,omitempty"`
	WriteLagSeconds     int64      `json:"writeLagSeconds,omitempty"`
	EventsWritten       uint64     `json:"eventsWritten"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type HeartbeatUpdate struct {
	HostID        string
	HostName      string
	InputMode     string
	LastError     string
	ClearError    bool
	LastSeenAt    time.Time
	LastEventTime *time.Time
	LastWriteAt   *time.Time
	EventsWritten uint64
}

type Repository interface {
	List(ctx context.Context, now time.Time) ([]Heartbeat, error)
	Upsert(ctx context.Context, update HeartbeatUpdate) error
}

type Handler struct {
	repository Repository
	token      string
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

func NewHandlerWithToken(repository Repository, token string) *Handler {
	return &Handler{repository: repository, token: strings.TrimSpace(token)}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repository.List(r.Context(), time.Now().UTC())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.repository == nil || h.token == "" {
		http.Error(w, "collector heartbeat is not configured", http.StatusNotFound)
		return
	}
	if bearerToken(r.Header.Get("Authorization")) != h.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		HostID        string     `json:"hostId"`
		HostName      string     `json:"hostName"`
		InputMode     string     `json:"inputMode"`
		LastError     string     `json:"lastError"`
		ClearError    bool       `json:"clearError"`
		LastSeenAt    time.Time  `json:"lastSeenAt"`
		LastEventTime *time.Time `json:"lastEventTime"`
		LastWriteAt   *time.Time `json:"lastWriteAt"`
		EventsWritten uint64     `json:"eventsWritten"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.HostID) == "" {
		http.Error(w, "hostId is required", http.StatusBadRequest)
		return
	}
	if request.LastSeenAt.IsZero() {
		request.LastSeenAt = time.Now().UTC()
	}
	if err := h.repository.Upsert(r.Context(), HeartbeatUpdate{
		HostID:        strings.TrimSpace(request.HostID),
		HostName:      strings.TrimSpace(request.HostName),
		InputMode:     strings.TrimSpace(request.InputMode),
		LastError:     request.LastError,
		ClearError:    request.ClearError,
		LastSeenAt:    request.LastSeenAt,
		LastEventTime: request.LastEventTime,
		LastWriteAt:   request.LastWriteAt,
		EventsWritten: request.EventsWritten,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]bool{"accepted": true})
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func Status(lastSeenAt time.Time, now time.Time) string {
	if lastSeenAt.IsZero() || now.Sub(lastSeenAt) > 2*time.Minute {
		return "offline"
	}
	return "online"
}

func Enrich(item Heartbeat, now time.Time) Heartbeat {
	item.Status = Status(item.LastSeenAt, now)
	if item.InputMode == "" {
		item.InputMode = "file"
	}
	item.HealthLevel = "healthy"
	item.Message = "采集正常"
	item.HeartbeatLagSeconds = lagSeconds(item.LastSeenAt, now)
	if item.LastEventTime != nil {
		item.EventLagSeconds = lagSeconds(*item.LastEventTime, now)
	}
	if item.LastWriteAt != nil {
		item.WriteLagSeconds = lagSeconds(*item.LastWriteAt, now)
	}

	if item.Status == "offline" {
		item.HealthLevel = "critical"
		item.Message = "Collector 心跳超时"
		return item
	}
	if item.LastError != "" {
		item.HealthLevel = "warning"
		item.Message = item.LastError
		return item
	}
	if item.LastEventTime == nil {
		item.HealthLevel = "warning"
		item.Message = "尚未收到采集事件"
		return item
	}
	if item.EventLagSeconds > int64((10 * time.Minute).Seconds()) {
		item.HealthLevel = "warning"
		item.Message = "长时间未收到事件"
		return item
	}
	if item.LastWriteAt == nil {
		item.HealthLevel = "warning"
		item.Message = "尚未写入事件"
		return item
	}
	if item.WriteLagSeconds > int64((10 * time.Minute).Seconds()) {
		item.HealthLevel = "warning"
		item.Message = "长时间未写入事件"
	}
	return item
}

func lagSeconds(value time.Time, now time.Time) int64 {
	if value.IsZero() || now.Before(value) {
		return 0
	}
	return int64(now.Sub(value).Seconds())
}
