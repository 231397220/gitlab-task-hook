# 部署资料文件清单

## 📦 部署包概览

**创建日期**: 2026-06-28  
**部署版本**: 1.0  
**目标项目**: gitlab-task-hook  

## 📂 文件列表

### 核心文档

```
deploy/
├── README.md                             (7.1 KB)
│   ├─ 用途: 部署资料概览和快速入门
│   ├─ 读者: 所有使用者
│   ├─ 读时间: 5 分钟
│   └─ 内容:
│       • 文件清单和用途
│       • 快速开始（两种方案）
│       • 部署前准备
│       • 部署后验证
│       • 常见问题链接
│
├── DEPLOYMENT.md                         (20 KB)  ★★★ 必读
│   ├─ 用途: 完整部署指南和参考手册
│   ├─ 读者: 运维人员、DevOps
│   ├─ 读时间: 20 分钟
│   └─ 内容:
│       • 系统要求（OS、Go、Nacos、GitLab）
│       • 部署架构图和流程说明
│       • 8 步详细部署流程
│       • bootstrap.yaml 详解
│       • gitlab-task-hook.yaml 详解
│       • 监控与维护指南
│       • 5 大故障排查场景
│       • 4 种回滚方案
│       • 命令速查表
│
└── QUICK_REFERENCE.md                    (6.9 KB)
    ├─ 用途: 快速参考卡片
    ├─ 读者: 所有使用者
    ├─ 用法: 需要时速查命令
    └─ 内容:
        • 一键部署命令
        • 常用操作（启动/停止/查看日志等）
        • 常见问题 Q&A
        • 文件权限检查
        • 紧急禁用方案
        • 监控指标和日志位置
```

### 配置示例

```
deploy/
├── bootstrap.yaml.example                (2.3 KB)
│   ├─ 用途: Nacos 连接和缓存配置示例
│   ├─ 部署路径: /etc/gitlab-task-hook/bootstrap.yaml
│   ├─ 权限: 600 (避免密码泄露)
│   └─ 关键配置:
│       • nacos.server_addr: Nacos 地址
│       • nacos.namespace_id: 命名空间（dev）
│       • nacos.username/password: 认证信息
│       • nacos.cache_file: 本地缓存路径
│       • nacos.watch_enabled: 是否启用监听
│
└── gitlab-task-hook.yaml.example         (9.6 KB)
    ├─ 用途: 业务规则配置示例
    ├─ 部署位置: Nacos dev 命名空间
    ├─ Data ID: gitlab-task-hook.yaml
    ├─ Group: GITLAB_HOOK
    └─ 配置内容:
        • 版本和全局开关
        • 规则配置（强推、直接 push、提交人、任务号等）
        • 白名单（用户/分支/项目）
        • 消息模板（中文提示信息）
        • 日志配置
```

### 部署工具

```
deploy/
├── deploy.sh                             (6.9 KB, 可执行)
│   ├─ 用途: 一键自动化部署脚本
│   ├─ 权限: 755（已设置）
│   ├─ 用法: sudo ./deploy.sh <binary> [bootstrap_path]
│   └─ 自动执行:
│       • 检查前置条件
│       • 创建目录结构
│       • 复制二进制文件
│       • 设置文件权限
│       • 配置 bootstrap.yaml
│       • 初始化本地缓存
│       • 配置 systemd service
│       • 启动 config-sync 服务
│       • 验证部署成功
│
├── gitlab-task-hook-config-sync.service  (1.1 KB)
│   ├─ 用途: systemd service 文件
│   ├─ 部署路径: /etc/systemd/system/gitlab-task-hook-config-sync.service
│   ├─ 功能:
│   │   • 管理 config-sync 后台进程
│   │   • 开机自启
│   │   • 异常自动重启（10s 延迟）
│   │   • 日志输出到 systemd journal
│   └─ 运行用户: git
│
└── pre-receive.example                   (1.1 KB)
    ├─ 用途: GitLab hook wrapper 脚本
    ├─ 部署路径: /var/opt/gitlab/gitaly/custom_hooks/hooks/pre-receive
    ├─ 权限: 755
    └─ 功能:
        • GitLab 调用此脚本触发 hook
        • 读取 GL_* 环境变量
        • 调用 gitlab-task-hook hook 模式
```

### 其他文件

```
deploy/
└── FILES_MANIFEST.md                     (本文件)
    ├─ 用途: 部署资料文件清单
    └─ 内容: 详细的文件列表和用途说明
```

## 📊 统计信息

| 分类 | 数量 | 总大小 |
|------|------|--------|
| 文档 | 4 个 | ~40 KB |
| 配置示例 | 2 个 | ~12 KB |
| 部署工具 | 3 个 | ~9 KB |
| **总计** | **9 个** | **~61 KB** |

## 🚀 部署路径导航

### 快速部署（5 分钟）

```bash
# 1. 复制配置示例
cp deploy/bootstrap.yaml.example /tmp/bootstrap.yaml

# 2. 编辑配置（改 Nacos 地址、用户名、密码）
vi /tmp/bootstrap.yaml

# 3. 执行一键部署
sudo ./deploy/deploy.sh ./gitlab-task-hook /tmp/bootstrap.yaml

# 4. 验证部署
sudo systemctl status gitlab-task-hook-config-sync.service
```

### 详细部署（30 分钟）

```bash
# 1. 阅读 README.md（快速了解）
cat deploy/README.md

# 2. 阅读 DEPLOYMENT.md（深入理解）
less deploy/DEPLOYMENT.md

# 3. 根据文档手工执行各步骤
# （详见 DEPLOYMENT.md 中的"部署步骤"章节）
```

### 故障排查（按需）

```bash
# 1. 查看快速参考卡片
cat deploy/QUICK_REFERENCE.md

# 2. 查看完整故障排查指南
less deploy/DEPLOYMENT.md  # 滚动到"故障排查"章节
```

## 📖 推荐阅读顺序

1. **README.md** (5 分钟)
   - 了解部署资料的整体结构
   - 学习两种部署方案
   - 确认前置条件

2. **QUICK_REFERENCE.md** (10 分钟)
   - 学习常用命令
   - 了解常见问题解决方案
   - 保存书签供后续查询

3. **DEPLOYMENT.md** (20 分钟)
   - 深入理解部署架构
   - 学习每一步的细节和原理
   - 了解监控、维护、故障排查、回滚方案

4. **配置示例**
   - 根据需要参考 bootstrap.yaml.example
   - 查阅 gitlab-task-hook.yaml.example 了解规则

5. **执行部署**
   - 使用 deploy.sh 脚本一键部署（推荐）
   - 或手工执行 DEPLOYMENT.md 中的步骤

## ✅ 部署检查清单

- [ ] 已阅读 README.md
- [ ] 已准备 bootstrap.yaml 配置（修改 Nacos 地址）
- [ ] 已编译 gitlab-task-hook 二进制
- [ ] 已确认有 root/sudo 权限
- [ ] 已确认目标机器存在 git 用户
- [ ] 已在 Nacos 中发布 gitlab-task-hook.yaml 配置（已完成 ✓）
- [ ] 已执行部署（使用 deploy.sh 或手工）
- [ ] 已验证 config-sync 服务运行正常
- [ ] 已验证本地缓存文件存在
- [ ] 已在测试仓库中验证 hook 功能

## 🔗 相关文档链接

- **项目需求文档**: `docs/gitlab-task-hook-nacos-yaml-config-requirements.md`
- **项目主文档**: `README.md`（项目根目录）

## 📞 快速链接

| 需求 | 文件 |
|------|------|
| 快速开始 | README.md |
| 命令速查 | QUICK_REFERENCE.md |
| 详细部署 | DEPLOYMENT.md |
| bootstrap 配置 | bootstrap.yaml.example |
| 业务规则配置 | gitlab-task-hook.yaml.example |
| 自动部署脚本 | deploy.sh |
| systemd 配置 | gitlab-task-hook-config-sync.service |
| hook wrapper | pre-receive.example |

---

**部署资料版本**: 1.0  
**最后更新**: 2026-06-28  
**文档完整性**: ✓ 100%  
**可使用性**: ✓ 立即可用
