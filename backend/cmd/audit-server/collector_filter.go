package main

import (
	"fmt"
	"regexp"
	"strings"

	"diting/backend/internal/audit"
	"diting/backend/internal/systemconfig"
)

type collectorNoiseFilter struct {
	Enabled               bool
	IgnoreProcessNames    []string
	IgnoreCommandKeywords []string
	IgnoreUsers           []string
	KeepSeverities        []string
	Rules                 []systemconfig.CollectorFilterRule
}

// collectorNoiseFilterFromSystemConfig converts persisted system config into the runtime filter.
func collectorNoiseFilterFromSystemConfig(cfg systemconfig.CollectorFilterConfig) collectorNoiseFilter {
	return collectorNoiseFilter{
		Enabled:               cfg.Enabled,
		IgnoreProcessNames:    cfg.IgnoreProcessNames,
		IgnoreCommandKeywords: cfg.IgnoreCommandKeywords,
		IgnoreUsers:           cfg.IgnoreUsers,
		KeepSeverities:        cfg.KeepSeverities,
		Rules:                 cfg.Rules,
	}
}

// ShouldDrop returns true when the event matches any enabled collector filter rule.
func (f collectorNoiseFilter) ShouldDrop(event audit.Event) bool {
	if !f.Enabled || f.hasAuditRuleHit(event) || f.shouldKeep(event) {
		return false
	}
	return f.shouldDropByRules(event)
}

// ShouldDropBeforeEnrichment drops explicit platform self-noise before audit rules can
// promote it to a protected severity.
func (f collectorNoiseFilter) ShouldDropBeforeEnrichment(event audit.Event) bool {
	if !f.Enabled || f.shouldKeep(event) {
		return false
	}
	if f.shouldDropByLegacyFields(event) {
		return false
	}
	for _, rule := range f.Rules {
		if rule.ID == "pre-diting-self-vite-noise" && f.ruleMatches(rule, event) {
			return true
		}
	}
	return false
}

func (f collectorNoiseFilter) hasAuditRuleHit(event audit.Event) bool {
	return len(event.RuleIDs) > 0 || len(event.RuleNames) > 0 || len(event.RuleMatches) > 0
}

func (f collectorNoiseFilter) shouldDropByRules(event audit.Event) bool {
	if f.shouldDropByLegacyFields(event) {
		return true
	}
	for _, rule := range f.Rules {
		if f.ruleMatches(rule, event) {
			return true
		}
	}
	return false
}

func (f collectorNoiseFilter) shouldDropByLegacyFields(event audit.Event) bool {
	if containsFold(f.IgnoreProcessNames, event.ProcessName) {
		return true
	}
	if containsFold(f.IgnoreUsers, event.Username) || containsFold(f.IgnoreUsers, event.LoginUsername) {
		return true
	}
	cmdline := strings.ToLower(event.Cmdline)
	for _, keyword := range f.IgnoreCommandKeywords {
		if keyword != "" && strings.Contains(cmdline, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (f collectorNoiseFilter) ruleMatches(rule systemconfig.CollectorFilterRule, event audit.Event) bool {
	if !rule.Enabled || len(rule.Conditions) == 0 {
		return false
	}
	for _, condition := range rule.Conditions {
		if !conditionMatches(condition, event) {
			return false
		}
	}
	return true
}

func conditionMatches(condition systemconfig.CollectorFilterCondition, event audit.Event) bool {
	actual := collectorFilterFieldValue(event, condition.Field)
	switch condition.Op {
	case "eq":
		return strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(condition.Value))
	case "contains":
		return strings.Contains(strings.ToLower(actual), strings.ToLower(strings.TrimSpace(condition.Value)))
	case "prefix":
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(actual)), strings.ToLower(strings.TrimSpace(condition.Value)))
	case "regex":
		matched, err := regexp.MatchString(condition.Value, actual)
		return err == nil && matched
	case "in":
		return containsFold(condition.Values, actual)
	default:
		return false
	}
}

func collectorFilterFieldValue(event audit.Event, field string) string {
	switch field {
	case "event_type":
		return event.EventType
	case "severity":
		return event.Severity
	case "process_name":
		return event.ProcessName
	case "cmdline":
		return event.Cmdline
	case "parent_process_name":
		return event.ParentProcessName
	case "username":
		return event.Username
	case "login_username":
		return event.LoginUsername
	case "file_path":
		return event.FilePath
	case "file_operation":
		return event.FileOperation
	case "dst_ip":
		return event.DstIP
	case "dst_port":
		if event.DstPort == 0 {
			return ""
		}
		return fmt.Sprintf("%d", event.DstPort)
	case "protocol":
		return event.Protocol
	case "domain":
		return event.Domain
	default:
		return ""
	}
}

// shouldKeep preserves protected severities even when a drop rule matches.
func (f collectorNoiseFilter) shouldKeep(event audit.Event) bool {
	keepSeverities := f.KeepSeverities
	if len(keepSeverities) == 0 {
		keepSeverities = []string{"high", "critical"}
	}
	return containsFold(keepSeverities, event.Severity)
}

// containsFold checks case-insensitive membership after trimming whitespace.
func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
