package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"diting/backend/internal/audit"
)

type APIWriter struct {
	url           string
	token         string
	client        *http.Client
	maxAttempts   int
	retryInterval time.Duration
	bufferLimit   int
	mu            sync.Mutex
	buffer        []audit.Event
	droppedEvents uint64
	lastBuffered  bool
}

type APIHeartbeat struct {
	HostID         string     `json:"hostId"`
	HostName       string     `json:"hostName"`
	InputMode      string     `json:"inputMode"`
	LastError      string     `json:"lastError,omitempty"`
	ClearError     bool       `json:"clearError"`
	LastSeenAt     time.Time  `json:"lastSeenAt,omitempty"`
	LastEventTime  *time.Time `json:"lastEventTime,omitempty"`
	LastWriteAt    *time.Time `json:"lastWriteAt,omitempty"`
	EventsWritten  uint64     `json:"eventsWritten"`
	BufferedEvents int        `json:"bufferedEvents"`
	DroppedEvents  uint64     `json:"droppedEvents"`
}

func NewAPIWriter(url string, token string) *APIWriter {
	return &APIWriter{
		url:           strings.TrimSpace(url),
		token:         strings.TrimSpace(token),
		client:        &http.Client{Timeout: 30 * time.Second},
		maxAttempts:   3,
		retryInterval: 2 * time.Second,
		bufferLimit:   1000,
	}
}

func (w *APIWriter) SetRetryPolicy(maxAttempts int, retryInterval time.Duration) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if retryInterval < 0 {
		retryInterval = 0
	}
	w.maxAttempts = maxAttempts
	w.retryInterval = retryInterval
}

func (w *APIWriter) SetBufferLimit(limit int) {
	if limit < 0 {
		limit = 0
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bufferLimit = limit
	w.trimBufferLocked()
}

func (w *APIWriter) BufferedEvents() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.buffer)
}

func (w *APIWriter) DroppedEvents() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.droppedEvents
}

func (w *APIWriter) BufferedEventIDs() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	ids := make([]string, 0, len(w.buffer))
	for _, event := range w.buffer {
		ids = append(ids, event.EventID)
	}
	return ids
}

func (w *APIWriter) LastWriteBuffered() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastBuffered
}

func (w *APIWriter) Write(ctx context.Context, events []audit.Event) error {
	if len(events) == 0 {
		return nil
	}
	w.setLastBuffered(false)
	if err := w.flushBuffered(ctx); err != nil {
		w.enqueueEvents(events)
		w.setLastBuffered(true)
		return nil
	}
	if err := w.postJSON(ctx, w.url, map[string][]audit.Event{"events": events}); err != nil {
		if isRetriableAPIError(err) {
			w.enqueueEvents(events)
			w.setLastBuffered(true)
			return nil
		}
		return err
	}
	return nil
}

func (w *APIWriter) setLastBuffered(value bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastBuffered = value
}

func (w *APIWriter) WriteHeartbeat(ctx context.Context, heartbeat APIHeartbeat) error {
	if heartbeat.LastSeenAt.IsZero() {
		heartbeat.LastSeenAt = time.Now().UTC()
	}
	return w.postJSON(ctx, heartbeatURL(w.url), heartbeat)
}

func (w *APIWriter) postJSON(ctx context.Context, url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	attempts := w.maxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		lastErr = w.doPostJSON(ctx, url, body)
		if lastErr == nil {
			return nil
		}
		if !isRetriableAPIError(lastErr) || attempt == attempts {
			return lastErr
		}
		if err := sleepWithContext(ctx, w.retryInterval); err != nil {
			return err
		}
	}
	return lastErr
}

func heartbeatURL(eventsURL string) string {
	trimmed := strings.TrimSpace(eventsURL)
	if strings.HasSuffix(trimmed, "/events") {
		return strings.TrimSuffix(trimmed, "/events") + "/heartbeat"
	}
	return strings.TrimRight(trimmed, "/") + "/heartbeat"
}

func (w *APIWriter) flushBuffered(ctx context.Context) error {
	for {
		batch := w.nextBufferedBatch()
		if len(batch) == 0 {
			return nil
		}
		if err := w.postJSON(ctx, w.url, map[string][]audit.Event{"events": batch}); err != nil {
			return err
		}
		w.removeBuffered(len(batch))
	}
}

func (w *APIWriter) nextBufferedBatch() []audit.Event {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buffer) == 0 {
		return nil
	}
	batch := make([]audit.Event, len(w.buffer))
	copy(batch, w.buffer)
	return batch
}

func (w *APIWriter) removeBuffered(count int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if count >= len(w.buffer) {
		w.buffer = nil
		return
	}
	w.buffer = append([]audit.Event{}, w.buffer[count:]...)
}

func (w *APIWriter) enqueueEvents(events []audit.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.bufferLimit <= 0 {
		w.droppedEvents += uint64(len(events))
		return
	}
	w.buffer = append(w.buffer, events...)
	w.trimBufferLocked()
}

func (w *APIWriter) trimBufferLocked() {
	for len(w.buffer) > w.bufferLimit {
		index := lowestPriorityEventIndex(w.buffer)
		w.buffer = append(w.buffer[:index], w.buffer[index+1:]...)
		w.droppedEvents++
	}
}

func lowestPriorityEventIndex(events []audit.Event) int {
	index := 0
	priority := severityPriority(events[0].Severity)
	for i := 1; i < len(events); i++ {
		nextPriority := severityPriority(events[i].Severity)
		if nextPriority < priority {
			index = i
			priority = nextPriority
		}
	}
	return index
}

func severityPriority(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium", "warning":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func (w *APIWriter) doPostJSON(ctx context.Context, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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
		return apiStatusError{statusCode: resp.StatusCode}
	}
	return nil
}

type apiStatusError struct {
	statusCode int
}

func (e apiStatusError) Error() string {
	return fmt.Sprintf("ingest api status %d", e.statusCode)
}

func isRetriableAPIError(err error) bool {
	if statusErr, ok := err.(apiStatusError); ok {
		return statusErr.statusCode == http.StatusTooManyRequests || statusErr.statusCode >= 500
	}
	return true
}

func sleepWithContext(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return nil
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
