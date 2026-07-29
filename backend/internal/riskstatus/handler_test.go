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

func TestIgnoredSimilarCollectorRuleNormalizesRuncVolatileCommand(t *testing.T) {
	rule := ignoredSimilarCollectorRule("fp-runc", ignoredSimilarEvent{
		EventType:   "process_exec",
		ProcessName: "runc",
		Cmdline:     "/usr/bin/runc --root /var/run/docker/runtime-runc/moby --log /var/run/docker/containerd/daemon/io.containerd.runtime.v2.task/moby/4799b5c8d705acec661e8d5ec3a86f7d69c67fe96f875bd95a9a12d528679f1c/log.json --log-format json exec --process /tmp/runc-process17961974 --detach --pid-file /var/run/docker/containerd/daemon/io.containerd.runtime.v2.task/moby/4799b5c8d705acec661e8d5ec3a86f7d69c67fe96f875bd95a9a12d528679f1c/4411b8e924f20c62b1d24c68f4dfdfd0f9369a0fc5f690765e0d83cb0abec79f.pid 4799b5c8d705acec661e8d5ec3a86f7d69c67fe96f875bd95a9a12d528679f1c",
	})

	for _, condition := range rule.Conditions {
		if condition.Field == "cmdline" {
			if condition.Op != "regex" {
				t.Fatalf("expected runc cmdline condition to use regex, got %#v", condition)
			}
			if !bytes.Contains([]byte(condition.Value), []byte(`runc\b.*\bexec\b`)) || bytes.Contains([]byte(condition.Value), []byte("4799b5c8")) || bytes.Contains([]byte(condition.Value), []byte("17961974")) {
				t.Fatalf("expected normalized runc regex, got %q", condition.Value)
			}
			return
		}
	}
	t.Fatalf("expected cmdline condition in %#v", rule.Conditions)
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
