# 操作日志审计平台技术设计文档

## 1. 文档目标

本文档用于指导操作日志审计平台的第一版开发。

平台底座依靠 Tetragon 的内核级安全审计和执行引擎，审计明细数据存储在 ClickHouse，业务配置类数据存储在 PostgreSQL，后端使用 Go，前端使用 React。

第一版目标不是一次性做完整安全运营平台，而是先做出稳定可用的审计闭环：

```text
Tetragon 采集事件
  -> Go Collector 标准化事件
  -> ClickHouse 存储审计明细
  -> PostgreSQL 存储规则、用户、配置
  -> Go API Server 提供查询和管理接口
  -> React Web 展示日志、详情、统计和规则
```

## 2. 建设范围

### 2.1 第一版范围

第一版优先完成以下能力：

- 接入 Tetragon 进程执行事件。
- 将 Tetragon 原始事件转换为统一审计事件模型。
- 批量写入 ClickHouse。
- 使用 PostgreSQL 存储用户、角色、规则、系统配置。
- 提供操作日志查询接口。
- 提供事件详情接口。
- 提供基础统计接口。
- 提供规则管理接口。
- React 前端支持日志检索、事件详情、首页概览、规则管理。
- 支持 Docker Compose 本地开发环境。

### 2.2 暂不进入第一版的能力

以下能力可以作为后续迭代：

- Tetragon 阻断策略下发。
- 文件访问事件深度审计。
- 网络连接事件深度审计。
- DNS 审计。
- 复杂聚合告警。
- 多租户。
- 多集群统一管理。
- 审计报告自动生成。
- SOAR / SIEM 对接。

## 3. 技术选型和理由

### 3.1 Tetragon

Tetragon 负责内核级运行时事件采集。

选择理由：

- 基于 eBPF，可以在内核层捕获进程执行、文件访问、网络连接等行为。
- 事件可信度高，不依赖业务应用主动打日志。
- 适合 Kubernetes 和容器环境，可以携带 Pod、Namespace、Container 等元数据。
- 支持 TracingPolicy，后续可以按需扩展采集范围。
- 后续可以从审计扩展到执行控制，例如阻断高危命令。

第一版建议只采集 `process_exec` 相关事件，把命令执行审计做扎实。

### 3.2 ClickHouse

ClickHouse 存储审计明细日志。

选择理由：

- 审计日志是典型的追加写入数据，ClickHouse 写入吞吐高。
- 列式存储适合按时间、主机、事件类型、Namespace、命令等字段查询。
- 压缩率好，适合保存大量日志。
- 聚合性能好，可以直接做趋势、TOP N、分布统计。
- TTL 能力适合日志自动过期清理。

ClickHouse 只存高吞吐审计明细和必要的统计数据，不承担用户、规则、权限等业务配置数据。

### 3.3 PostgreSQL

PostgreSQL 存储业务配置类数据。

选择理由：

- 用户、角色、规则、系统配置属于强结构化业务数据。
- 这些数据需要事务、唯一约束、外键关系、更新和删除能力。
- PostgreSQL 查询表达能力强，适合管理后台业务数据。
- 将业务配置和审计明细拆开，可以避免 ClickHouse 承担不适合的 OLTP 场景。

PostgreSQL 存储范围：

- 用户账号。
- 角色权限。
- 审计规则。
- 系统配置。
- 登录会话或刷新令牌。
- 告警配置。
- 处置备注。

### 3.4 Go

Go 用于 Collector 和 API Server。

选择理由：

- 并发模型简单，适合持续消费 Tetragon 事件流。
- 性能稳定，适合日志接入和批量写入。
- 部署简单，可以编译为单二进制。
- 云原生生态成熟，和 Kubernetes、gRPC、ClickHouse 结合方便。

推荐库：

- HTTP 框架：Gin。
- ClickHouse 驱动：clickhouse-go/v2。
- PostgreSQL 驱动和 ORM：pgx + sqlc，或者 GORM。
- 配置：Viper。
- 日志：zap。
- 认证：JWT。
- 参数校验：go-playground/validator。
- API 文档：swaggo 或 OpenAPI。

建议第一版优先使用 `pgx` 或 `GORM`。如果想提高 SQL 可控性，用 `pgx + sqlc`；如果想开发速度快，用 `GORM`。

### 3.5 React

React 用于管理后台。

选择理由：

- 适合构建复杂筛选、表格、详情抽屉、图表看板。
- 组件生态成熟。
- 和 Ant Design 结合可以快速做出可用后台。

推荐栈：

- React。
- TypeScript。
- Vite。
- Ant Design。
- React Router。
- Axios。
- Zustand。
- ECharts。
- dayjs。

## 4. 总体架构

```text
+---------------------+
| Linux / K8s Node     |
+----------+----------+
           |
           v
+---------------------+
| Tetragon             |
| eBPF event engine    |
+----------+----------+
           |
           | gRPC stream / JSON event
           v
+---------------------+
| Go Collector         |
| parse / normalize    |
| enrich / classify    |
+----+-----------+----+
     |           |
     | audit log | config query
     v           v
+----------+   +----------------+
|ClickHouse|   | PostgreSQL      |
|audit log |   | config / users  |
+-----+----+   +--------+-------+
      |                 |
      +--------+--------+
               |
               v
+---------------------+
| Go API Server        |
| query / rules / auth |
+----------+----------+
           |
           v
+---------------------+
| React Web Console    |
+---------------------+
```

## 5. 服务拆分

### 5.1 audit-collector

职责：

- 连接 Tetragon 事件源。
- 持续消费事件。
- 将 Tetragon 事件转换为内部 `AuditEvent`。
- 补充字段，例如风险等级、标签、规则命中结果。
- 批量写入 ClickHouse。
- 记录采集和写入指标。

第一版 Collector 可以独立进程运行，也可以先和 API Server 放在同一个 Go 项目中，通过不同启动命令区分：

```text
audit-server api
audit-server collector
```

### 5.2 audit-api

职责：

- 用户登录和认证。
- 审计日志查询。
- 审计事件详情。
- 首页统计。
- 规则管理。
- 用户和角色管理。
- 系统配置管理。

### 5.3 audit-web

职责：

- 登录页面。
- 首页看板。
- 操作日志检索。
- 事件详情。
- 规则管理。
- 用户管理。
- 系统配置。

## 6. 推荐项目目录

建议使用前后端分离的单仓库结构：

```text
DiTing/
  backend/
    cmd/
      audit-server/
        main.go
    internal/
      app/
      collector/
      config/
      clickhouse/
      postgres/
      auth/
      audit/
      rule/
      user/
      stats/
      server/
    migrations/
      postgres/
      clickhouse/
    configs/
      config.example.yaml
    go.mod
    go.sum

  frontend/
    src/
      api/
      app/
      components/
      layouts/
      pages/
        dashboard/
        audit-events/
        rules/
        users/
        settings/
      router/
      stores/
      types/
      utils/
    package.json
    vite.config.ts

  deploy/
    docker-compose.yaml
    clickhouse/
    postgres/
    tetragon/

  docs/
    operation-audit-platform-technical-design.md
```

## 7. 数据流设计

### 7.1 事件采集链路

```text
Tetragon event
  -> Collector receives event
  -> Parse event type
  -> Convert to AuditEvent
  -> Load enabled rules from PostgreSQL cache
  -> Match rules
  -> Fill severity / risk_score / tags
  -> Batch write to ClickHouse
```

### 7.2 查询链路

```text
React query form
  -> Go API validates query params
  -> Build ClickHouse SQL with required time range
  -> Query audit_events
  -> Return paged list
  -> React table renders result
```

### 7.3 规则管理链路

```text
React rule form
  -> Go API validates rule
  -> PostgreSQL saves rule
  -> Collector periodically refreshes enabled rules
  -> New events use latest rule set
```

## 8. ClickHouse 设计

### 8.1 audit_events 表

第一版使用统一大宽表，便于前端和接口统一查询。

```sql
CREATE TABLE IF NOT EXISTS audit_events
(
    event_id String,
    event_time DateTime64(3),
    event_date Date,
    ingest_time DateTime64(3),

    event_type LowCardinality(String),
    action LowCardinality(String),
    severity LowCardinality(String),
    risk_score UInt8,
    tags Array(String),

    host_name String,
    host_ip String,
    node_name String,

    namespace String,
    pod_name String,
    container_id String,
    container_name String,
    image String,

    pid UInt32,
    ppid UInt32,
    process_name String,
    binary_path String,
    cmdline String,
    cwd String,

    parent_process_name String,
    parent_binary_path String,
    parent_cmdline String,

    uid UInt32,
    gid UInt32,
    username String,

    file_path String,
    file_operation LowCardinality(String),

    src_ip String,
    src_port UInt16,
    dst_ip String,
    dst_port UInt16,
    protocol LowCardinality(String),
    domain String,

    rule_ids Array(String),
    rule_names Array(String),

    raw_event String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_date, event_type, severity, host_name, namespace, pod_name, event_time)
TTL event_date + INTERVAL 90 DAY;
```

字段说明：

- `event_id`：后端生成的唯一事件 ID。
- `event_time`：事件发生时间。
- `ingest_time`：平台入库时间。
- `event_type`：事件类型，例如 `process_exec`。
- `action`：行为，例如 `exec`、`open`、`connect`。
- `severity`：风险等级。
- `risk_score`：风险分数，0 到 100。
- `tags`：事件标签。
- `rule_ids`：命中的规则 ID。
- `raw_event`：原始 Tetragon JSON，便于排查。

### 8.2 查询约束

API 查询 ClickHouse 时必须有时间范围。

默认查询：

- 默认开始时间：当前时间前 24 小时。
- 默认结束时间：当前时间。
- 默认分页大小：50。
- 最大分页大小：500。

避免前端直接触发无条件全表扫描。

## 9. PostgreSQL 设计

### 9.1 users 表

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    email VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

### 9.2 roles 表

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

### 9.3 user_roles 表

```sql
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
```

### 9.4 audit_rules 表

```sql
CREATE TABLE audit_rules (
    id UUID PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_type VARCHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    severity VARCHAR(32) NOT NULL,
    risk_score INTEGER NOT NULL,
    match_expr JSONB NOT NULL,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_audit_rules_event_type ON audit_rules(event_type);
CREATE INDEX idx_audit_rules_enabled ON audit_rules(enabled);
```

`match_expr` 示例：

```json
{
  "operator": "and",
  "conditions": [
    {
      "field": "cmdline",
      "op": "contains",
      "value": "bash -i"
    },
    {
      "field": "event_type",
      "op": "eq",
      "value": "process_exec"
    }
  ]
}
```

### 9.5 system_configs 表

```sql
CREATE TABLE system_configs (
    key VARCHAR(128) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL
);
```

## 10. 后端接口设计

### 10.1 认证接口

```text
POST /api/v1/auth/login
GET  /api/v1/auth/me
POST /api/v1/auth/logout
```

登录请求：

```json
{
  "username": "admin",
  "password": "admin123"
}
```

登录响应：

```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "username": "admin",
    "displayName": "管理员",
    "roles": ["admin"]
  }
}
```

### 10.2 审计日志查询

```text
GET /api/v1/audit/events
```

查询参数：

```text
start_time=2026-07-09T00:00:00+08:00
end_time=2026-07-09T23:59:59+08:00
event_type=process_exec
severity=high
host_name=node-1
namespace=default
pod_name=nginx-xxx
username=root
keyword=bash
page=1
page_size=50
```

响应：

```json
{
  "items": [
    {
      "eventId": "evt_01",
      "eventTime": "2026-07-09T10:00:00+08:00",
      "eventType": "process_exec",
      "severity": "high",
      "riskScore": 80,
      "hostName": "node-1",
      "namespace": "default",
      "podName": "nginx-xxx",
      "username": "root",
      "processName": "bash",
      "cmdline": "bash -i",
      "tags": ["reverse-shell"]
    }
  ],
  "page": 1,
  "pageSize": 50,
  "total": 1
}
```

### 10.3 审计事件详情

```text
GET /api/v1/audit/events/{event_id}
```

返回内容包括：

- 基础事件信息。
- 主机信息。
- 容器信息。
- 进程信息。
- 父进程信息。
- 用户信息。
- 命中的规则。
- 原始事件 JSON。

### 10.4 统计接口

```text
GET /api/v1/stats/overview
GET /api/v1/stats/event-trend
GET /api/v1/stats/top-hosts
GET /api/v1/stats/top-commands
GET /api/v1/stats/top-namespaces
```

### 10.5 规则接口

```text
GET    /api/v1/rules
POST   /api/v1/rules
GET    /api/v1/rules/{id}
PUT    /api/v1/rules/{id}
DELETE /api/v1/rules/{id}
```

创建规则请求：

```json
{
  "name": "反弹 shell 命令",
  "description": "检测 bash -i 等常见反弹 shell 行为",
  "eventType": "process_exec",
  "enabled": true,
  "severity": "high",
  "riskScore": 85,
  "matchExpr": {
    "operator": "or",
    "conditions": [
      {
        "field": "cmdline",
        "op": "contains",
        "value": "bash -i"
      },
      {
        "field": "cmdline",
        "op": "contains",
        "value": "nc -e"
      }
    ]
  },
  "tags": ["reverse-shell", "suspicious-command"]
}
```

## 11. 规则匹配设计

第一版规则引擎不要做太复杂，支持以下操作符即可：

- `eq`
- `neq`
- `contains`
- `prefix`
- `suffix`
- `in`
- `regex`

支持逻辑关系：

- `and`
- `or`

规则匹配在 Collector 内完成。Collector 每 30 秒从 PostgreSQL 刷新一次启用规则，缓存在内存中。

如果规则刷新失败，Collector 继续使用上一版规则，并记录错误日志。

## 12. 前端页面设计

### 12.1 登录页

功能：

- 用户名密码登录。
- 登录成功保存 JWT。
- 请求接口时自动带上 Authorization Header。

### 12.2 首页看板

模块：

- 今日事件总数。
- 高危事件数。
- 活跃主机数。
- 活跃 Namespace 数。
- 事件趋势图。
- 风险等级分布。
- TOP 主机。
- TOP 命令。
- 最近高危事件。

### 12.3 操作日志页面

核心能力：

- 时间范围筛选。
- 事件类型筛选。
- 风险等级筛选。
- 主机筛选。
- Namespace / Pod 筛选。
- 用户筛选。
- 关键字搜索。
- 表格分页。
- 点击打开事件详情抽屉。

### 12.4 事件详情抽屉

展示：

- 基础信息。
- 主机信息。
- 容器信息。
- 进程树信息。
- 命令行。
- 规则命中。
- 原始 JSON。

### 12.5 规则管理页面

功能：

- 规则列表。
- 新建规则。
- 编辑规则。
- 启用 / 禁用规则。
- 删除规则。
- 规则测试。

第一版规则测试可以只在前端做简单 JSON 输入和后端返回是否匹配。

## 13. 本地开发环境

推荐使用 Docker Compose 启动依赖：

```text
PostgreSQL
ClickHouse
audit-api
audit-web
```

Tetragon 在本地开发时可以先不强依赖，使用样例 JSON 文件模拟事件输入。

开发阶段建议准备：

- `sample-events/process_exec.json`
- `sample-events/sensitive_command.json`
- `sample-events/container_exec.json`

Collector 支持两种输入模式：

```yaml
collector:
  input_mode: file
  sample_file: ./sample-events/process_exec.json
```

```yaml
collector:
  input_mode: tetragon_grpc
  tetragon_endpoint: 127.0.0.1:54321
```

## 14. 第一版开发顺序

### 14.1 后端基础工程

- 初始化 Go module。
- 加载配置文件。
- 初始化 zap 日志。
- 初始化 PostgreSQL 连接。
- 初始化 ClickHouse 连接。
- 启动 Gin HTTP Server。
- 提供健康检查接口。

### 14.2 数据库迁移

- 编写 PostgreSQL migration。
- 编写 ClickHouse migration。
- 提供本地初始化脚本。

### 14.3 审计事件模型

- 定义 `AuditEvent` 结构。
- 定义 Tetragon 事件解析器接口。
- 实现样例 JSON 解析。
- 实现 ClickHouse 批量写入。

### 14.4 日志查询接口

- 实现查询参数校验。
- 实现 ClickHouse 查询构造。
- 实现分页返回。
- 实现事件详情查询。

### 14.5 规则管理

- 实现规则 CRUD。
- 实现规则表达式结构。
- 实现 Collector 内存规则缓存。
- 实现规则匹配。

### 14.6 前端基础工程

- 初始化 React + TypeScript + Vite。
- 接入 Ant Design。
- 接入 React Router。
- 接入 Axios。
- 实现登录页和基础布局。

### 14.7 前端审计页面

- 实现日志查询表单。
- 实现日志表格。
- 实现事件详情抽屉。
- 实现分页和筛选。

### 14.8 首页和规则页面

- 实现首页统计卡片和图表。
- 实现规则列表。
- 实现规则创建和编辑表单。

## 15. 测试策略

### 15.1 后端测试

重点测试：

- Tetragon 事件解析。
- `AuditEvent` 字段映射。
- 规则匹配。
- 查询参数校验。
- ClickHouse SQL 构造。
- PostgreSQL 规则 CRUD。

### 15.2 前端测试

重点测试：

- 查询参数是否正确传给 API。
- 日志表格是否正确渲染。
- 详情抽屉是否展示完整字段。
- 规则表单校验是否有效。

### 15.3 集成测试

准备样例事件：

- 普通命令执行。
- root 用户执行命令。
- 容器内执行 shell。
- 可疑命令执行。
- 命中规则的高危命令。

验证链路：

```text
sample event
  -> collector
  -> ClickHouse
  -> API query
  -> Web table
```

## 16. 风险和处理

### 16.1 ClickHouse 查询慢

处理方式：

- 强制时间范围。
- 控制分页大小。
- 优化 `ORDER BY` 字段。
- 高频统计可以加物化视图。

### 16.2 Tetragon 事件量过大

处理方式：

- 第一版只采集必要事件。
- Collector 批量写入。
- 增加本地缓冲。
- 对低价值事件采样。

### 16.3 规则匹配拖慢 Collector

处理方式：

- 规则缓存在内存。
- 先按 `event_type` 分组规则。
- 简单操作符优先。
- 正则规则限制数量。

### 16.4 原始事件字段变化

处理方式：

- 保留 `raw_event`。
- 解析器对缺失字段给默认值。
- 解析失败记录错误日志和失败计数。

### 16.5 PostgreSQL 和 ClickHouse 数据不一致

审计事件不依赖 PostgreSQL 事务。规则命中结果写入 ClickHouse 后就是事件当时的判断结果。规则后续修改不反向更新历史事件。

## 17. MVP 验收标准

MVP 完成时应满足：

- 可以通过样例 Tetragon JSON 导入进程执行事件。
- ClickHouse 中可以查询到标准化后的审计事件。
- PostgreSQL 中可以管理规则。
- Collector 可以根据规则给事件打风险等级和标签。
- 前端可以登录。
- 前端可以查询操作日志。
- 前端可以查看事件详情。
- 前端可以查看基础统计。
- 前端可以新增、编辑、启用、禁用规则。

## 18. 后续演进

第二阶段：

- 接入 Tetragon gRPC。
- 增加文件访问事件。
- 增加网络连接事件。
- 增加 DNS 事件。
- 增加告警通知。

第三阶段：

- 多集群管理。
- 审计报告。
- 操作处置流程。
- 策略命中分析。
- 和企业微信、钉钉、邮件集成。

第四阶段：

- Tetragon enforcement 策略下发。
- 高危命令阻断。
- 敏感文件访问阻断。
- 策略灰度和回滚。

