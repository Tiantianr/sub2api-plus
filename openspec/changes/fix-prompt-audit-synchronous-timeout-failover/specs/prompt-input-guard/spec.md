## RENAMED Requirements

- FROM: `### Requirement: 同步分片必须共享总预算并完整覆盖`
- TO: `### Requirement: 同步分片必须使用节点级预算并完整覆盖`

## MODIFIED Requirements

### Requirement: 同步分片必须使用节点级预算并完整覆盖

系统 SHALL 将每个启用节点的 timeout 应用于该节点对一个分片的一次调用。不同分片与不同节点尝试 MUST 获得各自的节点预算；任一必要分片失败、超时或无合法结果时 MUST fail-closed，父请求取消 MUST 立即终止剩余尝试。

#### Scenario: 所有分片均为安全

- **WHEN** 每个非空分片都通过至少一个节点返回 Safe 或允许的 Warn
- **THEN** 请求 MAY 进入下一阶段

#### Scenario: 中间分片阻断

- **WHEN** 任一分片返回 Block
- **THEN** evaluator MAY 立即早停
- **THEN** 请求 MUST 被阻断且不得部分转发

#### Scenario: 最后一个必要分片失败

- **WHEN** 前面分片安全但最后一个必要分片耗尽节点或响应无效
- **THEN** 系统 MUST 返回 unavailable/invalid_response
- **THEN** 系统 MUST NOT 根据部分结果放行

### Requirement: 同步节点故障切换必须有序且 fail-closed

系统 SHALL 按配置顺序尝试启用节点。连接失败、429、5xx 和节点超时 MUST 在父请求仍有效且存在后续节点时切换到下一节点，后续节点 MUST 获得自己的完整配置超时；401/403、严格解析失败或耗尽节点 MUST 结束为不可用/非法响应。同步模式 MUST NOT 提供隐式 fail-open。

#### Scenario: 首节点耗尽自身超时而次节点成功

- **WHEN** 首节点达到自己的完整配置超时
- **AND** 次节点在自己的配置超时内返回合法结果
- **THEN** 系统 MUST 使用次节点结果
- **THEN** failover 指标 MUST 增加

#### Scenario: 父请求在首节点期间取消

- **WHEN** 父请求在首节点调用期间被取消
- **THEN** 系统 MUST 停止同步审计且不得调用次节点
- **THEN** 系统 MUST NOT 根据任何部分结果放行

#### Scenario: 认证失败

- **WHEN** 节点返回 401 或 403
- **THEN** 系统 MUST 视为不可重试配置错误
- **THEN** 请求 MUST 返回 503 而不是按 Safe 放行

#### Scenario: 所有节点容量饱和

- **WHEN** 全局或每节点 bulkhead 均无法接受 evaluation
- **THEN** 系统 MUST 快速返回 `prompt_guard_unavailable`
- **THEN** 系统 MUST 不无限排队
