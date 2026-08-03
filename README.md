# DiTing

DiTing 是一套面向 Linux 主机与容器环境的安全审计平台，用于采集 Tetragon 事件、执行审计规则匹配、沉淀操作日志、识别风险事件，并提供风险处置、采集过滤、服务状态监控、AI 风险复核和拦截策略管理能力。

项目当前主要用于上线前预演和生产预发初始化：重点解决采集数据量过大、风险噪声淹没有效告警、人工处置后同类噪声仍持续入库等问题。

## 功能概览

- **审计事件采集**：支持从 Tetragon 日志文件或 gRPC 读取事件。
- **操作日志分析**：按命令、用户、主机、规则命中等维度查看审计事件。
- **风险事件调查**：聚焦 `medium/high/critical` 风险，支持处置状态、主机、用户、事件类型、关键字筛选。
- **风险处置**：支持单条处理、批量处理、误报、忽略当前、忽略同类。
- **采集过滤**：支持按进程、命令、用户、登录用户、父进程、工作目录、文件、网络等条件过滤采集数据。
- **可信动作过滤**：可配置“某个来源执行某个高危动作是安全的”，用于减少监控/巡检类固定动作噪声。
- **AI 风险复核**：支持 MiniMax-M3 和 OpenAI-compatible 模型服务，对风险事件做人工处理前的辅助判断。
- **服务状态告警**：右上角只展示采集节点离线、采集异常、Tetragon 不可访问等服务状态问题。
- **拦截策略管理**：支持 Tetragon policy 下发与同步。
- **预发清库脚本**：支持清理采集运行数据和审计规则，同时保留采集配置、用户、角色、操作日志和拦截策略。

## 技术栈

### 后端

- Go
- PostgreSQL：用户、配置、规则、处置状态、AI 分析、服务状态等业务数据
- ClickHouse：高吞吐审计事件与聚合统计
- Redis：响应缓存和部分查询缓存，可按配置启用
- Tetragon：Linux 主机/容器安全事件来源

### 前端

- React 18
- TypeScript
- Vite
- Ant Design
- Axios
- ECharts

## 目录结构

```text
.
├── backend/                 # Go 后端
│   ├── cmd/audit-server/     # API、collector、migration 等命令入口
│   ├── configs/              # 配置样例和本地配置
│   ├── internal/             # 业务模块
│   └── migrations/           # PostgreSQL / ClickHouse 迁移
├── frontend/                 # React 前端
│   ├── src/
│   ├── public/
│   └── package.json
├── scripts/                  # 启停、迁移、清库、构建脚本
├── logs/                     # 运行日志
├── run/                      # 运行时 pid 文件，启动后生成
└── dist/                     # 构建产物
```

后端主要模块：

```text
backend/internal/audit             审计事件查询
backend/internal/collector         采集器输入、身份补全、规则匹配、写入
backend/internal/collectorhealth   采集节点健康状态
backend/internal/rule              审计规则
backend/internal/riskstatus        风险事件处置状态
backend/internal/riskanalysis      AI 风险复核
backend/internal/systemconfig      系统配置、采集过滤、AI 配置
backend/internal/enforcement       拦截策略
backend/internal/stats             统计分析
backend/internal/useradmin         用户管理
```

## 环境要求

开发和预发环境建议：

- Linux 主机用于运行采集器和 Tetragon
- Go 1.22+ 或项目当前 `go.mod` 兼容版本
- Node.js 18+
- npm
- PostgreSQL
- ClickHouse
- Redis，可选
- Tetragon，可选但生产采集需要

Windows 上可开发前端和后端代码；采集器、Tetragon 和 Linux 清库/启停脚本主要面向 Linux 环境。

## 配置

复制配置样例：

```bash
cp backend/configs/config.example.yaml backend/configs/config.yaml
```

核心配置项：

```yaml
server:
  port: 8080

jwt:
  secret: change-me

postgres:
  host: 127.0.0.1
  port: 5432
  database: diting
  username: diting
  password: diting

clickhouse:
  addr: 127.0.0.1:9000
  database: diting

redis:
  enabled: true
  addr: 127.0.0.1:6379

collector:
  input_mode: file
  output_mode: clickhouse
  ingest_url: http://127.0.0.1:8080/api/v1/ingest/events
  tetragon_log_file: /data/tetragon/logs/tetragon.log
  tetragon_grpc_addr: 127.0.0.1:54321
  host_id: server-001
  host_name: diting-test-host
  token: change-me-collector-token
```

说明：

- `server.port` 是后端 API 实际监听端口。
- `collector.input_mode` 支持 `file` 和 gRPC 相关输入模式，具体以当前代码配置解析为准。
- `collector.output_mode` 支持直接写 ClickHouse 或通过 API 写入。
- `collector.token` 用于采集器上报鉴权。
- `jwt.secret` 同时用于系统鉴权和 AI Key 加密派生密钥，生产环境必须修改。

## 启动

### Linux 一键启动

```bash
scripts/start-linux.sh --migrate
```

默认行为：

- 构建后端二进制到 `bin/audit-server`
- 可选执行 PostgreSQL 和 ClickHouse 迁移
- 启动 API
- 启动 Collector
- 启动前端 Vite

默认地址：

```text
API: http://127.0.0.1:8089/healthz
Web: http://127.0.0.1:5174
```

脚本常用参数：

```bash
scripts/start-linux.sh --config backend/configs/config.yaml
scripts/start-linux.sh --web-port 5174
scripts/start-linux.sh --api-port 8089
scripts/start-linux.sh --skip-collector
scripts/start-linux.sh --skip-frontend
scripts/start-linux.sh --migrate
```

停止：

```bash
scripts/stop-linux.sh
```

### 手动启动后端

```bash
cd backend
go run ./cmd/audit-server api --config configs/config.yaml
```

启动采集器：

```bash
cd backend
go run ./cmd/audit-server collector --config configs/config.yaml
```

### 手动启动前端

```bash
cd frontend
npm install
npm run dev -- --port 5174 --strictPort
```

前端默认通过 Vite proxy 转发 `/api` 到后端服务。

## 数据库迁移

执行全部迁移：

```bash
scripts/migrate-linux.sh --config backend/configs/config.yaml
```

只执行 PostgreSQL：

```bash
scripts/migrate-linux.sh --only postgres
```

只执行 ClickHouse：

```bash
scripts/migrate-linux.sh --only clickhouse
```

## 预发清库

上线预演时可清理采集运行数据和审计规则：

```bash
scripts/clear-test-data-linux.sh --config backend/configs/config.yaml --yes
```

会清理：

- ClickHouse `audit_events`
- ClickHouse `audit_*_hourly` 聚合表
- PostgreSQL AI 分析结果
- PostgreSQL 风险处置状态
- PostgreSQL 采集心跳
- PostgreSQL 主机资产
- PostgreSQL 审计规则

会保留：

- 采集配置
- 系统用户、角色
- 操作日志
- 拦截策略

这个脚本适合预发环境清理脏数据后重新做采集策略预演。生产环境使用前必须确认数据保留要求。

## 采集过滤建议

系统支持两类过滤：

1. **普通过滤**：用于过滤低价值普通噪声。
2. **可信动作过滤**：用于明确“某个来源执行某个动作是安全的”，例如监控 Agent 固定执行 Docker 查询命令。

生产初始化建议：

- root 用户不要简单全量过滤，应保留敏感文件、权限变更、网络连接、异常执行链等高价值行为。
- 普通用户采集应更严格，保留命令执行、敏感路径访问、提权、下载执行、反弹 Shell、容器逃逸相关行为。
- zabbix、prometheus、monitor-agent 等监控用户或进程可过滤常规巡检动作，但不要过滤它们执行的未知高危动作。
- 对固定安全动作，使用“可信动作过滤”：来源条件 + 动作条件必须同时满足。

示例思路：

```text
父进程名 = monitor-agent
命令行 包含 /usr/bin/docker ps
启用可信动作过滤
```

这样只过滤 `monitor-agent` 触发的固定 Docker 巡检，不会放过其他来源执行的 Docker 高危动作。

## 风险事件处置

风险事件页默认聚焦待处理风险，支持：

- 时间范围
- 风险等级
- 事件类型
- 用户
- 主机
- 关键字
- 处置状态

处置状态：

- `未处理`
- `已处理`
- `误报`
- `忽略当前`
- `忽略同类`
- `已关闭`

支持批量选择当前页事件并一键处理。处理后页面会自动刷新当前页，后续未处理事件会补位到当前页。

`忽略同类` 会基于风险指纹处理同类事件，适合确认固定巡检、固定监控动作不是风险后减少重复处理。

## AI 风险复核

AI 配置在前端 `配置管理 -> AI 配置` 中维护，API Key 会加密入库，前端不会回显明文。

当前支持：

- MiniMax-M3
- OpenAI-compatible Chat Completions 模型服务

MiniMax-M3 会走 MiniMax Responses API：

- Base URL 默认：`https://api.minimaxi.com/v1`
- Model 默认：`MiniMax-M3`
- 使用 `reasoning.effort = none`
- 读取 `output_text`

配置页支持测试模型服务是否可用。风险事件页支持单条 AI 分析和重分析，详情抽屉中可查看完整 AI 判断、证据、建议和原始输出。

AI 分析只做辅助判断，不自动处置、不自动降级风险。最终结果仍由人工确认。

## 服务状态告警

右上角 `服务状态` 只显示平台运行状态问题，不显示风险事件数量。

当前来源：

- 采集节点离线
- Collector 心跳超时
- 采集异常
- Tetragon 服务不可访问或最近错误
- 长时间未收到事件
- 长时间未写入

点击服务状态项会跳转到 `配置管理 -> 采集状态` 页面查看详情。

## 拦截策略

拦截策略通过配置管理页面维护，采集器可按配置同步 Tetragon policy。

相关配置：

```yaml
collector:
  enforcement_enabled: false
  enforcement_policy_dir: /data/tetragon/policies
  enforcement_sync_interval_seconds: 30
  tetragon_restart_command: docker compose -f /data/tetragon/docker-compose.yaml restart tetragon
```

生产启用前建议先在预发环境验证：

- policy 文件是否正确下发
- Tetragon 是否能加载策略
- 重启命令是否可执行
- 策略是否只拦截明确确认的高危行为

## 构建

后端测试：

```bash
cd backend
go test ./...
```

前端构建：

```bash
cd frontend
npm run build
```

构建 Linux 采集器：

```bash
scripts/build-collector-linux.sh --arch amd64 --test
scripts/build-collector-linux.sh --arch arm64
```

输出默认位于：

```text
dist/collector-linux-amd64
dist/collector-linux-arm64
```

## 日志

通过 `scripts/start-linux.sh` 启动时，日志写入：

```text
logs/api.out.log
logs/api.err.log
logs/collector.out.log
logs/collector.err.log
logs/web.out.log
logs/web.err.log
```

后端错误日志会同时写入 stderr，便于排查 `api.err.log`。

常见排查：

```bash
tail -f logs/api.err.log
tail -f logs/collector.err.log
tail -f logs/web.err.log
```

## 常见问题

### 前端能打开但接口 502/404

确认后端 API 是否启动、端口是否与 Vite proxy 一致：

```bash
curl http://127.0.0.1:8089/healthz
```

### 采集节点显示离线

检查：

- Collector 进程是否运行
- `collector.host_id` / `collector.host_name` 是否正确
- Collector 到 API 或 ClickHouse 是否可达
- Tetragon 日志文件路径或 gRPC 地址是否正确

### Tetragon 不可访问

检查：

- Tetragon 容器或 systemd 服务是否运行
- `collector.tetragon_log_file` 是否存在并持续写入
- `collector.tetragon_grpc_addr` 是否可连接
- Collector 运行用户是否有权限读取日志或访问 socket

### 风险事件数据量过大

优先处理：

- 调整采集过滤
- 对监控巡检动作配置可信动作过滤
- 降低 root 常规低价值事件采集
- 保留敏感文件、高危命令、异常网络和权限相关事件

不要简单按 root 全量过滤，否则会漏掉真实高危行为。

### AI 分析超时

在 `配置管理 -> AI 配置` 中调大超时秒数，或降低最大输出 Token。

MiniMax-M3 建议：

```text
Base URL: https://api.minimaxi.com/v1
Model: MiniMax-M3
Timeout: 120-300 秒
Max Tokens: 500-800
```

## 上线前检查清单

- [ ] PostgreSQL / ClickHouse 连接正常
- [ ] 数据库迁移完成
- [ ] JWT Secret 已修改
- [ ] Collector Token 已修改
- [ ] 采集节点服务状态正常
- [ ] Tetragon 事件可采集
- [ ] 采集过滤已按生产策略初始化
- [ ] 审计规则已按生产策略初始化
- [ ] 风险事件页面能按主机、用户、等级筛选
- [ ] 误报和忽略同类处置符合预期
- [ ] AI 配置测试通过
- [ ] 拦截策略在预发环境验证通过
- [ ] 清库脚本不会清理需要保留的系统配置

