INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000101',
    'Reverse shell command',
    'Detects common reverse shell command patterns.',
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
    'Download and execute',
    'Detects shell pipelines that download and execute remote scripts.',
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
    'Privilege switch command',
    'Detects sudo, su, and passwd command execution.',
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
    'Sensitive file access',
    'Detects access to sensitive Linux account and SSH files.',
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
    'Dangerous permission change',
    'Detects broad permission changes such as chmod 777.',
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
    'Container control command',
    'Detects docker and kubectl command execution on audited hosts.',
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
