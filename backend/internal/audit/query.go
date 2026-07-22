package audit

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Query struct {
	StartTime     time.Time
	EndTime       time.Time
	EventType     string
	Severity      string
	SeverityIn    []string
	HostName      string
	Namespace     string
	PodName       string
	Username      string
	LoginUsername string
	ExecUsername  string
	Keyword       string
	Cmdline       string
	FilePath      string
	DstIP         string
	DstPort       int
	EventIDs      []string
	Page          int
	PageSize      int
}

func ParseQuery(r *http.Request) (Query, error) {
	values := r.URL.Query()
	now := time.Now().UTC()
	query := Query{
		StartTime:     now.Add(-24 * time.Hour),
		EndTime:       now,
		Page:          parsePositiveInt(values.Get("page"), 1),
		PageSize:      parsePositiveInt(values.Get("page_size"), 10),
		EventType:     strings.TrimSpace(values.Get("event_type")),
		Severity:      strings.TrimSpace(values.Get("severity")),
		SeverityIn:    parseCSV(values.Get("severity_in")),
		HostName:      strings.TrimSpace(values.Get("host_name")),
		Namespace:     strings.TrimSpace(values.Get("namespace")),
		PodName:       strings.TrimSpace(values.Get("pod_name")),
		Username:      strings.TrimSpace(values.Get("username")),
		LoginUsername: strings.TrimSpace(values.Get("login_username")),
		ExecUsername:  strings.TrimSpace(values.Get("exec_username")),
		Keyword:       strings.TrimSpace(values.Get("keyword")),
		Cmdline:       strings.TrimSpace(values.Get("cmdline")),
		FilePath:      strings.TrimSpace(values.Get("file_path")),
		DstIP:         strings.TrimSpace(values.Get("dst_ip")),
		DstPort:       parsePositiveInt(values.Get("dst_port"), 0),
		EventIDs:      parseCSV(values.Get("event_ids")),
	}
	if query.PageSize > 500 {
		query.PageSize = 500
	}

	if raw := values.Get("start_time"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return Query{}, fmt.Errorf("invalid start_time: %w", err)
		}
		query.StartTime = parsed
	}
	if raw := values.Get("end_time"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return Query{}, fmt.Errorf("invalid end_time: %w", err)
		}
		query.EndTime = parsed
	}
	return query, nil
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
