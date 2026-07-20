package clickhouse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"diting/backend/internal/stats"
)

type RuleCounter interface {
	CountEnabledRules(ctx context.Context) (uint64, error)
}

type StatsRepository struct {
	client      *HTTPClient
	ruleCounter RuleCounter
}

func NewStatsRepository(client *HTTPClient, ruleCounter RuleCounter) *StatsRepository {
	return &StatsRepository{client: client, ruleCounter: ruleCounter}
}

func (r *StatsRepository) Overview(ctx context.Context, query stats.Query) (stats.Overview, error) {
	sql := fmt.Sprintf(`SELECT
	count() AS total_events,
	countIf(severity IN ('high', 'critical')) AS high_risk_events,
	uniqExact(if(host_name != '', host_name, if(host_id != '', host_id, node_name))) AS active_hosts
FROM %s
WHERE %s
FORMAT JSONEachRow`, r.table(), statsWhere(query))
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return stats.Overview{}, err
	}

	var row struct {
		TotalEvents    flexibleUint64 `json:"total_events"`
		HighRiskEvents flexibleUint64 `json:"high_risk_events"`
		ActiveHosts    flexibleUint64 `json:"active_hosts"`
	}
	if err := firstJSONRow(data, &row); err != nil {
		return stats.Overview{}, err
	}

	activeRules := uint64(0)
	if r.ruleCounter != nil {
		activeRules, err = r.ruleCounter.CountEnabledRules(ctx)
		if err != nil {
			return stats.Overview{}, err
		}
	}
	return stats.Overview{
		TotalEvents:    uint64(row.TotalEvents),
		HighRiskEvents: uint64(row.HighRiskEvents),
		ActiveHosts:    uint64(row.ActiveHosts),
		ActiveRules:    activeRules,
	}, nil
}

type flexibleUint64 uint64

func (v *flexibleUint64) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), `"`)
	if raw == "" || raw == "null" {
		*v = 0
		return nil
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return err
	}
	*v = flexibleUint64(parsed)
	return nil
}

func (r *StatsRepository) EventTrend(ctx context.Context, query stats.Query) ([]stats.TrendPoint, error) {
	sql := fmt.Sprintf(`SELECT
	formatDateTime(toStartOfHour(toTimeZone(event_time, 'Asia/Shanghai')), '%%Y-%%m-%%d %%H:00:00') AS time,
	count() AS count
FROM %s
WHERE %s
GROUP BY time
ORDER BY time ASC
FORMAT JSONEachRow`, r.table(), statsWhere(query))
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[trendRow](data)
	if err != nil {
		return nil, err
	}
	points := make([]stats.TrendPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, stats.TrendPoint{Time: row.Time, Count: uint64(row.Count)})
	}
	return points, nil
}

func (r *StatsRepository) TopCommands(ctx context.Context, query stats.Query) ([]stats.TopItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	sql := fmt.Sprintf(`SELECT
	process_name AS name,
	count() AS count
FROM %s
WHERE %s AND event_type = 'process_exec' AND process_name != ''
GROUP BY process_name
ORDER BY count DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), limit)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[topItemRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.TopItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.TopItem{Name: row.Name, Count: uint64(row.Count)})
	}
	return items, nil
}

func (r *StatsRepository) TopHosts(ctx context.Context, query stats.Query) ([]stats.TopItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	sql := fmt.Sprintf(`SELECT
	if(host_name != '', host_name, if(host_id != '', host_id, node_name)) AS name,
	count() AS count
FROM %s
WHERE %s AND event_type = 'process_exec' AND name != ''
GROUP BY name
ORDER BY count DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), limit)
	return r.topItems(ctx, sql)
}

func (r *StatsRepository) TopNamespaces(ctx context.Context, query stats.Query) ([]stats.TopItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	sql := fmt.Sprintf(`SELECT
	namespace AS name,
	count() AS count
FROM %s
WHERE %s AND event_type = 'process_exec' AND namespace != ''
GROUP BY namespace
ORDER BY count DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), limit)
	return r.topItems(ctx, sql)
}

func (r *StatsRepository) topItems(ctx context.Context, sql string) ([]stats.TopItem, error) {
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[topItemRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.TopItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.TopItem{Name: row.Name, Count: uint64(row.Count)})
	}
	return items, nil
}

func (r *StatsRepository) CommandStats(ctx context.Context, query stats.Query) ([]stats.CommandItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	conditions := []string{
		statsWhere(query),
		"event_type = 'process_exec'",
		"cmdline != ''",
	}
	if query.Keyword != "" {
		keyword := escapeSQL(query.Keyword)
		conditions = append(conditions, "(positionCaseInsensitive(cmdline, '"+keyword+"') > 0 OR positionCaseInsensitive(process_name, '"+keyword+"') > 0)")
	}
	if query.Username != "" {
		username := escapeSQL(query.Username)
		conditions = append(conditions, "(username = '"+username+"' OR login_username = '"+username+"')")
	}
	if query.HostName != "" {
		hostName := escapeSQL(query.HostName)
		conditions = append(conditions, "(host_id = '"+hostName+"' OR node_name = '"+hostName+"' OR host_name = '"+hostName+"')")
	}
	sql := fmt.Sprintf(`SELECT
	process_name,
	cmdline,
	username,
	login_username,
	argMax(host_id, event_time) AS latest_host_id,
	argMax(host_name, event_time) AS latest_host_name,
	argMax(node_name, event_time) AS latest_node_name,
	uniqExact(if(host_id != '', host_id, if(node_name != '', node_name, host_name))) AS host_count,
	count() AS command_count,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen,
	max(event_time) AS last_seen_sort
FROM %s
WHERE %s
GROUP BY process_name, cmdline, username, login_username
ORDER BY last_seen_sort DESC, command_count DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), strings.Join(conditions, " AND "), limit)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[commandItemRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.CommandItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.CommandItem{
			ProcessName:   row.ProcessName,
			Cmdline:       row.Cmdline,
			Username:      row.Username,
			LoginUsername: row.LoginUsername,
			HostID:        row.HostID,
			HostName:      row.HostName,
			NodeName:      row.NodeName,
			HostCount:     uint64(row.HostCount),
			Count:         row.commandCount(),
			FirstSeen:     row.FirstSeen,
			LastSeen:      row.LastSeen,
		})
	}
	return items, nil
}

func (r *StatsRepository) UserAudits(ctx context.Context, query stats.Query) ([]stats.UserAuditItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	conditions := []string{
		statsWhere(query),
		"event_type = 'process_exec'",
		"audit_user != ''",
	}
	if query.Keyword != "" {
		keyword := escapeSQL(query.Keyword)
		conditions = append(conditions, "positionCaseInsensitive(audit_user, '"+keyword+"') > 0")
	}
	if query.HostName != "" {
		hostName := escapeSQL(query.HostName)
		conditions = append(conditions, "(host_id = '"+hostName+"' OR node_name = '"+hostName+"' OR host_name = '"+hostName+"')")
	}
	sql := fmt.Sprintf(`SELECT
	audit_user AS username,
	count() AS command_count,
	uniqExact(node_name) AS active_hosts,
	countIf(severity IN ('high', 'critical')) AS high_risk_events,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen
FROM
(
	SELECT
		if(login_username != '', login_username, username) AS audit_user,
		host_id,
		node_name,
		host_name,
		severity,
		event_time,
		event_type
	FROM %s
	WHERE %s
)
WHERE %s
GROUP BY audit_user
ORDER BY command_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), strings.Join(conditions[1:], " AND "), limit)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[userAuditRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.UserAuditItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.UserAuditItem{
			Username:       row.Username,
			CommandCount:   uint64(row.CommandCount),
			ActiveHosts:    uint64(row.ActiveHosts),
			HighRiskEvents: uint64(row.HighRiskEvents),
			FirstSeen:      row.FirstSeen,
			LastSeen:       row.LastSeen,
		})
	}
	return items, nil
}

func (r *StatsRepository) HostAudits(ctx context.Context, query stats.Query) ([]stats.HostAuditItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	conditions := []string{
		"event_type = 'process_exec'",
		"audit_host != ''",
	}
	if query.Keyword != "" {
		keyword := escapeSQL(query.Keyword)
		conditions = append(conditions, "positionCaseInsensitive(audit_host, '"+keyword+"') > 0")
	}
	sql := fmt.Sprintf(`SELECT
	audit_host_key AS host_id,
	anyLast(audit_host) AS host_name,
	anyLast(node_name) AS node_name,
	count() AS command_count,
	uniqExact(audit_user) AS active_users,
	countIf(severity IN ('high', 'critical')) AS high_risk_events,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen
FROM
(
	SELECT
		if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS audit_host_key,
		if(host_name != '', host_name, if(host_id != '', host_id, node_name)) AS audit_host,
		node_name,
		if(login_username != '', login_username, username) AS audit_user,
		severity,
		event_time,
		event_type
	FROM %s
	WHERE %s
)
WHERE %s
GROUP BY audit_host_key
ORDER BY command_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), strings.Join(conditions, " AND "), limit)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[hostAuditRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.HostAuditItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.HostAuditItem{
			HostID:         row.HostID,
			HostName:       row.HostName,
			NodeName:       row.NodeName,
			CommandCount:   uint64(row.CommandCount),
			ActiveUsers:    uint64(row.ActiveUsers),
			HighRiskEvents: uint64(row.HighRiskEvents),
			FirstSeen:      row.FirstSeen,
			LastSeen:       row.LastSeen,
		})
	}
	return items, nil
}

func (r *StatsRepository) HostUsers(ctx context.Context, query stats.Query) ([]stats.HostUserItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	conditions := []string{"event_type = 'process_exec'", "audit_user != ''"}
	if query.HostName != "" {
		hostName := escapeSQL(query.HostName)
		conditions = append(conditions, "(host_id = '"+hostName+"' OR node_name = '"+hostName+"' OR host_name = '"+hostName+"')")
	}
	sql := fmt.Sprintf(`SELECT
	audit_user AS username,
	count() AS command_count,
	countIf(severity IN ('high', 'critical')) AS high_risk_events,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen
FROM
(
	SELECT
		host_id,
		node_name,
		host_name,
		if(login_username != '', login_username, username) AS audit_user,
		severity,
		event_time,
		event_type
	FROM %s
	WHERE %s
)
WHERE %s
GROUP BY audit_user
ORDER BY command_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), strings.Join(conditions, " AND "), limit)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[hostUserRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.HostUserItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.HostUserItem{
			Username:       row.Username,
			CommandCount:   uint64(row.CommandCount),
			HighRiskEvents: uint64(row.HighRiskEvents),
			FirstSeen:      row.FirstSeen,
			LastSeen:       row.LastSeen,
		})
	}
	return items, nil
}

func (r *StatsRepository) HostBehavior(ctx context.Context, query stats.Query) (stats.HostBehavior, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	hostFilter := ""
	if query.HostName != "" {
		hostName := escapeSQL(query.HostName)
		hostFilter = " AND (host_id = '" + hostName + "' OR node_name = '" + hostName + "' OR host_name = '" + hostName + "')"
	}
	sensitiveFilePath := `(file_path IN ('/etc/passwd', '/etc/shadow', '/etc/sudoers', '/etc/group', '/etc/gshadow', '/etc/ssh/sshd_config') OR file_path LIKE '/etc/sudoers.d/%' OR file_path LIKE '/etc/ssh/%' OR file_path LIKE '/root/%' OR file_path LIKE '/home/%/.ssh/%' OR file_path LIKE '/var/log/auth.log%' OR file_path LIKE '/var/log/secure%')`
	fileSQL := fmt.Sprintf(`SELECT
	file_path AS name,
	count() AS count,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen
FROM %s
WHERE %s%s AND event_type = 'file_access' AND file_path != '' AND file_path NOT IN ('/etc', '/proc', '/sys', '/dev') AND file_path NOT LIKE '/proc/%%' AND file_path NOT LIKE '/sys/%%' AND file_path NOT LIKE '/dev/%%' AND %s
GROUP BY file_path
ORDER BY count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), hostFilter, sensitiveFilePath, limit)
	filePaths, err := r.behaviorItems(ctx, fileSQL)
	if err != nil {
		return stats.HostBehavior{}, err
	}

	networkSQL := fmt.Sprintf(`SELECT
	concat(dst_ip, if(dst_port = 0, '', concat(':', toString(dst_port)))) AS name,
	count() AS count,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen
FROM %s
WHERE %s%s AND event_type = 'network_connect' AND dst_ip != '' AND dst_ip != 'invalid IP' AND IPv4StringToNumOrNull(dst_ip) IS NOT NULL
GROUP BY name
ORDER BY count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), hostFilter, limit)
	network, err := r.behaviorItems(ctx, networkSQL)
	if err != nil {
		return stats.HostBehavior{}, err
	}

	eventTypeSQL := fmt.Sprintf(`SELECT
	event_type AS name,
	count() AS count,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen
FROM %s
WHERE %s%s AND event_type != 'process_exec' AND event_type != ''
GROUP BY event_type
ORDER BY count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), hostFilter, limit)
	eventTypes, err := r.behaviorItems(ctx, eventTypeSQL)
	if err != nil {
		return stats.HostBehavior{}, err
	}

	return stats.HostBehavior{FilePaths: filePaths, Network: network, EventTypes: eventTypes}, nil
}

func (r *StatsRepository) behaviorItems(ctx context.Context, sql string) ([]stats.BehaviorItem, error) {
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[behaviorItemRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.BehaviorItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.BehaviorItem{
			Name:      row.Name,
			Count:     uint64(row.Count),
			FirstSeen: row.FirstSeen,
			LastSeen:  row.LastSeen,
		})
	}
	return items, nil
}

func (r *StatsRepository) RuleHits(ctx context.Context, query stats.Query) ([]stats.RuleHitItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	conditions := []string{"rule_name != ''"}
	if query.Keyword != "" {
		keyword := escapeSQL(query.Keyword)
		conditions = append(conditions, "positionCaseInsensitive(rule_name, '"+keyword+"') > 0")
	}
	sql := fmt.Sprintf(`SELECT
	rule_name,
	count() AS hit_count,
	uniqExact(audit_host) AS active_hosts,
	uniqExact(audit_user) AS active_users,
	min(event_time) AS first_seen,
	max(event_time) AS last_seen
FROM
(
	SELECT
		arrayJoin(rule_names) AS rule_name,
		if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS audit_host,
		if(login_username != '', login_username, username) AS audit_user,
		event_time
	FROM %s
	WHERE %s AND length(rule_names) > 0
)
WHERE %s
GROUP BY rule_name
ORDER BY hit_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.table(), statsWhere(query), strings.Join(conditions, " AND "), limit)
	data, err := r.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	rows, err := decodeJSONRows[ruleHitRow](data)
	if err != nil {
		return nil, err
	}
	items := make([]stats.RuleHitItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stats.RuleHitItem{
			RuleName:    row.RuleName,
			HitCount:    uint64(row.HitCount),
			ActiveHosts: uint64(row.ActiveHosts),
			ActiveUsers: uint64(row.ActiveUsers),
			FirstSeen:   row.FirstSeen,
			LastSeen:    row.LastSeen,
		})
	}
	return items, nil
}

type trendRow struct {
	Time  string         `json:"time"`
	Count flexibleUint64 `json:"count"`
}

type topItemRow struct {
	Name  string         `json:"name"`
	Count flexibleUint64 `json:"count"`
}

type commandItemRow struct {
	ProcessName   string         `json:"process_name"`
	Cmdline       string         `json:"cmdline"`
	Username      string         `json:"username"`
	LoginUsername string         `json:"login_username"`
	HostID        string         `json:"latest_host_id"`
	HostName      string         `json:"latest_host_name"`
	NodeName      string         `json:"latest_node_name"`
	HostCount     flexibleUint64 `json:"host_count"`
	CommandCount  flexibleUint64 `json:"command_count"`
	Count         flexibleUint64 `json:"count"`
	FirstSeen     string         `json:"first_seen"`
	LastSeen      string         `json:"last_seen"`
}

func (r commandItemRow) commandCount() uint64 {
	if r.CommandCount != 0 {
		return uint64(r.CommandCount)
	}
	return uint64(r.Count)
}

type userAuditRow struct {
	Username       string         `json:"username"`
	CommandCount   flexibleUint64 `json:"command_count"`
	ActiveHosts    flexibleUint64 `json:"active_hosts"`
	HighRiskEvents flexibleUint64 `json:"high_risk_events"`
	FirstSeen      string         `json:"first_seen"`
	LastSeen       string         `json:"last_seen"`
}

type hostAuditRow struct {
	HostID         string         `json:"host_id"`
	HostName       string         `json:"host_name"`
	NodeName       string         `json:"node_name"`
	CommandCount   flexibleUint64 `json:"command_count"`
	ActiveUsers    flexibleUint64 `json:"active_users"`
	HighRiskEvents flexibleUint64 `json:"high_risk_events"`
	FirstSeen      string         `json:"first_seen"`
	LastSeen       string         `json:"last_seen"`
}

type hostUserRow struct {
	Username       string         `json:"username"`
	CommandCount   flexibleUint64 `json:"command_count"`
	HighRiskEvents flexibleUint64 `json:"high_risk_events"`
	FirstSeen      string         `json:"first_seen"`
	LastSeen       string         `json:"last_seen"`
}

type behaviorItemRow struct {
	Name      string         `json:"name"`
	Count     flexibleUint64 `json:"count"`
	FirstSeen string         `json:"first_seen"`
	LastSeen  string         `json:"last_seen"`
}

type ruleHitRow struct {
	RuleName    string         `json:"rule_name"`
	HitCount    flexibleUint64 `json:"hit_count"`
	ActiveHosts flexibleUint64 `json:"active_hosts"`
	ActiveUsers flexibleUint64 `json:"active_users"`
	FirstSeen   string         `json:"first_seen"`
	LastSeen    string         `json:"last_seen"`
}

func (r *StatsRepository) table() string {
	if r.client.config.Database == "" {
		return "audit_events"
	}
	return r.client.config.Database + ".audit_events"
}

func statsWhere(query stats.Query) string {
	return fmt.Sprintf("event_time >= parseDateTime64BestEffort('%s', 3) AND event_time <= parseDateTime64BestEffort('%s', 3)",
		formatDateTime64(query.StartTime),
		formatDateTime64(query.EndTime),
	)
}

func firstJSONRow(data []byte, value any) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		return json.Unmarshal([]byte(line), value)
	}
	return scanner.Err()
}

func decodeJSONRows[T any](data []byte) ([]T, error) {
	result := []T{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
