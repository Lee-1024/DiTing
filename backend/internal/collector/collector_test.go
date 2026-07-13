package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"diting/backend/internal/audit"
)

type fakeWriter struct {
	events []audit.Event
}

func (f *fakeWriter) Write(_ context.Context, events []audit.Event) error {
	f.events = append(f.events, events...)
	return nil
}

func TestFileCollectorParsesJSONLinesAndFlushes(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "tetragon.jsonl")
	line := `{"time":"2026-07-09T13:00:00Z","process_exec":{"process":{"exec_id":"evt-1","pid":1,"binary":"/usr/bin/id","arguments":"","pod":{}},"parent":{"pid":0}},"node_name":"node-1"}`
	content := line + "\n" + line + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	writer := &fakeWriter{}
	collector := NewFileCollector(filePath, 2, writer)
	if err := collector.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if len(writer.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(writer.events))
	}
}

func TestFileCollectorSkipsUnsupportedEvents(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "tetragon.jsonl")
	supported := `{"time":"2026-07-09T13:00:00Z","process_exec":{"process":{"exec_id":"evt-1","pid":1,"binary":"/usr/bin/id","arguments":"","pod":{}},"parent":{"pid":0}},"node_name":"node-1"}`
	unsupported := `{"unknown_event":{"value":true},"time":"2026-07-09T13:00:01Z"}`
	if err := os.WriteFile(filePath, []byte(unsupported+"\n"+supported+"\n"), 0644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	writer := &fakeWriter{}
	collector := NewFileCollector(filePath, 10, writer)
	if err := collector.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if len(writer.events) != 1 {
		t.Fatalf("expected 1 supported event, got %d", len(writer.events))
	}
}

func TestFileCollectorTailReadsOnlyAppendedLines(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "tetragon.jsonl")
	first := `{"time":"2026-07-09T13:00:00Z","process_exec":{"process":{"exec_id":"evt-1","pid":1,"binary":"/usr/bin/id","arguments":"","pod":{}},"parent":{"pid":0}},"node_name":"node-1"}`
	second := `{"time":"2026-07-09T13:00:01Z","process_exec":{"process":{"exec_id":"evt-2","pid":2,"binary":"/usr/bin/whoami","arguments":"","pod":{}},"parent":{"pid":0}},"node_name":"node-1"}`
	if err := os.WriteFile(filePath, []byte(first+"\n"), 0644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	writer := &fakeWriter{}
	collector := NewFileCollector(filePath, 1, writer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- collector.Tail(ctx, 10*time.Millisecond)
	}()

	waitForNoEvents(t, writer)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open sample file: %v", err)
	}
	if _, err := file.WriteString(second + "\n"); err != nil {
		t.Fatalf("append sample file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close sample file: %v", err)
	}

	waitForEvents(t, writer, 1)
	if writer.events[0].ProcessName != "whoami" {
		t.Fatalf("expected only appended whoami event, got %q", writer.events[0].ProcessName)
	}
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Tail returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("tail did not stop after context cancellation")
	}
}

func waitForNoEvents(t *testing.T, writer *fakeWriter) {
	t.Helper()
	time.Sleep(50 * time.Millisecond)
	if len(writer.events) != 0 {
		t.Fatalf("expected no existing events to be tailed, got %d", len(writer.events))
	}
}

func waitForEvents(t *testing.T, writer *fakeWriter, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(writer.events) >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected at least %d events, got %d", count, len(writer.events))
}
