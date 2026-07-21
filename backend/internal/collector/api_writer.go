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
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string][]audit.Event{"events": events}); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, &body)
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
