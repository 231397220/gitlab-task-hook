# GitLab Task Hook 配置中心改造需求文档：YAML 配置文件 + Nacos 拉取 + 本地缓存

> 文档用途：本文件用于指导 Claude Code / Codex 在现有 `gitlab-task-hook` Go CLI 基础上完成配置中心化改造、架构设计、方案设计、编码实现与测试。
>
> 建议保存路径：`docs/gitlab-task-hook-nacos-yaml-config-requirements.md`
>
> 目标程序：`gitlab-task-hook`
>
> 适用场景：GitLab / Gitaly `pre-receive` server hook。

---

## 1. 背景

现有 `gitlab-task-hook` 已完成 Go CLI 开发，实现了 GitLab `pre-receive` hook 代码提交治理能力，包括：

1. `root` 用户跳过所有校验。
2. 删除分支、tag 放行。
3. 禁止 non fast-forward / 强推。
4. 指定分支禁止直接 push，只允许 MR / Web 合并。
5. `GL_PROTOCOL=web` 时跳过 push 类校验。
6. 校验提交人与 push 人一致。
7. 对指定分支校验 commit subject 任务号。
8. 支持用户、分支、项目白名单。
9. 支持 merge commit 豁免。
10. 支持 `HOOK_MODE=enforce|warn`。
11. 只校验本次 push 新引入仓库的 commit。

当前问题：

- 配置项写死在 Go 代码中，修改规则需要重新编译、部署 binary。
- 白名单、分支正则、任务号正则、提示策略等无法由运维人员动态调整。
- 多 Gitaly 节点部署时，配置一致性和变更生效成本较高。
- 希望统一由 Nacos 管理配置，程序从 Nacos 拉取配置后缓存到本地，hook 执行时从本地配置读取。

---

## 2. 改造目标

### 2.1 总体目标

将现有 Go CLI 的配置项从代码常量改造为 YAML 配置文件，并支持从 Nacos 配置中心自动拉取配置、缓存到本地。GitLab hook 执行时优先读取本地缓存配置，避免每次 push 都依赖 Nacos 网络调用。

### 2.2 关键目标

1. 配置项统一使用 YAML 格式。
2. 配置内容存储在 Nacos 中。
3. 程序支持从 Nacos 拉取配置。
4. 拉取成功后将配置缓存到本地文件。
5. hook 执行时每次从本地缓存读取配置。
6. Nacos 配置修改后，程序能够重新拉取并更新本地缓存。
7. 新配置更新后，下一次 hook 执行自动生效。
8. Nacos 不可用时，不影响 Git push 主链路，继续使用本地缓存配置。
9. 本地缓存不存在或配置非法时，按照 fail-safe 策略处理。
10. 支持多 Gitaly 节点独立同步配置。

---

## 3. 重要架构约束

### 3.1 pre-receive hook 是短生命周期进程

`gitlab-task-hook` 作为 GitLab `pre-receive` hook 被 Gitaly 调用时，是一次 push 触发一次进程执行。进程完成校验后会退出。

因此不能假设 hook 主进程可以长期驻留并持续监听 Nacos。

错误设计示例：

```text
每次 git push 时启动 hook -> hook 内部开启 Nacos 长轮询 -> 等待配置变化
```

该设计不合理，因为：

- hook 是同步阻塞在 git push 主链路上的。
- 长轮询会增加 push 延迟。
- Nacos 异常会影响代码提交。
- hook 进程结束后监听也随之消失。

### 3.2 推荐架构：hook 执行与配置同步分离

必须将程序分为两个执行模式：

1. `hook` 模式：执行 GitLab pre-receive 校验，只读取本地缓存配置，不直接访问 Nacos。
2. `config-sync` 模式：独立运行，负责从 Nacos 拉取配置、监听配置变化、更新本地缓存。

推荐架构：

```text
Nacos 配置中心
      ↓
config-sync 长驻进程 / systemd service / 定时任务
      ↓
本地 YAML 缓存文件
      ↓
GitLab pre-receive hook
      ↓
gitlab-task-hook hook 模式读取本地 YAML 并执行校验
```

### 3.3 不允许 push 主链路强依赖 Nacos

hook 模式禁止在每次 push 时同步访问 Nacos。

原因：

- Nacos 延迟会直接影响研发提交代码。
- Nacos 故障会导致 GitLab push 不可用。
- Git hook 应尽量快、稳定、少外部依赖。

---

## 4. 程序执行模式设计

### 4.1 二进制名称

```bash
gitlab-task-hook
```

### 4.2 支持命令

程序至少支持以下命令：

```bash
gitlab-task-hook hook

gitlab-task-hook config-sync --once

gitlab-task-hook config-sync --watch

gitlab-task-hook config-validate --file /path/to/config.yaml

gitlab-task-hook version
```

### 4.3 默认命令兼容

为了兼容 GitLab hook wrapper，如果用户直接执行：

```bash
gitlab-task-hook
```

默认等价于：

```bash
gitlab-task-hook hook
```

---

## 5. hook 模式需求

### 5.1 功能

`hook` 模式负责执行 GitLab pre-receive 校验。

执行流程：

1. 读取本地 YAML 缓存配置。
2. 校验配置结构是否合法。
3. 读取 GitLab 传入的 stdin：`<old-value> <new-value> <ref-name>`。
4. 读取环境变量：`GL_USERNAME`、`GL_PROJECT_PATH`、`GL_PROTOCOL`、`HOOK_MODE`。
5. 根据配置和现有规则执行校验。
6. 输出错误提示到 stderr。
7. 根据 `enforce` / `warn` 返回退出码。

### 5.2 hook 模式禁止事项

hook 模式不得：

- 访问 Nacos。
- 启动长轮询。
- 写入 Nacos 配置。
- 长时间阻塞等待配置更新。

### 5.3 本地配置读取路径

默认读取：

```text
/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml
```

支持通过环境变量覆盖：

```bash
GITLAB_TASK_HOOK_CONFIG=/path/to/gitlab-task-hook.yaml
```

### 5.4 本地配置读取失败策略

读取本地配置失败时，处理策略必须明确。

推荐策略：

| 场景 | 行为 |
|---|---|
| 配置文件存在且合法 | 正常执行校验 |
| 配置文件不存在 | 使用内置最小安全默认配置，并输出告警 |
| 配置文件存在但 YAML 解析失败 | 拒绝 push，exit 1 |
| 配置结构非法 | 拒绝 push，exit 1 |
| 配置版本不兼容 | 拒绝 push，exit 1 |

说明：

- 配置不存在时允许使用内置默认配置，是为了首次部署时具备可启动能力。
- 配置存在但非法时必须 fail closed，避免错误配置导致关键规则失效。

---

## 6. config-sync 模式需求

### 6.1 功能

`config-sync` 模式负责从 Nacos 拉取 YAML 配置，并写入本地缓存文件。

支持两种运行方式：

```bash
gitlab-task-hook config-sync --once
```

执行一次拉取，成功后退出。

```bash
gitlab-task-hook config-sync --watch
```

作为长驻进程运行，监听 Nacos 配置变化，变化后重新拉取并更新本地缓存。

### 6.2 once 模式

`--once` 模式流程：

1. 读取 Nacos 连接配置。
2. 从 Nacos 获取指定 dataId / group / namespace 的配置内容。
3. 校验配置 YAML 是否可解析。
4. 校验配置结构是否合法。
5. 原子写入本地缓存文件。
6. 输出同步成功日志。
7. 退出码为 0。

失败时：

- 不覆盖旧缓存。
- 输出错误日志。
- 退出码为非 0。

### 6.3 watch 模式

`--watch` 模式流程：

1. 启动后先执行一次完整拉取。
2. 拉取成功后写入本地缓存。
3. 进入监听循环。
4. 监听到 Nacos 配置变更后，重新拉取完整配置。
5. 校验成功后原子更新本地缓存。
6. 校验失败时保留旧配置。
7. 监听异常时自动重试。

### 6.4 Nacos 监听实现要求

如果当前使用 Nacos 2.x，可使用 Nacos OpenAPI 的配置监听能力或 Go SDK。

推荐优先级：

1. 优先使用成熟 Go Nacos SDK。
2. 如果项目不允许第三方依赖，可使用 Nacos OpenAPI 实现 GET 配置和监听配置。
3. 如果目标 Nacos 版本为 3.x，需要注意 HTTP long polling 的兼容性，必要时使用官方 SDK 长连接能力。

### 6.5 监听降级策略

如果监听能力不可用，必须支持定时轮询兜底。

配置项：

```yaml
nacos:
  poll_interval_seconds: 30
```

轮询逻辑：

1. 定时拉取 Nacos 配置。
2. 计算内容 MD5。
3. 如果 MD5 变化，则校验并更新本地缓存。
4. 如果 MD5 未变化，不写文件。

---

## 7. Nacos 配置连接要求

### 7.1 Nacos 元配置来源

程序需要知道连接哪个 Nacos、读取哪个配置项。这部分不能从 Nacos 本身读取，必须来自本地 bootstrap 配置或环境变量。

推荐 bootstrap 配置路径：

```text
/etc/gitlab-task-hook/bootstrap.yaml
```

也支持环境变量覆盖：

```bash
GITLAB_TASK_HOOK_BOOTSTRAP=/path/to/bootstrap.yaml
```

### 7.2 bootstrap.yaml 示例

```yaml
nacos:
  enabled: true
  server_addr: "http://127.0.0.1:8848"
  namespace_id: ""
  group: "GITLAB_HOOK"
  data_id: "gitlab-task-hook.yaml"
  username: ""
  password: ""
  access_key: ""
  secret_key: ""
  timeout_seconds: 5
  watch_enabled: true
  poll_interval_seconds: 30
  cache_file: "/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml"
  cache_meta_file: "/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta"
  log_file: "/var/log/gitlab-task-hook/config-sync.log"
```

### 7.3 字段说明

| 字段 | 是否必填 | 说明 |
|---|---:|---|
| `enabled` | 是 | 是否启用 Nacos 同步 |
| `server_addr` | 是 | Nacos 地址，如 `http://nacos.example.com:8848` |
| `namespace_id` | 否 | Nacos namespace / tenant |
| `group` | 是 | Nacos group |
| `data_id` | 是 | Nacos dataId |
| `username` | 否 | Nacos 用户名 |
| `password` | 否 | Nacos 密码 |
| `access_key` | 否 | AK 鉴权，按实际环境可选 |
| `secret_key` | 否 | SK 鉴权，按实际环境可选 |
| `timeout_seconds` | 否 | HTTP / SDK 超时时间 |
| `watch_enabled` | 否 | 是否启用监听 |
| `poll_interval_seconds` | 否 | 监听不可用时的轮询间隔 |
| `cache_file` | 是 | 本地 YAML 缓存文件路径 |
| `cache_meta_file` | 否 | 本地缓存元数据文件 |
| `log_file` | 否 | config-sync 日志文件 |

### 7.4 敏感信息处理要求

- `password`、`secret_key` 不允许打印明文日志。
- 日志中必须脱敏。
- 文件权限建议：`600`。
- 文件属主建议：`git:git` 或运行 config-sync 的专用用户。

---

## 8. 本地缓存设计

### 8.1 缓存文件

主缓存文件：

```text
/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml
```

元数据文件：

```text
/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta
```

### 8.2 元数据文件内容

建议 JSON 格式：

```json
{
  "data_id": "gitlab-task-hook.yaml",
  "group": "GITLAB_HOOK",
  "namespace_id": "",
  "md5": "e10adc3949ba59abbe56e057f20f883e",
  "version": "2026-06-28T10:30:00+08:00",
  "last_sync_time": "2026-06-28T10:31:02+08:00",
  "source": "nacos",
  "sync_status": "success"
}
```

### 8.3 原子写入要求

config-sync 更新本地缓存时必须原子写入，避免 hook 进程读到半截文件。

推荐流程：

1. 写入临时文件：`gitlab-task-hook.yaml.tmp.<pid>`。
2. fsync 临时文件。
3. 校验临时文件可读可解析。
4. rename 覆盖正式文件。
5. 写入 meta 临时文件。
6. rename 覆盖 meta 文件。

### 8.4 文件锁要求

为避免多个 config-sync 进程并发写配置，必须使用文件锁。

锁文件建议：

```text
/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.lock
```

行为：

- 获取锁成功：执行同步。
- 获取锁失败：退出或等待，具体由参数控制。

### 8.5 权限要求

目录权限：

```bash
chown -R git:git /var/opt/gitlab/gitaly/custom_hooks/config
chmod 750 /var/opt/gitlab/gitaly/custom_hooks/config
```

配置文件权限：

```bash
chmod 640 /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml
chmod 640 /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta
```

---

## 9. YAML 配置结构需求

### 9.1 顶层结构

Nacos 中存储的配置内容必须是完整 YAML。

示例：

```yaml
version: "1.0"
enabled: true
mode:
  default: "enforce"

rules:
  root_bypass:
    enabled: true
    usernames:
      - "root"

  non_fast_forward:
    enabled: true

  deny_direct_push:
    enabled: true
    branch_regex: "(?i)^refs/heads/(master|sit_.*|uat_.*)$"
    allow_protocols:
      - "web"
    deny_protocols:
      - "http"
      - "ssh"
      - ""

  web_bypass_push_checks:
    enabled: true
    protocols:
      - "web"

  committer_match_push_user:
    enabled: true
    compare_strategy: "email_prefix"
    skip_merge_commit: true
    case_insensitive: true

  task_id:
    enabled: true
    branch_regex: "(?i)^refs/heads/(feature|dev)(/|_|-|$)"
    subject_regex: "\\[#(TSK|DEF)-[^\\[\\]]+\\]"
    check_subject_only: true
    exempt_merge_commit: true
    exempt_message_regex: ""

whitelist:
  users: []
  branch_regex: "(?i)^refs/heads/(init/|migrate/|tmp/)"
  projects:
    - "demo-service"
    - "legacy-repo"

messages:
  language: "zh-CN"
  show_fix_guide: true

logging:
  level: "info"
  stderr: true
```

---

## 10. YAML 字段详细说明

### 10.1 通用字段

| 字段 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `version` | string | `1.0` | 配置版本 |
| `enabled` | bool | true | 全局是否启用校验 |
| `mode.default` | string | enforce | 默认模式，支持 enforce / warn |

### 10.2 root_bypass

```yaml
rules:
  root_bypass:
    enabled: true
    usernames:
      - "root"
```

说明：

- 命中后跳过所有校验。
- 用户名大小写不敏感。

### 10.3 non_fast_forward

```yaml
rules:
  non_fast_forward:
    enabled: true
```

说明：

- 是否启用强推校验。
- 建议生产始终开启。

### 10.4 deny_direct_push

```yaml
rules:
  deny_direct_push:
    enabled: true
    branch_regex: "(?i)^refs/heads/(master|sit_.*|uat_.*)$"
    allow_protocols:
      - "web"
    deny_protocols:
      - "http"
      - "ssh"
      - ""
```

说明：

- 命中 `branch_regex` 且协议在 `deny_protocols` 中时拒绝。
- `GL_PROTOCOL` 为空时按直接 push 处理。
- `web` 允许，用于 MR / Web 合并。

### 10.5 web_bypass_push_checks

```yaml
rules:
  web_bypass_push_checks:
    enabled: true
    protocols:
      - "web"
```

说明：

- 命中后跳过提交人与 push 人一致性校验、任务号校验等 push 类规则。
- 强推校验和指定分支禁止直接 push 规则应在该规则前执行。

### 10.6 committer_match_push_user

```yaml
rules:
  committer_match_push_user:
    enabled: true
    compare_strategy: "email_prefix"
    skip_merge_commit: true
    case_insensitive: true
```

说明：

- `email_prefix`：取 commit committer email 的 `@` 前缀，与 `GL_USERNAME` 比较。
- `skip_merge_commit=true`：merge commit 不做该校验。

### 10.7 task_id

```yaml
rules:
  task_id:
    enabled: true
    branch_regex: "(?i)^refs/heads/(feature|dev)(/|_|-|$)"
    subject_regex: "\\[#(TSK|DEF)-[^\\[\\]]+\\]"
    check_subject_only: true
    exempt_merge_commit: true
    exempt_message_regex: ""
```

说明：

- `branch_regex`：哪些分支需要任务号校验。
- `subject_regex`：任务号格式。
- `check_subject_only=true`：仅校验 commit subject，不校验 body。
- `exempt_merge_commit=true`：merge commit 跳过任务号校验。
- `exempt_message_regex`：特殊 message 白名单。

### 10.8 whitelist

```yaml
whitelist:
  users: []
  branch_regex: "(?i)^refs/heads/(init/|migrate/|tmp/)"
  projects:
    - "demo-service"
    - "legacy-repo"
```

说明：

- 白名单只跳过任务号校验。
- 不跳过强推校验。
- 不跳过指定分支禁止直接 push 校验。
- 不跳过提交人与 push 人一致性校验。
- 项目白名单按 `GL_PROJECT_PATH` 最后一段 repo name 匹配。

---

## 11. 配置校验要求

程序启动或同步配置时必须校验 YAML。

### 11.1 必须校验项

1. YAML 能成功解析。
2. `version` 是支持的版本。
3. 正则字段能成功编译。
4. `mode.default` 只能是 `enforce` 或 `warn`。
5. `deny_direct_push.allow_protocols`、`deny_direct_push.deny_protocols` 不允许同时为空。
6. `committer_match_push_user.compare_strategy` 当前只支持 `email_prefix`。
7. `whitelist.projects` 必须为字符串数组。
8. `whitelist.users` 必须为字符串数组。

### 11.2 配置非法处理

| 发生位置 | 行为 |
|---|---|
| config-sync 拉取到非法配置 | 不覆盖本地旧配置，记录错误 |
| hook 模式读取到非法配置 | 拒绝 push，输出配置错误 |
| 本地配置不存在 | 使用内置最小默认配置，输出告警 |

---

## 12. 规则执行顺序

使用 YAML 配置后，规则执行顺序仍必须固定，不允许由配置改变。

固定顺序：

1. 读取配置。
2. 如果全局 `enabled=false`，直接放行。
3. root 用户跳过所有校验。
4. 删除 ref 放行。
5. tag ref 放行。
6. non fast-forward 校验。
7. 指定分支禁止直接 push 校验。
8. `GL_PROTOCOL=web` 时跳过后续 push 类校验。
9. 计算本次 push 新引入 commit。
10. 提交人与 push 人一致性校验。
11. 判断是否命中任务号校验分支。
12. 用户 / 分支 / 项目白名单跳过任务号校验。
13. merge commit 任务号豁免。
14. message 白名单豁免。
15. commit subject 任务号校验。

---

## 13. Nacos OpenAPI 兼容性说明

### 13.1 获取配置

Nacos OpenAPI 支持通过配置管理接口获取配置内容。程序应根据 `server_addr`、`data_id`、`group`、`namespace_id` 读取 YAML 配置。

### 13.2 监听配置

Nacos OpenAPI 支持监听配置变化，监听到变化后可重新获取配置并刷新本地缓存。

### 13.3 版本注意

如果目标环境是 Nacos 2.x，通常可使用 Nacos 1.x 兼容 OpenAPI 或 Go SDK。若目标环境升级为 Nacos 3.x，应确认 HTTP long polling 是否仍可用；如果不可用，应切换为官方 SDK 长连接监听或定时轮询方案。

---

## 14. 错误处理与可用性要求

### 14.1 Nacos 不可用

config-sync 访问 Nacos 失败时：

- 保留本地旧缓存。
- 记录错误日志。
- 按重试策略继续尝试。
- 不影响 hook 模式。

### 14.2 本地缓存过旧

建议在 meta 中记录 `last_sync_time`。

可选策略：

```yaml
cache:
  max_stale_minutes: 1440
  stale_policy: "warn"
```

当前需求建议：

- 缓存过旧只告警，不阻断 push。
- 除非配置明确要求 fail closed。

### 14.3 配置更新失败

如果新配置拉取成功但校验失败：

- 不覆盖旧配置。
- 记录错误。
- 下一轮继续尝试。

---

## 15. 日志要求

### 15.1 hook 模式日志

hook 模式应尽量少输出。

只在以下场景输出到 stderr：

- 规则违规提示。
- 配置文件不存在告警。
- 配置非法错误。

### 15.2 config-sync 日志

config-sync 应输出结构化日志，至少包含：

- 启动日志。
- Nacos 地址、dataId、group、namespace，敏感信息脱敏。
- 拉取成功。
- 配置 MD5。
- 本地缓存写入成功。
- 配置未变化。
- 监听异常。
- 重试次数。
- 配置校验失败原因。

### 15.3 敏感信息脱敏

禁止打印：

- password 明文。
- secret_key 明文。
- access token 明文。

---

## 16. systemd 部署建议

### 16.1 config-sync service

```ini
[Unit]
Description=GitLab Task Hook Nacos Config Sync
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=git
Group=git
ExecStart=/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook config-sync --watch --bootstrap /etc/gitlab-task-hook/bootstrap.yaml
Restart=always
RestartSec=10
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

### 16.2 GitLab hook wrapper

```sh
#!/bin/sh
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook hook
```

---

## 17. 测试要求

### 17.1 单元测试

必须覆盖：

1. YAML 解析成功。
2. YAML 正则编译失败。
3. 配置版本不支持。
4. `mode.default` 非法。
5. 项目白名单解析。
6. 分支白名单解析。
7. 受保护分支禁止 push 规则。
8. `GL_PROTOCOL=web` 放行。
9. 提交人与 push 人一致性校验。
10. 任务号正则校验。
11. 本地配置不存在时使用默认配置。
12. 本地配置非法时拒绝。
13. Nacos 拉取成功后原子写入缓存。
14. 新配置非法时不覆盖旧缓存。
15. meta 文件写入。

### 17.2 集成测试

建议用临时 Git 仓库模拟：

1. 普通 push 到 dev，任务号合规，通过。
2. 普通 push 到 dev，任务号不合规，拒绝。
3. 普通 push 到 master，拒绝，提示使用 MR。
4. `GL_PROTOCOL=web` 写入 master，通过。
5. committer email 前缀和 `GL_USERNAME` 不一致，拒绝。
6. 项目白名单命中，只跳过任务号，不跳过提交人一致性。
7. 修改 YAML 配置后重新运行 hook，规则变化生效。
8. config-sync 拉取新配置后，本地缓存更新。

### 17.3 Nacos 测试

如果 CI 中无法启动真实 Nacos，则应通过接口抽象 mock Nacos client。

NacosClient 接口建议：

```go
type NacosClient interface {
    GetConfig(ctx context.Context, dataID, group, namespace string) ([]byte, error)
    ListenConfig(ctx context.Context, dataID, group, namespace string, onChange func([]byte)) error
}
```

---

## 18. 代码架构建议

建议目录：

```text
gitlab-task-hook/
├── cmd/
│   ├── hook.go
│   ├── config_sync.go
│   ├── config_validate.go
│   └── version.go
├── internal/
│   ├── config/
│   │   ├── model.go
│   │   ├── loader.go
│   │   ├── validator.go
│   │   ├── defaults.go
│   │   └── cache.go
│   ├── nacos/
│   │   ├── client.go
│   │   ├── openapi_client.go
│   │   └── syncer.go
│   ├── hook/
│   │   ├── input.go
│   │   ├── env.go
│   │   ├── engine.go
│   │   ├── rules.go
│   │   └── messages.go
│   ├── git/
│   │   └── git.go
│   └── log/
│       └── log.go
├── docs/
│   └── gitlab-task-hook-nacos-yaml-config-requirements.md
├── README.md
└── go.mod
```

---

## 19. 交付物要求

Claude Code / Codex 必须交付：

1. 更新后的架构设计说明。
2. YAML 配置模型设计。
3. Nacos 同步方案设计。
4. 本地缓存与原子写入实现。
5. Go 代码实现。
6. 单元测试。
7. 集成测试说明。
8. 示例 `bootstrap.yaml`。
9. 示例 `gitlab-task-hook.yaml`。
10. systemd service 示例。
11. GitLab hook wrapper 示例。
12. README 更新。
13. 回滚方案。

---

## 20. Claude Code 执行要求

请 Claude Code 按以下阶段执行：

### 阶段 1：只做设计，不编码

输出：

- 配置中心改造架构设计。
- YAML schema 设计。
- Nacos 同步流程设计。
- hook 主链路读取本地缓存设计。
- 错误处理设计。
- 测试方案。

### 阶段 2：确认设计后编码

实现：

- config model。
- local cache loader。
- config validator。
- config-sync once/watch。
- hook 模式读取 YAML。
- 规则引擎使用 YAML 配置。

### 阶段 3：测试

完成：

- 单元测试。
- mock Nacos 测试。
- 本地 Git 仓库集成测试说明。

### 阶段 4：文档

完成：

- README。
- 部署说明。
- systemd 示例。
- 回滚说明。

---

## 21. 给 Claude Code 的入口指令

```text
请读取 docs/gitlab-task-hook-nacos-yaml-config-requirements.md。

当前已有 gitlab-task-hook Go CLI，现需新增 YAML 配置文件和 Nacos 配置中心同步能力。

请先不要直接编码，先输出架构设计和方案设计，必须包括：
1. hook 模式与 config-sync 模式拆分设计；
2. YAML 配置结构设计；
3. Nacos 拉取和监听方案；
4. 本地缓存文件和原子写入方案；
5. Nacos 不可用时的降级方案；
6. 配置校验和错误处理；
7. 单元测试和集成测试方案；
8. 对现有规则引擎的改造点。

设计确认后，再开始修改代码。
```

---

## 22. 验收标准

1. 程序可通过本地 YAML 配置驱动所有现有规则。
2. 修改 YAML 后，重新执行 hook 能读取新规则。
3. config-sync 能从 Nacos 拉取配置并缓存到本地。
4. Nacos 配置变更后，本地缓存能自动更新。
5. Nacos 不可用时，hook 仍使用本地缓存配置执行。
6. 非法 Nacos 配置不会覆盖本地旧配置。
7. 本地非法配置会导致 hook fail closed。
8. 所有规则单元测试通过。
9. README 中有部署和回滚说明。
