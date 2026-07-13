package rule

import "diting/backend/internal/audit"

func ApplyRules(event audit.Event, rules []Rule) audit.Event {
	for _, candidate := range rules {
		if !candidate.Enabled {
			continue
		}
		if candidate.EventType != "" && candidate.EventType != event.EventType {
			continue
		}
		if !Match(candidate.MatchExpr, event) {
			continue
		}

		event.RuleIDs = appendUnique(event.RuleIDs, candidate.ID)
		event.RuleNames = appendUnique(event.RuleNames, candidate.Name)
		for _, tag := range candidate.Tags {
			event.Tags = appendUnique(event.Tags, tag)
		}
		if candidate.RiskScore > int(event.RiskScore) {
			event.RiskScore = uint8(candidate.RiskScore)
			event.Severity = candidate.Severity
		}
	}
	return event
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
