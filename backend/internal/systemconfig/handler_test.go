package systemconfig

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerReturnsCollectorFilterConfig(t *testing.T) {
	repository := NewMemoryRepository()
	_ = repository.SaveCollectorFilter(t.Context(), CollectorFilterConfig{
		Enabled:               true,
		IgnoreProcessNames:    []string{"node_exporter"},
		IgnoreCommandKeywords: []string{"/metrics"},
		IgnoreUsers:           []string{"prometheus"},
		KeepSeverities:        []string{"high", "critical"},
	})
	handler := NewHandler(repository)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system-configs/collector-filter", nil)
	rec := httptest.NewRecorder()
	handler.GetCollectorFilter(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response CollectorFilterConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Enabled || response.IgnoreProcessNames[0] != "node_exporter" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestHandlerSavesCollectorFilterConfig(t *testing.T) {
	repository := NewMemoryRepository()
	handler := NewHandler(repository)
	body := bytes.NewBufferString(`{"enabled":true,"ignoreProcessNames":["node_exporter"],"ignoreCommandKeywords":["/metrics"],"ignoreUsers":["prometheus"],"keepSeverities":["high","critical"]}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/system-configs/collector-filter", body)
	rec := httptest.NewRecorder()
	handler.SaveCollectorFilter(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	config, err := repository.GetCollectorFilter(t.Context())
	if err != nil {
		t.Fatalf("GetCollectorFilter returned error: %v", err)
	}
	if !config.Enabled || config.IgnoreUsers[0] != "prometheus" {
		t.Fatalf("unexpected saved config: %#v", config)
	}
}

func TestHandlerValidatesCollectorFilterSeverity(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"enabled":true,"keepSeverities":["urgent"]}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/system-configs/collector-filter", body)
	rec := httptest.NewRecorder()
	handler.SaveCollectorFilter(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
