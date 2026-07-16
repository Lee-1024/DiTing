package rule

import (
	"regexp"
	"strings"

	"diting/backend/internal/audit"
)

type Expression struct {
	Operator   string      `json:"operator"`
	Conditions []Condition `json:"conditions"`
}

type Condition struct {
	Field  string   `json:"field"`
	Op     string   `json:"op"`
	Value  string   `json:"value"`
	Values []string `json:"values"`
}

func Match(expr Expression, event audit.Event) bool {
	return len(MatchConditions(expr, event)) > 0
}

func MatchConditions(expr Expression, event audit.Event) []audit.RuleMatch {
	if len(expr.Conditions) == 0 {
		return nil
	}

	operator := strings.ToLower(expr.Operator)
	if operator == "" {
		operator = "and"
	}

	if operator == "or" {
		for _, condition := range expr.Conditions {
			if match, actual := matchCondition(condition, event); match {
				return []audit.RuleMatch{conditionMatch(condition, actual)}
			}
		}
		return nil
	}

	matches := make([]audit.RuleMatch, 0, len(expr.Conditions))
	for _, condition := range expr.Conditions {
		match, actual := matchCondition(condition, event)
		if !match {
			return nil
		}
		matches = append(matches, conditionMatch(condition, actual))
	}
	return matches
}

func conditionMatch(condition Condition, actual string) audit.RuleMatch {
	return audit.RuleMatch{
		Field:    condition.Field,
		Operator: condition.Op,
		Value:    conditionValue(condition),
		Actual:   actual,
	}
}

func conditionValue(condition Condition) string {
	if len(condition.Values) > 0 {
		return strings.Join(condition.Values, ",")
	}
	return condition.Value
}

func matchCondition(condition Condition, event audit.Event) (bool, string) {
	actual := fieldValue(condition.Field, event)
	switch strings.ToLower(condition.Op) {
	case "eq":
		return actual == condition.Value, actual
	case "neq":
		return actual != condition.Value, actual
	case "contains":
		return strings.Contains(actual, condition.Value), actual
	case "prefix":
		return strings.HasPrefix(actual, condition.Value), actual
	case "suffix":
		return strings.HasSuffix(actual, condition.Value), actual
	case "in":
		for _, value := range condition.Values {
			if actual == value {
				return true, actual
			}
		}
		return false, actual
	case "regex":
		matched, err := regexp.MatchString(condition.Value, actual)
		return err == nil && matched, actual
	default:
		return false, actual
	}
}

func fieldValue(field string, event audit.Event) string {
	switch field {
	case "event_type":
		return event.EventType
	case "action":
		return event.Action
	case "severity":
		return event.Severity
	case "host_name":
		return event.HostName
	case "namespace":
		return event.Namespace
	case "pod_name":
		return event.PodName
	case "container_id":
		return event.ContainerID
	case "process_name":
		return event.ProcessName
	case "binary_path":
		return event.BinaryPath
	case "cmdline":
		return event.Cmdline
	case "username":
		return event.Username
	case "file_path":
		return event.FilePath
	case "dst_ip":
		return event.DstIP
	case "domain":
		return event.Domain
	default:
		return ""
	}
}
