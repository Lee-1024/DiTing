UPDATE diting_audit_rules
SET name = '反弹 Shell 命令',
    description = '检测 bash -i、nc -e、/dev/tcp 等常见反弹 Shell 行为。',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000101'
  AND name = 'Reverse shell command';

UPDATE diting_audit_rules
SET name = '下载并执行脚本',
    description = '检测 curl、wget 下载后通过 sh/bash 执行远程脚本的行为。',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000102'
  AND name = 'Download and execute';

UPDATE diting_audit_rules
SET name = '权限切换命令',
    description = '检测 sudo、su、passwd 等权限切换或账号相关命令。',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000103'
  AND name = 'Privilege switch command';

UPDATE diting_audit_rules
SET name = '敏感文件访问',
    description = '检测访问 /etc/shadow、/etc/passwd、SSH 私钥或 authorized_keys 等敏感文件的命令。',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000104'
  AND name = 'Sensitive file access';

UPDATE diting_audit_rules
SET name = '危险权限变更',
    description = '检测 chmod 777、递归放宽权限或修改 root 属主等高风险权限变更。',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000105'
  AND name = 'Dangerous permission change';

UPDATE diting_audit_rules
SET name = '容器控制命令',
    description = '检测 docker、kubectl、ctr、crictl 等容器管理命令。',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000106'
  AND name = 'Container control command';
