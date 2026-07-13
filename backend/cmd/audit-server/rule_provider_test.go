package main

import (
	"context"
	"testing"

	"diting/backend/internal/audit"
	"diting/backend/internal/rule"
)

type fakeRuleProvider struct {
	calls int
	sets  [][]rule.Rule
}

func (f *fakeRuleProvider) Rules(_ context.Context) ([]rule.Rule, error) {
	index := f.calls
	if index >= len(f.sets) {
		index = len(f.sets) - 1
	}
	f.calls++
	return f.sets[index], nil
}

func TestRefreshingRuleWriterUsesLatestRules(t *testing.T) {
	sink := &fakeEventSink{}
	provider := &fakeRuleProvider{sets: [][]rule.Rule{
		{},
		{{
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
		}},
	}}
	writer := newRefreshingRuleWriter(sink, provider)

	if err := writer.Refresh(context.Background()); err != nil {
		t.Fatalf("first refresh returned error: %v", err)
	}
	if err := writer.Write(context.Background(), []audit.Event{{EventType: "process_exec", Cmdline: "bash -i", Severity: "info"}}); err != nil {
		t.Fatalf("first write returned error: %v", err)
	}
	if sink.events[0].Severity != "info" {
		t.Fatalf("expected first event to remain info, got %q", sink.events[0].Severity)
	}

	if err := writer.Refresh(context.Background()); err != nil {
		t.Fatalf("second refresh returned error: %v", err)
	}
	if err := writer.Write(context.Background(), []audit.Event{{EventType: "process_exec", Cmdline: "bash -i", Severity: "info"}}); err != nil {
		t.Fatalf("second write returned error: %v", err)
	}
	if sink.events[1].Severity != "high" {
		t.Fatalf("expected second event to use refreshed high rule, got %q", sink.events[1].Severity)
	}
}
