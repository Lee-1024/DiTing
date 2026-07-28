DROP VIEW IF EXISTS diting.mv_audit_host_behavior_event_type_hourly;
DROP VIEW IF EXISTS diting.mv_audit_host_behavior_network_hourly;
DROP VIEW IF EXISTS diting.mv_audit_host_behavior_file_hourly;
DROP VIEW IF EXISTS diting.mv_audit_operation_groups_hourly;
DROP VIEW IF EXISTS diting.mv_audit_rule_hit_stats_hourly;
DROP VIEW IF EXISTS diting.mv_audit_command_stats_hourly;
DROP VIEW IF EXISTS diting.mv_audit_user_stats_hourly;
DROP VIEW IF EXISTS diting.mv_audit_host_stats_hourly;
DROP VIEW IF EXISTS diting.mv_audit_overview_hourly;

DROP TABLE IF EXISTS diting.audit_host_behavior_hourly;
DROP TABLE IF EXISTS diting.audit_operation_groups_hourly;
DROP TABLE IF EXISTS diting.audit_rule_hit_stats_hourly;
DROP TABLE IF EXISTS diting.audit_command_stats_hourly;
DROP TABLE IF EXISTS diting.audit_user_stats_hourly;
DROP TABLE IF EXISTS diting.audit_host_stats_hourly;
DROP TABLE IF EXISTS diting.audit_overview_hourly;

CREATE TABLE IF NOT EXISTS diting.audit_overview_hourly
(
    hour DateTime,
    event_type LowCardinality(String),
    severity LowCardinality(String),
    event_count AggregateFunction(count),
    active_hosts AggregateFunction(uniq, String)
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, event_type, severity);

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_overview_hourly
TO diting.audit_overview_hourly
AS
SELECT
    toStartOfHour(event_time) AS hour,
    event_type,
    severity,
    countState() AS event_count,
    uniqState(if(host_id != '', host_id, if(node_name != '', node_name, host_name))) AS active_hosts
FROM diting.audit_events
GROUP BY hour, event_type, severity;

CREATE TABLE IF NOT EXISTS diting.audit_host_stats_hourly
(
    hour DateTime,
    host_key String,
    host_name String,
    node_name String,
    command_count AggregateFunction(count),
    active_users AggregateFunction(uniq, String),
    high_risk_events AggregateFunction(countIf, UInt8),
    first_seen AggregateFunction(min, DateTime64(3)),
    last_seen AggregateFunction(max, DateTime64(3))
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, host_key);

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_host_stats_hourly
TO diting.audit_host_stats_hourly
AS
SELECT
    hour,
    host_key,
    anyLast(raw_host_name) AS host_name,
    anyLast(raw_node_name) AS node_name,
    countState() AS command_count,
    uniqState(audit_user) AS active_users,
    countIfState(is_high_risk) AS high_risk_events,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM
(
    SELECT
        toStartOfHour(event_time) AS hour,
        if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS host_key,
        host_name AS raw_host_name,
        node_name AS raw_node_name,
        if(login_username != '', login_username, username) AS audit_user,
        severity IN ('high', 'critical') AS is_high_risk,
        event_time
    FROM diting.audit_events
    WHERE event_type = 'process_exec'
)
GROUP BY hour, host_key;

CREATE TABLE IF NOT EXISTS diting.audit_user_stats_hourly
(
    hour DateTime,
    audit_user String,
    command_count AggregateFunction(count),
    active_hosts AggregateFunction(uniq, String),
    high_risk_events AggregateFunction(countIf, UInt8),
    first_seen AggregateFunction(min, DateTime64(3)),
    last_seen AggregateFunction(max, DateTime64(3))
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, audit_user);

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_user_stats_hourly
TO diting.audit_user_stats_hourly
AS
SELECT
    toStartOfHour(event_time) AS hour,
    if(login_username != '', login_username, username) AS audit_user,
    countState() AS command_count,
    uniqState(if(host_id != '', host_id, if(node_name != '', node_name, host_name))) AS active_hosts,
    countIfState(severity IN ('high', 'critical')) AS high_risk_events,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM diting.audit_events
WHERE event_type = 'process_exec' AND audit_user != ''
GROUP BY hour, audit_user;

CREATE TABLE IF NOT EXISTS diting.audit_command_stats_hourly
(
    hour DateTime,
    process_name String,
    cmdline String,
    audit_user String,
    latest_host_id AggregateFunction(argMax, String, DateTime64(3)),
    latest_host_name AggregateFunction(argMax, String, DateTime64(3)),
    latest_node_name AggregateFunction(argMax, String, DateTime64(3)),
    host_count AggregateFunction(uniq, String),
    command_count AggregateFunction(count),
    first_seen AggregateFunction(min, DateTime64(3)),
    last_seen AggregateFunction(max, DateTime64(3))
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, process_name, cityHash64(cmdline), audit_user);

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_command_stats_hourly
TO diting.audit_command_stats_hourly
AS
SELECT
    toStartOfHour(event_time) AS hour,
    process_name,
    cmdline,
    if(login_username != '', login_username, username) AS audit_user,
    argMaxState(host_id, event_time) AS latest_host_id,
    argMaxState(host_name, event_time) AS latest_host_name,
    argMaxState(node_name, event_time) AS latest_node_name,
    uniqState(if(host_id != '', host_id, if(node_name != '', node_name, host_name))) AS host_count,
    countState() AS command_count,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM diting.audit_events
WHERE event_type = 'process_exec' AND cmdline != ''
GROUP BY hour, process_name, cmdline, audit_user;

CREATE TABLE IF NOT EXISTS diting.audit_rule_hit_stats_hourly
(
    hour DateTime,
    rule_name String,
    hit_count AggregateFunction(count),
    active_hosts AggregateFunction(uniq, String),
    active_users AggregateFunction(uniq, String),
    first_seen AggregateFunction(min, DateTime64(3)),
    last_seen AggregateFunction(max, DateTime64(3))
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, rule_name);

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_rule_hit_stats_hourly
TO diting.audit_rule_hit_stats_hourly
AS
SELECT
    toStartOfHour(event_time) AS hour,
    arrayJoin(rule_names) AS rule_name,
    countState() AS hit_count,
    uniqState(if(host_id != '', host_id, if(node_name != '', node_name, host_name))) AS active_hosts,
    uniqState(if(login_username != '', login_username, username)) AS active_users,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM diting.audit_events
WHERE length(rule_names) > 0
GROUP BY hour, rule_name;

CREATE TABLE IF NOT EXISTS diting.audit_operation_groups_hourly
(
    hour DateTime,
    event_second DateTime,
    audit_host_key String,
    namespace String,
    pod_name String,
    login_username String,
    username String,
    process_name String,
    cmdline String,
    host_name String,
    host_id String,
    node_name String,
    representative_event_id AggregateFunction(argMax, String, DateTime64(3)),
    representative_event_time AggregateFunction(argMax, DateTime64(3), DateTime64(3)),
    representative_event_type AggregateFunction(argMax, String, DateTime64(3)),
    representative_severity AggregateFunction(argMax, String, DateTime64(3)),
    representative_tags AggregateFunction(argMax, Array(String), DateTime64(3)),
    event_count AggregateFunction(sum, UInt64),
    event_types AggregateFunction(groupUniqArray, String),
    severities AggregateFunction(groupUniqArray, String),
    file_paths AggregateFunction(groupUniqArray, String),
    max_severity_rank AggregateFunction(max, UInt8),
    first_seen AggregateFunction(min, DateTime64(3)),
    last_seen AggregateFunction(max, DateTime64(3))
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, event_second, audit_host_key, namespace, pod_name, login_username, username, process_name, cityHash64(cmdline));

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_operation_groups_hourly
TO diting.audit_operation_groups_hourly
AS
SELECT
    hour,
    event_second,
    audit_host_key,
    namespace,
    pod_name,
    login_username,
    username,
    process_name,
    cmdline,
    anyLast(raw_host_name) AS host_name,
    anyLast(raw_host_id) AS host_id,
    anyLast(raw_node_name) AS node_name,
    argMaxState(event_id, event_time) AS representative_event_id,
    argMaxState(event_time, event_time) AS representative_event_time,
    argMaxState(event_type, event_time) AS representative_event_type,
    argMaxState(severity, event_time) AS representative_severity,
    argMaxState(tags, event_time) AS representative_tags,
    sumState(toUInt64(1)) AS event_count,
    groupUniqArrayState(event_type) AS event_types,
    groupUniqArrayState(severity) AS severities,
    groupUniqArrayStateIf(file_path, file_path != '') AS file_paths,
    maxState(toUInt8(indexOf(['info', 'low', 'medium', 'high', 'critical'], severity))) AS max_severity_rank,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM
(
    SELECT
        toStartOfHour(event_time) AS hour,
        toStartOfSecond(event_time) AS event_second,
        if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS audit_host_key,
        namespace,
        pod_name,
        login_username,
        username,
        process_name,
        cmdline,
        host_name AS raw_host_name,
        host_id AS raw_host_id,
        node_name AS raw_node_name,
        event_id,
        event_time,
        event_type,
        severity,
        tags,
        file_path
    FROM diting.audit_events
)
GROUP BY hour, event_second, audit_host_key, namespace, pod_name, login_username, username, process_name, cmdline;

CREATE TABLE IF NOT EXISTS diting.audit_host_behavior_hourly
(
    hour DateTime,
    host_key String,
    behavior_type LowCardinality(String),
    behavior_name String,
    hit_count AggregateFunction(count),
    first_seen AggregateFunction(min, DateTime64(3)),
    last_seen AggregateFunction(max, DateTime64(3))
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, host_key, behavior_type, cityHash64(behavior_name));

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_host_behavior_file_hourly
TO diting.audit_host_behavior_hourly
AS
SELECT
    toStartOfHour(event_time) AS hour,
    if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS host_key,
    'file_path' AS behavior_type,
    file_path AS behavior_name,
    countState() AS hit_count,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM diting.audit_events
WHERE event_type = 'file_access' AND file_path != ''
GROUP BY hour, host_key, behavior_type, behavior_name;

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_host_behavior_network_hourly
TO diting.audit_host_behavior_hourly
AS
SELECT
    toStartOfHour(event_time) AS hour,
    if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS host_key,
    'network' AS behavior_type,
    concat(if(position(dst_ip, ':') > 0, concat('[', dst_ip, ']'), dst_ip), if(dst_port = 0, '', concat(':', toString(dst_port)))) AS behavior_name,
    countState() AS hit_count,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM diting.audit_events
WHERE event_type = 'network_connect' AND dst_ip != ''
GROUP BY hour, host_key, behavior_type, behavior_name;

CREATE MATERIALIZED VIEW IF NOT EXISTS diting.mv_audit_host_behavior_event_type_hourly
TO diting.audit_host_behavior_hourly
AS
SELECT
    toStartOfHour(event_time) AS hour,
    if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS host_key,
    'event_type' AS behavior_type,
    event_type AS behavior_name,
    countState() AS hit_count,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM diting.audit_events
WHERE event_type != ''
GROUP BY hour, host_key, behavior_type, behavior_name;
