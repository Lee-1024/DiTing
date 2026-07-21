INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000401',
    'Shell 下载工具外联链路',
    '检测 bash/sh 等 Shell 父进程拉起 curl、wget、nc、ncat、socat、telnet 等工具发起网络连接。',
    'network_connect',
    true,
    'high',
    85,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"parent_process_name","op":"in","values":["bash","sh","dash","zsh"]},{"field":"process_name","op":"in","values":["curl","wget","nc","ncat","socat","telnet"]}]}'::jsonb,
    '["process-chain","network","download-tool"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000402',
    'Web 服务拉起 Shell',
    '检测 nginx、apache、httpd 等 Web 服务父进程拉起 sh、bash、dash、zsh 的可疑链路。',
    'process_exec',
    true,
    'high',
    88,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"process_exec"},{"field":"parent_process_name","op":"in","values":["nginx","apache","apache2","httpd"]},{"field":"process_name","op":"in","values":["sh","bash","dash","zsh"]}]}'::jsonb,
    '["process-chain","webshell"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000403',
    'Shell 拉起解释器外联',
    '检测 bash/sh 等 Shell 父进程拉起 python、perl、php、ruby、node 等解释器发起网络连接。',
    'network_connect',
    true,
    'high',
    82,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"parent_process_name","op":"in","values":["bash","sh","dash","zsh"]},{"field":"process_name","op":"in","values":["python","python3","perl","php","ruby","node"]}]}'::jsonb,
    '["process-chain","network","interpreter"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
