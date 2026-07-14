# DiTing Development Runbook

## Quick Start

Start API, Collector, and Web together:

```powershell
.\scripts\start-dev.ps1 -Config .\backend\configs\config.yaml
```

Start API and Web only:

```powershell
.\scripts\start-dev.ps1 -Config .\backend\configs\config.yaml -SkipCollector
```

Use another frontend port when `5173` is occupied:

```powershell
.\scripts\start-dev.ps1 -Config .\backend\configs\config.yaml -WebPort 5174
```

Stop development processes:

```powershell
.\scripts\stop-dev.ps1
```

Logs are written to `logs/*.out.log` and `logs/*.err.log`.

Build backend and frontend:

```powershell
.\scripts\build.ps1
```

## Backend

Run tests from the backend directory:

```bash
cd backend
go test ./...
```

If Go tries to write cache files outside the workspace, use workspace-local cache variables:

```bash
cd backend
set GOCACHE=E:\goProject\DiTing\.cache\go-build
set GOMODCACHE=E:\goProject\DiTing\.cache\gomod
set GOTELEMETRY=off
set GOENV=E:\goProject\DiTing\.cache\goenv
go test ./...
```

Run API server:

```bash
cd backend
go run ./cmd/audit-server api --config ./configs/config.yaml
```

Health check:

```bash
curl http://127.0.0.1:8080/healthz
```

## Databases

Start PostgreSQL and ClickHouse:

```bash
docker compose -f deploy/docker-compose.yaml up -d
```

The compose file mounts migrations into container init directories:

- PostgreSQL: `backend/migrations/postgres`
- ClickHouse: `backend/migrations/clickhouse`

## Tetragon File Input

The default collector file path in `backend/configs/config.example.yaml` is:

```text
/data/tetragon/logs/tetragon.log
```

Your Docker-installed Tetragon should export JSON events to this path or the config should be changed to match the real path.

The collector also uses `collector.passwd_file` to map Linux UID/AUID values to usernames. On a Linux server this should normally be:

```text
/etc/passwd
```

When developing locally from a copied server log, copy the audited server's `/etc/passwd` to:

```text
backend/sample-events/passwd
```

Then set `collector.passwd_file` in `backend/configs/config.yaml` to `./sample-events/passwd`. The copied passwd file is ignored by git.

Run one import pass for a copied sample log:

```bash
cd backend
go run ./cmd/audit-server collector-once --config ./configs/config.yaml
```

Run continuous tail mode for a live log file:

```bash
cd backend
go run ./cmd/audit-server collector --config ./configs/config.yaml
```

Initialize ClickHouse schema:

```bash
cd backend
go run ./cmd/audit-server migrate-clickhouse --config ./configs/config.yaml
```

## Frontend

Install dependencies:

```bash
cd frontend
npm install
```

Build:

```bash
npm run build
```

Run development server:

```bash
npm run dev
```

The Vite dev server proxies `/api` and `/healthz` to `http://127.0.0.1:8089`.

## Production

See `docs/production-deployment.md`.
# Linux 实时数据测试启动

在 Linux 服务器上测试实时 Tetragon 日志时，先确认 `backend/configs/config.yaml` 中：

- `collector.tetragon_log_file` 指向真实日志，例如 `/data/tetragon/logs/tetragon.log`
- `collector.passwd_file` 指向服务器的 passwd 快照，例如 `/data/tetragon/passwd`
- `collector.host_id` 设置为稳定主机 ID，例如 `/etc/machine-id` 的值或自定义资产编号
- `collector.host_name` 设置为页面展示名称，例如 `app-prod-01`
- ClickHouse 和 PostgreSQL 地址可从服务器访问

启动实时测试服务：

```bash
chmod +x scripts/start-linux.sh scripts/stop-linux.sh
scripts/start-linux.sh --config backend/configs/config.yaml --web-port 5174
```

这个脚本会启动 Vite dev server，因此 `/api` 会按 `frontend/vite.config.ts` 代理到本机 `8089` API。

首次部署或表结构变更后可带迁移：

```bash
scripts/start-linux.sh --config backend/configs/config.yaml --web-port 5174 --migrate
```

也可以单独执行迁移脚本：

```bash
chmod +x scripts/migrate-linux.sh
scripts/migrate-linux.sh --config backend/configs/config.yaml
```

只执行 ClickHouse 迁移：

```bash
scripts/migrate-linux.sh --config backend/configs/config.yaml --only clickhouse
```

停止：

```bash
scripts/stop-linux.sh --web-port 5174
```

日志位置：

```text
logs/api.out.log
logs/api.err.log
logs/collector.out.log
logs/collector.err.log
logs/web.out.log
logs/web.err.log
```
