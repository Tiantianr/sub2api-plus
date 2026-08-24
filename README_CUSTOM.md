# Sub2API Plus 二开与上游同步流程

本文档只描述 `Tiantianr/sub2api-plus` 的二开流程。原项目的开发、测试、迁移和发布约束仍以
[`AGENTS.md`](AGENTS.md)、[`CONTRIBUTING.md`](CONTRIBUTING.md)、
[`UPSTREAM.md`](UPSTREAM.md) 和 [`docs/RELEASING.md`](docs/RELEASING.md) 为准。

## 仓库关系

```text
Wei-Shaw/sub2api
        ↓ 由 LuckyKuang 维护同步
LuckyKuang/sub2api-plus
        ↓ 由本仓库按正式版本同步
Tiantianr/sub2api-plus
        ↓ 构建自有 Release 和镜像
ID3 生产环境
```

- `origin`：`https://github.com/Tiantianr/sub2api-plus.git`，个人二开仓库。
- `upstream`：`https://github.com/LuckyKuang/sub2api-plus.git`，只读上游。
- 不直接同步 `Wei-Shaw/sub2api`。Plus 已经维护官方基线与自身改动，绕过它会丢失或破坏 Plus 适配。
- 本仓库不配置 GitLab remote，不使用公司 GitLab 账号、邮箱、Token、SSH Key 或提交身份。
- 本地提交身份只在当前仓库配置为 GitHub 账号 `Tiantianr` 的 noreply 邮箱，不修改全局 Git 配置。

检查远端与本地身份：

```bash
git remote -v
git config --local --get user.name
git config --local --get user.email
git config --local --get user.useConfigOnly
```

仓库自带的受保护 push/release 工具默认锁定原维护仓库。在本 fork 中，每次调用前必须显式设置：

```bash
export SUB2API_EXPECTED_REPOSITORY=Tiantianr/sub2api-plus
export SUB2API_CUSTOM_ITERATION_MIN=901
```

## 分支规则

- `main`：已验证、可发布的二开主线，不直接提交或推送。
- `feature/<topic>`：单一二开功能。
- `fix/<topic>`：缺陷修复。
- `sync/upstream-<version>`：同步一个上游正式版本的临时分支。
- `release/<version>`：准备自有发布时使用，发布结束后删除。

每项改动保持为小而独立的提交。优先使用现有设置和扩展点；品牌、Logo、菜单等后台可配置内容不改源码。
不要把多个无关功能塞进一个长期 `custom` 分支，也不要重写已经发布的 `main` 历史。

## 日常二开

从最新个人主线创建功能分支：

```bash
git switch main
git pull --ff-only origin main
git switch -c feature/<topic>
```

实现期间运行与改动范围匹配的检查。提交 PR 前按
[`CONTRIBUTING.md`](CONTRIBUTING.md) 完成完整验证；前后端常用入口为：

```bash
pnpm --dir frontend install --frozen-lockfile
make test
```

涉及 Ent schema 时必须重新生成 Ent 和 Wire；涉及数据库时只能增加新的前向 migration，禁止修改已经发布的 migration。
英语和中文前端文案必须一起更新。

## 同步上游

只同步 LuckyKuang 已发布且准备采用的正式标签，不直接把上游 `main` 的未发布内容部署到生产环境。

```bash
git fetch upstream --tags --prune
git switch main
git pull --ff-only origin main
git switch -c sync/upstream-vX.Y.Z-custom.NNN
git merge --no-ff 'vX.Y.Z+custom.NNN'
```

解决冲突时保留本仓库明确需要的二开行为，同时接受上游安全修复、迁移和接口调整。完成后：

1. 更新本文件中的“版本映射”。
2. 检查上游 Release Notes、`UPSTREAM.md`、配置示例和 migration。
3. 运行完整测试和构建。
4. 使用 PR 合并到个人 `main`。
5. 合并和验证完成前不发布、不部署。

不要使用 rebase 重写已发布主线，也不要逐个 cherry-pick 大批上游提交；以正式标签为边界进行一次 merge，冲突来源更清楚。

## 自有版本

现有更新器和发布工具要求 `vX.Y.Z+custom.NNN`。为减少对发布工具的修改，本仓库保留 `901-999`
作为个人二开迭代区间：

```text
当前 Plus 基线：v0.1.178+custom.003
个人已发布版本：v0.1.178+custom.901
同一官方基线的下一版本：v0.1.178+custom.902
新官方基线首版：v0.1.179+custom.901
```

一个官方基线最多允许 99 个个人版本；接近该上限或上游开始使用 `900` 区间时，再扩展版本格式和校验工具。
不得复用、移动或覆盖已经发布的标签。

### 版本映射

| 个人版本 | LuckyKuang 基线 | 状态 |
| --- | --- | --- |
| `v0.1.178+custom.902` | `v0.1.178+custom.003` | 计划发布 |
| `v0.1.178+custom.901` | `v0.1.178+custom.001` | 已发布 |
| `v0.1.178+custom.001` | 同名上游版本 | 已发布回退基线 |

## 一键更新边界

本 fork 已将一键更新源切换到 `Tiantianr/sub2api-plus`，并已完成
`v0.1.178+custom.901` 的 Release 与公开镜像验证。新候选版本在完整发布验证前，界面不会有对应资产。
ID3 尚未切换到个人不可变镜像，因此生产环境仍不得使用“升级”或“版本回退”。

本 fork 的每个版本发布必须满足以下条件：

1. 后端更新源及前端仓库、镜像地址保持为个人发布仓库。
2. 由个人 CI 生成匹配平台的二进制、`checksums.txt`、GitHub Release 和 GHCR 镜像。
3. GHCR 不可变标签必须公开，并提供 ID3 当前使用的 `linux/arm64` 镜像。
4. 保持发布构建的 `BuildType=release`；普通源码构建不会提供在线升级。

生产使用前还必须将 ID3 Compose 固定到同版本的个人不可变镜像，不能只依赖容器可写层中的自更新二进制。
在上述链路完成前，不使用管理界面的“升级”或“版本回退”操作发布二开版本。

私有源码仓库不能直接复用当前匿名 GitHub Release 下载逻辑。需要保持源码私有时，应使用独立发布仓库，或先实现经过审查的认证下载；Token 只能通过运行时 Secret 注入，禁止写入源码、镜像或 README。

## GitHub CI 与发布流水线

`submit-pr` 在本机运行 Go unit、Vitest、版本、文档、迁移和部署静态 preflight；功能分支和 PR 的 Linux Actions
负责 integration、lint、生产构建、Docker、仓库策略和安全扫描。受保护 `main` 同时预构建精确提交的
Linux arm64 镜像 artifact；显式发布标签后，Release workflow 复用该验证结果并在五分钟执行预算内推送镜像，
不重复完整测试矩阵。发布仅允许从已经合并并验证的 `main` 提交创建不可变标签；禁止直接运行 `git push --tags`。

提交最终 PR：

```bash
export SUB2API_EXPECTED_REPOSITORY=Tiantianr/sub2api-plus
export SUB2API_CUSTOM_ITERATION_MIN=901
python3 skills/push-cli/scripts/push_cli.py submit-pr
```

PR 和合并后 Actions 全部通过后，按 [`docs/RELEASING.md`](docs/RELEASING.md) 分阶段执行：

```bash
python3 skills/release-cli/scripts/release_cli.py promote-pr \
  --tag vX.Y.Z+custom.NNN --pr <number> --notes-file release-notes.md

python3 skills/release-cli/scripts/release_cli.py validate \
  --tag vX.Y.Z+custom.NNN --pr <number> --notes-file release-notes.md

python3 skills/release-cli/scripts/release_cli.py tag \
  --tag vX.Y.Z+custom.NNN --pr <number> --notes-file release-notes.md

python3 skills/release-cli/scripts/release_cli.py publish \
  --tag vX.Y.Z+custom.NNN

python3 skills/release-cli/scripts/release_cli.py monitor \
  --tag vX.Y.Z+custom.NNN

python3 skills/release-cli/scripts/release_cli.py verify \
  --tag vX.Y.Z+custom.NNN
```

成功发布后生成：

```text
GitHub Release: https://github.com/Tiantianr/sub2api-plus/releases/tag/vX.Y.Z+custom.NNN
GHCR:          ghcr.io/tiantianr/sub2api-plus:vX.Y.Z-custom.NNN
Platform:      linux/arm64
```

GHCR 包首次创建后设为 public 并验证匿名 `docker pull`；后续版本继承包可见性。Release 完成后再通过单独 PR
将 `UPSTREAM.md` 状态由 `planned` 改为 `published`。

## ID3 发布流程

ID3 是生产运行环境，不在服务器或容器里直接修改源代码。生产发布使用个人 CI 生成的不可变镜像，并遵循以下顺序：

1. 备份 PostgreSQL、配置文件和当前 Compose，记录当前镜像标签及摘要。
2. 在独立数据库和 Redis 上验证候选镜像；验证环境不得连接生产数据库。
3. 检查 migration、登录、管理后台、API 转发、SSE、WebSocket 和后台任务。
4. 更新 ID3 Compose 为个人镜像的 `版本标签@sha256:digest`，只重建 `sub2api` 应用容器。
5. 等待健康检查通过，再执行公网冒烟测试。
6. 保留旧镜像和升级前数据库备份，直到新版本稳定。

应用内回退只替换二进制，不会回退数据库 migration。发生不兼容 migration 时，必须同时恢复升级前数据库备份，不能只切换旧镜像。

## 当前生产注意事项

ID3 当前 Compose 仍固定旧镜像，而容器内二进制已通过原项目的一键更新升级。普通容器重启会保留可写层，容器重建则可能回到旧镜像版本。
在首次二开部署前，应先备份生产数据，并让 Compose 镜像与当前运行版本一致；该调整必须作为单独的生产变更执行，不与上游同步混在一起。

## 禁止事项

- 不向 `main` 直接推送。
- 不在 ID3 容器内编辑或替换源码作为正式改动。
- 不让验证环境和生产环境共用数据库或 Redis。
- 不修改已经执行的 migration。
- 不把凭据、生产数据、服务器地址或私有配置提交到仓库。
- 不直接点击上游一键升级覆盖个人二开版本。
- 不添加 GitLab remote，不使用公司 GitLab 身份进行提交、签名或推送。
