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
		matches := MatchConditions(candidate.MatchExpr, event)
		if len(matches) == 0 {
			continue
		}

		event.RuleIDs = appendUnique(event.RuleIDs, candidate.ID)
		event.RuleNames = appendUnique(event.RuleNames, candidate.Name)
		for _, match := range matches {
			match.RuleID = candidate.ID
			match.RuleName = candidate.Name
			event.RuleMatches = append(event.RuleMatches, match)
		}
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
