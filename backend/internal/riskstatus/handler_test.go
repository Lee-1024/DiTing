package riskstatus

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"diting/backend/internal/systemconfig"
)

func TestHandlerAddsCollectorFilterRuleForIgnoredSimilarEvent(t *testing.T) {
	dispositions := &fakeDispositionRepository{}
	collectorFilter := systemconfig.NewMemoryRepository()
	handler := NewHandler(dispositions)
	handler.SetCollectorFilterRepository(collectorFilter)

	body := bytes.NewBufferString(`{
		"status":"ignore_similar",
		"note":"无害监控命令",
		"scope":"similar",
		"fingerprint":"fp-1",
		"event":{
			"eventType":"process_exec",
			"severity":"high",
			"username":"deploy",
			"loginUsername":"operator",
			"processName":"bash",
			"cmdline":"bash -c /opt/app/healthcheck",
			"ruleIds":["rule-1"]
		}
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/risk-dispositions/event-1", body)
	req.SetPathValue("event_id", "event-1")
	rec := httptest.NewRecorder()

	handler.Upsert(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	config, err := collectorFilter.GetCollectorFilter(t.Context())
	if err != nil {
		t.Fatalf("GetCollectorFilter returned error: %v", err)
	}
	data, _ := json.Marshal(config)
	if !bytes.Contains(data, []byte("risk-ignore-similar-")) || !bytes.Contains(data, []byte("bash -c /opt/app/healthcheck")) {
		t.Fatalf("expected ignored similar collector rule, got %s", data)
	}
}

type fakeDispositionRepository struct {
	disposition Disposition
}

func (f *fakeDispositionRepository) List(_ context.Context, _ string, _ int) ([]Disposition, error) {
	return []Disposition{}, nil
}

func (f *fakeDispositionRepository) ListByEventIDs(_ context.Context, _ []string) (map[string]Disposition, error) {
	return map[string]Disposition{}, nil
}

func (f *fakeDispositionRepository) ListByFingerprints(_ context.Context, _ []string) (map[string]Disposition, error) {
	return map[string]Disposition{}, nil
}

func (f *fakeDispositionRepository) Upsert(_ context.Context, disposition Disposition) (Disposition, error) {
	now := time.Now().UTC()
	disposition.CreatedAt = now
	disposition.UpdatedAt = now
	f.disposition = disposition
	return disposition, nil
}
