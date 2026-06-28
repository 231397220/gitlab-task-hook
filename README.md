# gitlab-task-hook

GitLab `pre-receive` server hook —— Go CLI 实现。

在 `git push` 时执行服务端代码提交规范门禁，支持通过 **Nacos 配置中心**动态下发规则，无需重新编译或重启 hook 进程。

---

## 功能

| 规则 | 说明 |
|---|---|
| root 用户跳过 | 用户名命中白名单时跳过所有校验 |
| 禁止强推 | non fast-forward / rewrite history |
| 受保护分支 | 指定分支禁止直接 push，只允许 MR/Web 合并 |
| 提交人一致性 | committer email 前缀必须与 GL_USERNAME 一致 |
| 任务号校验 | 指定分支的 commit subject 必须含 `[#TSK-xxx]` 或 `[#DEF-xxx]` |
| 白名单 | 用户 / 分支 / 项目 / message 白名单豁免任务号校验 |
| merge commit 豁免 | merge commit 可跳过提交人和任务号校验 |
| GL_PROTOCOL=web 跳过 | MR 合并、Web IDE 提交跳过 push 类校验 |
| enforce / warn 模式 | enforce 拒绝 push；warn 仅提示不阻断 |

---

## 消息提示优化

push 被拒绝时，hook 向 stderr 输出详细的违规信息和修复指引。为帮助技术能力弱的开发者快速自助解决问题，所有四类提示已优化完善：

### 四个场景的拒绝提示

| 场景 | 触发条件 | 包含内容 |
|------|---------|--------|
| **禁止强制推送** | non fast-forward 提交 | ✓ 当前状态（项目/用户/分支）<br>✓ 修复步骤（git pull --rebase）<br>✓ 冲突处理说明 |
| **禁止直接 push 到受保护分支** | 直接 push 到 master/sit/uat 等 | ✓ 当前状态 + 协议信息<br>✓ 修复步骤（创建功能分支→发起 MR）<br>✓ GitLab 操作指引 |
| **提交人与 push 用户不一致** | committer email 不匹配 | ✓ 当前状态 + 具体 commit ID<br>✓ 修复步骤（配置 git user + amend/rebase）<br>✓ 单提交与多提交修改方法 |
| **缺少任务 ID** | commit subject 无 `[#TSK-xxx]` | ✓ 当前状态 + 具体 commit ID 和 subject<br>✓ 修复步骤（amend/rebase）<br>✓ 多种场景修改指引 |

所有提示支持**运维动态更新**，无需发版部署（详见[消息模板配置](#消息模板配置)）。

---

## 架构

```
Nacos 配置中心
      ↓
config-sync 长驻进程（systemd service）
      ↓  原子写入本地 YAML 缓存
/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml
      ↓  hook 模式只读，不访问 Nacos
GitLab pre-receive hook
      ↓
gitlab-task-hook hook
```

- **hook 模式**：短生命周期，只读本地缓存，零网络依赖
- **config-sync 模式**：长驻进程，负责从 Nacos 拉取配置并写入本地缓存

---

## 子命令

```bash
# 执行 pre-receive 校验（等价于无参数启动，向后兼容）
gitlab-task-hook hook

# 拉取一次 Nacos 配置后退出（适合 cron / 首次初始化）
gitlab-task-hook config-sync --once [--bootstrap /path/to/bootstrap.yaml]

# 长驻进程，监听 Nacos 配置变化并实时更新本地缓存
gitlab-task-hook config-sync --watch [--bootstrap /path/to/bootstrap.yaml]

# 校验 YAML 配置文件合法性
gitlab-task-hook config-validate --file /path/to/config.yaml

# 打印版本
gitlab-task-hook version
```

---

## 配置文件

### 1. bootstrap.yaml — Nacos 连接配置

告知程序去哪里连接 Nacos、拉取哪条配置。由 `config-sync` 启动时读取，`hook` 模式不读此文件。

**默认路径**：`/etc/gitlab-task-hook/bootstrap.yaml`

**覆盖方式**：
- `--bootstrap /path/to/bootstrap.yaml` 命令行参数
- `GITLAB_TASK_HOOK_BOOTSTRAP` 环境变量

模板文件：[`scripts/bootstrap.yaml`](scripts/bootstrap.yaml)

```yaml
nacos:
  enabled: true
  server_addr: "http://127.0.0.1:8848"
  namespace_id: ""
  group: "GITLAB_HOOK"
  data_id: "gitlab-task-hook.yaml"
  username: ""
  password: ""
  timeout_seconds: 5
  watch_enabled: true
  poll_interval_seconds: 30
  cache_file: "/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml"
  cache_meta_file: "/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta"
  log_file: "/var/log/gitlab-task-hook/config-sync.log"
```

> `password` 和 `secret_key` 不会打印到任何日志，文件权限建议 `600`。

### 2. gitlab-task-hook.yaml — hook 规则配置

存入 Nacos（group=`GITLAB_HOOK`，dataId=`gitlab-task-hook.yaml`），由 `config-sync` 拉取后缓存到本地，`hook` 模式直接读本地缓存。

**默认缓存路径**：`/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml`

**覆盖方式**：`GITLAB_TASK_HOOK_CONFIG` 环境变量

模板文件：[`scripts/gitlab-task-hook.yaml`](scripts/gitlab-task-hook.yaml)

**修改配置生效流程**：在 Nacos 控制台发布新配置 → `config-sync` 监听到变化 → 原子写入本地缓存 → 下一次 `git push` 触发 hook 时自动读取新配置，无需重启任何进程。

### 3. 消息模板配置（可选）

hook 拒绝 push 时输出的提示信息可在 YAML 配置中自定义，无需修改代码或重新部署。

在 `gitlab-task-hook.yaml` 中新增 `messages.templates` 部分（空字符串表示使用内置默认模板）：

```yaml
messages:
  language: "zh-CN"
  show_fix_guide: true
  templates:
    # 禁止强制推送 — 空字符串 = 使用内置默认模板
    non_fast_forward: |
      禁止强制推送（non fast-forward / rewrite history）
      
      project: {{.ProjectPath}}
      repo:    {{.RepoName}}
      user:    {{.Username}}
      branch:  {{.BranchName}}
      
      【如何处理】
      您的推送被拒绝，因为本地历史与远端不一致。
      1) 拉取远端最新并本地追加：
         git pull --rebase origin {{.BranchName}}
      ...
    
    # 禁止直接 push 到受保护分支
    direct_push_denied: |
      当前分支禁止直接 push，请通过 Merge Request 合并代码。
      ...
    
    # 提交人与 push 用户不一致
    committer_mismatch: |
      提交人信息不符合规范：commit 提交人与 push 用户不一致
      ...
    
    # 缺少任务 ID
    task_id_missing: |
      提交信息不符合规范：缺少任务ID（[#TSK-...] 或 [#DEF-...]）
      ...
```

**模板语法**：Go `text/template`，支持以下变量：

| 变量 | 说明 | 所有场景可用 |
|------|------|-----------|
| `{{.ProjectPath}}` | 项目完整路径，如 `group/my-repo` | ✓ |
| `{{.RepoName}}` | 项目名称（路径最后一段） | ✓ |
| `{{.Username}}` | GitLab 用户名 | ✓ |
| `{{.BranchName}}` | 分支短名，如 `master` 或 `feature/auth` | ✓ |
| `{{.Protocol}}` | 操作协议：`http`/`ssh`/`web` | 场景1 |
| `{{.CommitID}}` | 问题 commit 的 SHA | 场景2/3 |
| `{{.CommitSubject}}` | commit subject（第一行）| 场景2 |
| `{{.CommitterEmail}}` | committer email | 场景3 |

**修改生效流程**：
1. 在 Nacos 控制台修改 `gitlab-task-hook.yaml` 的 `messages.templates` 部分
2. `config-sync` 检测变化 → 验证 YAML 合法性 → 原子写入本地缓存
3. 下一次 `git push` 触发 hook 时，自动使用新模板渲染错误提示

**降级策略**：
- 模板为空 → 使用内置默认模板
- 模板语法非法 → `config-sync` 拒绝写入，保留旧配置
- 模板渲染错误 → hook 自动回退内置默认模板，不影响 push 校验

---

## 本地缓存与降级策略

| 缓存状态 | hook 行为 |
|---|---|
| 文件存在且合法 | 正常执行全部规则 |
| 文件不存在 | 使用内置最小默认配置（仅启用强推校验），stderr 输出告警 |
| YAML 解析失败 | 拒绝所有 push（exit 1），输出错误 |
| 配置结构非法 | 拒绝所有 push（exit 1），输出错误 |

**Nacos 不可用时**：`config-sync` 保留旧缓存，记录错误日志，按退避策略自动重试，不影响 hook 正常运行。

**新配置非法时**：`config-sync` 不覆盖旧缓存，本地缓存始终保持上一份合法配置。

---

## 构建

```bash
# 当前平台
go build -o gitlab-task-hook ./cmd/gitlab-task-hook

# Linux x86_64（生产常用）
GOOS=linux GOARCH=amd64 go build \
  -ldflags "-X main.version=v1.1.0" \
  -o gitlab-task-hook \
  ./cmd/gitlab-task-hook

# Linux ARM64
GOOS=linux GOARCH=arm64 go build \
  -ldflags "-X main.version=v1.1.0" \
  -o gitlab-task-hook \
  ./cmd/gitlab-task-hook

# 查看版本
./gitlab-task-hook version
```

---

## 测试

```bash
# 单元测试
go test ./...

# 集成测试（需先构建二进制）
go build -o gitlab-task-hook ./cmd/gitlab-task-hook
./scripts/integration_test.sh ./gitlab-task-hook
```

---

## 部署

### 目录准备

```bash
HOOKS_BASE=/var/opt/gitlab/gitaly/custom_hooks

mkdir -p $HOOKS_BASE/bin
mkdir -p $HOOKS_BASE/config
chown -R git:git $HOOKS_BASE/config
chmod 750 $HOOKS_BASE/config

mkdir -p /var/log/gitlab-task-hook
chown git:git /var/log/gitlab-task-hook

mkdir -p /etc/gitlab-task-hook
```

### 上传二进制

```bash
HOOKS_BASE=/var/opt/gitlab/gitaly/custom_hooks
VERSION=v1.1.0

cp gitlab-task-hook $HOOKS_BASE/bin/gitlab-task-hook-${VERSION}
chown git:git $HOOKS_BASE/bin/gitlab-task-hook-${VERSION}
chmod 755     $HOOKS_BASE/bin/gitlab-task-hook-${VERSION}

ln -sf gitlab-task-hook-${VERSION} $HOOKS_BASE/bin/gitlab-task-hook
```

或使用部署脚本：

```bash
./scripts/deploy.sh ./gitlab-task-hook v1.1.0
```

### 写入 bootstrap 配置

```bash
cp scripts/bootstrap.yaml /etc/gitlab-task-hook/bootstrap.yaml
vim /etc/gitlab-task-hook/bootstrap.yaml   # 修改 server_addr 等字段

chmod 600 /etc/gitlab-task-hook/bootstrap.yaml
chown git:git /etc/gitlab-task-hook/bootstrap.yaml
```

### 发布规则配置到 Nacos

在 Nacos 控制台（或通过 OpenAPI）发布配置：

- **Namespace**：bootstrap.yaml 中的 `namespace_id`
- **Group**：`GITLAB_HOOK`
- **Data ID**：`gitlab-task-hook.yaml`
- **内容**：参考 [`scripts/gitlab-task-hook.yaml`](scripts/gitlab-task-hook.yaml)

> **提示**：配置中的 `messages.templates` 部分用于自定义 push 拒绝时的提示文案，可选项。详见[消息模板配置](#消息模板配置)。

### 首次拉取配置（验证连通性）

```bash
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 验证本地缓存已写入
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml

# 验证配置合法
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-validate \
  --file /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml
```

### 部署 config-sync systemd 服务

```ini
# /etc/systemd/system/gitlab-task-hook-config-sync.service
[Unit]
Description=GitLab Task Hook Nacos Config Sync
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=git
Group=git
ExecStart=/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --watch --bootstrap /etc/gitlab-task-hook/bootstrap.yaml
Restart=always
RestartSec=10
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable gitlab-task-hook-config-sync
systemctl start  gitlab-task-hook-config-sync
systemctl status gitlab-task-hook-config-sync
```

### 部署 GitLab hook wrapper

```bash
HOOKS_BASE=/var/opt/gitlab/gitaly/custom_hooks
mkdir -p $HOOKS_BASE/pre-receive.d

cat > $HOOKS_BASE/pre-receive.d/01-task-hook <<'EOF'
#!/bin/sh
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook hook
EOF

chown git:git $HOOKS_BASE/pre-receive.d/01-task-hook
chmod 755     $HOOKS_BASE/pre-receive.d/01-task-hook
```

### 目录结构总览

```
/etc/gitlab-task-hook/
└── bootstrap.yaml                         ← Nacos 连接配置（600, git:git）

/var/opt/gitlab/gitaly/custom_hooks/
├── bin/
│   ├── gitlab-task-hook                   → gitlab-task-hook-v1.1.0（软链接）
│   ├── gitlab-task-hook-v1.0.0
│   └── gitlab-task-hook-v1.1.0
├── config/
│   ├── gitlab-task-hook.yaml              ← 本地规则缓存（config-sync 写入）
│   ├── gitlab-task-hook.yaml.meta         ← 同步元数据（MD5、时间戳）
│   └── gitlab-task-hook.lock              ← 写锁（自动管理）
└── pre-receive.d/
    └── 01-task-hook                       ← GitLab hook wrapper

/var/log/gitlab-task-hook/
└── config-sync.log                        ← config-sync 结构化日志
```

---

## 回滚

### 方式 A：禁用 wrapper（最快，秒级生效）

```bash
mv /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-hook \
   /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-hook.disabled
```

恢复：

```bash
mv /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-hook.disabled \
   /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-hook
```

### 方式 B：切回上一版本二进制

```bash
ln -sf gitlab-task-hook-v1.0.0 \
  /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
```

### 方式 C：临时切换 warn 模式（不阻断 push，仅提示）

```bash
# 在 wrapper 脚本中设置环境变量（无需重新编译或重启）
cat > /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-hook <<'EOF'
#!/bin/sh
export HOOK_MODE=warn
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook hook
EOF
```

### 方式 D：回滚规则配置（Nacos 控制台）

在 Nacos 控制台对 `gitlab-task-hook.yaml` 执行「历史版本回滚」，`config-sync` 监听到变化后自动更新本地缓存，下一次 push 即生效。

---

## 运维操作

### 手动更新本地缓存

```bash
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml
```

### 查看同步状态

```bash
# 查看最后同步时间和 MD5
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta

# 查看 config-sync 日志
tail -f /var/log/gitlab-task-hook/config-sync.log

# 查看 systemd 服务状态
systemctl status gitlab-task-hook-config-sync
```

### 验证 hook 配置

```bash
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-validate \
  --file /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml
```

---

## 环境变量参考

### GitLab 自动注入（无需手动设置）

| 变量 | 说明 |
|---|---|
| `GL_USERNAME` | 当前操作的 GitLab 用户名 |
| `GL_PROJECT_PATH` | 当前项目路径，如 `group/demo-service` |
| `GL_PROTOCOL` | 操作来源：`http`、`ssh`、`web` |

### 程序支持的环境变量

| 变量 | 默认 | 说明 |
|---|---|---|
| `HOOK_MODE` | 读 YAML `mode.default` | `enforce` 或 `warn`，覆盖 YAML 配置，适合临时切换 |
| `GITLAB_TASK_HOOK_CONFIG` | `/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml` | 本地规则缓存路径 |
| `GITLAB_TASK_HOOK_BOOTSTRAP` | `/etc/gitlab-task-hook/bootstrap.yaml` | Nacos 连接配置路径 |

---

## 规则执行顺序

顺序由代码固定，YAML 配置不能改变执行顺序，只能通过 `enabled` 字段开关规则。

| 步骤 | 规则 | 配置项 |
|---:|---|---|
| 1 | 全局 `enabled=false` → 放行 | `enabled` |
| 2 | root 用户 → 跳过所有校验 | `rules.root_bypass` |
| 3 | 删除 ref → 放行 | 固定 |
| 4 | tag ref → 放行 | 固定 |
| 5 | 禁止强推 | `rules.non_fast_forward` |
| 6 | 受保护分支禁止直接 push | `rules.deny_direct_push` |
| 7 | GL_PROTOCOL 在 web 协议列表 → 跳过后续 | `rules.web_bypass_push_checks` |
| 8 | 计算本次 push 新引入 commit | 固定 |
| 9 | 提交人与 push 用户一致性校验 | `rules.committer_match_push_user` |
| 10 | 分支不在任务号校验范围 → 跳过 | `rules.task_id.branch_regex` |
| 11 | 用户白名单 → 跳过任务号 | `whitelist.users` |
| 12 | 分支白名单 → 跳过任务号 | `whitelist.branch_regex` |
| 13 | 项目白名单 → 跳过任务号 | `whitelist.projects` |
| 14 | merge commit 豁免任务号 | `rules.task_id.exempt_merge_commit` |
| 15 | message 白名单 → 跳过任务号 | `rules.task_id.exempt_message_regex` |
| 16 | 任务号正则校验 | `rules.task_id.subject_regex` |

---

## 任务号格式

commit subject 必须包含（二选一）：

```
[#TSK-xxx]   示例：[#TSK-1001]
[#DEF-xxx]   示例：[#DEF-A20260001]
```

- `xxx` 至少 1 个字符，不含 `[` 或 `]`
- 默认只校验 subject（第一行），不校验 body
- 可通过 `task_id.subject_regex` 自定义格式
- 可通过 `task_id.exempt_message_regex` 配置豁免正则

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
