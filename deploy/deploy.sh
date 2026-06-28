#!/bin/bash
#
# GitLab Task Hook 快速部署脚本
#
# 用法：
#   ./deploy.sh <binary_path> [bootstrap_yaml_path]
#
# 示例：
#   ./deploy.sh ./gitlab-task-hook /etc/gitlab-task-hook/bootstrap.yaml
#   ./deploy.sh ./gitlab-task-hook  # 使用默认 bootstrap 路径
#
# 本脚本执行以下步骤：
# 1. 检查必要的文件和权限
# 2. 创建目录结构
# 3. 复制二进制文件
# 4. 复制 bootstrap 配置（如提供）
# 5. 配置 systemd service
# 6. 首次拉取 Nacos 配置
# 7. 启动 config-sync 服务
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# 配置项
BINARY_PATH="${1:-.}"
BOOTSTRAP_PATH="${2:-/etc/gitlab-task-hook/bootstrap.yaml}"
HOOK_BIN_DIR="/var/opt/gitlab/gitaly/custom_hooks/bin"
HOOK_CONFIG_DIR="/var/opt/gitlab/gitaly/custom_hooks/config"
HOOK_HOOKS_DIR="/var/opt/gitlab/gitaly/custom_hooks/hooks"
HOOK_LOG_DIR="/var/log/gitlab-task-hook"
CONFIG_BOOTSTRAP_DIR="/etc/gitlab-task-hook"
SYSTEMD_SERVICE_DIR="/etc/systemd/system"
SYSTEMD_SERVICE_FILE="gitlab-task-hook-config-sync.service"

# 检查是否以 root 身份运行
if [ "$EUID" -ne 0 ]; then
    log_error "此脚本必须以 root 身份运行，请使用 sudo"
    exit 1
fi

log_info "开始部署 GitLab Task Hook..."

# 第一步：检查二进制文件
if [ ! -f "$BINARY_PATH/gitlab-task-hook" ]; then
    log_error "gitlab-task-hook 二进制文件不存在：$BINARY_PATH/gitlab-task-hook"
    exit 1
fi

log_info "✓ 找到 gitlab-task-hook 二进制文件"

# 第二步：创建目录结构
log_info "创建目录结构..."
mkdir -p "$HOOK_BIN_DIR"
mkdir -p "$HOOK_CONFIG_DIR"
mkdir -p "$HOOK_HOOKS_DIR"
mkdir -p "$HOOK_LOG_DIR"
mkdir -p "$CONFIG_BOOTSTRAP_DIR"

# 第三步：设置目录权限（git 用户和组）
log_info "设置目录权限..."
chown -R git:git "$HOOK_BIN_DIR"
chown -R git:git "$HOOK_CONFIG_DIR"
chown -R git:git "$HOOK_HOOKS_DIR"
chown -R git:git "$HOOK_LOG_DIR"
chown -R git:git "$CONFIG_BOOTSTRAP_DIR"

chmod 750 "$HOOK_BIN_DIR"
chmod 750 "$HOOK_CONFIG_DIR"
chmod 750 "$HOOK_HOOKS_DIR"
chmod 750 "$HOOK_LOG_DIR"

# 第四步：复制二进制文件
log_info "部署二进制文件..."
cp "$BINARY_PATH/gitlab-task-hook" "$HOOK_BIN_DIR/gitlab-task-hook"
chmod 755 "$HOOK_BIN_DIR/gitlab-task-hook"
log_info "✓ 二进制文件已复制到 $HOOK_BIN_DIR"

# 第五步：检查 bootstrap.yaml
if [ ! -f "$BOOTSTRAP_PATH" ]; then
    log_warn "bootstrap.yaml 不存在：$BOOTSTRAP_PATH"
    log_info "请手工创建 bootstrap.yaml 配置文件"
    log_info "参考：deploy/bootstrap.yaml.example"
    read -p "继续部署？(y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_error "部署已取消"
        exit 1
    fi
else
    log_info "✓ 找到 bootstrap.yaml 配置文件"
    # 验证 bootstrap.yaml 格式
    if ! "$HOOK_BIN_DIR/gitlab-task-hook" config-validate --file "$BOOTSTRAP_PATH" &>/dev/null; then
        log_error "bootstrap.yaml 格式错误，请检查"
        exit 1
    fi
fi

# 第六步：配置 systemd service
log_info "配置 systemd service..."

# 检查 service 文件是否在当前目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_TEMPLATE="$SCRIPT_DIR/$SYSTEMD_SERVICE_FILE"

if [ -f "$SERVICE_TEMPLATE" ]; then
    cp "$SERVICE_TEMPLATE" "$SYSTEMD_SERVICE_DIR/$SYSTEMD_SERVICE_FILE"
    log_info "✓ systemd service 文件已配置"
else
    log_warn "未找到 systemd service 模板：$SERVICE_TEMPLATE"
    log_warn "请手工复制 deploy/$SYSTEMD_SERVICE_FILE 到 /etc/systemd/system/"
fi

# 第七步：初始化本地缓存
if [ -f "$BOOTSTRAP_PATH" ]; then
    log_info "从 Nacos 拉取初始配置..."
    if sudo -u git "$HOOK_BIN_DIR/gitlab-task-hook" \
        config-sync --once --bootstrap "$BOOTSTRAP_PATH"; then
        log_info "✓ 本地缓存初始化成功"
    else
        log_error "从 Nacos 拉取配置失败，请检查 bootstrap.yaml 和 Nacos 连接"
        log_warn "继续部署，但 hook 可能无法正常工作，直到配置成功拉取"
    fi
fi

# 第八步：部署 hook wrapper
log_info "配置 GitLab hook wrapper..."

HOOK_WRAPPER="$HOOK_HOOKS_DIR/pre-receive"
HOOK_TEMPLATE="$SCRIPT_DIR/pre-receive.example"

if [ -f "$HOOK_TEMPLATE" ]; then
    cp "$HOOK_TEMPLATE" "$HOOK_WRAPPER"
    chmod 755 "$HOOK_WRAPPER"
    chown git:git "$HOOK_WRAPPER"
    log_info "✓ hook wrapper 已配置"
else
    log_warn "未找到 hook wrapper 模板：$HOOK_TEMPLATE"
    log_warn "请手工创建 $HOOK_WRAPPER 脚本"
fi

# 第九步：启动 systemd service
log_info "启动 config-sync 服务..."

if [ -f "$SYSTEMD_SERVICE_DIR/$SYSTEMD_SERVICE_FILE" ]; then
    systemctl daemon-reload
    systemctl enable "$SYSTEMD_SERVICE_FILE"
    systemctl restart "$SYSTEMD_SERVICE_FILE"

    # 等待服务启动
    sleep 2

    if systemctl is-active --quiet "$SYSTEMD_SERVICE_FILE"; then
        log_info "✓ config-sync 服务已启动"
    else
        log_error "config-sync 服务启动失败"
        log_info "请查看日志：journalctl -u $SYSTEMD_SERVICE_FILE -n 50"
        exit 1
    fi
else
    log_warn "systemd service 文件不存在，跳过自动启动"
    log_info "请手工执行：sudo systemctl start $SYSTEMD_SERVICE_FILE"
fi

# 第十步：验证部署
log_info "验证部署..."

# 检查二进制文件
if [ -x "$HOOK_BIN_DIR/gitlab-task-hook" ]; then
    VERSION=$("$HOOK_BIN_DIR/gitlab-task-hook" version 2>/dev/null || echo "unknown")
    log_info "✓ gitlab-task-hook 版本：$VERSION"
else
    log_error "gitlab-task-hook 二进制文件不可执行"
    exit 1
fi

# 检查本地缓存
if [ -f "$HOOK_CONFIG_DIR/gitlab-task-hook.yaml" ]; then
    log_info "✓ 本地缓存文件存在"
else
    log_warn "⚠ 本地缓存文件不存在，hook 将使用内置默认配置"
fi

# 检查 hook wrapper
if [ -x "$HOOK_WRAPPER" ]; then
    log_info "✓ hook wrapper 已配置"
else
    log_warn "⚠ hook wrapper 未正确配置"
fi

# 总结
echo ""
log_info "================================"
log_info "部署完成！"
log_info "================================"
echo ""

echo "关键文件和目录："
echo "  二进制：      $HOOK_BIN_DIR/gitlab-task-hook"
echo "  Bootstrap：   $BOOTSTRAP_PATH"
echo "  本地缓存：    $HOOK_CONFIG_DIR/gitlab-task-hook.yaml"
echo "  Hook Wrapper: $HOOK_WRAPPER"
echo "  日志文件：    $HOOK_LOG_DIR/config-sync.log"
echo ""

echo "后续步骤："
echo "  1. 验证 config-sync 服务状态："
echo "     systemctl status gitlab-task-hook-config-sync.service"
echo ""
echo "  2. 查看实时日志："
echo "     journalctl -u gitlab-task-hook-config-sync.service -f"
echo ""
echo "  3. 在 Git 仓库中测试 push："
echo "     git push origin feature/TEST-001"
echo ""
echo "  4. 查看完整部署文档："
echo "     less deploy/DEPLOYMENT.md"
echo ""

log_info "部署成功！"
