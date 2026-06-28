# Product Requirements Document: Push 拒绝提示优化 & 消息模板动态化

**作者**: Sam  
**日期**: 2026-06-28  
**状态**: Draft  
**关联代码库**: gitlab-task-hook  

---

## 1. 执行摘要

gitlab-task-hook 在触发 push 规则拦截时，当前提示信息存在两个问题：
一是场景4（禁止强制推送）缺少修复指引，技术能力弱的开发者无从下手；
二是所有提示文案硬编码在 Go 源码中，运维无法在不发版的情况下修改。

本 PRD 覆盖两项并行改造：**提示内容完善（用户体验）** 与 **提示内容迁移至 Nacos（可运维性）**。

---

## 2. 背景与问题

### 2.1 用户侧问题

- 目标用户（非资深研发）在 push 被拒绝后，如果看不到明确的修复步骤，会陷入困惑，需要频繁寻求同事或运维介入。
- 场景4（强制推送拦截）目前的提示仅列出上下文信息（project / repo / user / branch），**没有任何修复指引**，是四个场景中唯一缺失该章节的。

### 2.2 运维侧问题

- 四类提示文案完全硬编码于 `internal/message/message.go`。
- 任何文案调整（新增注意事项、修改 GitLab 地址、更换任务系统前缀等）都需要**重新编译、打包、部署 hook**，变更成本高且有发布风险。
- 项目已集成 Nacos 配置中心（`internal/nacos/` 包），具备热更新基础能力，但消息层尚未接入。

---

## 3. 目标与非目标

### 目标

1. 补全场景4的修复指引，确保四个场景的提示信息结构完整统一。
2. 所有场景的操作步骤对**非技术用户**可读、可执行，无需 Google。
3. 将全部消息模板从 Go 代码迁移至 Nacos YAML 配置，支持热更新。
4. hook 运行时优先使用 Nacos 模板，Nacos 不可用时回退到内置默认模板。

### 非目标

- 不改变四个规则本身的判断逻辑（rule.go 规则顺序不变）。
- 不新增拦截规则。
- 不对 warn 模式的输出行为做改变（warn 下仍然打印消息、但不阻断 push）。
- 不引入新的配置中心（Nacos 已存在，不替换）。
- 不做 UI 界面，提示仍输出到 stderr 文本。

### 成功指标

| 指标 | 当前 | 目标 | 衡量方式 |
|------|------|------|---------|
| 四个场景提示信息均含操作指引 | 3/4 | 4/4 | 代码 review 确认 |
| 消息模板更新无需发版 | 不支持 | 支持 | 运维演练：改 Nacos 文案 → 下次 push 生效 |
| Nacos 不可用时 hook 正常工作 | N/A | 支持（回退默认） | 单测 + 手动断开 Nacos 验证 |

---

## 4. 目标用户

| 角色 | 描述 | 痛点 |
|------|------|------|
| 普通开发者 | 非资深研发，日常 git 操作熟悉基础命令，但遇到 rebase/amend 等操作需要指引 | push 被拒绝后不知道怎么改 |
| 运维/管理员 | 负责部署和维护 hook，有 Nacos 操作权限 | 每次改文案要走发布流程 |
| 团队负责人 | 制定分支规范的人 | 需要及时调整拦截提示以反映最新规范 |

---

## 5. 用户故事与需求

### P0 — 必须有

| # | 用户故事 | 验收标准 |
|---|---------|---------|
| P0-1 | 作为普通开发者，当我强制推送被拒时，我需要看到明确的修复步骤 | 场景4提示包含【如何处理】章节，步骤包含 `git pull --rebase` 和 `git push`，并说明冲突处理方式 |
| P0-2 | 作为普通开发者，被拒绝时我需要知道是哪个 commit 出了问题（场景2/3） | 提示中显示 commitid 和 commit subject（场景2）/ committer email（场景3） |
| P0-3 | 作为运维，我可以修改 Nacos 中的消息模板，下次 push 触发拦截时生效新文案 | 修改 Nacos YAML → 触发 push → stderr 输出新文案，全程无需重启或部署 |
| P0-4 | 作为运维，即使 Nacos 暂时不可用，hook 仍正常工作并使用内置默认模板 | 断开 Nacos 连接后，push 拦截依然生效，提示内容使用代码内置默认值 |
| P0-5 | 作为运维，消息模板支持动态变量（项目名、用户名、分支名等） | Nacos 模板中 `{{.BranchName}}` 等占位符在输出时正确替换为实际值 |

### P1 — 应该有

| # | 用户故事 | 验收标准 |
|---|---------|---------|
| P1-1 | 作为普通开发者，场景1的指引中直接给出目标分支名，而不是占位符 `<target-branch>` | 场景1提示的操作步骤中显示实际分支名，如 `git checkout -b feature/xxx origin/main` |
| P1-2 | 作为普通开发者，场景2修改多提交的指引中直接给出实际目标分支名 | `git merge-base origin/<branch>` 中替换为实际分支名 |
| P1-3 | 作为管理员，我可以在 Nacos 中只覆盖部分场景的模板，其余保持默认 | Nacos 配置中未定义的场景 key 自动使用代码内置默认模板 |

### P2 — 以后考虑

| # | 用户故事 | 验收标准 |
|---|---------|---------|
| P2-1 | 作为管理员，可以配置不同项目使用不同的消息模板 | 暂不实现，当前统一模板 |
| P2-2 | 作为开发者，提示信息支持多语言（中/英） | 暂不实现 |

---

## 6. 功能规格

### 6.1 消息结构统一规范

四个场景的提示格式统一为三段式：

```
<标题行：一句话说明违规原因>

【规范要求】
<规则说明>

<上下文信息块>

【如何处理】
<编号步骤，每步一条命令，命令行缩进>
```

上下文信息块字段对照：

| 字段 | 场景1 | 场景2 | 场景3 | 场景4 |
|------|-------|-------|-------|-------|
| project | ✓ | ✓ | ✓ | ✓ |
| repo | ✓ | ✓ | ✓ | ✓ |
| user | ✓ | ✓ | ✓ | ✓ |
| branch | ✓ | ✓ | ✓ | ✓ |
| protocol | ✓ | — | — | — |
| commitid | — | ✓ | ✓ | — |
| commit subject | — | ✓ | — | — |
| committer email | — | — | ✓ | — |

### 6.2 场景4 补充内容规格（P0-1）

当前缺失，新增内容：

```
禁止强制推送（non fast-forward / rewrite history）

project: {{.ProjectPath}}
repo:    {{.RepoName}}
user:    {{.Username}}
branch:  {{.BranchName}}

【如何处理】
您的推送被拒绝是因为本地历史与远端不一致（rebase 或 reset 导致）。

1) 拉取远端最新代码并合并本地改动：
   git pull --rebase origin {{.BranchName}}

2) 如果出现冲突，逐个解决冲突文件后继续：
   # 解决冲突后
   git add <冲突文件>
   git rebase --continue

3) 推送：
   git push origin {{.BranchName}}

注意：请勿使用 git push --force，这会覆盖他人的提交。
如确认需要强制推送（例如个人分支整理），请联系管理员临时解除限制。
```

### 6.3 场景1 protocol 字段

`ViolationContext` 结构体新增 `Protocol string` 字段。  
`rule.go` 中构建 `msgCtx` 时赋值：`msgCtx.Protocol = e.GLProtocol`。

### 6.4 Nacos 消息模板配置规格

**Nacos DataID**: `gitlab-hook-messages`（可通过 YAML 配置覆盖）  
**Group**: 与现有 hook 配置使用同一 Group  
**格式**: YAML

```yaml
# gitlab-hook-messages (Nacos DataID)
messages:
  non_fast_forward: |
    禁止强制推送（non fast-forward / rewrite history）

    project: {{.ProjectPath}}
    repo:    {{.RepoName}}
    user:    {{.Username}}
    branch:  {{.BranchName}}

    【如何处理】
    ...（完整模板内容）

  direct_push_denied: |
    当前分支禁止直接 push，请通过 Merge Request 合并代码。
    ...

  committer_mismatch: |
    提交人信息不符合规范：commit 提交人与 push 用户不一致
    ...

  task_id_missing: |
    提交信息不符合规范：缺少任务ID（[#TSK-...] 或 [#DEF-...]）
    ...
```

**占位符对照表**（Go `text/template` 语法）：

| 占位符 | 对应字段 | 所有场景可用 |
|--------|---------|------------|
| `{{.ProjectPath}}` | GL_PROJECT_PATH | ✓ |
| `{{.RepoName}}` | 项目路径最后一段 | ✓ |
| `{{.Username}}` | GL_USERNAME | ✓ |
| `{{.BranchName}}` | 短分支名 | ✓ |
| `{{.Protocol}}` | GL_PROTOCOL | 场景1 |
| `{{.CommitID}}` | commit SHA | 场景2/3 |
| `{{.CommitSubject}}` | commit subject | 场景2 |
| `{{.CommitterEmail}}` | committer email | 场景3 |

### 6.5 消息加载与热更新流程

```
hook 启动
  └─> 尝试从 Nacos 加载 gitlab-hook-messages
       ├─ 成功 → 解析 YAML，编译 Go 模板，存入内存
       └─ 失败/超时 → 使用代码内置默认模板，打印 warning 到 stderr

Nacos 推送变更（Watch 回调）
  └─> 重新解析 YAML，重新编译模板，原子替换内存中的模板对象

push 触发 hook
  └─> 从内存模板渲染消息，填入 ViolationContext，输出到 stderr
```

**线程安全**：模板对象使用 `sync.RWMutex` 或 `atomic.Value` 保护，Watch 回调写入时不阻塞正在渲染的 goroutine。

### 6.6 函数重命名（建议）

| 旧名称 | 新名称 |
|--------|--------|
| `ForcePush()` | `NonFastForwardMessage()` |
| `ProtectedBranchDirect()` | `DirectPushDeniedMessage()` |
| `CommitterMismatch()` | `CommitterMismatchMessage()` |
| `MissingTaskID()` | `TaskIDMissingMessage()` |

---

## 7. 开放问题

| 问题 | Owner | 截止 |
|------|-------|------|
| Nacos DataID 和 Group 的命名规范由谁确认？ | 运维 | 开发前 |
| 消息模板 YAML 是否需要版本号字段，以支持回滚？ | 架构 | 开发前 |
| 场景4 注意事项中"联系管理员临时解除限制"，是否需要附上联系方式模板变量？ | 产品 | 开发前 |
| Nacos Watch 超时时间阈值是多少？ | 运维 | 开发前 |

---

## 8. 交付范围与阶段

### Phase 1：消息内容完善（独立可交付）

- 补全场景4的【如何处理】章节
- 场景1/2 指引中替换为实际分支名（P1-1/P1-2）
- `ViolationContext` 新增 `Protocol` 字段
- 函数重命名（可选）
- 更新 `message_test.go` golden string

**验收**：触发四个场景，stderr 内容符合规格；所有现有测试通过。

### Phase 2：Nacos 消息模板动态化

- `internal/message/` 新增模板加载器（`loader.go`）
- 从 Nacos 拉取 `gitlab-hook-messages`，解析并编译 Go 模板
- Watch 回调实现热更新，`atomic.Value` 保证并发安全
- Nacos 不可用时回退内置默认模板（内置模板即 Phase 1 的完善版本）
- `internal/config/model.go` 新增 `MessagesDataID` 配置项（可选，有默认值）
- 集成测试：mock Nacos 推送新模板，验证下次渲染使用新内容

**验收**：
1. 修改 Nacos 中模板文案 → 不重启 hook → 下次 push 拦截输出新文案
2. 断开 Nacos → push 拦截正常工作 → 使用内置默认模板
3. 模板中非法 Go 模板语法 → hook 打印 warning，回退默认模板，不崩溃

---

## 附录：各场景完整消息模板参考

（供开发实现时使用，最终文案以 Nacos 配置为准）

### 场景1 — 禁止直接 push 到受保护分支

```
当前分支禁止直接 push，请通过 Merge Request 合并代码。

【规范要求】
该分支属于受保护分支，只允许通过 GitLab Merge Request 合并，
不允许本地直接 git push。

project:  {{.ProjectPath}}
repo:     {{.RepoName}}
user:     {{.Username}}
branch:   {{.BranchName}}
protocol: {{.Protocol}}

【如何处理】
1) 从目标分支创建新的功能分支：
   git checkout -b feature/<你的功能名> origin/{{.BranchName}}

2) 在新分支上完成开发，然后推送功能分支：
   git push origin feature/<你的功能名>

3) 登录 GitLab，发起 Merge Request，目标分支选择：
   {{.BranchName}}

4) 等待审核通过后，由 GitLab 自动合并。
```

### 场景2 — Commit 信息缺少任务 ID

```
提交信息不符合规范：缺少任务ID（[#TSK-...] 或 [#DEF-...]）

【规范要求】commit subject 必须包含任务ID（二选一）
  [#TSK-xxx]  或  [#DEF-xxx]

project:  {{.ProjectPath}}
repo:     {{.RepoName}}
user:     {{.Username}}
branch:   {{.BranchName}}
commitid: {{.CommitID}}
commit:   {{.CommitSubject}}

【如何修改】
以下操作会重写提交历史，请先做好保护：
1) 确认工作区干净：
   git status
2) 创建备份分支（出错可恢复）：
   git branch backup/before-fix HEAD

A) 只修改最近一次提交：
   git commit --amend -m "<原提交信息> [#TSK-xxx]"
   git push --force-with-lease

B) 修改多次历史提交（rebase 方式）：
   方式1：修改最近 N 次提交（把 5 替换为实际数量）
   git rebase -i HEAD~5
   # 将需要修改的行从 pick 改为 reword，保存后按提示逐个修改

   方式2：从分叉点开始（推荐，不用数数量）
   base=$(git merge-base origin/{{.BranchName}} HEAD)
   git rebase -i "$base"

   完成后推送：
   git push --force-with-lease
```

### 场景3 — 提交人与 push 用户不一致

```
提交人信息不符合规范：commit 提交人与 push 用户不一致

【规范要求】
commit 的 committer 必须与当前 push 的 GitLab 用户一致。

project:         {{.ProjectPath}}
repo:            {{.RepoName}}
push user:       {{.Username}}
committer email: {{.CommitterEmail}}
branch:          {{.BranchName}}
commitid:        {{.CommitID}}

【如何修改】
1) 查看当前本地 Git 用户配置：
   git config user.name
   git config user.email

2) 将本地用户改为与 GitLab 一致：
   git config user.name "<GitLab用户名>"
   git config user.email "<GitLab用户名>@<公司域名>"

3) 修正最近一次提交的 committer：
   git commit --amend --reset-author --no-edit

4) 如需修改多次历史提交：
   git rebase -i HEAD~5
   # 将需要修改的行标记为 edit，逐个执行：
   git commit --amend --reset-author --no-edit
   git rebase --continue

5) 推送修正后的提交：
   git push --force-with-lease
```

### 场景4 — 禁止强制推送

```
禁止强制推送（non fast-forward / rewrite history）

project: {{.ProjectPath}}
repo:    {{.RepoName}}
user:    {{.Username}}
branch:  {{.BranchName}}

【如何处理】
您的推送被拒绝，因为本地历史与远端不一致（可能是执行了 rebase 或 reset）。

1) 拉取远端最新代码并将本地提交追加在后面：
   git pull --rebase origin {{.BranchName}}

2) 如果出现冲突，解决冲突后继续：
   # 编辑冲突文件，解决后执行：
   git add <冲突文件>
   git rebase --continue

3) 冲突全部解决后，正常推送：
   git push origin {{.BranchName}}

注意：请勿使用 git push --force，这会覆盖其他人的提交。
如您确认需要强制推送（例如整理个人开发分支），请联系管理员处理。
```
