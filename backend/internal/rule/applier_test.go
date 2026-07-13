package rule

import (
	"testing"

	"diting/backend/internal/audit"
)

func TestApplyRulesEnrichesMatchingEvent(t *testing.T) {
	event := audit.Event{
		EventType: "process_exec",
		Cmdline:   "/usr/bin/bash -i",
		Severity:  "info",
		RiskScore: 0,
	}
	rules := []Rule{
		{
			ID:        "rule-1",
			Name:      "reverse shell",
			EventType: "process_exec",
			Enabled:   true,
			Severity:  "high",
			RiskScore: 85,
			MatchExpr: Expression{
				Operator: "and",
				Conditions: []Condition{
					{Field: "cmdline", Op: "contains", Value: "bash -i"},
				},
			},
			Tags: []string{"reverse-shell"},
		},
	}

	enriched := ApplyRules(event, rules)

	if enriched.Severity != "high" {
		t.Fatalf("expected high severity, got %q", enriched.Severity)
	}
	if enriched.RiskScore != 85 {
		t.Fatalf("expected risk score 85, got %d", enriched.RiskScore)
	}
	if len(enriched.Tags) != 1 || enriched.Tags[0] != "reverse-shell" {
		t.Fatalf("expected reverse-shell tag, got %#v", enriched.Tags)
	}
	if len(enriched.RuleIDs) != 1 || enriched.RuleIDs[0] != "rule-1" {
		t.Fatalf("expected rule id rule-1, got %#v", enriched.RuleIDs)
	}
	if len(enriched.RuleNames) != 1 || enriched.RuleNames[0] != "reverse shell" {
		t.Fatalf("expected rule name, got %#v", enriched.RuleNames)
	}
}

func TestApplyRulesIgnoresDisabledRules(t *testing.T) {
	event := audit.Event{EventType: "process_exec", Cmdline: "bash -i", Severity: "info"}
	rules := []Rule{{
		ID:        "rule-1",
		Name:      "disabled",
		EventType: "process_exec",
		Enabled:   false,
		Severity:  "critical",
		RiskScore: 100,
		MatchExpr: Expression{Operator: "and", Conditions: []Condition{{Field: "cmdline", Op: "contains", Value: "bash -i"}}},
	}}

	enriched := ApplyRules(event, rules)

	if enriched.Severity != "info" {
		t.Fatalf("expected original severity info, got %q", enriched.Severity)
	}
	if len(enriched.RuleIDs) != 0 {
		t.Fatalf("expected no rule hits, got %#v", enriched.RuleIDs)
	}
}
