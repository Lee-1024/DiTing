INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES
(
    '00000000-0000-0000-0000-000000000306',
    'Tetragon 拦截策略触发',
    '检测所有带 diting-enforcement 标签的 Tetragon 进程类拦截事件，用于风险沉淀和右上角通知。',
    'process_kprobe',
    true,
    'critical',
    98,
    '{"operator":"and","conditions":[{"field":"tags","op":"contains","value":"diting-enforcement"}]}'::jsonb,
    '["tetragon","enforcement","blocked"]'::jsonb,
    NOW(),
    NOW()
),
(
    '00000000-0000-0000-0000-000000000307',
    'Tetragon 文件拦截策略触发',
    '检测所有带 diting-enforcement 标签的 Tetragon 文件、权限和删除拦截事件，用于风险沉淀和右上角通知。',
    'file_access',
    true,
    'critical',
    98,
    '{"operator":"and","conditions":[{"field":"tags","op":"contains","value":"diting-enforcement"}]}'::jsonb,
    '["tetragon","enforcement","blocked","file-access"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
