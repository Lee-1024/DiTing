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
	if !f.Enabled || f.shouldKeep(event) {
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
	if f.isRootRoutineEvent(event) {
		return !isExplicitHighRiskRootEvent(event)
	}
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

func (f collectorNoiseFilter) isRootRoutineEvent(event audit.Event) bool {
	if !strings.EqualFold(strings.TrimSpace(event.Username), "root") {
		return false
	}
	switch event.EventType {
	case "process_exec", "file_access", "network_connect":
	default:
		return false
	}
	severity := strings.ToLower(strings.TrimSpace(event.Severity))
	return severity == "" || severity == "info" || severity == "low" || severity == "medium"
}

func isExplicitHighRiskRootEvent(event audit.Event) bool {
	switch event.EventType {
	case "process_exec":
		return isExplicitHighRiskCommand(event)
	case "file_access":
		return isSensitiveFileMutation(event)
	case "network_connect":
		return isSuspiciousNetworkEvent(event)
	default:
		return false
	}
}

func isExplicitHighRiskCommand(event audit.Event) bool {
	cmdline := strings.ToLower(event.Cmdline)
	highRiskKeywords := []string{
		"bash -i",
		"sh -i",
		"nc -e",
		"ncat -e",
		"/dev/tcp/",
		"socat exec:",
		"| sh",
		"| bash",
		"chmod 777",
		"chmod -r 777",
		"chown root",
	}
	for _, keyword := range highRiskKeywords {
		if strings.Contains(cmdline, keyword) {
			return true
		}
	}
	return false
}

func isSensitiveFileMutation(event audit.Event) bool {
	filePath := event.FilePath
	if filePath == "" {
		return false
	}
	sensitive, err := regexp.MatchString(`(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\.ssh/|^/home/[^/]+/\.ssh/)`, filePath)
	if err != nil || !sensitive {
		return false
	}
	operation := event.FileOperation
	mutating, err := regexp.MatchString(`(?i)(write|truncate|create|creat|open.*wronly|open.*rdwr|unlink|unlinkat|rmdir|chmod|chown|fchmod|fchown|setxattr|removexattr|security_inode_unlink|security_inode_rmdir|security_inode_setattr)`, operation)
	return err == nil && mutating
}

func isSuspiciousNetworkEvent(event audit.Event) bool {
	switch event.DstPort {
	case 4444, 5555, 6666, 7777, 8888, 9999, 31337:
		return true
	default:
		return false
	}
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
