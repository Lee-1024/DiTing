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
	Field string   `json:"field"`
	Op    string   `json:"op"`
	Value string   `json:"value"`
	Values []string `json:"values"`
}

func Match(expr Expression, event audit.Event) bool {
	if len(expr.Conditions) == 0 {
		return false
	}

	operator := strings.ToLower(expr.Operator)
	if operator == "" {
		operator = "and"
	}

	if operator == "or" {
		for _, condition := range expr.Conditions {
			if matchCondition(condition, event) {
				return true
			}
		}
		return false
	}

	for _, condition := range expr.Conditions {
		if !matchCondition(condition, event) {
			return false
		}
	}
	return true
}

func matchCondition(condition Condition, event audit.Event) bool {
	actual := fieldValue(condition.Field, event)
	switch strings.ToLower(condition.Op) {
	case "eq":
		return actual == condition.Value
	case "neq":
		return actual != condition.Value
	case "contains":
		return strings.Contains(actual, condition.Value)
	case "prefix":
		return strings.HasPrefix(actual, condition.Value)
	case "suffix":
		return strings.HasSuffix(actual, condition.Value)
	case "in":
		for _, value := range condition.Values {
			if actual == value {
				return true
			}
		}
		return false
	case "regex":
		matched, err := regexp.MatchString(condition.Value, actual)
		return err == nil && matched
	default:
		return false
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
