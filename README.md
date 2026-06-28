# gitlab-task-hook

GitLab `pre-receive` server hook — Go CLI 实现。

在 `git push` 时执行服务端代码提交规范门禁，包括：

- 禁止 non fast-forward 强推
- 提交人与 push 用户一致性校验
- commit subject 任务号校验（`[#TSK-xxx]` / `[#DEF-xxx]`）
- 支持 root 跳过、web/MR 跳过、merge commit 豁免
- 支持用户/分支/项目/message 白名单
- 支持 `HOOK_MODE=warn`（仅提示，不阻断）

---

## 构建

### 快速构建（当前平台）

```bash
go build -o gitlab-task-hook ./cmd/gitlab-task-hook
```

### Linux x86_64

```bash
GOOS=linux GOARCH=amd64 go build \
  -ldflags "-X main.version=v1.0.0" \
  -o gitlab-task-hook \
  ./cmd/gitlab-task-hook
```

### Linux ARM64

```bash
GOOS=linux GOARCH=arm64 go build \
  -ldflags "-X main.version=v1.0.0" \
  -o gitlab-task-hook \
  ./cmd/gitlab-task-hook
```

### 查看版本

```bash
./gitlab-task-hook --version
# gitlab-task-hook version v1.0.0
```

---

## 单元测试

```bash
go test ./...
```

---

## 集成测试

先构建二进制，再运行集成测试脚本：

```bash
go build -o gitlab-task-hook ./cmd/gitlab-task-hook
./scripts/integration_test.sh ./gitlab-task-hook
```

脚本会在临时目录创建 bare git repo，模拟 12 个 pre-receive 场景：

| 场景 | 预期 |
|---|---|
| 合规 push | exit 0 |
| 缺任务号（enforce） | exit 1 |
| 缺任务号（warn） | exit 0 |
| root 用户 | exit 0 |
| tag ref | exit 0 |
| 删除分支 | exit 0 |
| 强推 | exit 1 |
| GL_PROTOCOL=web | exit 0 |
| 项目白名单 | exit 0 |
| committer 不一致 | exit 1 |
| merge commit 豁免 | exit 0 |
| 多 commit 只输出一次违规 | 只有 1 行提示 |

---

## 部署

### 方式一：使用部署脚本

```bash
./scripts/deploy.sh ./gitlab-task-hook v1.0.0
```

### 方式二：手动部署

```bash
# 1. 上传二进制（带版本后缀）
HOOKS_BASE=/var/opt/gitlab/gitaly/custom_hooks
mkdir -p $HOOKS_BASE/bin
cp gitlab-task-hook $HOOKS_BASE/bin/gitlab-task-hook-v1.0.0
chown git:git $HOOKS_BASE/bin/gitlab-task-hook-v1.0.0
chmod 755 $HOOKS_BASE/bin/gitlab-task-hook-v1.0.0

# 2. 创建软链接
ln -sf gitlab-task-hook-v1.0.0 $HOOKS_BASE/bin/gitlab-task-hook

# 3. 创建 wrapper
mkdir -p $HOOKS_BASE/pre-receive.d
cat > $HOOKS_BASE/pre-receive.d/01-task-id-check <<'EOF'
#!/bin/sh
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
EOF
chown git:git $HOOKS_BASE/pre-receive.d/01-task-id-check
chmod 755 $HOOKS_BASE/pre-receive.d/01-task-id-check
```

### 目录结构

```
/var/opt/gitlab/gitaly/custom_hooks/
├── bin/
│   ├── gitlab-task-hook          -> gitlab-task-hook-v1.1.0
│   ├── gitlab-task-hook-v1.0.0
│   └── gitlab-task-hook-v1.1.0
└── pre-receive.d/
    └── 01-task-id-check          (thin shell wrapper)
```

---

## 回滚

### 方式A：禁用 wrapper（最快，无感知）

```bash
mv /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check \
   /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check.disabled
```

恢复：

```bash
mv /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check.disabled \
   /var/opt/gitlab/gitaly/custom_hooks/pre-receive.d/01-task-id-check
```

### 方式B：切回上一版本

```bash
ln -sf gitlab-task-hook-v1.0.0 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
```

---

## 环境变量配置

### GitLab 自动注入（无需手动设置）

| 变量 | 说明 |
|---|---|
| `GL_USERNAME` | 当前操作的 GitLab 用户名 |
| `GL_PROJECT_PATH` | 当前项目路径，如 `group/demo-service` |
| `GL_PROTOCOL` | 操作来源：`http`、`ssh`、`web` |

### 自定义配置

| 变量 | 默认值 | 说明 |
|---|---|---|
| `HOOK_MODE` | `enforce` | `enforce`：违规阻断；`warn`：仅提示 |
| `WHITELIST_USERS` | 空 | 跳过任务号校验的用户，空格分隔，大小写不敏感 |
| `WHITELIST_PROJECT_NAMES` | 空 | 跳过任务号校验的项目名，逗号分隔，大小写不敏感 |
| `EXEMPT_MESSAGE_REGEX` | 空 | commit subject 匹配此正则则跳过任务号校验 |
| `EXEMPT_MERGE_COMMIT` | `true` | merge commit 是否豁免提交人和任务号校验 |

示例（在 Gitaly 服务器 `/etc/environment` 或 wrapper 脚本中设置）：

```sh
#!/bin/sh
export WHITELIST_PROJECT_NAMES="legacy-service,migrate-tool"
export HOOK_MODE=enforce
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
```

---

## 任务号格式

commit subject 必须包含（二选一）：

```
[#TSK-xxx]   e.g. [#TSK-1001]
[#DEF-xxx]   e.g. [#DEF-A20260001]
```

- `xxx` 至少 1 个字符，不含 `[` 或 `]`
- 仅校验 subject（第一行），不校验 body

---

## 规则优先级（摘要）

| 优先级 | 规则 |
|---:|---|
| 1 | root 用户跳过所有校验 |
| 2 | 删除分支放行 |
| 3 | tag 放行 |
| 4 | 禁止强推（non fast-forward） |
| 5 | GL_PROTOCOL=web 跳过 push 类校验 |
| 6 | 计算新引入 commit |
| 7 | merge commit 豁免 |
| 8 | 提交人与 push 用户一致性校验 |
| 9 | 分支不在任务号校验范围 → 跳过 |
| 10 | 用户白名单 → 跳过任务号校验 |
| 11 | 分支白名单 → 跳过任务号校验 |
| 12 | 项目白名单 → 跳过任务号校验 |
| 13 | message 白名单 → 跳过任务号校验 |
| 14 | 任务号校验 |
