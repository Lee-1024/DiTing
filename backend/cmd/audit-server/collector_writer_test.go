package main

import (
	"context"
	"testing"

	"diting/backend/internal/audit"
	"diting/backend/internal/rule"
)

type fakeEventSink struct {
	events []audit.Event
}

func (f *fakeEventSink) WriteEvents(_ context.Context, events []audit.Event) error {
	f.events = append(f.events, events...)
	return nil
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
