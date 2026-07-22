package clickhouse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

type AuditRepository struct {
	client *HTTPClient
}

func NewAuditRepository(client *HTTPClient) *AuditRepository {
	return &AuditRepository{client: client}
}

func (r *AuditRepository) ListEvents(ctx context.Context, query audit.Query) ([]audit.Event, int, error) {
	sql := buildListEventsSQL(r.client.config.Database, query)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, 0, err
	}

	events := []audit.Event{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		event, err := decodeEventRow(scanner.Bytes())
		if err != nil {
			return nil, 0, err
		}
		if eventMatchesQuery(event, query) {
			events = append(events, event)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}
	total, err := r.countEvents(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return collapseDuplicateListEvents(events), total, nil
}

func (r *AuditRepository) GetEvent(ctx context.Context, eventID string) (audit.Event, error) {
	sql := buildGetEventSQL(r.client.config.Database, eventID)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return audit.Event{}, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		return decodeEventRow(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		return audit.Event{}, err
	}
	return audit.Event{}, audit.ErrNotFound
}

func (r *AuditRepository) countEvents(ctx context.Context, query audit.Query) (int, error) {
	sql := buildCountEventsSQL(r.client.config.Database, query)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return 0, err
	}
	var row struct {
		Total flexibleUint64 `json:"total"`
	}
	if err := firstJSONRow(data, &row); err != nil {
		return 0, err
	}
	return int(row.Total), nil
}

func auditTable(database string) string {
	table := "audit_events"
	if database != "" {
		table = database + "." + table
	}
	return table
}

func buildListEventsSQL(database string, query audit.Query) string {
	table := auditTable(database)
	limit := query.PageSize
	if limit <= 0 {
		limit = 50
	}
	offset := 0
	if query.Page > 1 {
		offset = (query.Page - 1) * limit
	}

	where := buildAuditWhere(query)
	return fmt.Sprintf(`SELECT %s
FROM %s
WHERE %s
ORDER BY event_time DESC
LIMIT %d OFFSET %d
FORMAT JSONEachRow`, auditEventListSelectFields(), table, where, limit, offset)
}

func buildGetEventSQL(database string, eventID string) string {
	return fmt.Sprintf(`SELECT %s
FROM %s
WHERE event_id = '%s'
LIMIT 1
FORMAT JSONEachRow`, auditEventSelectFields(), auditTable(database), escapeSQL(eventID))
}

func auditEventSelectFields() string {
	return "event_id, event_time, event_type, severity, risk_score, host_name, host_id, node_name, namespace, pod_name, username, uid, gid, auid, euid, egid, login_username, process_name, binary_path, cmdline, cwd, parent_process_name, parent_binary_path, parent_cmdline, file_path, file_operation, src_ip, src_port, dst_ip, dst_port, protocol, domain, tags, rule_ids, rule_names, rule_matches, raw_event"
}

func auditEventListSelectFields() string {
	return "event_id, event_time, event_type, severity, risk_score, host_name, host_id, node_name, namespace, pod_name, username, uid, gid, auid, euid, egid, login_username, process_name, binary_path, cmdline, cwd, parent_process_name, parent_binary_path, parent_cmdline, file_path, file_operation, src_ip, src_port, dst_ip, dst_port, protocol, domain, tags, rule_ids, rule_names, '' AS rule_matches, '' AS raw_event"
}

func buildCountEventsSQL(database string, query audit.Query) string {
	return fmt.Sprintf(`SELECT count() AS total
FROM %s
WHERE %s
FORMAT JSONEachRow`, auditTable(database), buildAuditWhere(query))
}

func buildAuditWhere(query audit.Query) string {
	conditions := []string{
		fmt.Sprintf("event_time >= parseDateTime64BestEffort('%s', 3)", query.StartTime.Format(time.RFC3339Nano)),
		fmt.Sprintf("event_time <= parseDateTime64BestEffort('%s', 3)", query.EndTime.Format(time.RFC3339Nano)),
	}
	if query.EventType != "" {
		conditions = append(conditions, "event_type = '"+escapeSQL(query.EventType)+"'")
	}
	if query.Severity != "" {
		conditions = append(conditions, "severity = '"+escapeSQL(query.Severity)+"'")
	}
	if len(query.SeverityIn) > 0 {
		values := make([]string, 0, len(query.SeverityIn))
		for _, severity := range query.SeverityIn {
			values = append(values, "'"+escapeSQL(severity)+"'")
		}
		conditions = append(conditions, "severity IN ("+strings.Join(values, ", ")+")")
	}
	if query.HostName != "" {
		hostName := escapeSQL(query.HostName)
		conditions = append(conditions, "(host_id = '"+hostName+"' OR node_name = '"+hostName+"' OR host_name = '"+hostName+"')")
	}
	if query.Namespace != "" {
		conditions = append(conditions, "namespace = '"+escapeSQL(query.Namespace)+"'")
	}
	if query.PodName != "" {
		conditions = append(conditions, "pod_name = '"+escapeSQL(query.PodName)+"'")
	}
	if query.Username != "" {
		username := escapeSQL(query.Username)
		conditions = append(conditions, "(username = '"+username+"' OR login_username = '"+username+"')")
	}
	if query.LoginUsername != "" {
		conditions = append(conditions, "login_username = '"+escapeSQL(query.LoginUsername)+"'")
	}
	if query.ExecUsername != "" {
		conditions = append(conditions, "username = '"+escapeSQL(query.ExecUsername)+"'")
	}
	if query.Cmdline != "" {
		conditions = append(conditions, "cmdline = '"+escapeSQL(query.Cmdline)+"'")
	}
	if query.FilePath != "" {
		conditions = append(conditions, "file_path = '"+escapeSQL(query.FilePath)+"'")
	}
	if query.DstIP != "" {
		conditions = append(conditions, "dst_ip = '"+escapeSQL(query.DstIP)+"'")
	}
	if query.DstPort > 0 {
		conditions = append(conditions, fmt.Sprintf("dst_port = %d", query.DstPort))
	}
	if len(query.EventIDs) > 0 {
		values := make([]string, 0, len(query.EventIDs))
		for _, eventID := range query.EventIDs {
			values = append(values, "'"+escapeSQL(eventID)+"'")
		}
		conditions = append(conditions, "event_id IN ("+strings.Join(values, ", ")+")")
	}
	if query.Keyword != "" {
		keyword := escapeSQL(query.Keyword)
		conditions = append(conditions, "(positionCaseInsensitive(cmdline, '"+keyword+"') > 0 OR positionCaseInsensitive(process_name, '"+keyword+"') > 0 OR positionCaseInsensitive(username, '"+keyword+"') > 0 OR positionCaseInsensitive(login_username, '"+keyword+"') > 0 OR positionCaseInsensitive(file_path, '"+keyword+"') > 0 OR positionCaseInsensitive(file_operation, '"+keyword+"') > 0 OR positionCaseInsensitive(src_ip, '"+keyword+"') > 0 OR positionCaseInsensitive(dst_ip, '"+keyword+"') > 0 OR positionCaseInsensitive(protocol, '"+keyword+"') > 0 OR positionCaseInsensitive(domain, '"+keyword+"') > 0)")
	}
	return strings.Join(conditions, " AND ")
}

func eventMatchesQuery(event audit.Event, query audit.Query) bool {
	if query.EventType != "" && event.EventType != query.EventType {
		return false
	}
	if query.Severity != "" && event.Severity != query.Severity {
		return false
	}
	if len(query.SeverityIn) > 0 {
		matched := false
		for _, severity := range query.SeverityIn {
			if event.Severity == severity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if query.HostName != "" && event.HostID != query.HostName && event.NodeName != query.HostName && event.HostName != query.HostName {
		return false
	}
	if query.Namespace != "" && event.Namespace != query.Namespace {
		return false
	}
	if query.PodName != "" && event.PodName != query.PodName {
		return false
	}
	if query.Username != "" && event.Username != query.Username && event.LoginUsername != query.Username {
		return false
	}
	if query.LoginUsername != "" && event.LoginUsername != query.LoginUsername {
		return false
	}
	if query.ExecUsername != "" && event.Username != query.ExecUsername {
		return false
	}
	if query.Cmdline != "" && event.Cmdline != query.Cmdline {
		return false
	}
	if query.FilePath != "" && event.FilePath != query.FilePath {
		return false
	}
	if query.DstIP != "" && event.DstIP != query.DstIP {
		return false
	}
	if query.DstPort > 0 && int(event.DstPort) != query.DstPort {
		return false
	}
	if len(query.EventIDs) > 0 {
		matched := false
		for _, eventID := range query.EventIDs {
			if event.EventID == eventID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if query.Keyword != "" {
		keyword := strings.ToLower(query.Keyword)
		if !strings.Contains(strings.ToLower(event.Cmdline), keyword) &&
			!strings.Contains(strings.ToLower(event.ProcessName), keyword) &&
			!strings.Contains(strings.ToLower(event.Username), keyword) &&
			!strings.Contains(strings.ToLower(event.LoginUsername), keyword) &&
			!strings.Contains(strings.ToLower(event.FilePath), keyword) &&
			!strings.Contains(strings.ToLower(event.FileOperation), keyword) &&
			!strings.Contains(strings.ToLower(event.SrcIP), keyword) &&
			!strings.Contains(strings.ToLower(event.DstIP), keyword) &&
			!strings.Contains(strings.ToLower(event.Protocol), keyword) &&
			!strings.Contains(strings.ToLower(event.Domain), keyword) {
			return false
		}
	}
	return true
}

func collapseDuplicateListEvents(events []audit.Event) []audit.Event {
	if len(events) <= 1 {
		return events
	}
	seen := map[string]struct{}{}
	result := make([]audit.Event, 0, len(events))
	for _, event := range events {
		key := listEventDedupKey(event)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, event)
	}
	return result
}

func listEventDedupKey(event audit.Event) string {
	return strings.Join([]string{
		event.EventTime.Truncate(time.Second).Format(time.RFC3339),
		event.EventType,
		event.Action,
		firstNonEmpty(event.HostID, event.NodeName, event.HostName),
		event.LoginUsername,
		event.Username,
		event.ProcessName,
		event.Cmdline,
		event.DstIP,
		fmt.Sprintf("%d", event.DstPort),
	}, "\x00")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func escapeSQL(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

type eventRow struct {
	EventID           string   `json:"event_id"`
	EventTime         string   `json:"event_time"`
	EventType         string   `json:"event_type"`
	Severity          string   `json:"severity"`
	RiskScore         uint8    `json:"risk_score"`
	HostName          string   `json:"host_name"`
	HostID            string   `json:"host_id"`
	NodeName          string   `json:"node_name"`
	Namespace         string   `json:"namespace"`
	PodName           string   `json:"pod_name"`
	Username          string   `json:"username"`
	UID               uint32   `json:"uid"`
	GID               uint32   `json:"gid"`
	AUID              uint32   `json:"auid"`
	EUID              uint32   `json:"euid"`
	EGID              uint32   `json:"egid"`
	LoginUsername     string   `json:"login_username"`
	ProcessName       string   `json:"process_name"`
	BinaryPath        string   `json:"binary_path"`
	Cmdline           string   `json:"cmdline"`
	CWD               string   `json:"cwd"`
	ParentProcessName string   `json:"parent_process_name"`
	ParentBinaryPath  string   `json:"parent_binary_path"`
	ParentCmdline     string   `json:"parent_cmdline"`
	FilePath          string   `json:"file_path"`
	FileOperation     string   `json:"file_operation"`
	SrcIP             string   `json:"src_ip"`
	SrcPort           uint16   `json:"src_port"`
	DstIP             string   `json:"dst_ip"`
	DstPort           uint16   `json:"dst_port"`
	Protocol          string   `json:"protocol"`
	Domain            string   `json:"domain"`
	Tags              []string `json:"tags"`
	RuleIDs           []string `json:"rule_ids"`
	RuleNames         []string `json:"rule_names"`
	RuleMatches       string   `json:"rule_matches"`
	RawEvent          string   `json:"raw_event"`
}

func decodeEventRow(data []byte) (audit.Event, error) {
	var row eventRow
	if err := json.Unmarshal(data, &row); err != nil {
		return audit.Event{}, err
	}
	eventTime, _ := time.Parse("2006-01-02 15:04:05.000", row.EventTime)
	ruleMatches := []audit.RuleMatch{}
	if strings.TrimSpace(row.RuleMatches) != "" {
		_ = json.Unmarshal([]byte(row.RuleMatches), &ruleMatches)
	}
	return audit.Event{
		EventID: row.EventID, EventTime: eventTime, EventType: row.EventType, Severity: row.Severity, RiskScore: row.RiskScore,
		HostName: row.HostName, HostID: row.HostID, NodeName: row.NodeName, Namespace: row.Namespace, PodName: row.PodName,
		Username: row.Username, UID: row.UID, GID: row.GID, AUID: row.AUID, EUID: row.EUID, EGID: row.EGID, LoginUsername: row.LoginUsername,
		ProcessName: row.ProcessName, BinaryPath: row.BinaryPath, Cmdline: row.Cmdline, CWD: row.CWD,
		ParentProcessName: row.ParentProcessName, ParentBinaryPath: row.ParentBinaryPath, ParentCmdline: row.ParentCmdline,
		FilePath: row.FilePath, FileOperation: row.FileOperation,
		SrcIP: row.SrcIP, SrcPort: row.SrcPort, DstIP: row.DstIP, DstPort: row.DstPort, Protocol: row.Protocol, Domain: row.Domain,
		Tags: row.Tags, RuleIDs: row.RuleIDs, RuleNames: row.RuleNames, RuleMatches: ruleMatches, RawEvent: row.RawEvent,
	}, nil
}
