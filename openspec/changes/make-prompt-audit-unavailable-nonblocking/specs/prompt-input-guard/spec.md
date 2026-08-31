## MODIFIED Requirements

### Requirement: 同步分片必须使用节点级预算并完整覆盖

系统 SHALL 将每个启用节点的 timeout 应用于该节点对一个分片的一次调用。不同分片与不同节点尝试 MUST 获得各自的节点预算；父请求取消 MUST 立即终止剩余尝试。任一必要分片返回严格解析失败或无合法结果时 MUST fail-closed。任一必要分片最终返回 `prompt_guard_unavailable` 时，系统 MUST 保留部分结果不可作为完整 Safe 结果的语义，但 MUST 允许当前请求继续且 MUST NOT 向用户返回 Prompt Audit 不可用 503。

#### Scenario: 所有分片均为安全

- **WHEN** 每个非空分片都通过至少一个节点返回 Safe 或允许的 Warn
- **THEN** 请求 MAY 进入下一阶段

#### Scenario: 中间分片阻断

- **WHEN** 任一分片返回 Block
- **THEN** evaluator MAY 立即早停
- **THEN** 请求 MUST 被阻断且不得部分转发

#### Scenario: 最后一个必要分片不可用

- **WHEN** 前面分片未产生 Block
- **AND** 最后一个必要分片以任意 Guard 4xx、429、5xx、超时、连接失败、容量饱和或节点不可用结束为 `prompt_guard_unavailable`
- **THEN** 当前请求 MUST 可继续到后续独立安全检查或业务阶段
- **AND** 系统 MUST NOT 将已完成的部分结果记录为完整 Safe 结果
- **AND** 系统 MUST NOT 创建 Allow receipt

### Requirement: 同步节点故障切换必须有序且不可阻断业务请求

系统 SHALL 按配置顺序尝试启用节点。连接失败、429、5xx 和节点超时 MUST 在父请求仍有效且存在后续节点时切换到下一节点，后续节点 MUST 获得自己的完整配置超时。严格解析失败 MUST 结束为非法响应并 fail-closed。任何重试或切换结束后的最终 `prompt_guard_unavailable` MUST failure-allow；HTTP 状态、重试资格、恢复状态、节点容量和管理员配置 MUST NOT 将该结果转换成用户可见 503。该行为 MUST NOT 弱化 Content Moderation、内容提取、配置版本可信度、加密、无效 Guard 响应或已知风险判定。

#### Scenario: 首节点耗尽自身超时而次节点成功

- **WHEN** 首节点达到自己的完整配置超时
- **AND** 次节点在自己的配置超时内返回合法结果
- **THEN** 系统 MUST 使用次节点结果
- **AND** failover 指标 MUST 增加

#### Scenario: Guard 返回确定性客户端错误

- **WHEN** Guard 返回不可重试 4xx 并最终分类为 `prompt_guard_unavailable`
- **THEN** 请求 MUST 可继续
- **AND** failure-allowed 日志与指标 MUST 增加
- **AND** 请求 MUST NOT 创建 Allow receipt

#### Scenario: Guard 响应格式无效

- **WHEN** 任一必要 Guard 响应无法严格解析或包含未知结果
- **THEN** 请求 MUST 返回 `prompt_guard_invalid_response`
- **AND** 系统 MUST NOT 将非法响应重新分类为 unavailable 后放行

#### Scenario: 没有可执行 Guard 节点

- **WHEN** Prompt Audit 因节点缺失、凭据无效、客户端构造失败或容量饱和而最终返回 `prompt_guard_unavailable`
- **THEN** 当前请求 MUST 可继续
- **AND** 该请求 MUST NOT 获得 Safe 结果或 Allow receipt

#### Scenario: 内容提取失败

- **WHEN** 阻塞审计无法完整提取内容
- **THEN** 请求 MUST 在调用 Guard 和业务副作用前 fail-closed
- **AND** 该错误 MUST NOT 被标记为 `prompt_guard_unavailable`

#### Scenario: 用户处于强制恢复状态

- **WHEN** 用户携带未清除的强制深度复核状态且 Guard 最终返回 `prompt_guard_unavailable`
- **THEN** 当前请求 MUST 可继续且 MUST NOT 返回 Prompt Audit 不可用 503
- **AND** 恢复 finding MUST 保留且临时 claim MUST 释放
- **AND** 当前请求 MUST NOT 创建 Allow receipt

#### Scenario: Content Moderation 同时不可用或阻断

- **WHEN** Prompt Audit unavailable 被 failure-allow
- **AND** Content Moderation 独立返回阻断或不可用
- **THEN** Content Moderation 的决定 MUST 保持权威
- **AND** Prompt Audit failure-allow MUST NOT 覆盖该决定
