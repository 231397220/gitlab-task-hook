# GitLab Task Hook 部署指南

## 目录

1. [概述](#概述)
2. [系统要求](#系统要求)
3. [部署架构](#部署架构)
4. [部署步骤](#部署步骤)
5. [配置说明](#配置说明)
6. [监控与维护](#监控与维护)
7. [故障排查](#故障排查)
8. [回滚方案](#回滚方案)

---

## 概述

`gitlab-task-hook` 是一个 GitLab pre-receive hook，用于对代码提交进行治理，支持的功能包括：

- 强推（non fast-forward）检测
- 受保护分支直接 push 拦截
- 提交人与 push 用户一致性校验
- 任务 ID 格式校验
- 用户/分支/项目白名单

### 部署特点

- **配置中心化**：配置存储在 Nacos，由 `config-sync` 后台服务拉取并缓存到本地
- **hook 独立**：hook 执行时只读本地缓存配置，不依赖 Nacos 可用性
- **原子更新**：配置文件使用原子写入，避免 hook 读到不完整配置
- **自动重载**：修改 Nacos 配置后，下一次 git push 自动生效

---

## 系统要求

### 运行环境

- **操作系统**：Linux（RHEL 7.x+、CentOS 7.x+、Ubuntu 18.04+）
- **Go 版本**：1.16+（编译时）
- **Nacos 版本**：2.0.0+（建议 2.1.0+）
- **GitLab 版本**：13.0+

### 机器配置

| 配置项 | 要求 | 说明 |
|-------|------|------|
| CPU | 2+ 核 | config-sync 轮询不占用 CPU |
| 内存 | 512M+ | Go 进程典型占用 50-100M |
| 磁盘 | 10GB+ | 本地缓存占用 <10MB |

### 网络要求

- Gitaly 节点能访问 Nacos server（通常同机房）
- Nacos server 访问延迟 <500ms 推荐
- 支持 HTTP 和 HTTPS（根据 Nacos 配置）

---

## 部署架构

### 整体流程图

```
┌─────────────────────────────────────────────────────────┐
│                   Nacos 配置中心                        │
│  namespace: dev                                         │
│  dataId: gitlab-task-hook.yaml                          │
│  group: GITLAB_HOOK                                     │
└────────────────────────────┬────────────────────────────┘
                             │ HTTP API
                             ▼
         ┌───────────────────────────────────┐
         │   gitlab-task-hook config-sync    │
         │   （systemd service，长驻进程）    │
         │   └─ watch 监听配置变化           │
         │   └─ 原子更新本地缓存             │
         └────────────┬──────────────────────┘
                      │ 文件写入（原子操作）
                      ▼
    ┌─────────────────────────────────┐
    │  本地缓存（仅读）               │
    │  /var/opt/gitlab/.../           │
    │  └─ gitlab-task-hook.yaml       │
    │  └─ gitlab-task-hook.yaml.meta  │
    └────────────┬────────────────────┘
                 │ 每次 push 读取
                 ▼
    ┌─────────────────────────────────┐
    │  GitLab hook（hook 模式）       │
    │  git push 触发                  │
    │  执行 push 规则校验             │
    │  输出错误或放行                 │
    └─────────────────────────────────┘
```

### 关键目录结构

```
/var/opt/gitlab/gitaly/custom_hooks/
├── bin/
│   └── gitlab-task-hook              # 二进制文件
├── config/
│   ├── gitlab-task-hook.yaml         # 本地缓存（自动生成）
│   ├── gitlab-task-hook.yaml.meta    # 元数据文件（自动生成）
│   └── gitlab-task-hook.lock         # 分布式锁（自动生成）
└── hooks/
    └── pre-receive                   # GitLab hook wrapper

/etc/gitlab-task-hook/
└── bootstrap.yaml                     # bootstrap 配置（人工维护）

/var/log/gitlab-task-hook/
└── config-sync.log                   # config-sync 日志
```

---

## 部署步骤

### 第一步：编译二进制文件

假设你已有 gitlab-task-hook 源码在 `/path/to/gitlab-task-hook`

```bash
cd /path/to/gitlab-task-hook
go build -o gitlab-task-hook .
```

### 第二步：创建目录结构

在所有 Gitaly 节点上执行（用 root 或 git 用户）：

```bash
# 创建目录
mkdir -p /var/opt/gitlab/gitaly/custom_hooks/{bin,config,hooks}
mkdir -p /etc/gitlab-task-hook
mkdir -p /var/log/gitlab-task-hook

# 设置权限（假设 git 用户运行 GitLab Gitaly）
chown -R git:git /var/opt/gitlab/gitaly/custom_hooks
chown -R git:git /etc/gitlab-task-hook
chown -R git:git /var/log/gitlab-task-hook

chmod 750 /var/opt/gitlab/gitaly/custom_hooks/{bin,config,hooks}
chmod 640 /var/opt/gitlab/gitaly/custom_hooks/config/*
```

### 第三步：部署二进制文件

```bash
# 复制二进制文件到 bin 目录
cp gitlab-task-hook /var/opt/gitlab/gitaly/custom_hooks/bin/

# 设置执行权限
chmod 755 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook

# 验证
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook version
```

### 第四步：配置 bootstrap.yaml

参考 `bootstrap.yaml.example`，根据实际环境修改，复制到 `/etc/gitlab-task-hook/bootstrap.yaml`：

```bash
cp deploy/bootstrap.yaml.example /etc/gitlab-task-hook/bootstrap.yaml

# 编辑配置（特别是 Nacos 地址、用户名、密码）
vi /etc/gitlab-task-hook/bootstrap.yaml

# 验证 YAML 格式
gitlab-task-hook config-validate --file /etc/gitlab-task-hook/bootstrap.yaml

# 设置权限（避免密码泄露）
chmod 600 /etc/gitlab-task-hook/bootstrap.yaml
```

### 第五步：初始化本地缓存

首次执行时从 Nacos 拉取配置，缓存到本地：

```bash
# 以 git 用户身份执行（保持权限一致）
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once \
  --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 验证缓存文件是否生成
ls -la /var/opt/gitlab/gitaly/custom_hooks/config/
```

如果拉取失败，需要排查 Nacos 连接问题（见故障排查章节）。

### 第六步：启动 config-sync 后台服务

使用 systemd 管理 config-sync 进程（长驻）。参考 `gitlab-task-hook-config-sync.service` 示例：

```bash
# 复制 systemd service 文件
sudo cp deploy/gitlab-task-hook-config-sync.service /etc/systemd/system/

# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start gitlab-task-hook-config-sync.service

# 启用开机自启
sudo systemctl enable gitlab-task-hook-config-sync.service

# 查看服务状态
sudo systemctl status gitlab-task-hook-config-sync.service

# 查看日志
sudo journalctl -u gitlab-task-hook-config-sync.service -f
```

### 第七步：配置 GitLab hook wrapper

创建 pre-receive hook wrapper 脚本：

```bash
# 参考 pre-receive.example
cat > /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive << 'EOF'
#!/bin/bash
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook hook
EOF

# 设置执行权限
chmod 755 /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive

# 验证脚本可执行
/var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive --help || echo "hook mode started"
```

### 第八步：测试部署

#### 测试 1：验证 hook 模式可读本地缓存

```bash
# 手工触发一次 hook（使用空输入）
echo "" | /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook hook \
  GL_USERNAME=testuser \
  GL_PROJECT_PATH=group/test-repo \
  GL_PROTOCOL=ssh

# 应该看到错误或正常退出（取决于输入）
```

#### 测试 2：验证 config-sync 工作正常

```bash
# 查看最后同步时间
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta | jq .

# 查看配置内容
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml | head -20
```

#### 测试 3：在 Git 仓库中模拟 push

需要在有 GitLab Gitaly 的环境中创建测试 Git 仓库，执行 push 操作，观察 hook 返回值：

```bash
cd /path/to/test/repo
git config user.name "Test User"
git config user.email "testuser@example.com"

# 创建测试分支并 commit
git checkout -b feature/TEST-001
echo "test" > test.txt
git add test.txt
git commit -m "[#TSK-001] Test commit"

# push 到 dev 分支（假设允许）
git push origin feature/TEST-001

# 查看输出，应该通过或输出相应的错误提示
```

---

## 配置说明

### bootstrap.yaml 配置

bootstrap.yaml 是 config-sync 启动时读取的配置，包含 Nacos 连接信息和本地缓存路径。

详见 `bootstrap.yaml.example`，关键字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `nacos.enabled` | 是 | 是否启用 Nacos 同步 |
| `nacos.server_addr` | 是 | Nacos server 地址，如 `http://nacos.example.com:8848` |
| `nacos.namespace_id` | 否 | Nacos 命名空间 ID，如 `dev` |
| `nacos.group` | 是 | Nacos group，固定值 `GITLAB_HOOK` |
| `nacos.data_id` | 是 | Nacos dataId，固定值 `gitlab-task-hook.yaml` |
| `nacos.username` | 否 | Nacos 用户名（若启用认证） |
| `nacos.password` | 否 | Nacos 密码（若启用认证） |
| `nacos.cache_file` | 是 | 本地缓存路径 |
| `nacos.cache_meta_file` | 否 | 元数据文件路径 |
| `nacos.log_file` | 否 | config-sync 日志文件路径 |
| `nacos.timeout_seconds` | 否 | 超时时间，默认 5s |
| `nacos.watch_enabled` | 否 | 是否启用监听，默认 true |
| `nacos.poll_interval_seconds` | 否 | 轮询间隔，默认 30s |

### gitlab-task-hook.yaml 配置

这是实际的业务规则配置，存储在 Nacos，由 config-sync 拉取。

详见 Nacos 中的配置或 `gitlab-task-hook.yaml.example`。

关键规则：

- `rules.root_bypass`：root 用户跳过所有校验
- `rules.non_fast_forward`：禁止强推
- `rules.deny_direct_push`：受保护分支禁止直接 push
- `rules.committer_match_push_user`：提交人与 push 用户一致
- `rules.task_id`：commit subject 必须含任务号
- `whitelist`：豁免任务号校验的用户/分支/项目

---

## 监控与维护

### 日志文件

#### config-sync 日志

```bash
tail -f /var/log/gitlab-task-hook/config-sync.log
```

关注日志内容：
- `Pulling config from Nacos` - config-sync 启动
- `Config updated successfully` - 配置更新成功
- `Config unchanged` - 配置无变化（MD5 相同）
- `Failed to get config from Nacos` - Nacos 连接失败（不阻断 hook）

#### GitLab hook 日志

hook 模式的日志输出到 stderr，会被 GitLab 捕获显示给用户：

```bash
# 查看 GitLab 的 push 日志
sudo tail -f /var/log/gitlab/gitaly/gitaly.log

# 或在 GitLab 控制台查看 Activity 和 Push Events
```

### 监控指标

建议监控以下指标：

| 指标 | 告警阈值 | 说明 |
|------|---------|------|
| config-sync 进程状态 | 不运行 | systemd 进程存活性 |
| 配置缓存文件存在 | 缺失 | /var/opt/gitlab/.../gitlab-task-hook.yaml |
| 配置缓存更新时间 | >1小时 | 上次成功同步距离现在 |
| Nacos 连接失败次数 | >5 次/分钟 | 网络连通性 |
| bootstrap.yaml 权限 | >640 | 避免密码泄露 |

### 定期维护

#### 每周检查

- [ ] config-sync 进程运行状态
- [ ] 配置缓存文件权限
- [ ] Nacos 连接日志中是否有异常

#### 每月检查

- [ ] 本地缓存文件大小（应 <10MB）
- [ ] 日志文件大小，按需轮转
- [ ] 更新 gitlab-task-hook binary（若有新版本）

#### 配置变更流程

1. 在 Nacos dev 命名空间中修改 `gitlab-task-hook.yaml` 配置
2. 发布配置
3. config-sync 监听到变化，自动拉取并更新本地缓存（通常 1-2s 内）
4. 下一次 git push 使用新配置
5. 检查日志确认更新成功

---

## 故障排查

### 问题 1：config-sync 启动失败

#### 症状
```
systemctl status gitlab-task-hook-config-sync.service
# Job for gitlab-task-hook-config-sync.service failed
```

#### 排查步骤

```bash
# 1. 查看详细错误信息
journalctl -u gitlab-task-hook-config-sync.service -n 50

# 2. 手工执行 config-sync，查看错误
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 3. 检查 bootstrap.yaml 语法
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-validate --file /etc/gitlab-task-hook/bootstrap.yaml

# 4. 检查目录权限
ls -la /var/opt/gitlab/gitaly/custom_hooks/config/
```

#### 常见原因

| 原因 | 解决方案 |
|-----|--------|
| bootstrap.yaml 不存在 | 复制 bootstrap.yaml.example，修改 Nacos 地址 |
| Nacos 连接超时 | 检查网络连通性，ping Nacos server |
| 用户名/密码错误 | 确认 Nacos 认证信息 |
| 目录权限不足 | 检查 /var/opt/gitlab 权限，应为 git:git |
| 磁盘空间不足 | 检查 /var 分区空间 |

### 问题 2：hook 返回错误"配置文件不存在"

#### 症状

```
git push 时输出：
local config not found, using default built-in config
```

#### 排查步骤

```bash
# 1. 检查缓存文件是否存在
ls -la /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml

# 2. 检查 config-sync 日志
tail -100 /var/log/gitlab-task-hook/config-sync.log | grep -i error

# 3. 手工执行一次 config-sync 初始化
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 4. 确认文件权限
ls -la /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml*
```

#### 常见原因

| 原因 | 解决方案 |
|-----|--------|
| config-sync 尚未执行过 | 手工执行一次 config-sync --once |
| Nacos 中配置不存在 | 参考《Nacos 配置发布》章节 |
| 本地缓存被删除 | 重新执行 config-sync --once |
| 目录权限问题 | 确保 git 用户可读取该目录 |

### 问题 3：Nacos 不可用时 hook 被拦截

#### 症状

Nacos 宕机，但 git push 被拒绝（应该使用本地缓存）

#### 原因

hook 模式配置读取失败，可能是本地缓存损坏或权限问题

#### 解决方案

```bash
# 1. 检查本地缓存文件完整性
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml | head -20

# 2. 校验 YAML 格式
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-validate --file /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml

# 3. 检查文件权限，git 用户是否可读
stat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml

# 4. Nacos 恢复后，config-sync 会自动重新拉取
sudo systemctl restart gitlab-task-hook-config-sync.service
```

### 问题 4：hook 规则生效缓慢

#### 症状

修改 Nacos 配置后，过了 1 分钟才在 push 时生效

#### 原因

- config-sync 监听到变化需要 1-2s
- 旧的 hook 进程可能仍在读取旧配置（hook 是短生命周期）
- 如果 watch 不可用，依赖轮询，默认 30s 一次

#### 解决方案

```bash
# 1. 检查 config-sync 是否启用了 watch
cat /etc/gitlab-task-hook/bootstrap.yaml | grep watch_enabled

# 2. 如果 watch 未启用，可手工触发 Nacos 监听刷新
sudo systemctl restart gitlab-task-hook-config-sync.service

# 3. 强制重新拉取一次
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 4. 观察日志，确认拉取成功
tail -20 /var/log/gitlab-task-hook/config-sync.log
```

### 问题 5：多个 Gitaly 节点配置不一致

#### 症状

A 节点允许某个 push，B 节点拒绝

#### 原因

两个节点的本地缓存不同步，可能是：
- config-sync 在 A 节点运行，B 节点未启动
- bootstrap.yaml 配置不同

#### 解决方案

```bash
# 1. 检查所有 Gitaly 节点的 config-sync 状态
for node in gitaly1 gitaly2 gitaly3; do
  ssh $node systemctl status gitlab-task-hook-config-sync.service
done

# 2. 确保所有节点 bootstrap.yaml 一致
# 建议用配置管理工具（Ansible/Puppet）统一部署

# 3. 对所有节点同时执行 config-sync --once
for node in gitaly1 gitaly2 gitaly3; do
  ssh $node "sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
    config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml"
done

# 4. 验证所有节点缓存文件的 MD5 一致
for node in gitaly1 gitaly2 gitaly3; do
  echo "=== $node ===" 
  ssh $node "md5sum /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml"
done
```

---

## 回滚方案

### 场景 1：hook 规则配置错误，导致所有 push 被拦截

#### 快速恢复（1 分钟内）

1. 在 Nacos 中恢复之前的 `gitlab-task-hook.yaml` 配置
2. config-sync 自动拉取（1-2s 内）
3. 下一次 git push 使用恢复的配置

```bash
# Nacos 中回滚配置（参考 Nacos 控制台或 API）
curl -X POST "http://nacos.example.com:8848/nacos/v1/cs/configs" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "tenant=dev" \
  --data-urlencode "dataId=gitlab-task-hook.yaml" \
  --data-urlencode "group=GITLAB_HOOK" \
  --data-urlencode "content=<旧配置内容>" \
  --data-urlencode "username=nacos" \
  --data-urlencode "password=nacos"
```

### 场景 2：config-sync 故障，本地缓存损坏

#### 恢复步骤

```bash
# 1. 停止 config-sync 服务
sudo systemctl stop gitlab-task-hook-config-sync.service

# 2. 从备份恢复缓存文件（若有）
cp /backup/gitlab-task-hook.yaml.bak \
   /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml

# 3. 或手工从 Nacos 重新拉取
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 4. 重启 config-sync
sudo systemctl start gitlab-task-hook-config-sync.service

# 5. 验证恢复
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml | head -20
```

### 场景 3：需要完全关闭 hook 功能

#### 临时禁用 hook

修改本地缓存配置（或 Nacos 配置）：

```yaml
enabled: false  # 改为 false
```

这样 hook 会对所有 push 直接放行。

#### 永久关闭

```bash
# 1. 删除 pre-receive hook
rm /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive

# 2. 停止 config-sync 服务
sudo systemctl stop gitlab-task-hook-config-sync.service
sudo systemctl disable gitlab-task-hook-config-sync.service

# 3. 清理本地缓存（可选）
rm /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml*
```

### 场景 4：回滚到之前版本的二进制

```bash
# 1. 备份当前版本
cp /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
   /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook.v1.1.0

# 2. 从备份恢复旧版本
cp /backup/gitlab-task-hook.v1.0.0 \
   /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook

# 3. 重新启动 config-sync（如果有兼容性问题）
sudo systemctl restart gitlab-task-hook-config-sync.service

# 4. 验证
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook version
```

---

## 附录

### 文件清单

- `bootstrap.yaml.example` - bootstrap 配置示例
- `gitlab-task-hook.yaml.example` - gitlab-task-hook 配置示例
- `gitlab-task-hook-config-sync.service` - systemd service 文件
- `pre-receive.example` - GitLab hook wrapper 脚本
- `DEPLOYMENT.md` - 本部署文档

### 相关命令速查

```bash
# 查看版本
gitlab-task-hook version

# 校验本地配置
gitlab-task-hook config-validate --file /path/to/config.yaml

# 首次拉取配置（一次性）
gitlab-task-hook config-sync --once --bootstrap /path/to/bootstrap.yaml

# 启动 config-sync 监听（长驻）
gitlab-task-hook config-sync --watch --bootstrap /path/to/bootstrap.yaml

# 执行 hook 模式（通常由 GitLab 调用）
gitlab-task-hook hook

# 查看 systemd 服务状态
sudo systemctl status gitlab-task-hook-config-sync.service

# 查看实时日志
sudo journalctl -u gitlab-task-hook-config-sync.service -f

# 查看 config-sync 日志文件
tail -f /var/log/gitlab-task-hook/config-sync.log
```

### 联系方式

如有问题，请联系：
- DevOps 团队：devops@example.com
- GitLab 管理员：gitlab-admin@example.com

---

**最后更新时间**：2026-06-28  
**文档版本**：1.0
