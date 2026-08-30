## MODIFIED Requirements

### Requirement: 开启同步阻止必须有明确的风险确认

页面 SHALL 把 enabled 和 blocking_enabled 作为独立开关，并 SHALL
在独立的正常记录留存区域管理允许保存 Pass 事件的用户。未选择任何用户时 SHALL
默认不保存 Pass 事件。关闭 enabled 时 MUST 自动关闭并禁用
blocking_enabled；开启 blocking_enabled 时 MUST 展示二次确认，说明请求延迟、
Block 和 Guard 不可用的 fail-closed 行为。

#### Scenario: 管理员配置正常记录留存用户

- **WHEN** 管理员搜索并选择一个用户保存正常记录
- **THEN** 该名单 SHALL 使用独立版本保存
- **AND** 页面 SHALL 明确说明 Flag 和 Critical 记录始终保存

### Requirement: 页面必须提供防误操作的事件删除流程

页面 SHALL 支持单条删除、选中项批量删除、按筛选删除，以及固定
`decision=pass` 的正常记录清理入口。正常记录清理 MUST 先展示服务端预览，
并要求明确时间范围、matched_count、context count、estimated bytes、
snapshot_max_id、filter_hash、服务端认证 confirmation_token 和二次确认。

#### Scenario: 清理正常记录

- **WHEN** 管理员选择用户范围和明确时间范围
- **THEN** 页面 MUST 锁定 Pass 判定并先展示数量和预计后续备份减少量
- **THEN** 未成功展示预览前确认按钮 MUST 不可用
