# DiTing 当前状态与后续开发转接文档

更新时间：2026-07-14

## 1. 项目定位

DiTing 是一个基于 Tetragon 的 Linux 服务器操作日志审计平台。

当前架构：

```text
被审计 Linux 主机：
Tetragon -> tetragon.log -> DiTing Collector -> ClickHouse
                                      |
                                      -> PostgreSQL 读取审计规则

中心服务：
React 前端 -> Go API -> ClickHouse 查询审计明细和统计
                     -> PostgreSQL 查询用户、规则、处置状态、资产配置
```

当前技术栈：

- Tetragon：内核级进程执行审计数据来源
- Go：后端 API、Collector、迁移和清理命令
- React + Ant Design + ECharts：前端
- ClickHouse：审计事件明细和统计查询
- PostgreSQL：用户、规则、风险处置、主机资产等业务配置

## 2. 当前已实现功能

### 2.1 采集链路

- Collector 支持实时 tail Tetragon JSON 日志。
- Collector 启动时默认从文件末尾开始读，避免重启重复导入历史数据。
- 已支持日志文件被替换、截断、重建后的重新打开。
- 支持 `collector.passwd_file`，将 Linux `uid/auid/euid` 解析为用户名。
- 支持 `collector.host_id` 和 `collector.host_name`：
  - `host_id` 用作稳定主机标识。
  - `host_name` 用作页面展示名。
  - 未配置 `host_id` 时默认读 `/etc/machine-id`。
  - 未配置 `host_name` 时默认读系统 hostname。

### 2.2 审计事件

- 支持 `process_exec` 和 `process_exit` 事件解析。
- 审计事件写入 ClickHouse `audit_events`。
- 事件 ID 已改为基于原始事件行的稳定 hash，避免 exec/exit 事件 ID 冲突。
- 审计事件包含：
  - 时间
  - 主机 ID / 主机名 / Tetragon node_name
  - 登录用户 / 执行用户
  - 进程、命令、父进程
  - 命中规则 ID / 命中规则名称
  - 风险等级、风险分数、标签
  - 原始 Tetragon JSON

### 2.3 规则与风险

- PostgreSQL 存储审计规则。
- Collector 每 30 秒刷新启用规则。
- Collector 写入前进行规则匹配，命中后写入 ClickHouse。
- 风险事件页面支持：
  - 高危/严重事件筛选
  - 关键词筛选
  - 用户筛选
  - 处置状态：未处理、已确认、已忽略
- 风险处置状态存储在 PostgreSQL `diting_risk_dispositions`。

### 2.4 前端页面

已实现页面：

- 登录页
- 仪表盘
- 操作日志
- 风险事件
- 命令审计
- 用户审计
- 主机审计
- 规则管理
- 主机资产页

注意：

- 当前已经可以在 collector 配置中定义主机名，因此“主机资产页”在测试阶段可以暂时隐藏入口。
- 不建议直接删除主机资产能力，后续可升级为资产负责人、环境、IP、备注、归属部门等管理页面。

### 2.5 时间处理

- Tetragon JSON 时间通常是 UTC，例如 `2026-07-13T03:11:07Z`。
- ClickHouse 存储仍按 UTC 思路处理。
- 前端事件时间展示已统一格式化为浏览器本地时间，国内环境即 CST/UTC+8。
- 事件趋势接口已在后端 SQL 中按 `Asia/Shanghai` 小时桶聚合：

```sql
toStartOfHour(toTimeZone(event_time, 'Asia/Shanghai'))
```

### 2.6 Linux 脚本

已有脚本：

- `scripts/start-linux.sh`
  - 启动 API、Collector、前端
  - 使用 `setsid` 启动进程组
  - 记录 `run/*.pid` 和 `run/*.pgid`
- `scripts/stop-linux.sh`
  - 按进程组停止 API、Collector、前端
  - 对旧脚本残留的 Vite 5174 监听进程有谨慎清理逻辑
- `scripts/migrate-linux.sh`
  - 执行 PostgreSQL / ClickHouse 迁移
  - 支持 `--only postgres` 或 `--only clickhouse`
- `scripts/clear-test-data-linux.sh`
  - 测试阶段清理采集数据
  - 清空 ClickHouse `audit_events`
  - 清空 PostgreSQL `diting_risk_dispositions`
  - 不删除用户、规则、主机资产等配置

## 3. 关键配置

示例：

```yaml
server:
  port: 8089

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
  host_id: server-001
  host_name: diting-test-host
  flush_interval_seconds: 1
  batch_size: 1000
```

说明：

- 多台服务器部署时，每台服务器部署一套 Tetragon + Collector。
- API 和前端只需要中心部署一套。
- 所有 Collector 连接同一套 PostgreSQL 和 ClickHouse。
- Collector 当前不通过 API 上报，而是直接写 ClickHouse、读取 PostgreSQL 规则。

## 4. 常用命令

### 本地开发

```powershell
.\scripts\start-dev.ps1 -WebPort 5174
.\scripts\stop-dev.ps1
```

### Linux 启动

```bash
chmod +x scripts/start-linux.sh scripts/stop-linux.sh
scripts/start-linux.sh --config backend/configs/config.yaml --web-port 5174
```

### Linux 停止

```bash
scripts/stop-linux.sh --web-port 5174
```

### 执行迁移

```bash
chmod +x scripts/migrate-linux.sh
scripts/migrate-linux.sh --config backend/configs/config.yaml
```

只执行 ClickHouse 迁移：

```bash
scripts/migrate-linux.sh --config backend/configs/config.yaml --only clickhouse
```

### 清空测试数据后重采

```bash
chmod +x scripts/clear-test-data-linux.sh
scripts/stop-linux.sh --web-port 5174
scripts/clear-test-data-linux.sh --config backend/configs/config.yaml --yes
scripts/start-linux.sh --config backend/configs/config.yaml --web-port 5174
```

### 查看日志

```bash
tail -f logs/api.out.log
tail -f logs/api.err.log
tail -f logs/collector.out.log
tail -f logs/collector.err.log
tail -f logs/web.out.log
tail -f logs/web.err.log
```

## 5. 当前注意事项

### 5.1 测试数据污染

测试阶段已经多次更改主机标识、时间转换和规则命中逻辑，旧数据可能影响统计结果。

建议每次调整采集链路后执行：

```bash
scripts/clear-test-data-linux.sh --config backend/configs/config.yaml --yes
```

然后重新启动 Collector 采集。

### 5.2 主机资产页

当前 `collector.host_name` 已经可以定义展示主机名，因此主机资产页的 `nodeName/displayName` 能力和采集配置存在重叠。

建议下一步：

- 前端先隐藏“主机资产”菜单入口。
- 后端 API 和表暂时保留。
- 后续需要资产管理时，改成按 `host_id` 维护负责人、环境、IP、备注、部门等资产元数据。

### 5.3 Collector 直连数据库

当前 Collector 直接连接 ClickHouse 和 PostgreSQL。

优点：

- 简单，开发快。
- 多主机采集容易跑通。

风险：

- 每台被审计主机都需要配置数据库连接信息。
- 数据库网络和账号暴露面更大。

后续生产化建议改造为：

```text
Collector -> API Ingest 接口 -> ClickHouse
```

或：

```text
Collector -> Kafka/NATS -> 后端消费 -> ClickHouse
```

### 5.4 数据量增长

当前 ClickHouse `audit_events` 设置了 TTL：

```sql
TTL event_date + INTERVAL 90 DAY
```

后续建议：

- 明细事件保留 30/90 天。
- 新增按天聚合表，长期保存用户、主机、命令、风险等级统计。
- 再考虑采集侧过滤噪声命令。

## 6. 建议的后续开发计划

### 第一优先级：测试稳定性

1. 清空测试数据后重新采集。
2. 验证：
   - 实时日志是否持续写入。
   - `su ubuntu` 后执行命令是否能解析登录用户/执行用户。
   - 主机是否只显示一台。
   - 事件趋势时间是否为 CST。
   - 风险规则命中是否正常。

### 第二优先级：主机资产逻辑整理

1. 前端隐藏主机资产入口。
2. 主机审计页移除对资产 `displayName` 的依赖。
3. 后续若恢复资产页，改为按 `host_id` 管理资产。

### 第三优先级：采集过滤与降噪

1. 增加 collector 过滤配置：
   - 忽略进程名
   - 忽略命令关键词
   - 忽略用户
   - 保留高危/严重事件
2. 注意过滤规则默认关闭，避免误丢审计证据。

### 第四优先级：长期统计

1. 新增 ClickHouse 聚合表，例如 `audit_daily_stats`。
2. 新增定时聚合任务。
3. 首页趋势和统计可优先查聚合表。

### 第五优先级：生产化

1. Collector 改为上报 API，而不是直连数据库。
2. API 增加 ingest endpoint 和 collector token。
3. 增加 systemd 部署模板。
4. 增加日志轮转配置。
5. 增加健康检查和采集延迟监控。

## 7. 新会话接手建议

新会话开始时建议先做：

```powershell
git status --short
go test ./...
npm run build
```

然后优先处理：

1. 是否隐藏主机资产页。
2. 是否执行清库重采测试。
3. 是否新增采集降噪配置。
4. 是否把 Collector 通信方式从直连数据库改为 API ingest。

