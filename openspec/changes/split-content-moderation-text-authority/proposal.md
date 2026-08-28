## Why

Prompt Guard blocking 与 Content Moderation `pre_block` 当前会并行审核同一份直接用户文本。生产页面中的 `illicit`/`hash` 记录证明，管理员即使只关注 Prompt Audit，只要共享总开关和旧 Content Moderation 配置仍启用，两套文本裁决都会运行。重复调用增加延迟、费用和相互矛盾的命中，同时现有配置又无法只保留图片 API。

文本应有一个裁决权威，图片和本地快速规则仍需保留。切换期间还需要无执法副作用的影子结果来验证 Prompt Guard 是否漏掉 Moderation 独有命中。

## What Changes

- 增加 Content Moderation `text_api_mode`: `auto`、`blocking`、`observe`、`off`。
- `auto` 仅在 blocking Prompt Guard 确实覆盖当前请求时，把外部文本 Moderation 降为影子观察；Prompt Audit 范围外仍同步文本裁决。
- `observe` 文本结果只记录，不阻断、不写风险 hash、不发通知、不累计或触发封禁。
- `off` 不调用文本 Moderation API；图片仍按全局 `pre_block`/`observe` 模式执行。
- 本地关键词和 `pre_hash` 继续先于外部 API；图片或 blocking 提取/hash/API 依赖失败返回明确不可用并阻止上游。
- 风控记录页把 `hash_block` 明确显示为 Hash 拦截，避免与普通 API 命中混淆。

## Capabilities

### New Capabilities

- `content-moderation-text-authority`: 定义 Prompt Guard 文本权威、Moderation 文本过渡策略和图片裁决保留行为。

## Impact

- 修改 Content Moderation 配置 JSON/API/UI、Coordinator 请求级范围传递和 pre-block 执行分支。
- 旧配置缺少 `text_api_mode` 时规范化为 `auto`；没有数据库 migration。
- 不修改已存风险 hash，不自动改变生产配置，不部署、不发布。
