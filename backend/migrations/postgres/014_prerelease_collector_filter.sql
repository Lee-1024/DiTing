INSERT INTO diting_system_configs (key, value, description, updated_at)
VALUES (
    'collector_filter',
    '{
      "enabled": true,
      "ignoreProcessNames": [],
      "ignoreCommandKeywords": [],
      "ignoreUsers": [],
      "keepSeverities": ["high", "critical"],
      "rules": [
        {
          "id": "pre-root-process-low-risk",
          "name": "预发忽略 root 常规命令",
          "enabled": true,
          "conditions": [
            {"field": "event_type", "op": "eq", "value": "process_exec"},
            {"field": "username", "op": "eq", "value": "root"},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        },
        {
          "id": "pre-root-login-process-exit-low-risk",
          "name": "预发忽略 root 登录进程退出",
          "enabled": true,
          "conditions": [
            {"field": "event_type", "op": "eq", "value": "process_exit"},
            {"field": "login_username", "op": "eq", "value": "root"},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        },
        {
          "id": "pre-root-file-low-risk",
          "name": "预发忽略 root 常规文件访问",
          "enabled": true,
          "conditions": [
            {"field": "event_type", "op": "eq", "value": "file_access"},
            {"field": "username", "op": "eq", "value": "root"},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        },
        {
          "id": "pre-root-network-low-risk",
          "name": "预发忽略 root 低风险网络连接",
          "enabled": true,
          "conditions": [
            {"field": "event_type", "op": "eq", "value": "network_connect"},
            {"field": "username", "op": "eq", "value": "root"},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        },
        {
          "id": "pre-proc-sys-read-noise",
          "name": "预发忽略 proc/sys/dev 高频读取",
          "enabled": true,
          "conditions": [
            {"field": "event_type", "op": "eq", "value": "file_access"},
            {"field": "file_path", "op": "regex", "value": "^/(proc|sys)/|^/dev/(null|zero|random|urandom)$"},
            {"field": "file_operation", "op": "regex", "value": "(?i)(open|read|security_file_open|security_file_permission)"},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        },
        {
          "id": "pre-monitoring-agent-noise",
          "name": "预发忽略监控探针噪声",
          "enabled": true,
          "conditions": [
            {"field": "process_name", "op": "in", "values": ["kube-probe", "node_exporter", "prometheus", "telegraf", "grafana-agent", "zabbix_agentd", "zabbix_agent2"]},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        },
        {
          "id": "pre-monitoring-user-noise",
          "name": "预发忽略监控用户噪声",
          "enabled": true,
          "conditions": [
            {"field": "username", "op": "in", "values": ["zabbix", "prometheus"]},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        },
        {
          "id": "pre-diting-self-vite-noise",
          "name": "预发忽略 DiTing 自身 Vite 服务噪声",
          "enabled": true,
          "conditions": [
            {"field": "cmdline", "op": "contains", "value": "/data/DiTing/"},
            {"field": "cmdline", "op": "regex", "value": "(?i)(node_modules/.bin/vite|\\bnode\\b.*\\bvite\\b)"},
            {"field": "severity", "op": "in", "values": ["info", "low", "medium"]}
          ]
        }
      ]
    }'::jsonb,
    'Pre-release collector filter baseline: reduce root and host noise while preserving high and critical events.',
    NOW()
)
ON CONFLICT (key) DO NOTHING;
