# GitLab Task Hook 构建指南

## 快速开始

```bash
# 方法 1：使用构建脚本（推荐）
./build.sh

# 方法 2：使用 Makefile
make build

# 方法 3：直接使用 Go
go build -o gitlab-task-hook ..

# 验证
../gitlab-task-hook version
```

## 系统要求

### 编译环境

| 要求 | 版本 |
|------|------|
| Go | 1.16+ |
| Git | 2.0+ |
| Make | 4.0+（可选） |

### 运行环境

| 要求 | 说明 |
|------|------|
| OS | Linux (RHEL 7.x+, CentOS 7.x+, Ubuntu 18.04+) |
| libc | glibc 2.17+ |
| 架构 | amd64, arm64 |

## 构建工具

### 1. 构建脚本 (build.sh)

功能强大的 Bash 脚本，支持多种构建选项。

**用法:**
```bash
./build.sh [选项]

选项:
  --output <path>      输出目录（默认：当前目录）
  --version <version>  版本号，如 v1.0.0（默认：无）
  --all                构建所有平台（linux-amd64, linux-arm64）
  --optimize           构建优化版本（移除调试符号，减小大小）
  --static             静态链接（避免 glibc 依赖）
  --help              显示帮助信息
```

**示例:**
```bash
# 简单构建
./build.sh

# 输出到指定目录
./build.sh --output /tmp/builds

# 注入版本号
./build.sh --version v1.0.0

# 构建所有平台
./build.sh --all --version v1.0.0

# 优化版本（减小大小）
./build.sh --optimize

# 静态链接
./build.sh --static

# 完整示例
./build.sh --all --optimize --version v1.0.0 --output /tmp/releases
```

### 2. Makefile

提供快速的构建命令。

**常用命令:**
```bash
make build              # 构建当前平台
make build-all          # 构建所有平台
make build-optimized    # 优化版本（去掉调试符号）
make build-static       # 静态链接构建
make test               # 运行测试
make lint               # 代码检查
make fmt                # 代码格式化
make clean              # 清理构建产物
make help               # 显示帮助
```

**示例:**
```bash
# 构建
make build

# 构建所有平台并指定版本
make build-all VERSION=v1.0.0 OUTPUT_DIR=/tmp/releases

# 构建优化版本
make build-optimized

# 运行测试并构建
make test && make build
```

## 构建方法详解

### 方法 1：简单构建（推荐）

```bash
# 在项目根目录执行
go build -o gitlab-task-hook .

# 或在 build 目录中执行
cd build
make build
```

输出：当前目录下生成 `gitlab-task-hook` 二进制文件

### 方法 2：指定输出路径

```bash
# 使用脚本
./build/build.sh --output /path/to/output

# 使用 Makefile
make -C build build OUTPUT_DIR=/path/to/output

# 使用 Go
go build -o /path/to/output/gitlab-task-hook .
```

### 方法 3：跨平台构建

```bash
# 使用脚本
./build/build.sh --all

# 使用 Makefile
make -C build build-all

# 手工执行
GOOS=linux GOARCH=amd64 go build -o gitlab-task-hook-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o gitlab-task-hook-linux-arm64 .
```

### 方法 4：包含版本信息的构建

```bash
# 使用脚本
./build/build.sh --version v1.0.0

# 使用 Makefile
make -C build build VERSION=v1.0.0

# 手工执行
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')

go build \
  -ldflags "-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildTime=${BUILD_TIME}'" \
  -o gitlab-task-hook \
  .
```

## 构建优化

### 减小二进制大小

```bash
# 移除调试符号
./build/build.sh --optimize

# 使用 Makefile
make -C build build-optimized

# 或手工执行
go build -ldflags="-s -w" -o gitlab-task-hook .
```

### 静态链接（避免 glibc 依赖）

```bash
# 使用脚本
./build/build.sh --static

# 使用 Makefile
make -C build build-static

# 或手工执行
CGO_ENABLED=0 go build -o gitlab-task-hook .
```

## 验证构建

### 检查二进制信息

```bash
# 查看版本
./gitlab-task-hook version

# 查看帮助信息
./gitlab-task-hook --help

# 验证 hook 模式
echo "" | ./gitlab-task-hook hook

# 验证 config-validate 模式
./gitlab-task-hook config-validate --file /path/to/config.yaml
```

### 测试二进制文件

```bash
# 检查文件类型
file ./gitlab-task-hook

# 检查依赖库（非静态链接时）
ldd ./gitlab-task-hook

# 检查文件大小
ls -lh ./gitlab-task-hook
```

## 依赖管理

### 查看依赖

```bash
go mod graph
go mod why <module>
```

### 更新依赖

```bash
# 更新所有依赖到最新版本
go get -u ./...

# 更新特定依赖
go get -u github.com/example/module@latest

# 清理不使用的依赖
go mod tidy
```

### 生成依赖报告

```bash
go mod graph > dependencies.txt
```

## 单元测试

### 运行测试

```bash
# 运行所有测试
make -C build test

# 或直接执行
go test -v ./...

# 运行特定包的测试
go test -v ./internal/config

# 显示覆盖率
go test -cover ./...
```

### 生成覆盖率报告

```bash
# 使用 Makefile
make -C build test-coverage

# 或手工执行
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 代码检查

### 代码格式化

```bash
# 使用 Makefile
make -C build fmt

# 或手工执行
go fmt ./...
```

### go vet 检查

```bash
# 使用 Makefile
make -C build vet

# 或手工执行
go vet ./...
```

### golangci-lint 检查（推荐）

```bash
# 使用 Makefile
make -C build lint

# 或安装并执行
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run
```

## CI/CD 集成

### GitHub Actions 示例

参考 `.github/workflows/build.yml`（如项目中存在）

或手工创建：

```yaml
name: Build

on:
  push:
    branches: [ main ]
    tags: [ 'v*' ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [linux]
        arch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      
      - name: Build
        env:
          GOOS: ${{ matrix.os }}
          GOARCH: ${{ matrix.arch }}
        run: |
          go build -o gitlab-task-hook-${{ matrix.os }}-${{ matrix.arch }} .
      
      - name: Test
        run: go test -v ./...
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: gitlab-task-hook-${{ matrix.os }}-${{ matrix.arch }}
          path: gitlab-task-hook-${{ matrix.os }}-${{ matrix.arch }}
```

## 常见问题

### Q: 编译失败，提示找不到模块

**A:** 执行 `go mod download` 下载依赖

```bash
go mod download
./build/build.sh
```

### Q: 编译后二进制体积很大

**A:** 使用优化版本

```bash
./build/build.sh --optimize
# 或
make -C build build-optimized
```

### Q: 编译出的二进制在其他机器上无法运行

**A:** 使用静态链接

```bash
./build/build.sh --static
# 或
make -C build build-static
```

### Q: 如何构建特定版本

**A:** 使用 --version 选项

```bash
./build/build.sh --version v1.0.0
# 或
make -C build build VERSION=v1.0.0
```

## 发布构建

### 创建发布版本

```bash
# 构建所有平台
./build/build.sh --all --optimize --version v1.0.0 --output /tmp/releases

# 生成 checksums
cd /tmp/releases
sha256sum gitlab-task-hook-* > checksums.txt

# 生成 tar 包
tar -czf gitlab-task-hook-v1.0.0.tar.gz gitlab-task-hook-*
```

### 验证发布版本

```bash
# 验证所有二进制文件
for bin in gitlab-task-hook-*; do
  echo "=== $bin ==="
  ./$bin version
done

# 验证 checksums
sha256sum -c checksums.txt
```

## 部署构建产物

构建完成后，将二进制文件部署到服务器：

```bash
# 使用部署脚本
sudo ../deploy/deploy.sh ./gitlab-task-hook /path/to/bootstrap.yaml

# 或手工复制
sudo cp gitlab-task-hook /var/opt/gitlab/gitaly/custom_hooks/bin/
sudo chmod 755 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
```

详见 `../deploy/DEPLOYMENT.md`。

## 相关文档

- **部署指南**: [../deploy/DEPLOYMENT.md](../deploy/DEPLOYMENT.md)
- **项目 README**: [../README.md](../README.md)
- **需求文档**: [../docs/gitlab-task-hook-nacos-yaml-config-requirements.md](../docs/gitlab-task-hook-nacos-yaml-config-requirements.md)

## 快速参考

| 任务 | 命令 |
|------|------|
| 快速构建 | `./build.sh` 或 `make build` |
| 多平台构建 | `./build.sh --all` 或 `make build-all` |
| 优化构建 | `./build.sh --optimize` 或 `make build-optimized` |
| 静态链接 | `./build.sh --static` 或 `make build-static` |
| 运行测试 | `make test` 或 `go test ./...` |
| 代码检查 | `make lint` 或 `golangci-lint run` |
| 代码格式化 | `make fmt` 或 `go fmt ./...` |
| 清理产物 | `make clean` |
| 显示帮助 | `make help` 或 `./build.sh --help` |

---

**最后更新**: 2026-06-28  
**文档版本**: 1.0
