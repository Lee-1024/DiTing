package clickhouse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	sql := fmt.Sprintf(`SELECT
	process_name,
	cmdline,
	username,
	login_username,
	count() AS count,
	formatDateTime(toTimeZone(min(event_time), 'Asia/Shanghai'), '%%Y-%%m-%%d %%H:%%M:%%S') AS first_seen,
	formatDateTime(toTimeZone(max(event_time), 'Asia/Shanghai'), '%%Y-%%m-%%d %%H:%%M:%%S') AS last_seen
FROM %s
WHERE %s
GROUP BY process_name, cmdline, username, login_username
ORDER BY last_seen DESC, count DESC
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
			Count:         uint64(row.Count),
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
	sql := fmt.Sprintf(`SELECT
	audit_user AS username,
	count() AS command_count,
	uniqExact(node_name) AS active_hosts,
	countIf(severity IN ('high', 'critical')) AS high_risk_events,
	formatDateTime(toTimeZone(min(event_time), 'Asia/Shanghai'), '%%Y-%%m-%%d %%H:%%M:%%S') AS first_seen,
	formatDateTime(toTimeZone(max(event_time), 'Asia/Shanghai'), '%%Y-%%m-%%d %%H:%%M:%%S') AS last_seen
FROM
(
	SELECT
		if(login_username != '', login_username, username) AS audit_user,
		node_name,
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
	audit_host AS host_name,
	count() AS command_count,
	uniqExact(audit_user) AS active_users,
	countIf(severity IN ('high', 'critical')) AS high_risk_events,
	formatDateTime(toTimeZone(min(event_time), 'Asia/Shanghai'), '%%Y-%%m-%%d %%H:%%M:%%S') AS first_seen,
	formatDateTime(toTimeZone(max(event_time), 'Asia/Shanghai'), '%%Y-%%m-%%d %%H:%%M:%%S') AS last_seen
FROM
(
	SELECT
		if(host_name != '', host_name, if(host_id != '', host_id, node_name)) AS audit_host,
		if(login_username != '', login_username, username) AS audit_user,
		severity,
		event_time,
		event_type
	FROM %s
	WHERE %s
)
WHERE %s
GROUP BY audit_host
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
			HostName:       row.HostName,
			CommandCount:   uint64(row.CommandCount),
			ActiveUsers:    uint64(row.ActiveUsers),
			HighRiskEvents: uint64(row.HighRiskEvents),
			FirstSeen:      row.FirstSeen,
			LastSeen:       row.LastSeen,
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
	Count         flexibleUint64 `json:"count"`
	FirstSeen     string         `json:"first_seen"`
	LastSeen      string         `json:"last_seen"`
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
	HostName       string         `json:"host_name"`
	CommandCount   flexibleUint64 `json:"command_count"`
	ActiveUsers    flexibleUint64 `json:"active_users"`
	HighRiskEvents flexibleUint64 `json:"high_risk_events"`
	FirstSeen      string         `json:"first_seen"`
	LastSeen       string         `json:"last_seen"`
}

func (r *StatsRepository) table() string {
	if r.client.config.Database == "" {
		return "audit_events"
	}
	return r.client.config.Database + ".audit_events"
}

func statsWhere(query stats.Query) string {
	return fmt.Sprintf("event_time >= parseDateTime64BestEffort('%s', 3) AND event_time <= parseDateTime64BestEffort('%s', 3)",
		query.StartTime.Format(time.RFC3339Nano),
		query.EndTime.Format(time.RFC3339Nano),
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
