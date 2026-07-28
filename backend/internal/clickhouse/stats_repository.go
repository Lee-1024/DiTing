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

// NewStatsRepository 创建并初始化 New Stats Repository 实例。
func NewStatsRepository(client *HTTPClient, ruleCounter RuleCounter) *StatsRepository {
	return &StatsRepository{client: client, ruleCounter: ruleCounter}
}

// Overview 处理 Overview 相关逻辑。
func (r *StatsRepository) Overview(ctx context.Context, query stats.Query) (stats.Overview, error) {
	sql := fmt.Sprintf(`SELECT
	countMerge(event_count) AS total_events,
	countMergeIf(event_count, severity IN ('high', 'critical')) AS high_risk_events,
	uniqMerge(active_hosts) AS active_hosts
FROM %s
WHERE %s
FORMAT JSONEachRow`, r.overviewTable(), statsHourWhere(query))
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

// UnmarshalJSON 处理 Unmarshal JSON 相关逻辑。
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

// EventTrend 处理 Event Trend 相关逻辑。
func (r *StatsRepository) EventTrend(ctx context.Context, query stats.Query) ([]stats.TrendPoint, error) {
	sql := fmt.Sprintf(`SELECT
	formatDateTime(toStartOfHour(toTimeZone(hour, 'Asia/Shanghai')), '%%Y-%%m-%%d %%H:00:00') AS time,
	countMerge(event_count) AS count
FROM %s
WHERE %s
GROUP BY time
ORDER BY time ASC
FORMAT JSONEachRow`, r.overviewTable(), statsHourWhere(query))
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

// TopCommands 处理 Top Commands 相关逻辑。
func (r *StatsRepository) TopCommands(ctx context.Context, query stats.Query) ([]stats.TopItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	sql := fmt.Sprintf(`SELECT
	process_name AS name,
	countMerge(command_count) AS count
FROM %s
WHERE %s AND process_name != ''
GROUP BY process_name
ORDER BY count DESC
LIMIT %d
FORMAT JSONEachRow`, r.commandStatsTable(), statsHourWhere(query), limit)
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

// TopHosts 处理 Top Hosts 相关逻辑。
func (r *StatsRepository) TopHosts(ctx context.Context, query stats.Query) ([]stats.TopItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	sql := fmt.Sprintf(`SELECT
anyLast(host_name) AS name,
	countMerge(command_count) AS count
FROM %s
WHERE %s AND host_key != ''
GROUP BY host_key
ORDER BY count DESC
LIMIT %d
FORMAT JSONEachRow`, r.hostStatsTable(), statsHourWhere(query), limit)
	return r.topItems(ctx, sql)
}

// TopNamespaces 处理 Top Namespaces 相关逻辑。
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

// topItems 处理 top Items 相关逻辑。
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

// CommandStats 处理 Command Stats 相关逻辑。
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

// UserAudits 处理 User Audits 相关逻辑。
func (r *StatsRepository) UserAudits(ctx context.Context, query stats.Query) ([]stats.UserAuditItem, error) {
	if query.HostName != "" {
		return r.userAuditsRaw(ctx, query)
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	conditions := []string{
		statsHourWhere(query),
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
	countMerge(command_count) AS command_count,
	uniqMerge(active_hosts) AS active_hosts,
	countIfMerge(high_risk_events) AS high_risk_events,
	minMerge(first_seen) AS first_seen,
	maxMerge(last_seen) AS last_seen
FROM %s
WHERE %s
GROUP BY audit_user
ORDER BY command_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.userStatsTable(), strings.Join(conditions, " AND "), limit)
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

func (r *StatsRepository) userAuditsRaw(ctx context.Context, query stats.Query) ([]stats.UserAuditItem, error) {
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

// HostAudits 处理 Host Audits 相关逻辑。
func (r *StatsRepository) HostAudits(ctx context.Context, query stats.Query) ([]stats.HostAuditItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	conditions := []string{
		statsHourWhere(query),
		"host_key != ''",
	}
	if query.Keyword != "" {
		keyword := escapeSQL(query.Keyword)
		conditions = append(conditions, "(positionCaseInsensitive(host_key, '"+keyword+"') > 0 OR positionCaseInsensitive(host_name, '"+keyword+"') > 0 OR positionCaseInsensitive(node_name, '"+keyword+"') > 0)")
	}
	sql := fmt.Sprintf(`SELECT
	host_key AS host_id,
	anyLast(host_name) AS host_name,
	anyLast(node_name) AS node_name,
	countMerge(command_count) AS command_count,
	uniqMerge(active_users) AS active_users,
	countIfMerge(high_risk_events) AS high_risk_events,
	minMerge(first_seen) AS first_seen,
	maxMerge(last_seen) AS last_seen
FROM %s
WHERE %s
GROUP BY host_key
ORDER BY command_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.hostStatsTable(), strings.Join(conditions, " AND "), limit)
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

// HostUsers 处理 Host Users 相关逻辑。
func (r *StatsRepository) HostUsers(ctx context.Context, query stats.Query) ([]stats.HostUserItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	conditions := []string{statsHourWhere(query), "audit_user != ''"}
	if query.HostName != "" {
		hostName := escapeSQL(query.HostName)
		conditions = append(conditions, "(host_key = '"+hostName+"' OR host_name = '"+hostName+"' OR node_name = '"+hostName+"')")
	}
	sql := fmt.Sprintf(`SELECT
	audit_user AS username,
	countMerge(command_count) AS command_count,
	countIfMerge(high_risk_events) AS high_risk_events,
	minMerge(first_seen) AS first_seen,
	maxMerge(last_seen) AS last_seen
FROM %s
WHERE %s
GROUP BY audit_user
ORDER BY command_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.hostUserStatsTable(), strings.Join(conditions, " AND "), limit)
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

// HostBehavior 处理 Host Behavior 相关逻辑。
func (r *StatsRepository) HostBehavior(ctx context.Context, query stats.Query) (stats.HostBehavior, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	conditions := []string{statsHourWhere(query)}
	if query.HostName != "" {
		hostName := escapeSQL(query.HostName)
		conditions = append(conditions, "(host_key = '"+hostName+"' OR host_name = '"+hostName+"' OR node_name = '"+hostName+"')")
	}
	baseWhere := strings.Join(conditions, " AND ")
	filePaths, err := r.hostBehaviorItems(ctx, baseWhere, "file_path", sensitiveBehaviorNameFilter(), limit)
	if err != nil {
		return stats.HostBehavior{}, err
	}
	network, err := r.hostBehaviorItems(ctx, baseWhere, "network", "", limit)
	if err != nil {
		return stats.HostBehavior{}, err
	}
	eventTypes, err := r.hostBehaviorItems(ctx, baseWhere, "event_type", "", limit)
	if err != nil {
		return stats.HostBehavior{}, err
	}
	ruleHits, err := r.hostBehaviorItems(ctx, baseWhere, "rule_hit", "", limit)
	if err != nil {
		return stats.HostBehavior{}, err
	}

	return stats.HostBehavior{FilePaths: filePaths, Network: network, EventTypes: eventTypes, RuleHits: ruleHits}, nil
}

func (r *StatsRepository) hostBehaviorItems(ctx context.Context, baseWhere string, behaviorType string, extraWhere string, limit int) ([]stats.BehaviorItem, error) {
	conditions := []string{baseWhere, "behavior_type = '" + escapeSQL(behaviorType) + "'", "behavior_name != ''"}
	if extraWhere != "" {
		conditions = append(conditions, extraWhere)
	}
	sql := fmt.Sprintf(`SELECT
	behavior_name AS name,
	countMerge(hit_count) AS count,
	minMerge(first_seen) AS first_seen,
	maxMerge(last_seen) AS last_seen
FROM %s
WHERE %s
GROUP BY behavior_name
ORDER BY count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.hostBehaviorTable(), strings.Join(conditions, " AND "), limit)
	return r.behaviorItems(ctx, sql)
}

func sensitiveBehaviorNameFilter() string {
	return `(behavior_name IN ('/etc/passwd', '/etc/shadow', '/etc/sudoers', '/etc/group', '/etc/gshadow', '/etc/ssh/sshd_config') OR behavior_name LIKE '/etc/sudoers.d/%' OR behavior_name LIKE '/etc/ssh/%' OR behavior_name LIKE '/root/%' OR behavior_name LIKE '/home/%/.ssh/%' OR behavior_name LIKE '/var/log/auth.log%' OR behavior_name LIKE '/var/log/secure%') AND behavior_name NOT IN ('/etc', '/proc', '/sys', '/dev') AND behavior_name NOT LIKE '/proc/%' AND behavior_name NOT LIKE '/sys/%' AND behavior_name NOT LIKE '/dev/%'`
}

// behaviorItems 处理 behavior Items 相关逻辑。
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

// RuleHits 处理 Rule Hits 相关逻辑。
func (r *StatsRepository) RuleHits(ctx context.Context, query stats.Query) ([]stats.RuleHitItem, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	conditions := []string{statsHourWhere(query), "rule_name != ''"}
	if query.Keyword != "" {
		keyword := escapeSQL(query.Keyword)
		conditions = append(conditions, "positionCaseInsensitive(rule_name, '"+keyword+"') > 0")
	}
	sql := fmt.Sprintf(`SELECT
	rule_name,
	countMerge(hit_count) AS hit_count,
	uniqMerge(active_hosts) AS active_hosts,
	uniqMerge(active_users) AS active_users,
	minMerge(first_seen) AS first_seen,
	maxMerge(last_seen) AS last_seen
FROM %s
WHERE %s
GROUP BY rule_name
ORDER BY hit_count DESC, last_seen DESC
LIMIT %d
FORMAT JSONEachRow`, r.ruleHitStatsTable(), strings.Join(conditions, " AND "), limit)
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

// commandCount 处理 command Count 相关逻辑。
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

// table 处理 table 相关逻辑。
func (r *StatsRepository) table() string {
	if r.client.config.Database == "" {
		return "audit_events"
	}
	return r.client.config.Database + ".audit_events"
}

func (r *StatsRepository) overviewTable() string {
	if r.client.config.Database == "" {
		return "audit_overview_hourly"
	}
	return r.client.config.Database + ".audit_overview_hourly"
}

func (r *StatsRepository) hostStatsTable() string {
	if r.client.config.Database == "" {
		return "audit_host_stats_hourly"
	}
	return r.client.config.Database + ".audit_host_stats_hourly"
}

func (r *StatsRepository) userStatsTable() string {
	if r.client.config.Database == "" {
		return "audit_user_stats_hourly"
	}
	return r.client.config.Database + ".audit_user_stats_hourly"
}

func (r *StatsRepository) hostUserStatsTable() string {
	if r.client.config.Database == "" {
		return "audit_host_user_stats_hourly"
	}
	return r.client.config.Database + ".audit_host_user_stats_hourly"
}

func (r *StatsRepository) commandStatsTable() string {
	if r.client.config.Database == "" {
		return "audit_command_stats_hourly"
	}
	return r.client.config.Database + ".audit_command_stats_hourly"
}

func (r *StatsRepository) hostBehaviorTable() string {
	if r.client.config.Database == "" {
		return "audit_host_behavior_hourly"
	}
	return r.client.config.Database + ".audit_host_behavior_hourly"
}

func (r *StatsRepository) ruleHitStatsTable() string {
	if r.client.config.Database == "" {
		return "audit_rule_hit_stats_hourly"
	}
	return r.client.config.Database + ".audit_rule_hit_stats_hourly"
}

// statsWhere 处理 stats Where 相关逻辑。
func statsWhere(query stats.Query) string {
	return fmt.Sprintf("event_time >= parseDateTime64BestEffort('%s', 3) AND event_time <= parseDateTime64BestEffort('%s', 3)",
		formatDateTime64(query.StartTime),
		formatDateTime64(query.EndTime),
	)
}

func statsHourWhere(query stats.Query) string {
	return fmt.Sprintf("hour >= parseDateTimeBestEffort('%s') AND hour <= parseDateTimeBestEffort('%s')",
		query.StartTime.Format(time.RFC3339),
		query.EndTime.Format(time.RFC3339),
	)
}

// firstJSONRow 处理 first JSONRow 相关逻辑。
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
