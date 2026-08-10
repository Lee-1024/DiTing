# 持久化通知闭环设计

将顶部通知从“最近事件快照”改为后端持久化通知中心：红点只表示当前用户未读数，拦截事件由操作员处置闭环，Collector/Tetragon 状态告警在恢复后自动关闭，并保留完整历史。

## 数据与生命周期

- `diting_notifications` 保存通知来源、目标、状态、处置和恢复时间。
- `diting_notification_reads` 按通知和用户保存阅读时间。
- 拦截事件按 `event_id` 永久去重，已处置事件不得重开。
- 状态告警只对活跃事件去重；恢复后关闭，复发时创建新历史记录。
- 拦截事件成功写入后创建通知；Collector 离线或 Tetragon 异常由后台协调器创建，健康恢复时自动关闭。

## API 与前端

- `GET /api/v1/notifications?view=unread|pending|all&limit=20`
- `POST /api/v1/notifications/read-all`
- `POST /api/v1/notifications/{id}/read`
- `POST /api/v1/notifications/{id}/handle`
- 面板分为“未读 / 待处理 / 全部”，红点显示未读数，提供全部已读和三种处置操作。
- 点击通知先标记已读，再跳转对应事件详情；历史显示处置结果或恢复时间。

## 验证

覆盖逐用户阅读、永久事件去重、状态复发历史、处置、HTTP 身份、生产与恢复逻辑，以及前端展示和构建。