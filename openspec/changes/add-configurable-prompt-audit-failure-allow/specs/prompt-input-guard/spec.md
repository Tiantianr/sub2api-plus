## MODIFIED Requirements

### Requirement: 同步分片必须使用节点级预算并完整覆盖

系统 SHALL 将每个启用节点的 timeout 应用于该节点对一个分片的一次调用。不同分片与不同节点尝试 MUST 获得各自的节点预算；父请求取消 MUST 立即终止剩余尝试。任一必要分片返回严格解析失败或无合法结果时 MUST fail-closed。任一必要分片仅因节点耗尽、连接失败、429、5xx、401/403、容量饱和或超时而不可用时，系统 MUST 根据当前 `allow_on_guard_unavailable` 配置决定 fail-closed 或 failure-allow；该字段缺失时 MUST 默认 failure-allow，且 MUST NOT 将已完成的部分结果当作完整 Safe 结果。

#### Scenario: 所有分片均为安全

- **WHEN** 每个非空分片都通过至少一个节点返回 Safe 或允许的 Warn
- **THEN** 请求 MAY 进入下一阶段

#### Scenario: 中间分片阻断

- **WHEN** 任一分片返回 Block
- **THEN** evaluator MAY 立即早停
- **THEN** 请求 MUST 被阻断且不得部分转发

#### Scenario: 最后一个必要分片技术失败且开关关闭

- **WHEN** 前面分片安全但最后一个必要分片耗尽节点为 unavailable
- **AND** `allow_on_guard_unavailable` 被显式设为 false
- **THEN** 系统 MUST 返回 `prompt_guard_unavailable`
- **THEN** 系统 MUST NOT 根据部分结果放行

#### Scenario: 最后一个必要分片技术失败且开关开启

- **WHEN** 前面分片未产生 Block 且最后一个必要分片耗尽节点为 unavailable
- **AND** `allow_on_guard_unavailable` 为 true 或字段缺失
- **THEN** 普通同步请求 MAY 进入下一阶段
- **AND** 系统 MUST NOT 将该请求记录为 Safe 或创建 Allow receipt

### Requirement: 同步节点故障切换必须有序且显式决定失败行为

系统 SHALL 按配置顺序尝试启用节点。连接失败、429、5xx 和节点超时 MUST 在父请求仍有效且存在后续节点时切换到下一节点，后续节点 MUST 获得自己的完整配置超时。严格解析失败 MUST 结束为非法响应并 fail-closed。耗尽节点后的 eligible unavailable MUST 默认 failure-allow；只有管理员显式关闭 `allow_on_guard_unavailable` 时普通同步请求才 fail-closed。该配置 MUST NOT 影响 Content Moderation、内容提取、配置可信度、加密、深度恢复状态或已知风险判定。

#### Scenario: 首节点耗尽自身超时而次节点成功

- **WHEN** 首节点达到自己的完整配置超时
- **AND** 次节点在自己的配置超时内返回合法结果
- **THEN** 系统 MUST 使用次节点结果
- **THEN** failover 指标 MUST 增加

#### Scenario: 所有节点不可用且显式放行

- **WHEN** 所有节点最终返回 `prompt_guard_unavailable`
- **AND** 普通同步请求未显式关闭 `allow_on_guard_unavailable`
- **THEN** 请求 MUST 可继续到后续独立安全检查或业务阶段
- **AND** failure-allowed 日志与指标 MUST 增加
- **AND** 请求 MUST NOT 创建 Allow receipt

#### Scenario: Guard 响应格式无效

- **WHEN** 任一必要 Guard 响应无法严格解析或包含未知结果
- **THEN** 请求 MUST 返回 `prompt_guard_invalid_response`
- **AND** `allow_on_guard_unavailable` MUST NOT 放行该请求

#### Scenario: Guard 返回确定性客户端错误

- **WHEN** Guard 返回不属于 401/403 或 429 的其他 4xx 状态
- **THEN** 请求 MUST fail-closed
- **AND** `allow_on_guard_unavailable` MUST NOT 放行该请求

#### Scenario: 节点凭据无法解密

- **WHEN** 配置的节点凭据无法解密或没有可执行 Guard 节点
- **THEN** 请求 MUST 按配置或加密故障 fail-closed
- **AND** `allow_on_guard_unavailable` MUST NOT 放行该请求

#### Scenario: 内容提取失败

- **WHEN** 阻塞审计无法完整提取内容
- **THEN** 请求 MUST 在调用 Guard 和业务副作用前 fail-closed
- **AND** `allow_on_guard_unavailable` MUST NOT 放行该请求

#### Scenario: 用户处于强制恢复状态

- **WHEN** 用户携带未清除的强制深度复核状态且 Guard 不可用
- **THEN** 请求 MUST 保留恢复状态并 fail-closed
- **AND** `allow_on_guard_unavailable` MUST NOT 清除或越过恢复要求

#### Scenario: 故障放行后异步 Guard 恢复

- **WHEN** 普通同步请求因 Guard unavailable 被显式 failure-allow
- **AND** 随后的异步深度复核成功返回 Allow
- **THEN** 同步和异步阶段均 MUST NOT 为该请求创建 Allow receipt

#### Scenario: 所有节点容量饱和且开关关闭

- **WHEN** 全局或每节点 bulkhead 均无法接受 evaluation
- **AND** `allow_on_guard_unavailable` 被显式设为 false
- **THEN** 系统 MUST 快速返回 `prompt_guard_unavailable`
- **THEN** 系统 MUST 不无限排队
