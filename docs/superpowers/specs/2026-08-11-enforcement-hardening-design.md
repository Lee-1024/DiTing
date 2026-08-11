# 拦截能力强化完整规划

## 目标

将 DiTing 的拦截能力从当前“敏感文件编辑拦截”扩展为覆盖文件、命令、系统高危行为、容器运行时和策略运营闭环的分层防护体系。

设计目标不是一次性把所有行为都强制拦截，而是建立清晰的能力矩阵：哪些场景可以稳定拦截，哪些场景先审计再提升，哪些场景需要按主机内核和运行时能力降级。

## 当前状态

### 已具备能力

- 后端已持久化 `diting_enforcement_policies`，支持策略、部署状态和目标主机。
- 前端已有策略模板入口：`dangerous_command`、`sensitive_file`、`permission_change`、`delete_behavior`、`suspicious_process`。
- Collector 侧已实现 AppArmor 同步链路。
- 当前真正部署的强拦截能力只有 `sensitive_file + enforce`。
- Tetragon 主要承担观测、事件采集、策略触发证据和通知关联。
- 通知中心已支持拦截事件待处理闭环和 Collector/Tetragon 状态告警自动恢复。

### 主要限制

- 前端展示的多个模板尚未全部映射到 Collector 真实部署能力。
- Linux 5.4 环境下 Tetragon `Override` 类强拦截能力受内核和 LSM hook 支持限制。
- AppArmor 对文件访问控制稳定，但对命令参数、进程链、网络连接等语义表达有限。
- 当前 AppArmor 策略主要面向 sudo 及其子进程，直接 root 登录仍按既有设计放行。
- 拦截策略缺少统一的“主机能力评估、降级说明、模拟验证、误报反馈”运营层。

## 总体路线

采用混合引擎、分层防护：

- AppArmor 负责稳定强拦截，优先覆盖文件类和可用 profile 表达的操作。
- Tetragon 负责运行时观测、证据采集、风险识别、通知关联，以及高版本主机的可选强拦截增强。
- 后端统一策略模型、能力判断、部署状态、事件沉淀和处置闭环。
- 前端只展示当前主机可支持或可降级的能力，不再让“模板存在”误导为“已经能拦截”。

## 能力矩阵

| 能力域 | 典型场景 | 推荐引擎 | 首版模式 | 说明 |
| --- | --- | --- | --- | --- |
| 敏感文件读取 | 读取 `/etc/shadow`、密钥、kubeconfig | AppArmor + Tetragon 观测 | enforce | 稳定性最高，适合作为第一阶段扩展 |
| 敏感文件写入 | 编辑、覆盖、追加敏感配置 | AppArmor + Tetragon 观测 | enforce | 当前能力的自然延伸 |
| 文件删除 | `unlink`、`rmdir`、删除敏感目录 | AppArmor 优先，Tetragon 观测 | enforce | 避免依赖 `rm` 命令匹配 |
| 权限变更 | `chmod`、`chown`、`setxattr` | AppArmor/Tetragon 组合 | audit -> enforce | 路径解析和系统调用差异需要验证 |
| 危险命令 | `rm -rf /`、`mkfs`、`dd of=/dev/*` | AppArmor 路径限制 + Tetragon 语义识别 | audit -> selective enforce | 高置信命令可先拦截，其余先审计 |
| 下载执行链 | `curl|wget ... | sh` | Tetragon 观测，规则引擎判断 | audit | 参数和 shell 语义复杂，不宜首版强拦截 |
| 反弹 Shell | `bash -i`、`nc -e`、`socat exec` | Tetragon 观测，规则引擎判断 | audit -> warn/enforce | 需要避免误杀运维诊断命令 |
| 系统高危操作 | `mount`、`modprobe`、`insmod`、`unshare`、`nsenter` | Tetragon + AppArmor 局部限制 | audit -> selective enforce | 依赖主机用途和运维流程 |
| 持久化入口修改 | cron、systemd、SSH authorized_keys | AppArmor 文件保护 + Tetragon 观测 | enforce | 本质是敏感文件/目录保护 |
| 容器运行时 | `docker exec`、`docker run --privileged`、挂载 docker.sock | Tetragon 观测 + 命令/文件组合策略 | audit -> enforce | 环境差异大，适合后置 |
| Kubernetes 操作 | `kubectl exec`、修改 kubeconfig、访问 token | Tetragon 观测 + 文件保护 | audit -> enforce | 需结合集群角色和主机类型 |

## 分阶段规划

### 阶段 1：文件全生命周期拦截

目标：把现有 `sensitive_file` 从“编辑拦截”扩展成完整文件保护。

范围：

- 读取敏感文件。
- 写入敏感文件。
- 创建敏感路径文件。
- 删除敏感文件或目录。
- 重命名或移动敏感文件。
- 修改敏感文件权限和属主。

策略模型：

- `template`: `sensitive_file`
- `definition.filePaths`: 受保护路径。
- `definition.operations`: `read`、`write`、`create`、`delete`、`rename`、`chmod`、`chown`、`all`。
- `definition.userMatchMode`: `all`、`include`、`exclude_root`。

部署行为：

- AppArmor 作为强制拦截引擎。
- Tetragon observer 继续采集命中证据。
- 不支持的操作必须显示为 `unsupported` 或 `degraded`，不能标记为成功拦截。

验收标准：

- 对受保护路径的读、写、删、权限变更至少有可复现测试命令。
- 直接 root 放行和 sudo 子进程拦截语义保持现有设计一致。
- 拦截事件进入通知中心“待处理”。
- 恢复、失败和 unsupported 部署状态可在前端看到。

### 阶段 2：危险命令审计与高置信拦截

目标：让 `dangerous_command` 从前端模板变成真实能力。

首批命令分级：

- 高置信破坏类：`mkfs`、`dd of=/dev/*`、`shred`、`wipefs`。
- 高危删除类：`rm -rf /`、`rm -rf /*`、删除关键目录。
- 下载执行链：`curl`、`wget`、`bash`、`sh` 组合。
- 横向/扫描工具：`nmap`、`masscan`、`tcpdump`、`nc`。
- 反弹 Shell：`bash -i`、`nc -e`、`socat exec`。

策略模型：

- `template`: `dangerous_command`
- `definition.commandRules`: 命令、参数匹配、风险级别。
- `definition.enforcementLevel`: `audit`、`warn`、`enforce`。

部署行为：

- Linux 5.4 主机默认以 Tetragon 观测和后端规则识别为主。
- 可执行文件级别限制可由 AppArmor 承担。
- 参数语义复杂的命令默认先 `audit`，由处置反馈逐步提升。

验收标准：

- 前端明确显示“审计中”“可强拦截”“当前主机不支持强拦截”。
- 高置信规则命中进入通知中心。
- 误报处置可以反向生成忽略规则或建议降级。

### 阶段 3：系统高危行为防护

目标：覆盖提权、内核、挂载、命名空间和持久化入口修改。

范围：

- `mount`、`umount`。
- `modprobe`、`insmod`、`rmmod`。
- `unshare`、`nsenter`。
- 修改 `/etc/sudoers`、`/etc/sudoers.d`。
- 修改 systemd unit。
- 修改 cron。
- 修改 SSH authorized_keys。

策略模型：

- 新增 `system_sensitive_operation` 模板，或扩展 `permission_change` 与 `sensitive_file`。
- 按操作类型、路径、用户、主机角色组合配置。

部署行为：

- 文件和持久化入口优先落到 AppArmor。
- 进程和命名空间类先由 Tetragon 审计。
- 主机能力足够时再启用强拦截增强。

验收标准：

- 默认策略必须是审计优先。
- 每条策略必须能解释命中证据。
- 高危行为可进入风险事件和通知闭环。

### 阶段 4：容器与 Kubernetes 运行时防护

目标：覆盖容器逃逸和运行时滥用场景。

范围：

- `docker run --privileged`。
- 挂载 `/`、`/proc`、`/sys`、`/var/run/docker.sock`。
- `docker exec` 进入关键容器。
- `ctr`、`crictl`、`nerdctl` 高危操作。
- `kubectl exec`。
- 修改 kubeconfig。
- 读取 service account token。

策略模型：

- 新增 `container_runtime` 模板。
- 支持按主机角色、命名空间、容器运行时、命令规则配置。

部署行为：

- 首版以审计和告警为主。
- 敏感文件保护继续用 AppArmor。
- 命令链和运行时行为由 Tetragon 观测。

验收标准：

- 能识别 docker.sock 挂载、privileged 容器、kubectl exec。
- 不同主机角色支持不同默认策略。
- 支持从审计结果一键生成候选拦截策略。

### 阶段 5：策略运营闭环

目标：让拦截功能可安全运营，而不是只堆策略模板。

能力：

- 主机能力探测：内核版本、AppArmor 状态、Tetragon 状态、BPF override 能力。
- 策略部署矩阵：每台主机展示 `deployed`、`degraded`、`unsupported`、`failed`、`disabled`。
- 策略模拟：给出测试命令、预期结果和回滚方式。
- 命中统计：最近拦截次数、审计命中次数、误报次数。
- 处置反馈：已确认、误报、忽略同类。
- 风险升级：审计命中多次后建议提升到 `warn` 或 `enforce`。
- 紧急熔断：按策略、主机、模板批量禁用。

验收标准：

- 前端不再只按模板展示能力，而是按“当前主机实际支持能力”展示。
- 每条策略都有可解释的部署状态。
- 策略命中进入通知中心、风险事件和审计详情。
- 误报和忽略可以影响后续推荐和噪声过滤。

## 后端改造建议

### 策略模型

扩展 `definition` JSON，但保持现有表结构可兼容：

- `engine`: `apparmor`、`tetragon`、`hybrid`。
- `operations`: 文件或系统操作类型。
- `enforcementLevel`: `audit`、`warn`、`enforce`。
- `capabilityRequirements`: 需要的内核、AppArmor、Tetragon 能力。
- `fallbackMode`: 不支持强拦截时的降级方式。

### 部署状态

建议新增或扩展部署状态语义：

- `deployed`: 已按预期部署。
- `degraded`: 已部署降级能力，例如从 enforce 降级到 audit。
- `unsupported`: 当前主机不支持。
- `failed`: 部署失败。
- `disabled`: 已禁用。

### Collector

- AppArmor profile 生成器扩展为按操作类型生成规则。
- Tetragon observer 生成器按策略需要生成观测 policy。
- Enforcement syncer 必须对每条策略输出清晰的部署结果和原因。
- 不再把不支持模板简单忽略或误报为部署成功。

### 事件和通知

- 拦截事件继续带 `diting-enforcement`。
- 审计命中带 `diting-enforcement-audit` 或类似标签。
- 通知中心只对 enforce 阻断事件进入“待处理”。
- audit/warn 命中进入风险事件或普通告警，不强制人工处置。

## 前端改造建议

### 策略创建

- 模板不再只是静态表单，应显示支持引擎和主机兼容性。
- 文件模板增加操作类型多选。
- 危险命令模板增加规则级别和默认模式。
- 系统高危行为模板按操作域分组。
- 容器模板按运行时和主机角色分组。

### 策略列表

- 显示策略当前模式：审计、告警、拦截、禁用。
- 显示部署状态摘要：成功、降级、不支持、失败。
- 显示最近命中、最近拦截、最近误报。
- 支持一键查看关联审计事件和通知历史。

### 策略详情

- 展示每台主机的能力判断和部署结果。
- 展示测试命令和预期结果。
- 展示最近命中证据。
- 支持从误报处置生成忽略规则。

## 推荐实施顺序

1. 文件全生命周期拦截。
2. 策略部署状态语义补强，加入 `degraded` 和 `unsupported`。
3. 危险命令审计和高置信拦截。
4. 系统高危行为审计。
5. 容器运行时审计。
6. 策略运营闭环和自动推荐。
7. 按主机能力开放 Tetragon 高版本强拦截增强。

## 第一阶段最小可交付

第一阶段只做“文件全生命周期”：

- 后端保留 `sensitive_file` 模板，扩展 `definition.operations`。
- Collector AppArmor 生成器支持读、写、删除、权限变更。
- 前端在敏感文件策略中增加操作类型选择。
- 部署状态能区分成功、失败、不支持。
- AppArmor 拒绝事件进入现有通知中心。
- 增加端到端测试命令文档。

这一步完成后，DiTing 的拦截能力会从“能拦一个编辑动作”变成“能保护一组敏感路径的完整生命周期”，同时不会引入大量命令语义误杀风险。

## 风险与控制

- 误杀风险：所有新模板默认从 `audit` 或小范围 `enforce` 开始。
- 兼容性风险：每台主机必须输出能力评估，不支持时降级并说明原因。
- 证据缺失风险：强拦截和观测 policy 要配套部署。
- 运维中断风险：保留紧急禁用和自动回滚。
- 表达不一致风险：前端展示必须以后端/Collector 实际能力为准。

## 决策建议

采用混合引擎路线，并把第一阶段限定为文件全生命周期拦截。危险命令、系统高危行为、容器运行时都纳入总规划，但先以审计和风险沉淀进入系统，再根据主机能力与误报数据逐步开放强拦截。
