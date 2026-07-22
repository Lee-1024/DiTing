package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

type APIWriter struct {
	url    string
	token  string
	client *http.Client
}

type APIHeartbeat struct {
	HostID        string     `json:"hostId"`
	HostName      string     `json:"hostName"`
	InputMode     string     `json:"inputMode"`
	LastError     string     `json:"lastError,omitempty"`
	ClearError    bool       `json:"clearError"`
	LastSeenAt    time.Time  `json:"lastSeenAt,omitempty"`
	LastEventTime *time.Time `json:"lastEventTime,omitempty"`
	LastWriteAt   *time.Time `json:"lastWriteAt,omitempty"`
	EventsWritten uint64     `json:"eventsWritten"`
}

func NewAPIWriter(url string, token string) *APIWriter {
	return &APIWriter{
		url:    strings.TrimSpace(url),
		token:  strings.TrimSpace(token),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (w *APIWriter) Write(ctx context.Context, events []audit.Event) error {
	if len(events) == 0 {
		return nil
	}
	return w.postJSON(ctx, w.url, map[string][]audit.Event{"events": events})
}

func (w *APIWriter) WriteHeartbeat(ctx context.Context, heartbeat APIHeartbeat) error {
	if heartbeat.LastSeenAt.IsZero() {
		heartbeat.LastSeenAt = time.Now().UTC()
	}
	return w.postJSON(ctx, heartbeatURL(w.url), heartbeat)
}

func (w *APIWriter) postJSON(ctx context.Context, url string, payload any) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.token != "" {
		req.Header.Set("Authorization", "Bearer "+w.token)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ingest api status %d", resp.StatusCode)
	}
	return nil
}

func heartbeatURL(eventsURL string) string {
	trimmed := strings.TrimSpace(eventsURL)
	if strings.HasSuffix(trimmed, "/events") {
		return strings.TrimSuffix(trimmed, "/events") + "/heartbeat"
	}
	return strings.TrimRight(trimmed, "/") + "/heartbeat"
}
