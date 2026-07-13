package rule

import (
	"encoding/json"
	"testing"
)

func TestRuleJSONRoundTripForPostgres(t *testing.T) {
	rule := Rule{
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
	}

	matchExpr, tags, err := marshalRuleJSON(rule)
	if err != nil {
		t.Fatalf("marshalRuleJSON returned error: %v", err)
	}

	var expr Expression
	if err := json.Unmarshal(matchExpr, &expr); err != nil {
		t.Fatalf("unmarshal expression: %v", err)
	}
	if expr.Conditions[0].Value != "bash -i" {
		t.Fatalf("unexpected condition value %q", expr.Conditions[0].Value)
	}

	var decodedTags []string
	if err := json.Unmarshal(tags, &decodedTags); err != nil {
		t.Fatalf("unmarshal tags: %v", err)
	}
	if decodedTags[0] != "reverse-shell" {
		t.Fatalf("unexpected tag %q", decodedTags[0])
	}
}
