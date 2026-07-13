ALTER TABLE diting.audit_events ADD COLUMN IF NOT EXISTS auid UInt32 AFTER username;
ALTER TABLE diting.audit_events ADD COLUMN IF NOT EXISTS euid UInt32 AFTER auid;
ALTER TABLE diting.audit_events ADD COLUMN IF NOT EXISTS egid UInt32 AFTER euid;
ALTER TABLE diting.audit_events ADD COLUMN IF NOT EXISTS login_username String AFTER egid;
