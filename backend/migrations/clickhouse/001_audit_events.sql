CREATE DATABASE IF NOT EXISTS diting;

CREATE TABLE IF NOT EXISTS diting.audit_events
(
    event_id String,
    event_time DateTime64(3),
    event_date Date,
    ingest_time DateTime64(3),

    event_type LowCardinality(String),
    action LowCardinality(String),
    severity LowCardinality(String),
    risk_score UInt8,
    tags Array(String),

    host_name String,
    host_ip String,
    node_name String,

    namespace String,
    pod_name String,
    container_id String,
    container_name String,
    image String,

    pid UInt32,
    ppid UInt32,
    process_name String,
    binary_path String,
    cmdline String,
    cwd String,

    parent_process_name String,
    parent_binary_path String,
    parent_cmdline String,

    uid UInt32,
    gid UInt32,
    username String,
    auid UInt32,
    euid UInt32,
    egid UInt32,
    login_username String,

    file_path String,
    file_operation LowCardinality(String),

    src_ip String,
    src_port UInt16,
    dst_ip String,
    dst_port UInt16,
    protocol LowCardinality(String),
    domain String,

    rule_ids Array(String),
    rule_names Array(String),

    raw_event String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_date, event_type, severity, host_name, namespace, pod_name, event_time)
TTL event_date + INTERVAL 90 DAY;
