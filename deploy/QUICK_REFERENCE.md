# GitLab Task Hook 快速参考

## 部署命令速查

### 一键部署（推荐）

```bash
cd /path/to/gitlab-task-hook
sudo ./deploy/deploy.sh ./gitlab-task-hook /path/to/bootstrap.yaml
```

### 手工部署

```bash
# 1. 复制二进制
sudo cp gitlab-task-hook /var/opt/gitlab/gitaly/custom_hooks/bin/
sudo chmod 755 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook

# 2. 复制 bootstrap 配置
sudo cp deploy/bootstrap.yaml.example /etc/gitlab-task-hook/bootstrap.yaml
sudo chmod 600 /etc/gitlab-task-hook/bootstrap.yaml

# 3. 设置权限
sudo chown -R git:git /var/opt/gitlab/gitaly/custom_hooks /etc/gitlab-task-hook

# 4. 初始化缓存
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 5. 配置 hook wrapper
sudo cp deploy/pre-receive.example /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive
sudo chmod 755 /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive

# 6. 配置 systemd service
sudo cp deploy/gitlab-task-hook-config-sync.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable gitlab-task-hook-config-sync.service
sudo systemctl start gitlab-task-hook-config-sync.service
```

## 常用操作

### 查看服务状态

```bash
# 查看 config-sync 服务状态
sudo systemctl status gitlab-task-hook-config-sync.service

# 查看实时日志
sudo journalctl -u gitlab-task-hook-config-sync.service -f

# 查看最近 50 行日志
sudo journalctl -u gitlab-task-hook-config-sync.service -n 50
```

### 管理 config-sync 服务

```bash
# 启动服务
sudo systemctl start gitlab-task-hook-config-sync.service

# 停止服务
sudo systemctl stop gitlab-task-hook-config-sync.service

# 重启服务
sudo systemctl restart gitlab-task-hook-config-sync.service

# 启用开机自启
sudo systemctl enable gitlab-task-hook-config-sync.service

# 禁用开机自启
sudo systemctl disable gitlab-task-hook-config-sync.service
```

### 手工拉取配置

```bash
# 一次性拉取
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 查看拉取结果
echo $?  # 0 表示成功，其他表示失败
```

### 验证本地缓存

```bash
# 查看缓存文件（最近 20 行）
head -20 /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml

# 验证 YAML 格式
/var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-validate --file /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml

# 查看元数据（同步时间等）
cat /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml.meta | jq .
```

### 检查文件权限

```bash
# 检查目录权限
ls -la /var/opt/gitlab/gitaly/custom_hooks/
ls -la /var/opt/gitlab/gitaly/custom_hooks/{bin,config,hooks}

# 检查 bootstrap.yaml 权限
ls -la /etc/gitlab-task-hook/bootstrap.yaml  # 应该是 600

# 检查缓存文件权限
ls -la /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml*
```

## 常见问题

### Q: config-sync 启动失败

**A:** 查看日志找出原因
```bash
sudo journalctl -u gitlab-task-hook-config-sync.service -n 50 --all

# 常见原因及解决方案：
# 1. bootstrap.yaml 不存在 -> 复制 bootstrap.yaml.example 并编辑
# 2. Nacos 连接失败 -> 检查 Nacos 地址和网络连通性
# 3. 权限不足 -> 检查目录所有者是否为 git:git
```

### Q: hook 返回"配置文件不存在"

**A:** 初始化本地缓存
```bash
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 验证文件是否生成
ls -la /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml
```

### Q: 修改 Nacos 配置后没有生效

**A:** 等待 config-sync 拉取（通常 1-2s 内），或手工触发
```bash
# 手工拉取
sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
  config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml

# 检查日志
tail -20 /var/log/gitlab-task-hook/config-sync.log

# 下一次 git push 会使用新配置
```

### Q: 多个 Gitaly 节点配置不同步

**A:** 在所有节点同时拉取
```bash
for node in gitaly1 gitaly2 gitaly3; do
  ssh $node "sudo -u git /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook \
    config-sync --once --bootstrap /etc/gitlab-task-hook/bootstrap.yaml"
done

# 验证所有节点配置一致
for node in gitaly1 gitaly2 gitaly3; do
  echo "=== $node ==="
  ssh $node "md5sum /var/opt/gitlab/gitaly/custom_hooks/config/gitlab-task-hook.yaml"
done
```

## 配置变更流程

1. 登录 Nacos：http://localhost:8848/nacos
2. 进入 dev 命名空间
3. 找到 Data ID: `gitlab-task-hook.yaml`，Group: `GITLAB_HOOK`
4. 编辑并发布配置
5. config-sync 自动拉取（1-2s 内）
6. 下一次 git push 生效

## 紧急禁用 hook

```bash
# 临时禁用：在 Nacos 中修改 enabled: false
# 立即生效（不用重启）

# 完全关闭：删除 hook wrapper
sudo rm /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive

# 停止 config-sync 服务
sudo systemctl stop gitlab-task-hook-config-sync.service
```

## 监控项

| 项目 | 告警条件 | 检查方法 |
|-----|--------|--------|
| 服务状态 | config-sync 未运行 | `systemctl status gitlab-task-hook-config-sync.service` |
| 本地缓存 | 文件不存在或为空 | `ls -la /var/opt/gitlab/.../gitlab-task-hook.yaml` |
| 缓存更新 | 距离现在 >1 小时未更新 | `stat /var/opt/gitlab/.../gitlab-task-hook.yaml.meta` |
| Nacos 连接 | 连续 5+ 次拉取失败 | `grep -i error /var/log/gitlab-task-hook/config-sync.log` |
| 文件权限 | bootstrap.yaml 权限 >640 | `ls -la /etc/gitlab-task-hook/bootstrap.yaml` |

## 日志位置

| 日志 | 位置 | 查看命令 |
|-----|------|--------|
| config-sync 日志 | `/var/log/gitlab-task-hook/config-sync.log` | `tail -f /var/log/gitlab-task-hook/config-sync.log` |
| systemd 日志 | systemd journal | `journalctl -u gitlab-task-hook-config-sync.service -f` |
| GitLab 日志 | `/var/log/gitlab/gitaly/gitaly.log` | `tail -f /var/log/gitlab/gitaly/gitaly.log` |

## 目录结构

```
/var/opt/gitlab/gitaly/custom_hooks/
├── bin/
│   └── gitlab-task-hook              # 二进制文件
├── config/
│   ├── gitlab-task-hook.yaml         # 本地缓存
│   ├── gitlab-task-hook.yaml.meta    # 元数据
│   └── gitlab-task-hook.lock         # 分布式锁
└── hooks/
    └── pre-receive                   # hook wrapper

/etc/gitlab-task-hook/
└── bootstrap.yaml                     # bootstrap 配置

/var/log/gitlab-task-hook/
└── config-sync.log                   # config-sync 日志
```

## 有用的链接

- [完整部署指南](./DEPLOYMENT.md)
- [需求和设计文档](../docs/gitlab-task-hook-nacos-yaml-config-requirements.md)
- [项目 README](../README.md)

---

**快速参考卡片** | 最后更新：2026-06-28
