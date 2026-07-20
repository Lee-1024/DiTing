INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000201',
    '高危端口网络连接',
    '检测连接常见反连、代理、远控或非常规服务端口的网络行为。',
    'network_connect',
    true,
    'high',
    80,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"protocol","op":"eq","value":"tcp"},{"field":"dst_port","op":"in","values":["4444","5555","6666","7777","8888","9999","31337"]}]}'::jsonb,
    '["network","suspicious-port"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000202',
    '命令解释器发起网络连接',
    '检测 bash、sh、python、perl、php 等解释器进程直接发起网络连接。',
    'network_connect',
    true,
    'high',
    85,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"process_name","op":"in","values":["bash","sh","dash","zsh","python","python3","perl","php","ruby","node"]}]}'::jsonb,
    '["network","interpreter"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000203',
    '下载工具发起网络连接',
    '检测 curl、wget、nc、ncat、telnet 等工具发起网络连接。',
    'network_connect',
    true,
    'medium',
    60,
    '{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"process_name","op":"in","values":["curl","wget","nc","ncat","telnet","socat"]}]}'::jsonb,
    '["network","tool"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
