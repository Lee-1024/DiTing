package postgres

const ProductionPreReleaseBaselineSQL = `
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
        {"id":"pre-root-process-low-risk","name":"预发忽略 root 常规命令","enabled":true,"conditions":[{"field":"event_type","op":"eq","value":"process_exec"},{"field":"username","op":"eq","value":"root"},{"field":"severity","op":"in","values":["info","low","medium"]}]},
        {"id":"pre-root-file-low-risk","name":"预发忽略 root 常规文件访问","enabled":true,"conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"username","op":"eq","value":"root"},{"field":"severity","op":"in","values":["info","low","medium"]}]},
        {"id":"pre-root-network-low-risk","name":"预发忽略 root 低风险网络连接","enabled":true,"conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"username","op":"eq","value":"root"},{"field":"severity","op":"in","values":["info","low","medium"]}]},
        {"id":"pre-proc-sys-read-noise","name":"预发忽略 proc/sys/dev 高频读取","enabled":true,"conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"^/(proc|sys)/|^/dev/(null|zero|random|urandom)$"},{"field":"file_operation","op":"regex","value":"(?i)(open|read|security_file_open|security_file_permission)"},{"field":"severity","op":"in","values":["info","low","medium"]}]},
        {"id":"pre-monitoring-agent-noise","name":"预发忽略监控探针噪声","enabled":true,"conditions":[{"field":"process_name","op":"in","values":["kube-probe","node_exporter","prometheus","telegraf","grafana-agent"]},{"field":"severity","op":"in","values":["info","low","medium"]}]}
      ]
    }'::jsonb,
    'Production pre-release collector filter baseline.',
    NOW()
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = NOW();

INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
('10000000-0000-0000-0000-000000000101','生产-反弹 Shell 命令','检测 bash -i、nc -e、/dev/tcp、socat 等反弹 Shell 行为。','process_exec',true,'critical',96,'{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"bash -i"},{"field":"cmdline","op":"contains","value":"sh -i"},{"field":"cmdline","op":"contains","value":"nc -e"},{"field":"cmdline","op":"contains","value":"ncat -e"},{"field":"cmdline","op":"contains","value":"/dev/tcp/"},{"field":"cmdline","op":"contains","value":"socat exec:"}]}'::jsonb,'["production","reverse-shell","critical-command"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000102','生产-下载后执行','检测 curl/wget/python 获取远程内容后直接交给 sh/bash/python 执行。','process_exec',true,'critical',92,'{"operator":"or","conditions":[{"field":"cmdline","op":"regex","value":"(?i)(curl|wget)[^|&]*(\\||&&)\\s*(sh|bash)"},{"field":"cmdline","op":"regex","value":"(?i)(curl|wget)\\s+[^ ]+\\s+-O\\s+-\\s*\\|\\s*(sh|bash)"},{"field":"cmdline","op":"regex","value":"(?i)python[0-9.]*\\s+-c\\s+.*(urllib|requests|socket)"}]}'::jsonb,'["production","download-exec","critical-command"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000103','生产-提权与账号变更','检测 sudo、su、passwd、useradd、usermod、groupadd 等提权或账号变更。','process_exec',true,'high',82,'{"operator":"and","conditions":[{"field":"process_name","op":"in","values":["sudo","su","passwd","useradd","usermod","userdel","groupadd","groupmod","chage"]}]}'::jsonb,'["production","privilege","account"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000104','生产-危险权限变更','检测 chmod 777、递归放宽权限、修改 root 属主等高风险权限变更。','process_exec',true,'high',78,'{"operator":"or","conditions":[{"field":"cmdline","op":"regex","value":"(?i)chmod\\s+(-R\\s+)?777"},{"field":"cmdline","op":"regex","value":"(?i)chmod\\s+(-R\\s+)?[0-7]*7[0-7]*"},{"field":"cmdline","op":"contains","value":"chown root"}]}'::jsonb,'["production","permission","hardening"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000201','生产-高危端口外联','检测连接常见反连、代理、远控或非常规服务端口。','network_connect',true,'high',84,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"protocol","op":"eq","value":"tcp"},{"field":"dst_port","op":"in","values":["4444","5555","6666","7777","8888","9999","31337"]}]}'::jsonb,'["production","network","suspicious-port"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000202','生产-解释器直接外联','检测 Shell、Python、Perl、PHP、Ruby、Node 等解释器直接发起网络连接。','network_connect',true,'high',86,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"process_name","op":"in","values":["bash","sh","dash","zsh","python","python3","perl","php","ruby","node"]}]}'::jsonb,'["production","network","interpreter"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000301','生产-敏感文件读取','检测账号、sudo、SSH、计划任务等敏感路径读取。','file_access',true,'high',82,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"}]}'::jsonb,'["production","file-access","sensitive-file"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000302','生产-敏感文件写入','检测账号、sudo、SSH、计划任务等敏感路径写入、创建、截断。','file_access',true,'critical',94,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(write|truncate|create|creat|open.*wronly|open.*rdwr|security_file_permission|security_file_open)"}]}'::jsonb,'["production","file-access","sensitive-file","write"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000303','生产-敏感文件删除或权限变更','检测敏感路径删除、chmod、chown、扩展属性变更。','file_access',true,'critical',95,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(unlink|unlinkat|rmdir|chmod|chown|fchmod|fchown|setxattr|removexattr|security_inode_unlink|security_inode_rmdir|security_inode_setattr)"}]}'::jsonb,'["production","file-access","sensitive-file","mutation"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000401','生产-Web 服务拉起 Shell','检测 nginx/apache/httpd 等 Web 服务父进程拉起 Shell。','process_exec',true,'high',90,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"process_exec"},{"field":"parent_process_name","op":"in","values":["nginx","apache","apache2","httpd","php-fpm"]},{"field":"process_name","op":"in","values":["sh","bash","dash","zsh"]}]}'::jsonb,'["production","process-chain","webshell"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000402','生产-Shell 拉起下载工具外联','检测 Shell 父进程拉起 curl、wget、nc、ncat、socat、telnet 发起网络连接。','network_connect',true,'high',86,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"parent_process_name","op":"in","values":["bash","sh","dash","zsh"]},{"field":"process_name","op":"in","values":["curl","wget","nc","ncat","socat","telnet"]}]}'::jsonb,'["production","process-chain","network","download-tool"]'::jsonb,NOW(),NOW()),
('10000000-0000-0000-0000-000000000403','生产-Shell 拉起解释器外联','检测 Shell 父进程拉起解释器发起网络连接。','network_connect',true,'high',84,'{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"parent_process_name","op":"in","values":["bash","sh","dash","zsh"]},{"field":"process_name","op":"in","values":["python","python3","perl","php","ruby","node"]}]}'::jsonb,'["production","process-chain","network","interpreter"]'::jsonb,NOW(),NOW())
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    event_type = EXCLUDED.event_type,
    enabled = EXCLUDED.enabled,
    severity = EXCLUDED.severity,
    risk_score = EXCLUDED.risk_score,
    match_expr = EXCLUDED.match_expr,
    tags = EXCLUDED.tags,
    updated_at = NOW();
`
