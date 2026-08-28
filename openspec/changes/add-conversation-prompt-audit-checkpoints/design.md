## Context

客户端有两类会话形态：完整回放每轮 body，或仅用 `previous_response_id`/Live call 让上游持有历史。前者可验证正文连续性，后者只能验证服务器此前记录的父链。固定 latest/full 开关无法同时做到低重复和防回放绕过。

审核边界仍必须位于账号选择、计费、并发和上游写入之前。输出则必须在所有兼容转换、模型名恢复、工具名恢复和错误清洗之后捕获，才能代表下一轮客户端实际持有的 AI 内容。

## Decisions

### 1. Redis 状态只有 CLEAN 和 FULL_REQUIRED

每个 key 使用 API Key 与稳定会话信号哈希隔离。无显式信号的 Responses 请求生成临时 key，并可在成功输出后通过哈希化 response id 找回。Live 创建成功后用哈希化 call id 绑定同一 key。原始 session、response 和 call id 不进入状态 key、日志或审计事件。

`Begin` Lua 原子完成：拒绝已有回合租约、返回旧检查点、写入租约，并立即把状态改为 `FULL_REQUIRED`。因此进程崩溃、请求取消或遗漏 finalize 都不会遗留陈旧 `CLEAN`。

### 2. 完整回放必须验证历史连续性

成功检查点保存两段 SHA-256 指纹和段数：该请求完整 canonical 会话输入，以及捕获的 AI 输出 canonical 序列。下一次无父响应的完整回放只有在非当前历史恰好等于这两段连接时才可增量。插入、删除、改写、压缩或重排任何历史段都会退回 full。

system/developer instruction 和 tool definition 单独形成上下文 hash；存在的上下文变化强制 full。父响应 continuation 可以省略未变化的上下文并继承旧 hash。

### 3. 父响应是连续性索引，不是正文

已知 `previous_response_id` 映射到会话 key，且其哈希必须等于该 key 最新成功 response hash 才可增量；旧分支或未知父链强制 full。即使 parent 已知，只要请求同时携带非当前回放历史，仍必须通过两段历史指纹校验。该映射不声称能恢复缺失历史。客户端未在当前请求发送且 Redis 未保留的正文不可重新审核。

Live call alias 只确定控制会话身份；每个成功 response 仍更新上一轮输出。Sideband 内容帧重叠时失败关闭。

### 4. 增量输入由可信输出和当前输入组成

上一轮输出来自 gateway 最终下游捕获，客户端不能替换。当前输入来自共享 `auditcontent` document 中 `Current && ClientControlled` 的非静态文本，包括用户消息、当前 tool result、当前客户端声明的 assistant 文本和 prompt variable。任何一个 chunk 返回 Block 都阻断整个请求。

### 5. 成功终态是唯一提交点

HTTP/JSON/SSE 使用 gateway ResponseWriter；Responses WS 在最终 frame 写成功后观察；Live 在 upstream-to-downstream frame 写成功后观察。捕获上限为 500000 Unicode 字符和相应有界字节缓冲，媒体/base64/encrypted opaque 数据经共享 sanitizer 移除。

仅 2xx 非流 canonical 输出，或已成功写给客户端且带明确成功终态的完整 SSE/WS 输出可提交。空响应、未知输出 shape、`response.failed/incomplete/cancelled`、协议 error、缺终态、写失败、客户端断开、溢出或解析不确定都保持 `FULL_REQUIRED`。清洗后的输出经现有应用 SecretEncryptor 加密后才写 Redis；解密失败同样撤销增量资格。

### 6. 兼容字段不再控制 blocking

`blocking_latest_turn_only` 在滚动升级期间继续序列化，但新 blocking 运行时忽略它。控制台移除该控件，避免显示一个不再决定行为的设置。

## Operational Bounds

- 状态和父映射默认 TTL 为 2 小时；过期后下一请求 full。
- 回合租约固定为 2 小时，覆盖当前 1 小时 Live 与 15 分钟 WS 上限；进程崩溃后由 TTL 自动释放。只有未来支持超过 2 小时的传输时才增加续租。
- 输出超出边界时不截断后继续增量，而是撤销检查点资格。
- Redis 不可用时 blocking 请求返回审核不可用，不退化为无状态 latest。

## Verification

- miniredis 验证原子 begin/commit/fail、busy、stale lease、TTL 和父映射。
- 服务测试验证 first full、稳定 incremental、system 变化、未知/分叉父链、完整回放插入和 extraction failure。
- 输出测试覆盖 Responses terminal aggregate、Chat SSE delta、失败和缺终态。
- transport 测试验证 HTTP/SSE、Responses WS 三种模式与 Live 捕获的是最终下游 payload。
