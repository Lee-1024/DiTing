INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000101',
    '反弹 Shell 命令',
    '检测 bash -i、nc -e、/dev/tcp 等常见反弹 Shell 行为。',
    'process_exec',
    true,
    'critical',
    95,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"bash -i"},{"field":"cmdline","op":"contains","value":"nc -e"},{"field":"cmdline","op":"contains","value":"/dev/tcp/"},{"field":"cmdline","op":"contains","value":"python -c"},{"field":"cmdline","op":"contains","value":"perl -e"}]}'::jsonb,
    '["reverse-shell","critical-command"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000102',
    '下载并执行脚本',
    '检测 curl、wget 下载后通过 sh/bash 执行远程脚本的行为。',
    'process_exec',
    true,
    'critical',
    90,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"curl "},{"field":"cmdline","op":"contains","value":"wget "},{"field":"cmdline","op":"contains","value":"| sh"},{"field":"cmdline","op":"contains","value":"| bash"},{"field":"cmdline","op":"contains","value":"curl -fsSL"},{"field":"cmdline","op":"contains","value":"wget -qO-"}]}'::jsonb,
    '["download-exec","critical-command"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000103',
    '权限切换命令',
    '检测 sudo、su、passwd 等权限切换或账号相关命令。',
    'process_exec',
    true,
    'high',
    75,
    '{"operator":"or","conditions":[{"field":"process_name","op":"in","values":["sudo","su","passwd"]},{"field":"cmdline","op":"contains","value":"sudo "},{"field":"cmdline","op":"contains","value":"su -"}]}'::jsonb,
    '["privilege","account"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000104',
    '敏感文件访问',
    '检测访问 /etc/shadow、/etc/passwd、SSH 私钥或 authorized_keys 等敏感文件的命令。',
    'process_exec',
    true,
    'high',
    80,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"/etc/shadow"},{"field":"cmdline","op":"contains","value":"/etc/passwd"},{"field":"cmdline","op":"contains","value":"id_rsa"},{"field":"cmdline","op":"contains","value":"authorized_keys"}]}'::jsonb,
    '["sensitive-file","credential"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000105',
    '危险权限变更',
    '检测 chmod 777、递归放宽权限或修改 root 属主等高风险权限变更。',
    'process_exec',
    true,
    'high',
    70,
    '{"operator":"or","conditions":[{"field":"cmdline","op":"contains","value":"chmod 777"},{"field":"cmdline","op":"contains","value":"chmod -R 777"},{"field":"cmdline","op":"contains","value":"chown root"}]}'::jsonb,
    '["permission","hardening"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000106',
    '容器控制命令',
    '检测 docker、kubectl、ctr、crictl 等容器管理命令。',
    'process_exec',
    true,
    'high',
    65,
    '{"operator":"or","conditions":[{"field":"process_name","op":"in","values":["docker","kubectl","ctr","crictl"]},{"field":"cmdline","op":"contains","value":"docker "},{"field":"cmdline","op":"contains","value":"kubectl "}]}'::jsonb,
    '["container","admin-command"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
