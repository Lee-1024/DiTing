package collector

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

var (
	appArmorFieldPattern = regexp.MustCompile(`([a-zA-Z_]+)=("(?:[^"\\]|\\.)*"|[^\s]+)`)
	appArmorTimePattern  = regexp.MustCompile(`audit\(([0-9]+(?:\.[0-9]+)?):`)
)

func ParseAppArmorAuditEvent(line string) (audit.Event, error) {
	fields := parseAppArmorFields(line)
	if fields["apparmor"] != "DENIED" || fields["profile"] != appArmorProfileName {
		return audit.Event{}, ErrUnsupportedEvent
	}

	eventTime := parseAppArmorTime(line)
	pid := parseAppArmorUint(fields["pid"])
	uid := parseAppArmorUint(firstNonEmptyValue(fields["uid"], fields["fsuid"]))
	auid := parseAppArmorUint(fields["auid"])
	operation := fields["operation"]
	deniedMask := fields["denied_mask"]
	processName := fields["comm"]

	return audit.Event{
		EventID:       fmt.Sprintf("apparmor-%d-%d", eventTime.UnixNano(), pid),
		EventTime:     eventTime,
		EventDate:     time.Date(eventTime.Year(), eventTime.Month(), eventTime.Day(), 0, 0, 0, 0, time.UTC),
		IngestTime:    time.Now().UTC(),
		EventType:     "file_access",
		Action:        operation,
		Severity:      "critical",
		RiskScore:     98,
		Tags:          []string{"apparmor", "enforcement", "blocked", "file-access", "diting-enforcement"},
		PID:           uint32(pid),
		ProcessName:   processName,
		Cmdline:       processName,
		UID:           uint32(uid),
		EUID:          uint32(uid),
		AUID:          uint32(auid),
		FilePath:      fields["name"],
		FileOperation: deniedMask,
		RawEvent:      line,
	}, nil
}

func parseAppArmorFields(line string) map[string]string {
	fields := make(map[string]string)
	for _, match := range appArmorFieldPattern.FindAllStringSubmatch(line, -1) {
		value := match[2]
		if strings.HasPrefix(value, `"`) {
			if unquoted, err := strconv.Unquote(value); err == nil {
				value = unquoted
			}
		}
		fields[match[1]] = value
	}
	return fields
}

func parseAppArmorTime(line string) time.Time {
	match := appArmorTimePattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return time.Now().UTC()
	}
	seconds, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return time.Now().UTC()
	}
	whole := int64(seconds)
	nanos := int64((seconds - float64(whole)) * float64(time.Second))
	return time.Unix(whole, nanos).UTC()
}

func parseAppArmorUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 32)
	return parsed
}

func firstNonEmptyValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
