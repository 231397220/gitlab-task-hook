# GitLab Task Hook 部署资料

本目录包含 `gitlab-task-hook` 的完整部署资料和说明文档。

## 文件清单

### 📄 文档

| 文件 | 说明 |
|-----|------|
| **DEPLOYMENT.md** | 完整部署指南（必读）<br/>包含系统要求、部署步骤、配置说明、故障排查、回滚方案等 |
| **README.md** | 本文件，部署资料概览 |

### 📋 配置示例

| 文件 | 用途 | 部署路径 |
|-----|------|--------|
| **bootstrap.yaml.example** | Nacos 连接和缓存配置<br/>包含 Nacos 地址、认证信息、本地缓存路径等 | `/etc/gitlab-task-hook/bootstrap.yaml` |
| **gitlab-task-hook.yaml.example** | 业务规则配置<br/>包含各种 hook 规则（强推、任务号等）和提示模板<br/>存储在 Nacos dev 命名空间 | Nacos 中（不部署到本地） |

### 🔧 部署工具

| 文件 | 说明 |
|-----|------|
| **deploy.sh** | 快速部署脚本（可选）<br/>自动创建目录、复制文件、配置 systemd 等 |
| **gitlab-task-hook-config-sync.service** | systemd service 文件<br/>管理 config-sync 后台进程 |
| **pre-receive.example** | GitLab hook wrapper 脚本<br/>GitLab 调用此脚本触发 hook 校验 |

## 快速开始

### 方案 1：使用部署脚本（推荐，自动化）

```bash
# 前提：已编译出 gitlab-task-hook 二进制
cd /path/to/gitlab-task-hook

# 1. 编译二进制（如尚未编译）
go build -o gitlab-task-hook .

# 2. 根据环境修改配置示例
cp deploy/bootstrap.yaml.example /tmp/bootstrap.yaml
vi /tmp/bootstrap.yaml  # 修改 Nacos 地址、用户名、密码等

# 3. 执行部署脚本
sudo ./deploy/deploy.sh ./gitlab-task-hook /tmp/bootstrap.yaml
```

脚本会自动：
- 创建目录结构
- 复制二进制文件
- 配置权限
- 部署 systemd service
- 首次拉取 Nacos 配置
- 启动 config-sync 服务

### 方案 2：手工部署（完全控制）

```bash
# 1. 创建目录结构
sudo mkdir -p /var/opt/gitlab/gitaly/custom_hooks/{bin,config,hooks}
sudo mkdir -p /etc/gitlab-task-hook
sudo mkdir -p /var/log/gitlab-task-hook

# 2. 复制二进制文件
sudo cp gitlab-task-hook /var/opt/gitlab/gitaly/custom_hooks/bin/
sudo chmod 755 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook

# 3. 配置 bootstrap.yaml
sudo cp deploy/bootstrap.yaml.example /etc/gitlab-task-hook/bootstrap.yaml
sudo vi /etc/gitlab-task-hook/bootstrap.yaml  # 修改配置
sudo chmod 600 /etc/gitlab-task-hook/bootstrap.yaml

# 4. 设置权限
sudo chown -R git:git /var/opt/gitlab/gitaly/custom_hooks /etc/gitlab-task-hook /var/log/gitlab-task-hook

# 5. 首次拉取配置
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 6. 配置 hook wrapper
sudo cp deploy/pre-receive.example /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive
sudo chmod 755 /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive

# 7. 配置 systemd service
sudo cp deploy/gitlab-task-hook-config-sync.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable gitlab-task-hook-config-sync.service
sudo systemctl start gitlab-task-hook-config-sync.service
```

## 部署前准备

### 必要条件

1. **编译 gitlab-task-hook 二进制**
   ```bash
   go build -o gitlab-task-hook .
   ```

2. **Nacos 已启动并配置好 dev 命名空间**
   - 访问 http://localhost:8848/nacos
   - 用户名/密码：nacos/nacos
   - 确保 dev 命名空间存在

3. **在 Nacos 中发布配置**
   参考 [Nacos 配置发布指南](../docs/gitlab-task-hook-nacos-yaml-config-requirements.md#nacos-配置发布)

4. **Linux 机器以 root 身份访问**

5. **git 用户存在**（通常 GitLab 已自带）
   ```bash
   id git
   ```

### 配置 bootstrap.yaml

**关键配置项**（根据实际环境修改）：

```yaml
nacos:
  server_addr: "http://<nacos_server>:8848"    # 改为实际 Nacos 地址
  namespace_id: "dev"                          # 命名空间
  username: "nacos"                            # Nacos 用户名
  password: "nacos"                            # Nacos 密码
  cache_file: "/var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml"
```

## 部署后验证

### 1. 检查服务状态

```bash
# 查看 config-sync 服务状态
sudo systemctl status gitlab-task-hook-config-sync.service

# 查看实时日志
sudo journalctl -u gitlab-task-hook-config-sync.service -f
```

### 2. 验证本地缓存

```bash
# 查看缓存文件
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml | head -20

# 查看元数据
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta | jq .
```

### 3. 测试 hook

在 Git 仓库中执行 push：

```bash
git config user.email "testuser@example.com"
git checkout -b feature/TEST-001
echo "test" > test.txt
git add test.txt
git commit -m "[#TSK-001] Test commit"
git push origin feature/TEST-001
```

观察输出，应该通过或返回相应的错误提示。

### 4. 查看 GitLab 日志

```bash
# Gitaly 日志
sudo tail -f /var/log/gitlab/gitaly/gitaly.log
```

## 配置变更流程

### 修改规则

1. 登录 Nacos 控制台：http://localhost:8848/nacos
2. 进入 dev 命名空间
3. 查找 Data ID: `gitlab-task-hook.yaml`，Group: `GITLAB_HOOK`
4. 编辑配置（修改规则、白名单、提示信息等）
5. 点击"发布"
6. config-sync 自动拉取（1-2s 内）
7. 下一次 git push 使用新配置

## 故障排查

### config-sync 无法启动

查看详细错误：
```bash
sudo journalctl -u gitlab-task-hook-config-sync.service -n 50
```

常见原因：
- bootstrap.yaml 配置错误
- Nacos 连接失败
- 目录权限不足

详见 [DEPLOYMENT.md 中的故障排查章节](./DEPLOYMENT.md#故障排查)

### hook 返回"配置文件不存在"

```bash
# 手工执行一次 config-sync
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 查看日志
tail -20 /var/log/gitlab-task-hook/config-sync.log
```

### 多个 Gitaly 节点配置不同步

```bash
# 在所有节点执行同步
for node in gitaly1 gitaly2 gitaly3; do
  ssh $node "sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
    config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml"
done
```

## 回滚方案

### 临时禁用规则

在 Nacos 中修改配置：
```yaml
enabled: false  # 改为 false
```

所有 push 都会通过。

### 回滚到上一个配置版本

Nacos 通常支持配置历史版本，可在控制台直接恢复。

详见 [DEPLOYMENT.md 中的回滚方案章节](./DEPLOYMENT.md#回滚方案)

## 文档导航

- **[完整部署指南](./DEPLOYMENT.md)** - 详细的部署步骤和配置说明
- **[需求文档](../docs/gitlab-task-hook-nacos-yaml-config-requirements.md)** - 原始需求和设计文档
- **[README.md](../README.md)** - 项目主文档

## 获取帮助

- 查看日志：`journalctl -u gitlab-task-hook-config-sync.service -f`
- 查看部署文档：`less DEPLOYMENT.md`
- 检查配置示例：`cat bootstrap.yaml.example`
- 验证二进制：`./gitlab-task-hook version`

---

**最后更新**：2026-06-28  
**文档版本**：1.0
