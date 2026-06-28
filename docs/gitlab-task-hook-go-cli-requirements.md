# GitLab pre-receive Hook Go CLI 详细需求文档

文档名称：`gitlab-task-hook-go-cli-requirements.md`  
建议保存位置：`docs/gitlab-task-hook-go-cli-requirements.md`  
当前生成位置：`/mnt/data/gitlab-task-hook-go-cli-requirements.md`  
适用对象：Codex / Go 开发者 / DevOps 平台维护人员  
目标产物：`gitlab-task-hook` Go CLI 二进制程序  

---

## 1. 项目背景

当前 GitLab 代码提交规范通过 `pre-receive` server hook 的 Shell 脚本实现。该脚本部署在 Gitaly 的 `custom_hooks_dir/pre-receive.d/` 目录下，用于在用户执行 `git push` 时进行服务端校验。

现有 Shell 脚本已承载较多规则，包括：

1. root 用户跳过所有校验。
2. 禁止非 root 用户强制推送。
3. 区分 push 与 GitLab Web / MR 合并动作。
4. 仅对指定分支进行任务号校验。
5. 任务号支持 `[#TSK-...]` 和 `[#DEF-...]`。
6. 支持用户白名单、分支白名单、项目白名单。
7. 支持 merge commit 豁免。
8. 支持提交人与 push 人一致性校验。
9. 支持 `HOOK_MODE=enforce|warn`。
10. 只校验本次 push 新引入到仓库的 commit。
11. 发现第一个违规只输出一次提示并结束。

由于规则复杂度已经超过 Shell 脚本的舒适区，计划使用 Go 语言开发一个独立 CLI，作为 GitLab `pre-receive` hook 的核心校验程序。

---

## 2. 项目目标

### 2.1 总体目标

开发一个 Go 语言 CLI，名称建议为：

```bash
gitlab-task-hook
```

该 CLI 由 GitLab / Gitaly 在 `pre-receive` hook 阶段调用，用于统一执行代码 push 门禁规则。

### 2.2 设计目标

1. 替代当前复杂 Shell 脚本。
2. 保持当前业务规则不变。
3. 规则逻辑清晰、可测试、可维护。
4. 二进制单文件部署，无运行时依赖。
5. 支持后续扩展更多代码治理规则。
6. 适合多 Gitaly 节点统一分发部署。

---

## 3. 目标部署结构

建议目录结构：

```text
/var/opt/gitlab/gitaly/custom_hooks/
├── bin/
│   └── gitlab-task-hook
└── pre-receive.d/
    └── 01-task-id-check
```

其中 `01-task-id-check` 是极薄 Shell wrapper：

```sh
#!/bin/sh
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
```

权限要求：

```bash
chown git:git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
chmod 755 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook

chown git:git /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check
chmod 755 /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check
```

---

## 4. Hook 输入协议

### 4.1 stdin 输入格式

GitLab / Gitaly 触发 `pre-receive` hook 时，不通过命令行参数传递 ref 更新信息，而是通过标准输入 stdin 传入。

每行格式：

```text
<old-value> <new-value> <ref-name>
```

示例：

```text
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb refs/heads/dev/login
```

字段说明：

| 字段 | 含义 |
|---|---|
| `old-value` | 远端 ref 更新前指向的 commit SHA |
| `new-value` | 远端 ref 更新后要指向的 commit SHA |
| `ref-name` | 被更新的完整 ref 名称，例如 `refs/heads/dev/login` |

### 4.2 全 0 SHA

全 0 SHA 固定为：

```text
0000000000000000000000000000000000000000
```

含义：

| 场景 | old-value | new-value |
|---|---|---|
| 新建分支 | 全 0 SHA | 新 commit SHA |
| 删除分支 | 旧 commit SHA | 全 0 SHA |

### 4.3 多 ref 场景

一次 push 可能更新多个 ref，例如：

```bash
git push origin dev feature/login
```

此时 stdin 可能包含多行：

```text
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb refs/heads/dev
0000000000000000000000000000000000000000 cccccccccccccccccccccccccccccccccccccccc refs/heads/feature/login
```

程序必须循环读取所有行。

---

## 5. 环境变量

### 5.1 GitLab 注入环境变量

程序需要读取以下 GitLab/Gitaly hook 环境变量：

| 环境变量 | 说明 | 示例 |
|---|---|---|
| `GL_USERNAME` | 当前触发操作的 GitLab 用户名 | `root`、`zhangsan` |
| `GL_PROJECT_PATH` | 当前项目完整路径 | `group/subgroup/demo-service` |
| `GL_PROTOCOL` | 当前操作来源 | `http`、`ssh`、`web` |

### 5.2 自定义环境变量

当前明确要求仅支持：

| 环境变量 | 默认值 | 可选值 | 说明 |
|---|---|---|---|
| `HOOK_MODE` | `enforce` | `enforce` / `warn` | 控制违规时是否阻塞 |

行为：

| HOOK_MODE | 违规时行为 |
|---|---|
| 未设置 | 等同 `enforce`，输出提示并 `exit 1` |
| `enforce` | 输出提示并 `exit 1` |
| `warn` | 输出提示并 `exit 0` |

要求：

1. 提示文案中不要暴露 `warn` 模式。
2. 不输出“当前 warn 模式”。
3. 不输出“本次未阻断”。
4. 不做违规计数。
5. 不收集多个违规样例。
6. 遇到第一个违规立即输出一次提示并结束。

---

## 6. 内置默认配置

以下配置可作为 Go 代码中的默认常量。后续如需扩展，可再改为配置文件或更多环境变量。

### 6.1 分支校验范围

建议使用收紧版正则，避免误匹配 `develop`：

```regex
^refs/heads/(feature|dev)(/|_|-|$)
```

匹配示例：

| ref | 是否匹配 |
|---|---:|
| `refs/heads/dev` | 是 |
| `refs/heads/dev/login` | 是 |
| `refs/heads/dev-login` | 是 |
| `refs/heads/dev_login` | 是 |
| `refs/heads/feature` | 是 |
| `refs/heads/feature/login` | 是 |
| `refs/heads/feature-login` | 是 |
| `refs/heads/develop` | 否 |
| `refs/heads/master` | 否 |
| `refs/heads/main` | 否 |
| `refs/heads/SIT_A2026001` | 否 |

### 6.2 任务号格式

任务号支持：

```text
[#TSK-xxx]
[#DEF-xxx]
```

默认正则：

```regex
\[#(TSK|DEF)-[^\[\]]+\]
```

规则：

1. 必须以 `[#TSK-` 或 `[#DEF-` 开头。
2. 必须以 `]` 结尾。
3. `xxx` 至少 1 个字符。
4. `xxx` 不允许包含 `[` 或 `]`。
5. 只校验 commit subject，不校验 body。

合法示例：

```text
add login api [#TSK-1001]
fix bug [#DEF-A20260001]
[#TSK-anything] update service
```

非法示例：

```text
add login api
add login api #TSK-1001
add login api [TSK-1001]
add login api [#BUG-1001]
add login api [#TSK-]
```

### 6.3 用户白名单

默认：

```text
WHITELIST_USERS=''
```

语义：

1. 多个用户可用空格分隔。
2. 大小写不敏感。
3. 命中后仅跳过任务号校验。
4. 不跳过强推校验。
5. 不跳过提交人与 push 人一致性校验。
6. root 不走该普通白名单逻辑，root 单独处理。

### 6.4 分支白名单

默认正则：

```regex
^refs/heads/(init/|migrate/|tmp/)
```

语义：

1. 命中后跳过任务号校验。
2. 不跳过强推校验。
3. 不跳过提交人与 push 人一致性校验。

匹配示例：

| ref | 是否命中 |
|---|---:|
| `refs/heads/init/base` | 是 |
| `refs/heads/migrate/gitlab16` | 是 |
| `refs/heads/tmp/test` | 是 |
| `refs/heads/dev/test` | 否 |

### 6.5 项目白名单

默认：

```text
WHITELIST_PROJECT_NAMES=''
```

需求：

1. 项目指 Git 项目名称，即 `GL_PROJECT_PATH` 最后一段。
2. 多个项目通过英文逗号 `,` 分隔。
3. 允许逗号前后存在空格。
4. 大小写不敏感。
5. 命中后仅跳过任务号校验。
6. 不跳过强推校验。
7. 不跳过提交人与 push 人一致性校验。

示例：

```text
GL_PROJECT_PATH=group/subgroup/demo-service
repo name=demo-service
WHITELIST_PROJECT_NAMES=demo-service,legacy-repo,migrate-tool
```

匹配结果：

| GL_PROJECT_PATH | repo name | 是否命中 |
|---|---|---:|
| `group/demo-service` | `demo-service` | 是 |
| `group/subgroup/legacy-repo` | `legacy-repo` | 是 |
| `group/migrate-tool` | `migrate-tool` | 是 |
| `group/other-service` | `other-service` | 否 |

### 6.6 Message 白名单

默认：

```text
EXEMPT_MESSAGE_REGEX=''
```

语义：

1. 为空时不启用。
2. 非空时匹配 commit subject。
3. 命中后仅跳过任务号校验。
4. 不跳过强推校验。
5. 不跳过提交人与 push 人一致性校验。

### 6.7 Merge commit 豁免

默认：

```text
EXEMPT_MERGE_COMMIT=true
```

语义：

1. merge commit 跳过任务号校验。
2. merge commit 建议也跳过提交人与 push 人一致性校验，避免 IDE 合并冲突或 Git 工具自动生成 committer 场景误伤。

merge commit 判断方式：

```bash
git rev-list --parents -n 1 <commit>
```

如果输出字段数 >= 3，表示该 commit 至少有 2 个 parent，即 merge commit。

---

## 7. 规则优先级

Go CLI 必须按以下顺序执行规则：

| 优先级 | 规则 | 动作 |
|---:|---|---|
| 1 | `GL_USERNAME=root` | 跳过所有校验，直接放行 |
| 2 | 删除 ref | 放行 |
| 3 | tag ref | 放行 |
| 4 | non fast-forward | 根据 HOOK_MODE 阻塞或提示 |
| 5 | `GL_PROTOCOL=web` | 跳过后续 push 类校验 |
| 6 | 计算本次 push 新引入 commit | `git rev-list <new> --not --all` |
| 7 | merge commit 豁免 push 类 commit 校验 | 跳过该 commit 的提交人和任务号校验 |
| 8 | 提交人与 push 人一致性校验 | 不一致则违规 |
| 9 | 分支是否命中任务号校验范围 | 不命中则不做任务号校验 |
| 10 | 用户白名单 | 跳过任务号校验 |
| 11 | 分支白名单 | 跳过任务号校验 |
| 12 | 项目白名单 | 跳过任务号校验 |
| 13 | message 白名单 | 跳过任务号校验 |
| 14 | commit subject 任务号校验 | 不合规则违规 |

---

## 8. 详细功能需求

### 8.1 root 用户跳过所有校验

如果：

```text
GL_USERNAME=root
```

大小写不敏感，直接放行所有 ref 更新。

行为：

| 用户 | 行为 |
|---|---|
| `root` | 跳过所有校验 |
| `Root` | 跳过所有校验 |
| `ROOT` | 跳过所有校验 |
| `admin` | 正常校验 |
| 普通 Maintainer | 正常校验 |

### 8.2 删除 ref 放行

如果：

```text
new-value = 0000000000000000000000000000000000000000
```

直接放行。

### 8.3 tag 放行

如果：

```text
ref-name 以 refs/tags/ 开头
```

直接放行。

### 8.4 禁止 non fast-forward

对非 root 用户，如果 old-value 不是全 0，则执行：

```bash
git merge-base <old-value> <new-value>
```

判断：

1. 如果 merge-base 为空，违规。
2. 如果 merge-base 不等于 old-value，违规。
3. 否则是 fast-forward，放行。

违规输出强推提示。

### 8.5 Web / MR 合并动作不做 push 类校验

如果：

```text
GL_PROTOCOL=web
```

表示 GitLab Web 动作，例如 MR 合并、Web 操作。

行为：

1. 强推校验已在前面执行。
2. 后续跳过提交人与 push 人一致性校验。
3. 后续跳过任务号校验。

### 8.6 计算本次 push 新引入 commit

使用：

```bash
git rev-list <new-value> --not --all
```

含义：

```text
获取从 new-value 可达，但从当前仓库所有 refs 不可达的 commit。
```

也就是“本次 push 新引入到仓库的 commit”。

该设计用于避免合并历史分支时重新校验历史老 commit。

### 8.7 提交人与 push 人一致性校验

需求：

> 判断提交人信息和 push 人用户名相同。

建议校验 committer，不校验 author。

获取 committer email：

```bash
git log --pretty=format:%ce -1 <commit>
```

解析规则：

```text
zhangsan@example.com -> zhangsan
```

对比：

```text
committer email 的 @ 前缀 == GL_USERNAME
```

大小写不敏感。

适用范围：

| 场景 | 是否校验 |
|---|---:|
| root | 否 |
| 删除 ref | 否 |
| tag | 否 |
| GL_PROTOCOL=web | 否 |
| merge commit | 建议否 |
| 普通 push | 是 |
| 项目白名单 | 是 |
| 分支白名单 | 是 |
| 用户白名单 | 是 |
| 任务号校验范围外分支 | 是 |

失败时输出“提交人与 push 人不一致”提示。

### 8.8 任务号校验

仅当分支命中：

```regex
^refs/heads/(feature|dev)(/|_|-|$)
```

才做任务号校验。

获取 commit subject：

```bash
git log --pretty=format:%s -1 <commit>
```

subject 必须包含：

```text
[#TSK-xxx]
```

或：

```text
[#DEF-xxx]
```

正则：

```regex
\[#(TSK|DEF)-[^\[\]]+\]
```

失败时输出“提交信息不符合规范”提示。

---

## 9. 输出要求

### 9.1 输出位置

所有提示输出到 stderr。

Go 示例：

```go
fmt.Fprintln(os.Stderr, msg)
```

### 9.2 只输出一次

遇到第一个违规：

1. 输出一次提示。
2. 根据 HOOK_MODE 返回退出码。
3. 不继续检查后续 commit。
4. 不输出计数。
5. 不输出汇总。
6. 不输出多个样例。

### 9.3 强推提示文案

```text
禁止强制推送（non fast-forward / rewrite history）

project: <GL_PROJECT_PATH 或 unknown>
repo: <repo name 或 unknown>
user: <GL_USERNAME 或 unknown>
branch: <branch name>
```

### 9.4 提交人与 push 人不一致提示文案

```text
提交人信息不符合规范：commit 提交人与 push 用户不一致

【规范要求】
commit 的 committer 用户必须与当前 push 的 GitLab 用户一致。

project: <GL_PROJECT_PATH 或 unknown>
repo: <repo name 或 unknown>
push user: <GL_USERNAME 或 unknown>
committer email: <committer email 或 unknown>
branch: <branch name>

commitid: <commit id>

【如何修改】
请确认本地 Git 用户信息是否与 GitLab 用户一致：

1) 查看当前 Git 用户信息
   git config user.name
   git config user.email

2) 修改当前仓库的 Git 用户信息
   git config user.name "<GitLab用户名>"
   git config user.email "<GitLab用户名>@<your-domain>"

3) 修正最近一次提交的 committer 信息
   git commit --amend --reset-author

4) 如果需要修改多个历史提交，请使用 interactive rebase：
   git rebase -i HEAD~5
   # 将需要修改的 commit 标记为 edit
   # 逐个执行：
   git commit --amend --reset-author
   git rebase --continue

5) 推送修正后的提交
   git push --force-with-lease
```

### 9.5 任务号缺失提示文案

```text
提交信息不符合规范：缺少任务ID（[#TSK-...] 或 [#DEF-...]）

【规范要求】commit subject 必须包含任务ID（二选一）
  [#TSK-xxx]  或  [#DEF-xxx]

project: <GL_PROJECT_PATH 或 unknown>
repo: <repo name 或 unknown>
user: <GL_USERNAME 或 unknown>
branch: <branch name>

commitid: <commit id>
commit: <commit subject>

【如何修改】
以下操作可能涉及“重写历史（rewrite history）”。为避免代码丢失，请先确认：
1) 工作区干净（working tree clean）：git status
2) 建一个备份分支（backup branch）：git branch backup/<name> HEAD
   如误操作，可用 git reflog 找回提交

A) 只修改最近一次提交（amend）
   git commit --amend -m "<原提交信息> [#TSK-xxx]"
   # 或
   git commit --amend -m "<原提交信息> [#DEF-xxx]"
   git push --force-with-lease

B) 修改多次提交（interactive rebase / reword）
   方式1：按数量修改（示例：最近 5 个提交）
   git rebase -i HEAD~5
   # 将需要修改的提交从 pick 改为 reword，保存退出后按提示逐个修改
   git push --force-with-lease

   方式2：从分叉点开始（更稳，适合不知道数量）
   base=$(git merge-base origin/<target-branch> HEAD)
   git rebase -i "$base"
   git push --force-with-lease
```

---

## 10. 退出码要求

| 场景 | HOOK_MODE | 退出码 |
|---|---|---:|
| 无违规 | 任意 | `0` |
| root 用户 | 任意 | `0` |
| 删除分支 | 任意 | `0` |
| tag | 任意 | `0` |
| GL_PROTOCOL=web | 任意 | `0`，强推违规除外 |
| 强推违规 | enforce / 未设置 | `1` |
| 强推违规 | warn | `0` |
| 提交人与 push 人不一致 | enforce / 未设置 | `1` |
| 提交人与 push 人不一致 | warn | `0` |
| 缺少任务号 | enforce / 未设置 | `1` |
| 缺少任务号 | warn | `0` |

---

## 11. 真实 push 场景样例

### 11.1 普通 push 合规

```bash
git checkout dev/login
git commit -m "add login api [#TSK-1001]"
git push origin dev/login
```

环境：

```text
GL_USERNAME=zhangsan
GL_PROJECT_PATH=group/demo-service
GL_PROTOCOL=http
HOOK_MODE=enforce
```

stdin：

```text
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb refs/heads/dev/login
```

行为：

```text
通过，exit 0
```

### 11.2 普通 push 缺任务号

```bash
git checkout dev/login
git commit -m "add login api"
git push origin dev/login
```

行为：

```text
输出任务号缺失提示
HOOK_MODE=enforce 时 exit 1
```

### 11.3 提交人与 push 人不一致

环境：

```text
GL_USERNAME=zhangsan
```

commit committer email：

```text
lisi@example.com
```

行为：

```text
输出提交人与 push 人不一致提示
```

### 11.4 root 用户 push

环境：

```text
GL_USERNAME=root
```

行为：

```text
跳过所有校验，exit 0
```

### 11.5 MR 合并

环境：

```text
GL_PROTOCOL=web
```

行为：

```text
跳过提交人与任务号校验
exit 0
```

### 11.6 删除分支

stdin：

```text
cccccccccccccccccccccccccccccccccccccccc 0000000000000000000000000000000000000000 refs/heads/feature/login
```

行为：

```text
直接放行
```

### 11.7 项目白名单

环境：

```text
GL_PROJECT_PATH=group/legacy-service
```

配置：

```text
WHITELIST_PROJECT_NAMES=legacy-service,migrate-tool
```

行为：

```text
跳过任务号校验
仍执行强推校验和提交人与 push 人一致性校验
```

---

## 12. Go 架构设计建议

### 12.1 推荐代码结构

```text
gitlab-task-hook/
├── cmd/
│   └── gitlab-task-hook/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── env/
│   │   └── env.go
│   ├── git/
│   │   └── git.go
│   ├── hook/
│   │   ├── input.go
│   │   ├── rule.go
│   │   └── validator.go
│   └── message/
│       └── message.go
├── go.mod
└── README.md
```

### 12.2 模块职责

| 模块 | 职责 |
|---|---|
| `cmd/gitlab-task-hook/main.go` | 程序入口，读取 stdin，执行规则 |
| `internal/config` | 默认规则常量和模式配置 |
| `internal/env` | 读取并封装 GitLab 环境变量 |
| `internal/git` | 封装 git 命令调用 |
| `internal/hook/input.go` | 解析 pre-receive stdin |
| `internal/hook/rule.go` | 规则优先级编排 |
| `internal/hook/validator.go` | 具体校验函数 |
| `internal/message` | 输出文案 |

### 12.3 Git 命令封装

需要封装以下 git 命令：

```bash
git merge-base <old> <new>
git rev-list <new> --not --all
git rev-list --parents -n 1 <commit>
git log --pretty=format:%s -1 <commit>
git log --pretty=format:%ce -1 <commit>
```

要求：

1. 命令执行失败时返回明确错误。
2. 对空输出做防御处理。
3. 错误信息不要泄露敏感 token。
4. 所有命令默认在当前仓库上下文执行。

---

## 13. 测试要求

### 13.1 单元测试

至少覆盖以下纯函数：

| 测试项 | 输入 | 期望 |
|---|---|---|
| root 判断 | `root` | true |
| root 判断 | `Root` | true |
| root 判断 | `zhangsan` | false |
| repo name 提取 | `group/demo` | `demo` |
| repo name 提取 | `group/sub/demo` | `demo` |
| 项目白名单 | `demo`, `demo,app` | true |
| 项目白名单 | `demo`, `app,tool` | false |
| 任务号正则 | `fix [#TSK-1]` | true |
| 任务号正则 | `fix [#DEF-abc]` | true |
| 任务号正则 | `fix [#BUG-1]` | false |
| 分支匹配 | `refs/heads/dev` | true |
| 分支匹配 | `refs/heads/dev/login` | true |
| 分支匹配 | `refs/heads/develop` | false |
| 分支匹配 | `refs/heads/master` | false |
| email 前缀 | `zhangsan@example.com` | `zhangsan` |
| committer 对比 | `zhangsan@example.com` vs `zhangsan` | true |
| committer 对比 | `lisi@example.com` vs `zhangsan` | false |

### 13.2 集成测试

建议使用临时 bare repo 模拟 pre-receive 环境。

覆盖场景：

1. 合规 push 成功。
2. 缺任务号 push 失败。
3. `HOOK_MODE=warn` 时缺任务号但退出码为 0。
4. root 用户跳过所有校验。
5. tag 放行。
6. 删除分支放行。
7. 强推被拒绝。
8. `GL_PROTOCOL=web` 跳过 push 类校验。
9. 项目白名单跳过任务号校验。
10. 提交人与 push 人不一致被拒绝。
11. merge commit 被豁免。
12. 一次 push 多个 commit 时只输出一次违规提示。

---

## 14. 构建要求

### 14.1 x86_64 构建

```bash
GOOS=linux GOARCH=amd64 go build -o gitlab-task-hook ./cmd/gitlab-task-hook
```

### 14.2 ARM64 构建

```bash
GOOS=linux GOARCH=arm64 go build -o gitlab-task-hook ./cmd/gitlab-task-hook
```

### 14.3 版本信息建议

建议支持：

```bash
gitlab-task-hook --version
```

输出示例：

```text
gitlab-task-hook version v1.0.0
```

可通过 Go build ldflags 注入：

```bash
go build -ldflags "-X main.version=v1.0.0" -o gitlab-task-hook ./cmd/gitlab-task-hook
```

---

## 15. 部署要求

### 15.1 上传二进制

```bash
mkdir -p /var/opt/gitlab/gitaly/custom_hooks/bin
cp gitlab-task-hook /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
chown git:git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
chmod 755 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
```

### 15.2 创建 wrapper

```bash
mkdir -p /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d

cat > /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check <<'WRAPPER_EOF'
#!/bin/sh
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
WRAPPER_EOF

chown git:git /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check
chmod 755 /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check
```

### 15.3 验证 hook 是否执行

可临时增加 debug 输出，或使用不合规 commit 测试：

```bash
git commit -m "test commit"
git push origin dev/test
```

期望看到：

```text
remote: 提交信息不符合规范：缺少任务ID
```

---

## 16. 回滚方案

### 16.1 禁用 wrapper

```bash
mv /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check \
   /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check.disabled
```

### 16.2 回滚二进制版本

建议保留版本目录：

```text
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook-v1.0.0
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook-v1.1.0
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook -> gitlab-task-hook-v1.1.0
```

通过软链接切换版本。

---

## 17. 架构设计要求

Codex 在开发前需要先输出：

1. Go CLI 架构设计。
2. 模块拆分说明。
3. 规则执行流程图或文字流程。
4. 错误处理策略。
5. 测试方案。
6. 部署与回滚方案。

确认设计后，再实现代码。

---

## 18. 开发交付要求

Codex 最终需要交付：

1. 完整 Go 源码。
2. 单元测试代码。
3. 集成测试说明或脚本。
4. 构建命令。
5. 部署 wrapper。
6. README。
7. 版本号说明。
8. 示例配置说明。
9. 回滚说明。

---

## 19. 给 Codex 的直接指令

可以将下面这段作为 Codex 的入口指令：

```text
请读取 docs/gitlab-task-hook-go-cli-requirements.md，并基于该需求完成以下工作：

1. 先输出 Go CLI 的架构设计和方案设计，不要直接编码。
2. 设计内容必须包括：模块划分、规则执行顺序、错误处理、测试方案、部署方案和回滚方案。
3. 架构设计确认后，使用 Go 实现 gitlab-task-hook。
4. 程序用于 GitLab pre-receive server hook，从 stdin 读取 <old> <new> <ref>，从环境变量读取 GL_USERNAME、GL_PROJECT_PATH、GL_PROTOCOL、HOOK_MODE。
5. 严格按照需求文档中的规则优先级和提示文案实现。
6. 提供完整单元测试和必要集成测试说明。
7. 提供构建命令、部署 wrapper 和 README。
8. 不要引入第三方依赖，优先使用 Go 标准库。
```

---

## 20. 最终选型结论

### 20.1 Shell 是否合适

Shell 适合：

1. 快速验证规则。
2. 临时上线。
3. 应急修复。

但当前规则已经包含身份校验、任务号校验、分支/项目/用户白名单、push/web 区分、merge commit 豁免、历史 commit 过滤、warn/enforce 模式等能力。继续使用 Shell，后续可维护性和可测试性会较差。

### 20.2 Go 是否合适

Go 更适合当前长期生产场景：

1. 单二进制部署，无运行时依赖。
2. 性能稳定。
3. 规则可模块化。
4. 易做单元测试。
5. 多 Gitaly 节点可统一分发。
6. 适合版本化发布和回滚。

### 20.3 最终建议

```text
短期：Shell 继续验证规则。
中期：Go CLI 重写。
长期：Go binary + shell wrapper + 版本化发布。
```
## 新增需求：指定分支禁止直接 push，只允许 MR/Web 合并

### 需求背景
部分关键分支，例如 master、SIT_XXXXX、uat_xxxxx，不允许研发人员本地直接 git push 写入，只能通过 GitLab Merge Request 合并进入目标分支。

### 新增规则
新增“受保护分支禁止直接 push”规则。

当满足以下条件时，拒绝本次 ref 更新：
1. GL_USERNAME 不是 root；
2. ref-name 是分支 ref，即以 refs/heads/ 开头；
3. ref-name 命中 PUSH_DENY_BRANCH_REGEX；
4. GL_PROTOCOL 是 http 或 ssh，或者 GL_PROTOCOL 为空。

当 GL_PROTOCOL=web 时，认为是 GitLab Web/MR 类动作，允许写入该分支。

### 默认正则
新增配置项：

PUSH_DENY_BRANCH_REGEX

默认值：

(?i)^refs/heads/(master|sit_.*|uat_.*)$

说明：
- 匹配 refs/heads/master
- 匹配 refs/heads/SIT_20260101
- 匹配 refs/heads/sit_20260101
- 匹配 refs/heads/UAT_20260101
- 匹配 refs/heads/uat_20260101
- 大小写不敏感

### 规则优先级
该规则放在 non fast-forward 校验之后，GL_PROTOCOL=web 跳过 push 类校验之前。

更新后的规则顺序：
1. root 用户跳过所有校验。
2. 删除 ref 放行。
3. tag ref 放行。
4. non fast-forward 校验。
5. 指定分支禁止直接 push 校验。
6. GL_PROTOCOL=web 时跳过后续 push 类校验。
7. 计算本次 push 新引入 commit。
8. 提交人与 push 人一致性校验。
9. 任务号校验相关规则。

### 伪代码
if !IsRootUser(env.GLUsername) &&
   IsGitPush(env.GLProtocol) &&
   IsDirectPushDeniedBranch(refName, pushDenyBranchRegex) {
    fail(mode, DirectPushDeniedMessage(refName, env))
}

IsGitPush 逻辑：
- GL_PROTOCOL=http：true
- GL_PROTOCOL=ssh：true
- GL_PROTOCOL=""：true，环境变量缺失时按普通 push 处理
- GL_PROTOCOL=web：false

### 错误提示文案
当前分支禁止直接 push，请通过 Merge Request 合并代码。

【规范要求】
该分支属于受保护分支，只允许通过 GitLab Merge Request 合并，不允许本地直接 git push。

project: <GL_PROJECT_PATH 或 unknown>
repo: <repo name 或 unknown>
user: <GL_USERNAME 或 unknown>
branch: <branch name>
protocol: <GL_PROTOCOL 或 unknown>

【如何处理】
1) 请从目标分支拉取新分支进行开发：
   git checkout -b feature/<your-feature> origin/<target-branch>

2) 完成开发后推送 feature/dev 分支：
   git push origin feature/<your-feature>

3) 在 GitLab 页面发起 Merge Request，目标分支选择：
   <branch name>

4) 通过 MR 审核后再合并到目标分支。

### 注意事项
1. 用户白名单、项目白名单、分支白名单不绕过该规则。
2. 只有 root 用户跳过所有校验。
3. GL_PROTOCOL=web 不一定只代表 MR 合并，也可能包含 Web IDE 等 Web 写入动作。若要严格禁止 Web IDE 直接改关键分支，需要配合 GitLab Protected Branch 权限策略。