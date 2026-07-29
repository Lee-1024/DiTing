package main

import (
	"context"
	"testing"
	"time"

	"diting/backend/internal/audit"
	"diting/backend/internal/rule"
	"diting/backend/internal/systemconfig"
)

type fakeEventSink struct {
	events []audit.Event
}

func TestDedupingEventWriterDropsRepeatedEventsAcrossWrites(t *testing.T) {
	sink := &fakeCollectorEventWriter{}
	writer := newDedupingEventWriter(sink, 5*time.Second)
	eventTime := time.Date(2026, 7, 22, 14, 19, 8, int(100*time.Millisecond), time.UTC)
	event := audit.Event{
		EventTime:     eventTime,
		EventType:     "file_access",
		Action:        "sys_chmod",
		HostID:        "host-1",
		LoginUsername: "ubuntu",
		Username:      "ubuntu",
		ProcessName:   "chmod",
		Cmdline:       "/usr/bin/chmod 777 test",
		FilePath:      "test",
		FileOperation: "sys_chmod",
	}

	if err := writer.Write(context.Background(), []audit.Event{event}); err != nil {
		t.Fatalf("first Write returned error: %v", err)
	}
	event.EventID = "another-raw-event-id"
	event.EventTime = eventTime.Add(400 * time.Millisecond)
	if err := writer.Write(context.Background(), []audit.Event{event}); err != nil {
		t.Fatalf("second Write returned error: %v", err)
	}

	if len(sink.events) != 1 {
		t.Fatalf("expected duplicate event to be dropped across writes, got %d events", len(sink.events))
	}
}

type fakeCollectorEventWriter struct {
	events []audit.Event
}

func (f *fakeCollectorEventWriter) Write(_ context.Context, events []audit.Event) error {
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeEventSink) WriteEvents(_ context.Context, events []audit.Event) error {
	f.events = append(f.events, events...)
	return nil
}

type fakeCollectorFilterProvider struct {
	calls int
	sets  []systemconfig.CollectorFilterConfig
}

func (f *fakeCollectorFilterProvider) GetCollectorFilter(_ context.Context) (systemconfig.CollectorFilterConfig, error) {
	index := f.calls
	if index >= len(f.sets) {
		index = len(f.sets) - 1
	}
	f.calls++
	return f.sets[index], nil
}

func TestRuleApplyingWriterEnrichesEventsBeforeWrite(t *testing.T) {
	sink := &fakeEventSink{}
	writer := newRefreshingRuleWriter(sink, &fakeRuleProvider{sets: [][]rule.Rule{{{
		ID:        "rule-1",
		Name:      "reverse shell",
		EventType: "process_exec",
		Enabled:   true,
		Severity:  "high",
		RiskScore: 85,
		MatchExpr: rule.Expression{
			Operator:   "and",
			Conditions: []rule.Condition{{Field: "cmdline", Op: "contains", Value: "bash -i"}},
		},
		Tags: []string{"reverse-shell"},
	}}}})
	if err := writer.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	err := writer.Write(context.Background(), []audit.Event{{EventType: "process_exec", Cmdline: "bash -i", Severity: "info"}})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if len(sink.events) != 1 {
		t.Fatalf("expected one written event, got %d", len(sink.events))
	}
	if sink.events[0].Severity != "high" {
		t.Fatalf("expected enriched severity high, got %q", sink.events[0].Severity)
	}
	if len(sink.events[0].RuleIDs) != 1 || sink.events[0].RuleIDs[0] != "rule-1" {
		t.Fatalf("expected rule hit, got %#v", sink.events[0].RuleIDs)
	}
}

func TestRuleApplyingWriterFiltersNoiseAfterRuleEnrichment(t *testing.T) {
	sink := &fakeEventSink{}
	writer := newRefreshingRuleWriter(sink, &fakeRuleProvider{sets: [][]rule.Rule{{{
		ID:        "rule-1",
		Name:      "reverse shell",
		EventType: "process_exec",
		Enabled:   true,
		Severity:  "high",
		RiskScore: 85,
		MatchExpr: rule.Expression{
			Operator:   "and",
			Conditions: []rule.Condition{{Field: "cmdline", Op: "contains", Value: "bash -i"}},
		},
	}}}})
	writer.SetNoiseFilter(collectorNoiseFilter{
		Enabled:               true,
		IgnoreProcessNames:    []string{"bash"},
		IgnoreCommandKeywords: []string{"healthcheck"},
		IgnoreUsers:           []string{"prometheus"},
		KeepSeverities:        []string{"high", "critical"},
	})
	if err := writer.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	err := writer.Write(context.Background(), []audit.Event{
		{EventType: "process_exec", ProcessName: "curl", Cmdline: "curl http://127.0.0.1/healthcheck", Username: "app", Severity: "info"},
		{EventType: "process_exec", ProcessName: "node_exporter", Cmdline: "node_exporter", Username: "prometheus", Severity: "info"},
		{EventType: "process_exec", ProcessName: "bash", Cmdline: "bash -i", Username: "root", Severity: "info"},
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if len(sink.events) != 1 {
		t.Fatalf("expected only high-risk event to be written, got %d", len(sink.events))
	}
	if sink.events[0].Severity != "high" {
		t.Fatalf("expected high-risk event to be preserved after enrichment, got %q", sink.events[0].Severity)
	}
	if sink.events[0].Cmdline != "bash -i" {
		t.Fatalf("expected reverse shell event to be preserved, got %q", sink.events[0].Cmdline)
	}
}

func TestCollectorNoiseFilterUsesSystemConfig(t *testing.T) {
	filter := collectorNoiseFilterFromSystemConfig(systemconfig.CollectorFilterConfig{
		Enabled:               true,
		IgnoreProcessNames:    []string{"kube-probe"},
		IgnoreCommandKeywords: []string{"healthcheck"},
		IgnoreUsers:           []string{"prometheus"},
		KeepSeverities:        []string{"high", "critical"},
	})

	if !filter.Enabled {
		t.Fatalf("expected filter to be enabled")
	}
	if len(filter.IgnoreProcessNames) != 1 || filter.IgnoreProcessNames[0] != "kube-probe" {
		t.Fatalf("unexpected ignored process names: %#v", filter.IgnoreProcessNames)
	}
	if len(filter.IgnoreCommandKeywords) != 1 || filter.IgnoreCommandKeywords[0] != "healthcheck" {
		t.Fatalf("unexpected ignored command keywords: %#v", filter.IgnoreCommandKeywords)
	}
	if len(filter.IgnoreUsers) != 1 || filter.IgnoreUsers[0] != "prometheus" {
		t.Fatalf("unexpected ignored users: %#v", filter.IgnoreUsers)
	}
	if len(filter.KeepSeverities) != 2 || filter.KeepSeverities[0] != "high" || filter.KeepSeverities[1] != "critical" {
		t.Fatalf("unexpected kept severities: %#v", filter.KeepSeverities)
	}
}

func TestCollectorNoiseFilterDropsWhenAllRuleConditionsMatch(t *testing.T) {
	filter := collectorNoiseFilterFromSystemConfig(systemconfig.CollectorFilterConfig{
		Enabled:        true,
		KeepSeverities: []string{"high", "critical"},
		Rules: []systemconfig.CollectorFilterRule{{
			ID:      "rule-1",
			Name:    "ignore vite node",
			Enabled: true,
			Conditions: []systemconfig.CollectorFilterCondition{
				{Field: "process_name", Op: "eq", Value: "node"},
				{Field: "cmdline", Op: "contains", Value: "node_modules/.bin/vite"},
			},
		}},
	})

	if filter.ShouldDrop(audit.Event{ProcessName: "node", Cmdline: "/data/app/node_modules/.bin/vite --host", Severity: "info"}) != true {
		t.Fatalf("expected event matching all conditions to be dropped")
	}
	if filter.ShouldDrop(audit.Event{ProcessName: "node", Cmdline: "/usr/bin/node server.js", Severity: "info"}) != false {
		t.Fatalf("expected event missing one condition to be kept")
	}
}

func TestCollectorNoiseFilterDropsWhenAnyRuleMatches(t *testing.T) {
	filter := collectorNoiseFilterFromSystemConfig(systemconfig.CollectorFilterConfig{
		Enabled:        true,
		KeepSeverities: []string{"high", "critical"},
		Rules: []systemconfig.CollectorFilterRule{
			{
				ID:      "rule-1",
				Name:    "ignore kube probe",
				Enabled: true,
				Conditions: []systemconfig.CollectorFilterCondition{
					{Field: "process_name", Op: "eq", Value: "kube-probe"},
				},
			},
			{
				ID:      "rule-2",
				Name:    "ignore nobody low severity",
				Enabled: true,
				Conditions: []systemconfig.CollectorFilterCondition{
					{Field: "username", Op: "eq", Value: "nobody"},
					{Field: "severity", Op: "in", Values: []string{"info", "low"}},
				},
			},
		},
	})

	if !filter.ShouldDrop(audit.Event{ProcessName: "bash", Username: "nobody", Severity: "low"}) {
		t.Fatalf("expected second matching rule to drop event")
	}
	if filter.ShouldDrop(audit.Event{ProcessName: "bash", Username: "nobody", Severity: "medium"}) {
		t.Fatalf("expected non-matching rule conditions to keep event")
	}
}

func TestCollectorNoiseFilterPreservesKeptSeverityEvenWhenRuleMatches(t *testing.T) {
	filter := collectorNoiseFilterFromSystemConfig(systemconfig.CollectorFilterConfig{
		Enabled:        true,
		KeepSeverities: []string{"high", "critical"},
		Rules: []systemconfig.CollectorFilterRule{{
			ID:      "rule-1",
			Name:    "ignore node",
			Enabled: true,
			Conditions: []systemconfig.CollectorFilterCondition{
				{Field: "process_name", Op: "eq", Value: "node"},
			},
		}},
	})

	if filter.ShouldDrop(audit.Event{ProcessName: "node", Severity: "high"}) {
		t.Fatalf("expected kept severity to bypass collector filter")
	}
}

func TestCollectorNoiseFilterIgnoresDisabledRule(t *testing.T) {
	filter := collectorNoiseFilterFromSystemConfig(systemconfig.CollectorFilterConfig{
		Enabled:        true,
		KeepSeverities: []string{"high", "critical"},
		Rules: []systemconfig.CollectorFilterRule{{
			ID:      "rule-1",
			Name:    "disabled node rule",
			Enabled: false,
			Conditions: []systemconfig.CollectorFilterCondition{
				{Field: "process_name", Op: "eq", Value: "node"},
			},
		}},
	})

	if filter.ShouldDrop(audit.Event{ProcessName: "node", Severity: "info"}) {
		t.Fatalf("expected disabled rule to keep event")
	}
}

func TestCollectorNoiseFilterSupportsPreReleaseRootBaseline(t *testing.T) {
	filter := collectorNoiseFilterFromSystemConfig(systemconfig.PreReleaseCollectorFilterConfig())

	if !filter.ShouldDrop(audit.Event{EventType: "process_exec", Username: "root", LoginUsername: "root", ProcessName: "ls", Cmdline: "ls /var/log", Severity: "info"}) {
		t.Fatalf("expected root low-risk process exec to be dropped")
	}
	if filter.ShouldDrop(audit.Event{EventType: "file_access", Username: "root", LoginUsername: "root", ProcessName: "vim", FilePath: "/etc/shadow", FileOperation: "write", Severity: "critical"}) {
		t.Fatalf("expected critical root sensitive-file event to be preserved")
	}
	if filter.ShouldDrop(audit.Event{EventType: "process_exec", Username: "ubuntu", LoginUsername: "ubuntu", ProcessName: "ls", Cmdline: "ls /var/log", Severity: "info"}) {
		t.Fatalf("expected normal-user process exec to be preserved")
	}
	if !filter.ShouldDrop(audit.Event{EventType: "file_access", Username: "ubuntu", LoginUsername: "ubuntu", FilePath: "/proc/123/status", FileOperation: "open", Severity: "info"}) {
		t.Fatalf("expected normal-user proc read noise to be dropped")
	}
	if !filter.ShouldDrop(audit.Event{EventType: "process_exec", Username: "diting", ProcessName: "node", Cmdline: "/usr/local/bin/node /data/DiTing/frontend/node_modules/.bin/vite --host 0.0.0.0 --port 5174 --strictPort", Severity: "info"}) {
		t.Fatalf("expected DiTing vite process noise to be dropped")
	}
	if !filter.ShouldDrop(audit.Event{EventType: "network_connect", Username: "diting", ProcessName: "node", Cmdline: "/usr/local/bin/node /data/DiTing/frontend/node_modules/.bin/vite --host 0.0.0.0 --port 5174 --strictPort", DstIP: "127.0.0.1", DstPort: 5174, Protocol: "tcp", Severity: "info"}) {
		t.Fatalf("expected DiTing vite network noise to be dropped")
	}
	if !filter.ShouldDrop(audit.Event{EventType: "file_access", Username: "diting", ProcessName: "node", Cmdline: "/usr/local/bin/node /data/DiTing/frontend/node_modules/.bin/vite --host 0.0.0.0 --port 5174 --strictPort", FilePath: "/data/DiTing/frontend/src/main.tsx", FileOperation: "open", Severity: "info"}) {
		t.Fatalf("expected DiTing vite file access noise to be dropped")
	}
	if filter.ShouldDrop(audit.Event{EventType: "network_connect", Username: "diting", ProcessName: "node", Cmdline: "/usr/local/bin/node /data/DiTing/frontend/node_modules/.bin/vite --host 0.0.0.0 --port 5174 --strictPort", Severity: "high"}) {
		t.Fatalf("expected high severity DiTing vite event to be preserved")
	}
}

func TestCollectorNoiseFilterSupportsExtendedFieldsAndOperators(t *testing.T) {
	filter := collectorNoiseFilterFromSystemConfig(systemconfig.CollectorFilterConfig{
		Enabled:        true,
		KeepSeverities: []string{"high", "critical"},
		Rules: []systemconfig.CollectorFilterRule{{
			ID:      "proc-read",
			Name:    "ignore proc reads",
			Enabled: true,
			Conditions: []systemconfig.CollectorFilterCondition{
				{Field: "event_type", Op: "eq", Value: "file_access"},
				{Field: "file_path", Op: "prefix", Value: "/proc/"},
				{Field: "file_operation", Op: "regex", Value: "(?i)open|read"},
			},
		}},
	})

	if !filter.ShouldDrop(audit.Event{EventType: "file_access", FilePath: "/proc/123/status", FileOperation: "security_file_open", Severity: "info"}) {
		t.Fatalf("expected prefix and regex conditions to drop proc read")
	}
	if filter.ShouldDrop(audit.Event{EventType: "file_access", FilePath: "/etc/shadow", FileOperation: "security_file_open", Severity: "info"}) {
		t.Fatalf("expected non-matching prefix to keep event")
	}
}

func TestRuleApplyingWriterRefreshesCollectorFilterFromProvider(t *testing.T) {
	sink := &fakeEventSink{}
	provider := &fakeCollectorFilterProvider{sets: []systemconfig.CollectorFilterConfig{
		{
			Enabled:               true,
			IgnoreCommandKeywords: []string{"healthcheck"},
			KeepSeverities:        []string{"high", "critical"},
		},
		{
			Enabled:        false,
			KeepSeverities: []string{"high", "critical"},
		},
	}}
	writer := newRefreshingRuleWriter(sink, &fakeRuleProvider{sets: [][]rule.Rule{{}}})
	writer.SetCollectorFilterProvider(provider)

	if err := writer.Refresh(context.Background()); err != nil {
		t.Fatalf("first refresh returned error: %v", err)
	}
	if err := writer.Write(context.Background(), []audit.Event{{EventType: "process_exec", Cmdline: "curl /healthcheck", Severity: "info"}}); err != nil {
		t.Fatalf("first write returned error: %v", err)
	}
	if len(sink.events) != 0 {
		t.Fatalf("expected healthcheck event to be filtered, got %d events", len(sink.events))
	}

	if err := writer.Refresh(context.Background()); err != nil {
		t.Fatalf("second refresh returned error: %v", err)
	}
	if err := writer.Write(context.Background(), []audit.Event{{EventType: "process_exec", Cmdline: "curl /healthcheck", Severity: "info"}}); err != nil {
		t.Fatalf("second write returned error: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected healthcheck event to be written after disabling filter, got %d events", len(sink.events))
	}
}
