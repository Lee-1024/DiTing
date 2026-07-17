package collector

import (
	"context"
	"io"
	"testing"
	"time"

	"diting/backend/internal/audit"
	tetragon "github.com/cilium/tetragon/api/v1/tetragon"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGRPCCollectorWritesBatchesFromStream(t *testing.T) {
	writer := &recordingWriter{}
	stream := &fakeEventStream{events: []*tetragon.GetEventsResponse{
		grpcExecResponse("node-1", "/usr/bin/id", ""),
		grpcExecResponse("node-1", "/usr/bin/whoami", ""),
	}}
	collector := NewGRPCCollector("127.0.0.1:54321", 2, writer)
	collector.dial = func(context.Context, string) (eventStream, func() error, error) {
		return stream, func() error { return nil }, nil
	}

	if err := collector.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if len(writer.batches) != 1 || len(writer.batches[0]) != 2 {
		t.Fatalf("expected one batch with two events, got %#v", writer.batches)
	}
}

func TestGRPCCollectorReconnectsAfterStreamError(t *testing.T) {
	writer := &recordingWriter{}
	attempts := 0
	collector := NewGRPCCollector("127.0.0.1:54321", 1, writer)
	collector.reconnectInterval = time.Millisecond
	collector.dial = func(context.Context, string) (eventStream, func() error, error) {
		attempts++
		if attempts == 1 {
			return &fakeEventStream{err: io.ErrUnexpectedEOF}, func() error { return nil }, nil
		}
		return &fakeEventStream{events: []*tetragon.GetEventsResponse{grpcExecResponse("node-1", "/usr/bin/id", "")}}, func() error { return nil }, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	collector.afterWrite = cancel
	if err := collector.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected reconnect after stream error, got attempts=%d", attempts)
	}
	if len(writer.batches) != 1 {
		t.Fatalf("expected one written batch, got %d", len(writer.batches))
	}
}

type fakeEventStream struct {
	events []*tetragon.GetEventsResponse
	err    error
}

func (s *fakeEventStream) Recv() (*tetragon.GetEventsResponse, error) {
	if len(s.events) == 0 {
		if s.err != nil {
			return nil, s.err
		}
		return nil, io.EOF
	}
	next := s.events[0]
	s.events = s.events[1:]
	return next, nil
}

type recordingWriter struct {
	batches [][]audit.Event
}

func (w *recordingWriter) Write(_ context.Context, events []audit.Event) error {
	w.batches = append(w.batches, append([]audit.Event{}, events...))
	return nil
}

func grpcExecResponse(nodeName, binary, arguments string) *tetragon.GetEventsResponse {
	now := time.Date(2026, 7, 17, 7, 0, 0, 0, time.UTC)
	return &tetragon.GetEventsResponse{
		NodeName: nodeName,
		Time:     timestamppb.New(now),
		Event: &tetragon.GetEventsResponse_ProcessExec{ProcessExec: &tetragon.ProcessExec{
			Process: &tetragon.Process{Binary: binary, Arguments: arguments},
			Parent:  &tetragon.Process{Binary: "/bin/bash"},
		}},
	}
}
