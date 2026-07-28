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
    toStartOfHour(event_time) AS hour,
    if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS host_key,
    anyLast(host_name) AS host_name,
    anyLast(node_name) AS node_name,
    countState() AS command_count,
    uniqState(if(login_username != '', login_username, username)) AS active_users,
    countIfState(severity IN ('high', 'critical')) AS high_risk_events,
    minState(event_time) AS first_seen,
    maxState(event_time) AS last_seen
FROM diting.audit_events
WHERE event_type = 'process_exec'
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
