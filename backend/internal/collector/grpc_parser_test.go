package collector

import (
	"testing"
	"time"

	tetragon "github.com/cilium/tetragon/api/v1/tetragon"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestParseTetragonGRPCProcessExec(t *testing.T) {
	eventTime := time.Date(2026, 7, 17, 6, 30, 0, 0, time.UTC)
	event, err := ParseTetragonGRPCEvent(&tetragon.GetEventsResponse{
		NodeName: "node-1",
		Time:     timestamppb.New(eventTime),
		Event: &tetragon.GetEventsResponse_ProcessExec{ProcessExec: &tetragon.ProcessExec{
			Process: &tetragon.Process{
				Pid:       wrapperspb.UInt32(1234),
				Uid:       wrapperspb.UInt32(1000),
				Auid:      wrapperspb.UInt32(1000),
				Binary:    "/usr/bin/whoami",
				Arguments: "--version",
				Cwd:       "/home/ubuntu",
				ProcessCredentials: &tetragon.ProcessCredentials{
					Gid:  wrapperspb.UInt32(1000),
					Euid: wrapperspb.UInt32(1000),
					Egid: wrapperspb.UInt32(1000),
				},
				Pod: &tetragon.Pod{
					Namespace: "default",
					Name:      "demo",
					Container: &tetragon.Container{
						Id:    "container-1",
						Name:  "main",
						Image: &tetragon.Image{Name: "ubuntu:22.04"},
					},
				},
			},
			Parent: &tetragon.Process{
				Pid:       wrapperspb.UInt32(1200),
				Binary:    "/bin/bash",
				Arguments: "-l",
			},
		}},
	})
	if err != nil {
		t.Fatalf("ParseTetragonGRPCEvent returned error: %v", err)
	}

	if event.EventType != "process_exec" || event.Action != "exec" {
		t.Fatalf("expected process_exec event, got type=%s action=%s", event.EventType, event.Action)
	}
	if event.EventTime != eventTime || event.NodeName != "node-1" {
		t.Fatalf("unexpected event identity: time=%s node=%s", event.EventTime, event.NodeName)
	}
	if event.ProcessName != "whoami" || event.Cmdline != "/usr/bin/whoami --version" {
		t.Fatalf("unexpected command process=%s cmdline=%s", event.ProcessName, event.Cmdline)
	}
	if event.ParentProcessName != "bash" || event.ParentCmdline != "/bin/bash -l" {
		t.Fatalf("unexpected parent process=%s cmdline=%s", event.ParentProcessName, event.ParentCmdline)
	}
	if event.Namespace != "default" || event.PodName != "demo" || event.Image != "ubuntu:22.04" {
		t.Fatalf("unexpected pod context namespace=%s pod=%s image=%s", event.Namespace, event.PodName, event.Image)
	}
	if event.UID != 1000 || event.GID != 1000 || event.AUID != 1000 || event.EUID != 1000 || event.EGID != 1000 {
		t.Fatalf("unexpected credentials uid=%d gid=%d auid=%d euid=%d egid=%d", event.UID, event.GID, event.AUID, event.EUID, event.EGID)
	}
	if event.RawEvent == "" || event.EventID == "" {
		t.Fatal("expected raw event and stable event id to be set")
	}
}

func TestParseTetragonGRPCUnsupportedEvent(t *testing.T) {
	if _, err := ParseTetragonGRPCEvent(&tetragon.GetEventsResponse{}); err != ErrUnsupportedEvent {
		t.Fatalf("expected unsupported event error, got %v", err)
	}
}
