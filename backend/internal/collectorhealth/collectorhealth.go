package collectorhealth

import (
	"context"
	"encoding/json"
	"net/http"
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
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
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
