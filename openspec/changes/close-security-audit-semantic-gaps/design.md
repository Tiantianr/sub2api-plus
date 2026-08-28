## Context

当前安全审核直接消费 handler 收到的不可变请求体，这是正确的副作用顺序，但 raw body 中存在服务已经支持的兼容 alias 和账号相关 fallback。不能简单把完整上游 normalization 移到审核前：normalization 可能依赖账号并改变路由语义，也会破坏“先审核、后副作用”的边界。

## Decisions

### 1. 在 canonical extractor 表达等价语义

Responses raw body 仍保持不变。Extractor 直接复现 ingress 的内容优先级：非空原生 `input` 存在时忽略会被删除的 legacy `messages` 和字符串 `prompt`；否则把 legacy messages 按 Chat 内容语义提取，把字符串 prompt 视为当前用户 input。Reusable prompt object 继续走既有变量提取路径。

这样两套审核引擎看到实际可达上游的文本，但审核 gate 仍早于 compact normalization、账号选择、计费、并发和上游写入。

### 2. Alpha Search 审核所有 fallback 用户数据

账号在审核后才选择，因此审核时不能知道请求最终走 native alpha/search 还是 PAT Responses fallback。Canonical extractor 对 `commands`、`settings` 和 `input` 使用现有结构化清洗与确定性 JSON 编码，形成保守并集。媒体 URL、base64 和 encrypted payload 仍由共享清洗器排除。

### 3. 区分可见 reasoning 与不透明压缩状态

Chat `reasoning_content` 是可见文本，raw Chat 会转发，Responses bridge 也会把它变成 `<thinking>` 文本。用户角色上的该字段仍属于直接用户输入；assistant/model 角色上的该字段进入 `SourceReasoning`，不被 Content Moderation 归因为直接用户。

`compaction`、`compaction_summary` 和 `compaction_trigger` 是网关已支持的历史或控制项。Extractor 只跳过 `encrypted_content`、ID、status 等已知 opaque/control 字段；`summary` 或未来未知可见字段继续按结构化文本审核。纯 opaque 项不产生 incomplete，同一请求中的可见字段和兄弟用户内容照常审核。

### 4. 直接审核包含 reminder 的用户文本

`<system-reminder>` 来自不可信客户端 payload，不能作为跳过 Content Moderation 的信任标记。最小且安全的行为是审核完整直接用户文本，不尝试用字符串规则剥离块，因为无法证明标签来源可信或正确嵌套。

### 5. 静态 latest 开关由会话检查点取代

`blocking_latest_turn_only` 继续保留在 storage/public/update JSON 中，以支持滚动升级期间的新旧进程读取同一设置；控制台不再展示它。同步 blocking 的实际输入范围由 `add-conversation-prompt-audit-checkpoints` 定义：未知或失效检查点全量审核，可信连续会话只审核上一轮已捕获输出和本轮输入。

逐节点/逐 chunk timeout 和总预算仍另行设计，不与内容语义修复耦合。

## Verification

- Canonical extractor 覆盖 legacy alias 的 native-input 优先级、reasoning 和三类 compaction item。
- Content Moderation 与 Prompt Audit 对相同 production-shaped payload 断言各自选择语义。
- Alpha Search fallback 字段在账号选择前可见。
- HTTP/WS 审核顺序继续早于账号、计费、并发和 upstream write。
- 前端不再呈现被会话检查点取代的 latest-turn toggle。
