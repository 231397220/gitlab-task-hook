# GitLab Task Hook 构建指南

## 快速开始

```bash
# 编译二进制文件
go build -o gitlab-task-hook .

# 验证
./gitlab-task-hook version
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

## 构建方法

### 方法 1：简单构建（推荐）

```bash
# 在项目根目录执行
go build -o gitlab-task-hook .
```

输出：当前目录下生成 `gitlab-task-hook` 二进制文件

### 方法 2：指定输出路径

```bash
go build -o /path/to/output/gitlab-task-hook .
```

### 方法 3：跨平台构建

```bash
# 编译为 Linux amd64 版本
GOOS=linux GOARCH=amd64 go build -o gitlab-task-hook-linux-amd64 .

# 编译为 Linux arm64 版本
GOOS=linux GOARCH=arm64 go build -o gitlab-task-hook-linux-arm64 .
```

### 方法 4：包含版本信息的构建

```bash
# 获取版本号和 git commit
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')

# 使用 ldflags 注入版本信息
go build \
  -ldflags "-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildTime=${BUILD_TIME}'" \
  -o gitlab-task-hook \
  .
```

## 构建优化

### 减小二进制大小

```bash
# 移除调试符号
go build -ldflags="-s -w" -o gitlab-task-hook .

# 并使用 upx 进一步压缩（如已安装）
upx gitlab-task-hook
```

### 静态链接（避免依赖 glibc 版本）

```bash
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
# 检查依赖库
ldd ./gitlab-task-hook

# 检查是否为可执行文件
file ./gitlab-task-hook

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

## 持续构建

### GitHub Actions 示例

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

## 本地开发和测试

### 运行单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/config

# 显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 代码格式化

```bash
# 格式化所有 Go 文件
go fmt ./...

# 使用 goimports 组织导入
goimports -w .
```

### 代码检查

```bash
# 使用 go vet 检查
go vet ./...

# 使用 golint 检查代码风格
golint ./...

# 使用 golangci-lint（推荐）
golangci-lint run
```

## 常见问题

### Q: 编译失败，提示找不到模块

**A:** 执行 `go mod download` 下载依赖

```bash
go mod download
go build -o gitlab-task-hook .
```

### Q: 编译后二进制体积很大

**A:** 使用以下方式减小大小

```bash
# 移除调试符号
go build -ldflags="-s -w" -o gitlab-task-hook .

# 或使用 upx 压缩
upx --brute gitlab-task-hook
```

### Q: 编译出的二进制在其他机器上无法运行

**A:** 可能是 glibc 版本差异，使用静态链接

```bash
CGO_ENABLED=0 go build -o gitlab-task-hook .
```

### Q: 需要跨平台编译

**A:** 使用环境变量指定目标平台

```bash
GOOS=linux GOARCH=amd64 go build -o gitlab-task-hook .
GOOS=linux GOARCH=arm64 go build -o gitlab-task-hook-arm64 .
```

## 发布构建

### 创建发布版本

```bash
# 设置版本号
VERSION="v1.0.0"

# 编译多个平台的二进制
for ARCH in amd64 arm64; do
  GOOS=linux GOARCH=$ARCH go build \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o gitlab-task-hook-linux-$ARCH \
    .
done

# 创建 tar 包
tar -czf gitlab-task-hook-${VERSION}.tar.gz gitlab-task-hook-linux-*

# 生成 checksums
sha256sum gitlab-task-hook-linux-* > checksums.txt
```

### 检查清单

```bash
# 验证二进制信息
./gitlab-task-hook-linux-amd64 version

# 验证依赖
ldd ./gitlab-task-hook-linux-amd64

# 测试基本功能
echo "" | ./gitlab-task-hook-linux-amd64 hook
```

## 部署构建产物

构建完成后，将二进制文件部署到服务器：

```bash
# 使用部署脚本
sudo ./deploy/deploy.sh ./gitlab-task-hook /path/to/bootstrap.yaml

# 或手工复制
sudo cp gitlab-task-hook /var/opt/gitlab/gitaly/custom_hooks/bin/
sudo chmod 755 /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
```

详见 `deploy/DEPLOYMENT.md`。

## 相关文档

- **部署指南**: [deploy/DEPLOYMENT.md](./deploy/DEPLOYMENT.md)
- **项目 README**: [README.md](./README.md)
- **需求文档**: [docs/gitlab-task-hook-nacos-yaml-config-requirements.md](./docs/gitlab-task-hook-nacos-yaml-config-requirements.md)

---

**最后更新**: 2026-06-28  
**文档版本**: 1.0
