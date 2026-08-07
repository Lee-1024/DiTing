package collector

import (
	"strings"
	"testing"
)

func TestGenerateTetragonObserverPolicyIsMonitorOnly(t *testing.T) {
	policy, err := GenerateTetragonObserverPolicy([]string{"/etc/docker/daemon.json"})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"name: diting-apparmor-observer", "call: security_file_open", "return: true", "type: file", "action: Post", "diting-apparmor-observer"} {
		if !strings.Contains(policy, expected) {
			t.Fatalf("expected %q in observer policy:\n%s", expected, policy)
		}
	}
	for _, forbidden := range []string{"Sigkill", "Override"} {
		if strings.Contains(policy, forbidden) {
			t.Fatalf("observer policy must not enforce with %s:\n%s", forbidden, policy)
		}
	}
}

func TestParseTetragonObserverDenialCreatesEnforcementEvent(t *testing.T) {
	line := []byte(`{"process_kprobe":{"function_name":"security_file_open","policy_name":"diting-apparmor-observer","tags":["diting-apparmor-observer"],"args":[{"file_arg":{"path":"/etc/docker/daemon.json"}}],"return":{"int_arg":-13},"process":{"pid":10,"uid":0,"binary":"/usr/bin/vim","arguments":"/etc/docker/daemon.json","process_credentials":{"euid":0},"pod":{}},"parent":{"pid":9,"binary":"/usr/bin/sudo","arguments":"vim /etc/docker/daemon.json"}},"node_name":"node-1","time":"2026-08-07T06:00:00Z"}`)

	event, err := ParseTetragonEvent(line)
	if err != nil {
		t.Fatal(err)
	}
	if event.Severity != "critical" || event.RiskScore != 98 || !containsString(event.Tags, "diting-enforcement") {
		t.Fatalf("expected enforcement event, got severity=%s score=%d tags=%#v", event.Severity, event.RiskScore, event.Tags)
	}
}

func TestParseTetragonObserverSuccessDoesNotCreateEnforcementEvent(t *testing.T) {
	line := []byte(`{"process_kprobe":{"function_name":"security_file_open","policy_name":"diting-apparmor-observer","tags":["diting-apparmor-observer"],"args":[{"file_arg":{"path":"/etc/docker/daemon.json"}}],"return":{"int_arg":0},"process":{"pid":10,"uid":0,"binary":"/usr/bin/vim","process_credentials":{"euid":0},"pod":{}},"parent":{"pid":9}},"node_name":"node-1","time":"2026-08-07T06:00:00Z"}`)

	event, err := ParseTetragonEvent(line)
	if err != nil {
		t.Fatal(err)
	}
	if containsString(event.Tags, "diting-enforcement") || event.Severity == "critical" {
		t.Fatalf("successful access must not be reported as blocked: %#v", event)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
