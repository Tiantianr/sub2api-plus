## Why

安全审核 gate 位于路由和上游调用之前，但部分请求会在 gate 之后改变实际发送给上游的文本语义。Responses legacy aliases 在审核后转换为 `input`，Alpha Search 的 PAT fallback 会把完整 `commands`、`settings` 和 `input` 拼成用户提示词；canonical extractor 还遗漏 Chat `reasoning_content`，并把合法压缩控制项记录为 incomplete。Content Moderation 另有一条 `<system-reminder>` 整段丢弃规则，会连同块外真实用户文本一起跳过。

这些问题使“进入审核 hook”不等于“实际上游语义已审核”，需要在共享 canonical extractor 和直接用户选择层修复，而不是在账号或路由分支中重复解析。

## What Changes

- 让 Responses extractor 按实际 ingress 兼容规则处理 legacy `messages` 和字符串 `prompt`，同时保持原生 `input` 的优先级。
- 让 Alpha Search 在账号选择前覆盖 PAT fallback 会拼入用户提示词的完整 `commands`、`settings` 和 `input` JSON。
- 将 Chat `reasoning_content` 纳入全会话 Prompt Audit，并将 `compaction`、`compaction_summary`、`compaction_trigger` 统一分类为已知不透明历史/控制项。
- Content Moderation 审核完整直接用户文本，不再因出现 `<system-reminder>` 丢弃整段。
- 移除已被会话检查点策略取代的静态“仅最新输入”UI 开关；兼容字段继续保留在配置 API 中供滚动升级使用。
- 增加 production-shaped canonical、双引擎和入口顺序回归测试。

## Capabilities

### New Capabilities

- `security-audit-semantic-parity`: 定义审核输入与实际可达上游文本语义的一致性要求。

## Impact

- 修改 canonical extractor、Content Moderation 输入选择和 Prompt Audit 管理界面。
- 不新增数据库迁移，不改变账号选择、计费、路由、上游请求格式或生产配置。
- 会话级 blocking、输出检查点和失败关闭语义由配套的 `add-conversation-prompt-audit-checkpoints` 变更定义。
