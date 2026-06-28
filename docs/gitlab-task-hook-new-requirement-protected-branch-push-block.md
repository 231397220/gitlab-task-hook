# GitLab Task Hook 新增需求：指定分支禁止直接 Push，但允许合并

文档名称：`gitlab-task-hook-new-requirement-protected-branch-push-block.md`  
建议保存位置：`docs/gitlab-task-hook-new-requirement-protected-branch-push-block.md`  
适用项目：`gitlab-task-hook` Go CLI  
需求类型：新增规则  
目标：通过正则配置指定分支，禁止普通 `git push` 直接推送代码，但允许 GitLab Web/MR 合并代码进入该分支。

---

## 1. 需求背景

当前 `gitlab-task-hook` 已实现 commit message、提交人与 push 人一致性、强推禁止、root 跳过、web 合并跳过 push 类校验等规则。

现新增需求：

> 对指定分支禁止直接 `git push` 代码，但允许通过 MR 合并 / GitLab Web 合并代码。

该需求主要用于保护关键分支，例如：

- `main`
- `master`
- `release/*`
- `hotfix/*`
- `SIT_*`
- `UAT_*`

目标是强制研发人员通过 Merge Request 进入关键分支，避免绕过代码评审和合并流程。

---

## 2. 术语说明

| 术语 | 说明 |
|---|---|
| 直接 push | 用户通过 `git push` 经 SSH/HTTP 协议直接更新远端分支 |
| Web 合并 | GitLab Web 页面上的 Merge Request 合并、Squash Merge、Rebase Merge 等 Web 动作 |
| 受保护分支正则 | 用正则表达式配置需要禁止直接 push 的分支范围 |
| `GL_PROTOCOL` | GitLab server hook 注入的环境变量，常见值：`ssh`、`http`、`web` |

---

## 3. 新增配置项

新增一个分支正则配置项，用于指定哪些分支禁止直接 push。

建议配置项名称：

```text
BLOCK_DIRECT_PUSH_BRANCH_REGEX
```

默认值建议：

```regex
^refs/heads/(main|master|release/.*|hotfix/.*|SIT_.*|UAT_.*)$
```

### 3.1 配置语义

| 配置项 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `BLOCK_DIRECT_PUSH_BRANCH_REGEX` | 正则字符串 | `^refs/heads/(main|master|release/.*|hotfix/.*|SIT_.*|UAT_.*)$` | 匹配禁止直接 push 的完整 ref |

### 3.2 是否使用环境变量

如果当前项目设计坚持“除 `HOOK_MODE` 外不引入环境变量”，可以先作为 Go 代码内置常量实现。

如果希望后续无需重新编译即可调整分支范围，建议支持环境变量覆盖：

```bash
BLOCK_DIRECT_PUSH_BRANCH_REGEX='^refs/heads/(main|master|release/.*|hotfix/.*)$'
```

推荐实现方式：

1. 代码内置默认正则。
2. 如果环境变量 `BLOCK_DIRECT_PUSH_BRANCH_REGEX` 非空，则用环境变量覆盖默认值。
3. 如果正则编译失败，应返回明确错误并拒绝本次 push。

---

## 4. 规则定义

### 4.1 规则名称

指定分支禁止直接 push，允许 Web/MR 合并。

### 4.2 判断逻辑

当本次更新的 ref 满足：

```text
ref-name 匹配 BLOCK_DIRECT_PUSH_BRANCH_REGEX
```

且本次操作来源是普通 push：

```text
GL_PROTOCOL=ssh 或 GL_PROTOCOL=http
```

则判定为违规。

当本次操作来源是 Web 动作：

```text
GL_PROTOCOL=web
```

则允许通过。

### 4.3 root 用户例外

如果：

```text
GL_USERNAME=root
```

则仍然跳过所有校验，包括本规则。

### 4.4 删除分支处理

如果：

```text
new-value = 0000000000000000000000000000000000000000
```

表示删除 ref。

建议：删除受保护分支也应被禁止，除非 root 用户操作。

因此本规则应区分：

| 场景 | 建议行为 |
|---|---|
| root 删除受保护分支 | 放行 |
| 普通用户删除受保护分支 | 拒绝 |
| 普通用户删除非受保护分支 | 放行 |

如果当前希望保持旧逻辑“删除分支直接放行”，则该规则可以暂不拦截删除动作。但从生产治理角度看，建议对受保护分支删除也进行拦截。

推荐实现：

1. root 优先放行。
2. 如果 ref 命中 `BLOCK_DIRECT_PUSH_BRANCH_REGEX`，且 `GL_PROTOCOL` 不是 `web`，则拒绝。
3. 该判断应放在删除 ref 放行之前，避免普通用户删除关键分支。

---

## 5. 规则优先级调整

新增规则后，推荐最终执行顺序如下：

| 优先级 | 规则 | 动作 |
|---:|---|---|
| 1 | `GL_USERNAME=root` | 跳过所有校验 |
| 2 | 指定分支禁止直接 push | 命中受保护分支且非 `web` 来源则拒绝 |
| 3 | 删除非受保护 ref | 放行 |
| 4 | tag ref | 放行 |
| 5 | non fast-forward | 校验强推 |
| 6 | `GL_PROTOCOL=web` | 跳过后续 push 类校验 |
| 7 | 计算本次 push 新引入 commit | `git rev-list <new> --not --all` |
| 8 | 提交人与 push 人一致性校验 | 不一致则违规 |
| 9 | 分支是否在任务号校验范围 | 不在范围则放行任务号校验 |
| 10 | 用户/分支/项目白名单 | 跳过任务号校验 |
| 11 | merge commit 豁免 | 跳过任务号校验 |
| 12 | message 白名单 | 跳过任务号校验 |
| 13 | commit subject 任务号校验 | 不合规则违规 |

---

## 6. 违规提示文案

当用户直接 push 到受保护分支时，输出以下提示到 stderr：

```text
禁止直接 Push 到受保护分支

【规范要求】
当前分支不允许直接 git push，请通过 GitLab Merge Request 合并代码。

project: <GL_PROJECT_PATH 或 unknown>
repo: <repo name 或 unknown>
user: <GL_USERNAME 或 unknown>
branch: <branch name>
protocol: <GL_PROTOCOL 或 unknown>

【处理方式】
1) 请从当前分支创建 feature/dev 分支进行开发。
2) 将代码 push 到 feature/dev 分支。
3) 在 GitLab 页面创建 Merge Request。
4) 通过代码评审后合并到目标分支。

示例：
   git checkout -b feature/<name>
   git push origin feature/<name>
```

### 6.1 HOOK_MODE 行为

该规则是否受 `HOOK_MODE=warn` 影响，需要明确。

推荐：该规则受 `HOOK_MODE` 控制，与其它违规规则一致：

| HOOK_MODE | 行为 |
|---|---|
| `enforce` 或未设置 | 输出提示，`exit 1` |
| `warn` | 输出提示，`exit 0` |

如果业务认为“关键分支禁止直接 push”必须强制阻断，不允许 warn，可将该规则设计为始终 `exit 1`。但为保持现有模式一致性，默认建议仍受 `HOOK_MODE` 控制。

---

## 7. Go 实现建议

### 7.1 新增配置结构字段

```go
type Config struct {
    HookMode                    string
    CheckBranchRegex            *regexp.Regexp
    TaskIDRegex                 *regexp.Regexp
    WhitelistBranchRegex        *regexp.Regexp
    WhitelistProjectNames       []string
    BlockDirectPushBranchRegex  *regexp.Regexp
    ExemptMergeCommit           bool
}
```

### 7.2 新增判断函数

```go
func IsWebAction(protocol string) bool {
    return strings.EqualFold(protocol, "web")
}

func IsDirectPush(protocol string) bool {
    return strings.EqualFold(protocol, "ssh") || strings.EqualFold(protocol, "http") || protocol == ""
}

func IsBlockedDirectPushBranch(ref string, cfg Config) bool {
    if cfg.BlockDirectPushBranchRegex == nil {
        return false
    }
    return cfg.BlockDirectPushBranchRegex.MatchString(ref)
}
```

### 7.3 主流程伪代码

```go
for _, update := range updates {
    if IsRootUser(env.Username) {
        continue
    }

    if IsBlockedDirectPushBranch(update.Ref, cfg) && !IsWebAction(env.Protocol) {
        fail(mode, blockedDirectPushMessage(update.Ref, env))
    }

    if IsDeleteRef(update.NewSHA) {
        continue
    }

    if IsTagRef(update.Ref) {
        continue
    }

    if !IsNewRef(update.OldSHA) && !IsFastForward(update.OldSHA, update.NewSHA) {
        fail(mode, nonFastForwardMessage(update.Ref, env))
    }

    if IsWebAction(env.Protocol) {
        continue
    }

    commits := NewToRepoCommits(update.NewSHA)

    for _, commit := range commits {
        // 提交人与 push 人一致性校验
        // 任务号校验
    }
}
```

---

## 8. 测试用例

### 8.1 直接 push 到受保护分支

环境：

```text
GL_USERNAME=zhangsan
GL_PROTOCOL=ssh
GL_PROJECT_PATH=group/demo-service
```

stdin：

```text
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb refs/heads/master
```

期望：

```text
HOOK_MODE=enforce：exit 1
HOOK_MODE=warn：exit 0
输出“禁止直接 Push 到受保护分支”提示
```

### 8.2 HTTP push 到 release 分支

```text
GL_PROTOCOL=http
ref=refs/heads/release/2026-01
```

期望：拒绝或 warn。

### 8.3 Web/MR 合并到受保护分支

```text
GL_PROTOCOL=web
ref=refs/heads/master
```

期望：放行。

### 8.4 root 直接 push 到受保护分支

```text
GL_USERNAME=root
GL_PROTOCOL=ssh
ref=refs/heads/master
```

期望：放行。

### 8.5 普通用户 push 到非受保护分支

```text
GL_USERNAME=zhangsan
GL_PROTOCOL=ssh
ref=refs/heads/feature/login
```

期望：不触发“禁止直接 Push 到受保护分支”规则，继续执行后续提交人/任务号校验。

### 8.6 普通用户删除受保护分支

```text
GL_USERNAME=zhangsan
GL_PROTOCOL=ssh
new-value=0000000000000000000000000000000000000000
ref=refs/heads/master
```

推荐期望：拒绝。

### 8.7 普通用户删除非受保护分支

```text
GL_USERNAME=zhangsan
GL_PROTOCOL=ssh
new-value=0000000000000000000000000000000000000000
ref=refs/heads/feature/test
```

期望：放行。

---

## 9. 验收标准

1. 可以通过正则配置禁止直接 push 的分支。
2. 普通用户通过 SSH/HTTP 直接 push 到命中正则的分支时被拦截。
3. GitLab Web/MR 合并到命中正则的分支时允许通过。
4. root 用户仍然跳过所有校验。
5. 非受保护分支仍继续执行原有规则。
6. 受保护分支删除动作对普通用户应被拦截。
7. 提示文案清晰告知用户应通过 Merge Request 合并代码。
8. 单元测试覆盖正则匹配、协议判断、root 例外、删除受保护分支、warn/enforce 行为。

---

## 10. 给 Codex 的开发指令

```text
请在现有 gitlab-task-hook Go CLI 中新增规则：指定分支禁止直接 push，但允许 GitLab Web/MR 合并。

详细需求请读取：docs/gitlab-task-hook-new-requirement-protected-branch-push-block.md

实现要求：
1. 新增配置项 BLOCK_DIRECT_PUSH_BRANCH_REGEX，默认值为：
   ^refs/heads/(main|master|release/.*|hotfix/.*|SIT_.*|UAT_.*)$
2. 如果 ref-name 命中该正则，且 GL_PROTOCOL 不是 web，则判定为直接 push 到受保护分支。
3. 直接 push 到受保护分支时输出“禁止直接 Push 到受保护分支”提示。
4. HOOK_MODE=enforce 或未设置时 exit 1；HOOK_MODE=warn 时 exit 0。
5. GL_USERNAME=root 时跳过所有校验，包括本规则。
6. 该规则应在删除 ref 放行之前执行，避免普通用户删除受保护分支。
7. GL_PROTOCOL=web 时允许合并到受保护分支。
8. 保持原有规则不变。
9. 增加单元测试和必要的集成测试说明。
10. 更新 README 和需求文档。
```
