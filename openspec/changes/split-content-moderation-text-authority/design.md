## Context

`risk_control_enabled` 是 Prompt Audit 与 Content Moderation 共用的总入口，不代表只启用其中一套。Coordinator 在 Prompt blocking 时并行执行两个 engine，但此前 Legacy adapter 不知道 Prompt Guard 是否覆盖该分组，因此无法安全地自动取消文本执法。

Content Moderation 的 API 输入可以同时包含 text 和 image。要让文本影子、图片同步，必须拆成两个请求；仅把最终 Decision 改成 allow 仍会让影子命中写 hash、发邮件或封禁，不能称为观察。

## Decisions

### 1. 请求级传递 Prompt 文本权威

`PromptService.BlockingApplies` 只有在 active blocking config 可用且当前 group 在范围内时返回 true。Coordinator 通过可选能力接口把该事实传给 Legacy adapter。全局 blocking 但 group 不在范围、配置降级或 Prompt 未启用时均为 false。

### 2. auto 是兼容默认

旧 JSON 缺字段时 `text_api_mode=auto`。有效策略：

- Prompt 覆盖：等价 `observe`；
- Prompt 不覆盖：等价 `blocking`。

因此升级不会让 Prompt 范围外的既有 Content Moderation 文本保护消失。

### 3. 文本和图片分开调用

在 `pre_block` 下：

- `blocking`: 文本与图片保持一次同步组合调用；
- `observe`: 文本进入影子队列，图片单独同步；
- `off`: 文本不调用 API，图片单独同步。

在全局 `observe` 下，选中的文本和图片均进入普通异步观察。全局 `off` 仍关闭整个 Content Moderation。

`keyword_only` 继续表示文本只做关键词判断；它不关闭图片审核。纯文本未命中时不调用文本 API，带图片时图片仍按全局模式处理。

### 4. 影子命中无执法副作用

影子 task 调用同一评分和日志路径，但 `recordHash=false`、`applySideEffects=false`。它形成 `flagged + action=shadow` 对比记录；历史违规计数查询明确排除 `shadow`，不得阻断、通知、累计封禁或污染后续 hash 预检。

### 5. 图片和本地依赖失败关闭

`pre_block` 的不完整提取、hash cache 读取失败、缺 API Key、图片/同步 API 超时、取消、HTTP 错误或无效响应返回 `content_moderation_unavailable`。该结果不是内容策略命中，不计违规。

### 6. 管理界面明确来源

设置页展示四种文本 API 策略及图片行为。记录页单独标识 `hash_block`，`block + illicit/...` 表示外部 Moderation API，`cyber_policy` 表示业务上游 OpenAI 返回的网络安全策略。

## Verification

- 多模态测试断言 Prompt 覆盖时产生一个无执法文本影子调用和一个图片同步调用。
- `off` 测试断言文本不进 API、图片仍拦截、本地关键词仍拦截。
- Prompt 范围外测试断言 `auto` 仍同步阻断文本。
- 影子命中不得写风险 hash；图片依赖失败返回 unavailable。
- 前端组件、lint、typecheck 和双语字段保持一致。
