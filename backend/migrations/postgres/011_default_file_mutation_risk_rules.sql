INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000303',
    '敏感文件写入',
    '检测 Tetragon 文件事件中对账号、sudo、SSH、计划任务等敏感路径的写入、创建或截断行为。',
    'file_access',
    true,
    'critical',
    92,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(write|truncate|create|creat|open.*wronly|open.*rdwr|security_file_permission|security_file_open)"}]}'::jsonb,
    '["file-access","sensitive-file","write"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000304',
    '敏感文件权限变更',
    '检测 Tetragon 文件事件中对敏感路径的 chmod、chown、扩展属性等权限或属主变更行为。',
    'file_access',
    true,
    'critical',
    90,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(chmod|chown|fchmod|fchown|setxattr|removexattr|security_inode_setattr)"}]}'::jsonb,
    '["file-access","sensitive-file","permission"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000305',
    '敏感文件删除',
    '检测 Tetragon 文件事件中删除账号、sudo、SSH、计划任务等敏感路径的行为。',
    'file_access',
    true,
    'critical',
    94,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/(passwd|shadow|group|gshadow|sudoers|sudoers\\.d/|ssh/|crontab)|^/var/spool/cron/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"},{"field":"file_operation","op":"regex","value":"(?i)(unlink|unlinkat|rmdir|security_inode_unlink|security_inode_rmdir)"}]}'::jsonb,
    '["file-access","sensitive-file","delete"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
