package collectorhealth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Heartbeat struct {
	HostID        string     `json:"hostId"`
	HostName      string     `json:"hostName"`
	Status        string     `json:"status"`
	LastSeenAt    time.Time  `json:"lastSeenAt"`
	LastEventTime *time.Time `json:"lastEventTime,omitempty"`
	LastWriteAt   *time.Time `json:"lastWriteAt,omitempty"`
	EventsWritten uint64     `json:"eventsWritten"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type HeartbeatUpdate struct {
	HostID        string
	HostName      string
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
