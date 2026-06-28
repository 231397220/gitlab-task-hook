# GitLab Task Hook 构建工具

本文件夹包含完整的构建工具和文档。

## 📁 文件清单

### 文档

| 文件 | 用途 |
|------|------|
| **BUILD.md** | 完整的构建指南 |
| **README.md** | 本文件，构建工具概览 |

### 构建脚本和工具

| 文件 | 权限 | 用途 |
|------|------|------|
| **build.sh** | 755 | 一键自动化构建脚本（推荐） |
| **Makefile** | 644 | Makefile 构建工具 |

## 🚀 快速开始

### 方案 1：使用构建脚本（推荐）

```bash
# 进入构建目录
cd build

# 简单构建
./build.sh

# 或带选项构建
./build.sh --version v1.0.0 --optimize

# 构建所有平台
./build.sh --all --optimize --version v1.0.0 --output /tmp/releases
```

### 方案 2：使用 Makefile

```bash
# 进入构建目录
cd build

# 简单构建
make build

# 多平台构建
make build-all

# 优化版本
make build-optimized

# 帮助
make help
```

### 方案 3：直接使用 Go

```bash
# 在项目根目录执行
go build -o gitlab-task-hook .

# 验证
./gitlab-task-hook version
```

## 📖 构建脚本详解

### build.sh 功能

强大的 Bash 构建脚本，支持：

- ✓ 多平台编译（Linux amd64, arm64）
- ✓ 版本号注入
- ✓ 构建优化（移除调试符号）
- ✓ 静态链接
- ✓ 详细的日志输出
- ✓ 前置条件检查
- ✓ 二进制验证

### 常用命令

```bash
# 查看帮助
./build.sh --help

# 简单构建
./build.sh

# 指定输出目录
./build.sh --output /tmp/builds

# 注入版本号
./build.sh --version v1.0.0

# 多平台构建
./build.sh --all

# 优化版本（减小大小）
./build.sh --optimize

# 静态链接（无 glibc 依赖）
./build.sh --static

# 组合使用
./build.sh --all --optimize --version v1.0.0 --output /tmp/releases
```

### build.sh 选项说明

| 选项 | 说明 | 示例 |
|------|------|------|
| `--output <path>` | 输出目录 | `--output /tmp/builds` |
| `--version <version>` | 版本号 | `--version v1.0.0` |
| `--all` | 构建所有平台 | `--all` |
| `--optimize` | 优化版本 | `--optimize` |
| `--static` | 静态链接 | `--static` |
| `--help` | 显示帮助 | `--help` |

## 🔧 Makefile 详解

### Makefile 功能

提供快速的构建命令：

- ✓ 构建当前平台
- ✓ 多平台构建
- ✓ 优化构建
- ✓ 单元测试
- ✓ 代码检查和格式化
- ✓ 覆盖率报告

### 常用命令

```bash
# 查看帮助
make help

# 构建
make build

# 多平台构建
make build-all

# 优化版本
make build-optimized

# 静态链接
make build-static

# 运行测试
make test

# 代码检查
make lint

# 代码格式化
make fmt

# 清理产物
make clean

# 生成覆盖率报告
make test-coverage
```

### Makefile 变量

| 变量 | 说明 | 默认值 | 示例 |
|------|------|--------|------|
| `OUTPUT_DIR` | 输出目录 | `.` | `make build OUTPUT_DIR=/tmp` |
| `VERSION` | 版本号 | `dev` | `make build VERSION=v1.0.0` |

## 📊 构建流程

### 1. 前置条件检查

脚本会自动检查：
- Go 是否已安装
- go.mod 是否存在
- 输出目录是否存在或可创建

### 2. 版本信息获取

脚本会自动获取：
- 版本号（从 git tag 或参数）
- Git commit SHA（短形式）
- 构建时间

### 3. 编译构建

根据选项执行：
- 单平台或多平台编译
- 普通或优化构建
- 动态或静态链接

### 4. 验证和输出

构建后会：
- 验证二进制文件是否可执行
- 显示文件大小和信息
- 列出所有输出文件

## 💡 使用示例

### 示例 1：快速编译

```bash
cd build
./build.sh
```

输出：`gitlab-task-hook` 二进制文件在当前目录

### 示例 2：构建发布版本

```bash
cd build
./build.sh \
  --all \
  --optimize \
  --version v1.0.0 \
  --output /tmp/releases
```

输出：
- `/tmp/releases/gitlab-task-hook-linux-amd64`
- `/tmp/releases/gitlab-task-hook-linux-arm64`

### 示例 3：使用 Makefile 构建和测试

```bash
cd build

# 运行测试
make test

# 构建
make build

# 验证
./gitlab-task-hook version
```

### 示例 4：构建并部署

```bash
cd build

# 构建
./build.sh --version v1.0.0

# 部署
sudo ../deploy/deploy.sh ./gitlab-task-hook /path/to/bootstrap.yaml
```

## 🔍 故障排查

### 构建失败

检查以下项目：

```bash
# 1. 检查 Go 版本
go version

# 2. 检查 go.mod
cat go.mod

# 3. 下载依赖
go mod download

# 4. 重新构建
./build.sh
```

### 二进制无法运行

可能是 glibc 版本不兼容，使用静态链接：

```bash
./build.sh --static
```

### 二进制体积很大

使用优化选项：

```bash
./build.sh --optimize
```

## 📚 相关文档

- **详细构建指南**: [BUILD.md](./BUILD.md)
- **部署文档**: [../deploy/DEPLOYMENT.md](../deploy/DEPLOYMENT.md)
- **项目主文档**: [../README.md](../README.md)

## ✅ 检查清单

部署前检查：

- [ ] Go 已安装（1.16+）
- [ ] git 已安装
- [ ] 在项目根目录或 build 目录
- [ ] 已执行 `./build.sh` 或 `make build`
- [ ] 二进制文件已生成
- [ ] 已执行 `./gitlab-task-hook version` 验证

## 快速参考

| 需求 | 命令 |
|------|------|
| 快速构建 | `./build.sh` |
| 多平台构建 | `./build.sh --all --version v1.0.0` |
| 优化构建 | `./build.sh --optimize` |
| 运行测试 | `make test` |
| 查看帮助 | `./build.sh --help` 或 `make help` |
| 清理产物 | `make clean` |

---

**最后更新**: 2026-06-28  
**文档版本**: 1.0
