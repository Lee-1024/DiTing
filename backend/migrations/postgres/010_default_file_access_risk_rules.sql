INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000301',
    '敏感文件探针访问',
    '检测 Tetragon 文件访问事件中读取或打开 /etc/passwd、/etc/shadow、sudoers、SSH 配置与密钥等敏感路径。',
    'file_access',
    true,
    'high',
    80,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"in","values":["/etc/passwd","/etc/shadow","/etc/sudoers","/etc/group","/etc/gshadow","/etc/ssh/sshd_config"]}]}'::jsonb,
    '["file-access","sensitive-file"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000302',
    'SSH 敏感目录访问',
    '检测 Tetragon 文件访问事件中访问 SSH 配置、密钥或授权文件路径。',
    'file_access',
    true,
    'high',
    82,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"file_access"},{"field":"file_path","op":"regex","value":"(^/etc/ssh/|^/root/\\.ssh/|^/home/[^/]+/\\.ssh/)"}]}'::jsonb,
    '["file-access","ssh","credential"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
