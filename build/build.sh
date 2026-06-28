#!/bin/bash
#
# GitLab Task Hook 构建脚本
#
# 用法：
#   ./build.sh                          # 构建到当前目录
#   ./build.sh --output /path/to/dir    # 构建到指定目录
#   ./build.sh --version v1.0.0         # 注入版本号
#   ./build.sh --all                    # 构建所有平台
#   ./build.sh --optimize               # 构建优化版本（减小大小）
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置项
VERSION=""
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')
OUTPUT_DIR="."
BUILD_ALL=false
OPTIMIZE=false
BUILD_STATIC=false
PLATFORMS=()

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 打印帮助信息
print_help() {
    cat << 'EOF'
GitLab Task Hook 构建脚本

用法：
  ./build.sh [选项]

选项：
  --output <path>         输出目录（默认：当前目录）
  --version <version>     版本号，如 v1.0.0（默认：无）
  --all                   构建所有平台（linux-amd64, linux-arm64）
  --optimize              构建优化版本（移除调试符号，减小大小）
  --static                静态链接（避免 glibc 依赖）
  --help                  显示帮助信息

示例：
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

EOF
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --all)
                BUILD_ALL=true
                shift
                ;;
            --optimize)
                OPTIMIZE=true
                shift
                ;;
            --static)
                BUILD_STATIC=true
                shift
                ;;
            --help)
                print_help
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                print_help
                exit 1
                ;;
        esac
    done
}

# 检查前置条件
check_prerequisites() {
    log_info "检查前置条件..."

    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.16+"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_debug "Go 版本: $GO_VERSION"

    # 检查是否在项目根目录
    if [ ! -f "go.mod" ]; then
        log_error "go.mod 不存在，请在项目根目录执行此脚本"
        exit 1
    fi

    # 检查输出目录
    if [ ! -d "$OUTPUT_DIR" ]; then
        log_info "创建输出目录: $OUTPUT_DIR"
        mkdir -p "$OUTPUT_DIR"
    fi

    log_info "✓ 前置条件检查通过"
}

# 获取版本信息
get_version() {
    if [ -z "$VERSION" ]; then
        # 尝试从 git tag 获取
        if [ -d ".git" ]; then
            VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
        else
            VERSION="dev"
        fi
    fi
    log_debug "版本号: $VERSION"
}

# 构建二进制文件
build_binary() {
    local goos=$1
    local goarch=$2
    local output_file=$3

    log_info "编译: $goos/$goarch -> $output_file"

    # 构建 ldflags
    local ldflags="-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildTime=${BUILD_TIME}'"

    # 如果启用优化，移除调试符号
    if [ "$OPTIMIZE" = true ]; then
        ldflags="$ldflags -s -w"
        log_debug "启用优化（移除调试符号）"
    fi

    # 构建环境变量
    local env_vars="GOOS=$goos GOARCH=$goarch"

    if [ "$BUILD_STATIC" = true ]; then
        env_vars="$env_vars CGO_ENABLED=0"
        log_debug "启用静态链接"
    fi

    # 执行构建
    eval "$env_vars go build -ldflags=\"$ldflags\" -o \"$output_file\" ." || {
        log_error "构建失败: $output_file"
        return 1
    }

    # 验证输出文件
    if [ ! -f "$output_file" ]; then
        log_error "输出文件不存在: $output_file"
        return 1
    fi

    # 显示文件信息
    local file_size=$(ls -lh "$output_file" | awk '{print $5}')
    log_info "✓ 编译成功: $output_file ($file_size)"

    return 0
}

# 获取当前平台信息
get_current_platform() {
    local os=$(go env GOOS)
    local arch=$(go env GOARCH)
    echo "${os}-${arch}"
}

# 验证二进制文件
verify_binary() {
    local binary=$1

    if [ ! -x "$binary" ]; then
        log_error "二进制文件不可执行: $binary"
        return 1
    fi

    # 检查文件类型
    local file_type=$(file "$binary")
    log_debug "文件类型: $file_type"

    # 尝试运行 version 命令
    if ! "$binary" version &>/dev/null; then
        log_warn "⚠ 无法运行二进制文件的 version 命令"
        # 继续，不失败
    else
        log_info "✓ 二进制文件验证通过"
    fi

    return 0
}

# 主构建流程
main() {
    parse_args "$@"

    log_info "================================"
    log_info "GitLab Task Hook 构建脚本"
    log_info "================================"

    check_prerequisites
    get_version

    # 确定构建平台
    if [ "$BUILD_ALL" = true ]; then
        PLATFORMS=("linux:amd64" "linux:arm64")
        log_info "构建模式: 多平台"
    else
        local platform=$(get_current_platform)
        PLATFORMS=("$platform")
        log_info "构建模式: 当前平台 ($platform)"
    fi

    log_info ""
    log_info "构建配置:"
    echo "  版本号:       $VERSION"
    echo "  Commit:       $COMMIT"
    echo "  构建时间:     $BUILD_TIME"
    echo "  输出目录:     $OUTPUT_DIR"
    echo "  优化:         $([ "$OPTIMIZE" = true ] && echo "启用" || echo "禁用")"
    echo "  静态链接:     $([ "$BUILD_STATIC" = true ] && echo "启用" || echo "禁用")"
    echo "  构建平台数:   ${#PLATFORMS[@]}"
    log_info ""

    # 执行构建
    local build_count=0
    for platform in "${PLATFORMS[@]}"; do
        IFS=':' read -r goos goarch <<< "$platform"

        local binary_name="gitlab-task-hook"
        if [ "$BUILD_ALL" = true ]; then
            binary_name="gitlab-task-hook-${goos}-${goarch}"
        fi

        local output_file="${OUTPUT_DIR}/${binary_name}"

        if build_binary "$goos" "$goarch" "$output_file"; then
            verify_binary "$output_file"
            build_count=$((build_count + 1))
        else
            log_error "构建失败，中止"
            exit 1
        fi
    done

    log_info ""
    log_info "================================"
    log_info "构建完成！"
    log_info "================================"
    log_info "构建成功: $build_count 个二进制文件"
    log_info "输出目录: $OUTPUT_DIR"
    log_info ""

    # 列出输出文件
    log_info "输出文件:"
    if [ "$BUILD_ALL" = true ]; then
        for platform in "${PLATFORMS[@]}"; do
            IFS=':' read -r goos goarch <<< "$platform"
            local binary_name="gitlab-task-hook-${goos}-${goarch}"
            local output_file="${OUTPUT_DIR}/${binary_name}"
            if [ -f "$output_file" ]; then
                local size=$(ls -lh "$output_file" | awk '{print $5}')
                echo "  $binary_name ($size)"
            fi
        done
    else
        local output_file="${OUTPUT_DIR}/gitlab-task-hook"
        if [ -f "$output_file" ]; then
            local size=$(ls -lh "$output_file" | awk '{print $5}')
            echo "  gitlab-task-hook ($size)"
        fi
    fi

    log_info ""
    log_info "后续步骤:"
    echo "  1. 验证二进制文件:"
    echo "     $OUTPUT_DIR/gitlab-task-hook version"
    echo ""
    echo "  2. 部署到服务器:"
    echo "     sudo cp $OUTPUT_DIR/gitlab-task-hook /var/opt/gitlab/gitaly/custom_hooks/bin/"
    echo ""
    echo "  3. 参考部署文档:"
    echo "     less deploy/DEPLOYMENT.md"
    log_info ""
}

# 运行主函数
main "$@"
