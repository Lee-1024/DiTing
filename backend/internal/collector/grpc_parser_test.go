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

func TestParseTetragonGRPCProcessKprobeFileEvent(t *testing.T) {
	eventTime := time.Date(2026, 7, 20, 6, 30, 0, 0, time.UTC)
	event, err := ParseTetragonGRPCEvent(&tetragon.GetEventsResponse{
		NodeName: "node-1",
		Time:     timestamppb.New(eventTime),
		Event: &tetragon.GetEventsResponse_ProcessKprobe{ProcessKprobe: &tetragon.ProcessKprobe{
			FunctionName: "security_file_open",
			PolicyName:   "sensitive-files",
			Message:      "sensitive file read",
			Tags:         []string{"file", "sensitive"},
			Process: &tetragon.Process{
				Pid:    wrapperspb.UInt32(2345),
				Binary: "/usr/bin/cat",
			},
			Parent: &tetragon.Process{Binary: "/bin/bash"},
			Args: []*tetragon.KprobeArgument{{
				Label: "file",
				Arg: &tetragon.KprobeArgument_PathArg{PathArg: &tetragon.KprobePath{
					Path:       "/etc/shadow",
					Permission: "read",
				}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("ParseTetragonGRPCEvent returned error: %v", err)
	}

	if event.EventType != "file_access" || event.Action != "security_file_open" {
		t.Fatalf("expected file_access security_file_open, got type=%s action=%s", event.EventType, event.Action)
	}
	if event.FilePath != "/etc/shadow" || event.FileOperation != "read" {
		t.Fatalf("unexpected file context path=%s operation=%s", event.FilePath, event.FileOperation)
	}
	if event.ProcessName != "cat" || event.Tags[0] != "file" {
		t.Fatalf("unexpected process/tags process=%s tags=%#v", event.ProcessName, event.Tags)
	}
}

func TestParseTetragonGRPCProcessKprobeNetworkEvent(t *testing.T) {
	eventTime := time.Date(2026, 7, 20, 6, 30, 0, 0, time.UTC)
	event, err := ParseTetragonGRPCEvent(&tetragon.GetEventsResponse{
		NodeName: "node-1",
		Time:     timestamppb.New(eventTime),
		Event: &tetragon.GetEventsResponse_ProcessKprobe{ProcessKprobe: &tetragon.ProcessKprobe{
			FunctionName: "tcp_connect",
			PolicyName:   "network-connect",
			Process: &tetragon.Process{
				Pid:       wrapperspb.UInt32(2345),
				Binary:    "/usr/bin/curl",
				Arguments: "https://example.com",
			},
			Parent: &tetragon.Process{Binary: "/bin/bash"},
			Args: []*tetragon.KprobeArgument{{
				Label: "addr",
				Arg: &tetragon.KprobeArgument_SockaddrArg{SockaddrArg: &tetragon.KprobeSockaddr{
					Family: "AF_INET",
					Addr:   "93.184.216.34",
					Port:   443,
				}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("ParseTetragonGRPCEvent returned error: %v", err)
	}

	if event.EventType != "network_connect" || event.Action != "tcp_connect" {
		t.Fatalf("expected network_connect tcp_connect, got type=%s action=%s", event.EventType, event.Action)
	}
	if event.DstIP != "93.184.216.34" || event.DstPort != 443 || event.Protocol != "tcp" {
		t.Fatalf("unexpected network context dst=%s:%d protocol=%s", event.DstIP, event.DstPort, event.Protocol)
	}
}

func TestParseTetragonGRPCProcessKprobeIgnoresInvalidNetworkAddress(t *testing.T) {
	event, err := ParseTetragonGRPCEvent(&tetragon.GetEventsResponse{
		NodeName: "node-1",
		Time:     timestamppb.New(time.Date(2026, 7, 20, 6, 30, 0, 0, time.UTC)),
		Event: &tetragon.GetEventsResponse_ProcessKprobe{ProcessKprobe: &tetragon.ProcessKprobe{
			FunctionName: "tcp_connect",
			Process:      &tetragon.Process{Binary: "/usr/bin/curl"},
			Args: []*tetragon.KprobeArgument{{
				Label: "addr",
				Arg: &tetragon.KprobeArgument_SockaddrArg{SockaddrArg: &tetragon.KprobeSockaddr{
					Family: "AF_INET",
					Addr:   "invalid IP",
					Port:   443,
				}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("ParseTetragonGRPCEvent returned error: %v", err)
	}

	if event.EventType == "network_connect" || event.DstIP != "" || event.DstPort != 0 {
		t.Fatalf("expected invalid network address to be ignored, got type=%s dst=%s:%d", event.EventType, event.DstIP, event.DstPort)
	}
}
