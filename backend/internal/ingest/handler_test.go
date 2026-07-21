package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"diting/backend/internal/audit"
)

type fakeWriter struct {
	events []audit.Event
}

func (f *fakeWriter) WriteEvents(_ context.Context, events []audit.Event) error {
	f.events = append(f.events, events...)
	return nil
}

func TestHandlerRejectsMissingCollectorToken(t *testing.T) {
	writer := &fakeWriter{}
	handler := NewHandler(writer, "secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/events", strings.NewReader(`{"events":[]}`))
	rec := httptest.NewRecorder()

	handler.IngestEvents(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerWritesAuthorizedEvents(t *testing.T) {
	writer := &fakeWriter{}
	handler := NewHandler(writer, "secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/events", strings.NewReader(`{"events":[{"eventId":"evt-1","eventType":"process_exec","severity":"info","processName":"id","cmdline":"/usr/bin/id"}]}`))
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()

	handler.IngestEvents(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(writer.events) != 1 || writer.events[0].EventID != "evt-1" || writer.events[0].EventDate.IsZero() || writer.events[0].IngestTime.IsZero() {
		t.Fatalf("expected normalized event to be written, got %#v", writer.events)
	}
	if !strings.Contains(rec.Body.String(), `"accepted":1`) {
		t.Fatalf("expected accepted count response, got %s", rec.Body.String())
	}
}
