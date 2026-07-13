package clickhouse

import (
	"context"
	"testing"
	"time"

	"diting/backend/internal/audit"
)

type fakeBatch struct {
	rows [][]any
	sent bool
}

func (f *fakeBatch) Append(values ...any) error {
	f.rows = append(f.rows, values)
	return nil
}

func (f *fakeBatch) Send() error {
	f.sent = true
	return nil
}

type fakeBatchPreparer struct {
	batch *fakeBatch
	query string
}

func (f *fakeBatchPreparer) PrepareBatch(_ context.Context, query string) (Batch, error) {
	f.query = query
	return f.batch, nil
}

func TestAuditWriterWritesEventsInOneBatch(t *testing.T) {
	batch := &fakeBatch{}
	preparer := &fakeBatchPreparer{batch: batch}
	writer := NewAuditWriter(preparer)

	events := []audit.Event{
		{
			EventID: "evt-1", EventTime: time.Unix(1, 0), EventDate: time.Unix(0, 0), IngestTime: time.Unix(2, 0),
			EventType: "process_exec", Action: "exec", Severity: "info", ProcessName: "bash", Cmdline: "bash -c id",
		},
		{
			EventID: "evt-2", EventTime: time.Unix(3, 0), EventDate: time.Unix(0, 0), IngestTime: time.Unix(4, 0),
			EventType: "process_exec", Action: "exec", Severity: "high", ProcessName: "nc", Cmdline: "nc -e /bin/sh",
		},
	}

	if err := writer.Write(context.Background(), events); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if preparer.query == "" {
		t.Fatal("expected insert query to be prepared")
	}
	if len(batch.rows) != 2 {
		t.Fatalf("expected 2 appended rows, got %d", len(batch.rows))
	}
	if !batch.sent {
		t.Fatal("expected batch to be sent")
	}
	if batch.rows[0][0] != "evt-1" {
		t.Fatalf("expected first column to be event id, got %v", batch.rows[0][0])
	}
}
