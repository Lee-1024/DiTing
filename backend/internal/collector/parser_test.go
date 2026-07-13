package collector

import (
	"os"
	"strings"
	"testing"
)

func TestParseTetragonProcessExecEvent(t *testing.T) {
	data, err := os.ReadFile("../../sample-events/process_exec.jsonl")
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}

	event, err := ParseTetragonEvent([]byte(strings.TrimSpace(string(data))))
	if err != nil {
		t.Fatalf("ParseTetragonEvent returned error: %v", err)
	}

	if event.EventType != "process_exec" {
		t.Fatalf("expected process_exec, got %q", event.EventType)
	}
	if event.EventID == "" {
		t.Fatal("expected event id to be populated")
	}
	if event.EventTime.IsZero() {
		t.Fatal("expected event time to be populated")
	}
	if event.ProcessName != "bash" {
		t.Fatalf("expected process name bash, got %q", event.ProcessName)
	}
	if event.Cmdline != "/usr/bin/bash -c echo tetragon-test" {
		t.Fatalf("unexpected cmdline: %q", event.Cmdline)
	}
	if event.Namespace != "default" {
		t.Fatalf("expected namespace default, got %q", event.Namespace)
	}
	if event.ContainerID != "container-1" {
		t.Fatalf("expected container id container-1, got %q", event.ContainerID)
	}
	if event.HostName != "" {
		t.Fatalf("expected absent host name to default to empty string, got %q", event.HostName)
	}
}

func TestParseTetragonProcessExecEventWithLinuxCredentials(t *testing.T) {
	line := `{"process_exec":{"process":{"exec_id":"evt-1","pid":10,"uid":0,"gid":0,"auid":1000,"binary":"/usr/bin/bash","arguments":"-c id","process_credentials":{"uid":0,"gid":0,"euid":0,"egid":0},"pod":{}},"parent":{"pid":1,"binary":"/usr/sbin/sshd"}},"node_name":"node-1","time":"2026-07-09T07:08:20.928822560Z"}`

	event, err := ParseTetragonEvent([]byte(line))
	if err != nil {
		t.Fatalf("ParseTetragonEvent returned error: %v", err)
	}

	if event.UID != 0 || event.GID != 0 {
		t.Fatalf("expected uid/gid 0/0, got %d/%d", event.UID, event.GID)
	}
	if event.AUID != 1000 {
		t.Fatalf("expected auid 1000, got %d", event.AUID)
	}
	if event.EUID != 0 || event.EGID != 0 {
		t.Fatalf("expected euid/egid 0/0, got %d/%d", event.EUID, event.EGID)
	}
}

func TestParseTetragonProcessExitEvent(t *testing.T) {
	line := `{"process_exit":{"process":{"exec_id":"exit-1","pid":1578945,"uid":0,"binary":"[kworker/13:1-events]","start_time":"2026-07-09T06:46:50.337117152Z"},"parent":{"exec_id":"parent-1","pid":2,"uid":0,"binary":"[kthreadd]"},"time":"2026-07-09T07:08:19.971283237Z"},"node_name":"dd9f5f94c8e2","time":"2026-07-09T07:08:19.971296267Z"}`

	event, err := ParseTetragonEvent([]byte(line))
	if err != nil {
		t.Fatalf("ParseTetragonEvent returned error: %v", err)
	}

	if event.EventType != "process_exit" {
		t.Fatalf("expected process_exit, got %q", event.EventType)
	}
	if event.Action != "exit" {
		t.Fatalf("expected action exit, got %q", event.Action)
	}
	if event.EventID == "" {
		t.Fatal("expected event id to be populated")
	}
	if event.ProcessName != "[kworker/13:1-events]" {
		t.Fatalf("unexpected process name %q", event.ProcessName)
	}
	if event.NodeName != "dd9f5f94c8e2" {
		t.Fatalf("expected node name dd9f5f94c8e2, got %q", event.NodeName)
	}
}

func TestParseTetragonEventIDIsUniquePerRawEvent(t *testing.T) {
	execLine := `{"process_exec":{"process":{"exec_id":"same-exec","pid":10,"binary":"/usr/bin/bash","arguments":"-c id","pod":{}},"parent":{"pid":1}},"node_name":"node-1","time":"2026-07-09T07:08:20.928822560Z"}`
	exitLine := `{"process_exit":{"process":{"exec_id":"same-exec","pid":10,"binary":"/usr/bin/bash"},"parent":{"pid":1},"time":"2026-07-09T07:08:21.928822560Z"},"node_name":"node-1","time":"2026-07-09T07:08:21.928822560Z"}`

	execEvent, err := ParseTetragonEvent([]byte(execLine))
	if err != nil {
		t.Fatalf("parse exec event: %v", err)
	}
	exitEvent, err := ParseTetragonEvent([]byte(exitLine))
	if err != nil {
		t.Fatalf("parse exit event: %v", err)
	}

	if execEvent.EventID == exitEvent.EventID {
		t.Fatalf("expected different event ids for exec/exit with same exec_id, got %q", execEvent.EventID)
	}
}
