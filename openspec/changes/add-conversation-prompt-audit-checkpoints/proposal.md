## Why

同步 Prompt Audit 目前按单次请求静态选择 latest 或 full。latest 会漏掉客户端在完整回放历史中新增的风险文本；每次 full 又会反复审核 system prompt、AGENTS 注入和已通过历史，放大延迟、超时与费用。`previous_response_id` 只保存路由绑定，不包含可重新审核的正文，也不能充当会话审核检查点。

需要一个失败关闭的会话状态机：先确认完整上下文安全，再复用该结论；任何输入、输出、依赖或连续性异常都撤销增量资格。

## What Changes

- 为 blocking Prompt Audit 增加 Redis `CLEAN` / `FULL_REQUIRED` 检查点和单回合原子租约。
- 新会话、检查点失效、审核配置变化、system/tool 上下文变化、未知或分叉父链、历史回放指纹不一致时执行完整 canonical 文本审核。
- `CLEAN` 且连续性可验证时，只审核上一轮已清洗的真实下游 AI 输出与本轮当前客户端输入。
- 在 HTTP/JSON/SSE、Responses WebSocket 和 Live Sideband 的最终下游边界捕获输出；只有成功终态且输出完整可解析时提交新 `CLEAN`。
- Guard block/flag、超时、取消、无效响应、Redis 故障、内容提取不完整、并发重叠、输出截断或非成功终态均阻止检查点推进；blocking 输入提取失败直接阻断请求。
- 移除静态 latest/full 控件，保留旧 JSON 字段仅供滚动升级兼容。

## Capabilities

### New Capabilities

- `conversation-prompt-audit-checkpoints`: 定义会话级完整/增量审核、原子状态和下游输出检查点。

## Impact

- 修改 Prompt Audit blocking 热路径、Redis 临时状态、gateway response writer、Responses WS hooks、Live Sideband hooks 和管理界面文案。
- Redis 只保存应用层加密后的有界清洗输出、SHA-256 指纹、段数、配置版本和哈希化父链映射；不新增 PostgreSQL migration，不保存原始 session/response/call id。
- 检查点是优化，不是放行依赖：Redis 或连续性不可证明时请求失败关闭或退回完整审核，绝不静默跳过文本。
- 不修改生产配置、账号调度、计费、业务上游协议或发布状态。
