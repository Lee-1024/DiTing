# DiTing Production Deployment

## 1. Components

Production deployment includes:

- Tetragon: runs on audited Linux hosts and exports JSON events.
- DiTing API: serves REST APIs and authentication.
- DiTing Collector: tails Tetragon log and writes audit events to ClickHouse.
- PostgreSQL: stores users, roles, rules, and system configuration.
- ClickHouse: stores audit event details and supports analytics.
- Nginx: serves React static files and proxies API traffic.

## 2. Build

On the development machine:

```powershell
.\scripts\build.ps1
```

Artifacts:

- Backend binary: `backend/bin/audit-server.exe` on Windows builds.
- Frontend static files: `frontend/dist`.

For Linux production, build on Linux or cross-compile:

```bash
cd backend
GOOS=linux GOARCH=amd64 go build -o audit-server ./cmd/audit-server
cd ../frontend
npm run build
```

## 3. Server Layout

Recommended Linux paths:

```text
/opt/diting/bin/audit-server
/opt/diting/backend/migrations/
/opt/diting/web/
/etc/diting/config.yaml
/var/log/diting/
```

Create service user:

```bash
useradd --system --home /opt/diting --shell /sbin/nologin diting
mkdir -p /opt/diting/bin /opt/diting/backend /opt/diting/web /etc/diting /var/log/diting
chown -R diting:diting /opt/diting /var/log/diting
```

## 4. Config

Use `/etc/diting/config.yaml`:

```yaml
server:
  port: 8089
  mode: release

jwt:
  secret: replace-with-a-long-random-secret
  expires_hours: 24

postgres:
  host: 10.54.56.54
  port: 31060
  database: myappdb
  username: admin
  password: secure_password
  ssl_mode: disable

clickhouse:
  addr: 10.40.0.184:9002
  database: diting
  username: admin
  password: admin123456

collector:
  input_mode: file
  tetragon_log_file: /data/tetragon/logs/tetragon.log
  passwd_file: /etc/passwd
  host_id: app-prod-01
  host_name: app-prod-01
  flush_interval_seconds: 1
  batch_size: 1000
```

Change `jwt.secret` before production use.

## 5. Database Initialization

Run once after copying the backend binary and migrations:

```bash
cd /opt/diting/backend
/opt/diting/bin/audit-server migrate-postgres --config /etc/diting/config.yaml
/opt/diting/bin/audit-server migrate-clickhouse --config /etc/diting/config.yaml
```

Default login after migration:

```text
username: admin
password: admin123
```

Change the password before exposing the system.

## 6. Tetragon

Tetragon container example:

```yaml
services:
  tetragon:
    image: quay.io/cilium/tetragon:v1.7.0
    container_name: tetragon
    privileged: true
    pid: host
    cgroup: host
    restart: unless-stopped
    volumes:
      - /data/tetragon/logs:/data/tetragon/logs
      - /sys/kernel:/sys/kernel
      - /sys/kernel/btf/vmlinux:/var/lib/tetragon/btf:ro
    command:
      - /usr/bin/tetragon
      - --export-filename
      - /data/tetragon/logs/tetragon.log
      - --enable-process-cred
```

Verify:

```bash
tail -f /data/tetragon/logs/tetragon.log
```

`--enable-process-cred` is required for UID/EUID credential fields. DiTing also reads `/etc/passwd` through `collector.passwd_file` so audit records can show both the login user (`auid`) and the execution user (`uid/euid`).

## 7. systemd

Copy service files:

```bash
cp deploy/systemd/diting-api.service /etc/systemd/system/
cp deploy/systemd/diting-collector.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now diting-api
systemctl enable --now diting-collector
```

Check:

```bash
systemctl status diting-api
systemctl status diting-collector
journalctl -u diting-api -f
journalctl -u diting-collector -f
```

## 8. Nginx

Copy frontend files:

```bash
cp -r frontend/dist/* /opt/diting/web/
```

Install Nginx config:

```bash
cp deploy/nginx/diting-web.conf /etc/nginx/conf.d/diting-web.conf
nginx -t
systemctl reload nginx
```

## 9. Smoke Tests

API:

```bash
curl http://127.0.0.1:8089/healthz
```

Login:

```bash
curl -X POST http://127.0.0.1:8089/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
```

Collector:

```bash
journalctl -u diting-collector -f
```

ClickHouse event count:

```bash
curl 'http://CLICKHOUSE_HOST:8123/?query=SELECT%20count()%20FROM%20diting.audit_events'
```
