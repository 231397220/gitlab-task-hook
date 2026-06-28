#!/bin/sh
# Deploy gitlab-task-hook to a GitLab/Gitaly server.
# Usage: ./scripts/deploy.sh <binary-path> <version>
# Example: ./scripts/deploy.sh ./gitlab-task-hook v1.0.0

set -e

BINARY="${1:?binary path required (e.g. ./gitlab-task-hook)}"
VERSION="${2:?version required (e.g. v1.0.0)}"

HOOKS_BASE="/var/opt/gitlab/gitaly/custom_hooks"
BIN_DIR="$HOOKS_BASE/bin"
PRE_RECV_DIR="$HOOKS_BASE/pre-receive.d"
VERSIONED="$BIN_DIR/gitlab-task-hook-$VERSION"
SYMLINK="$BIN_DIR/gitlab-task-hook"
WRAPPER="$PRE_RECV_DIR/01-task-id-check"

# 1. Install binary with version suffix
mkdir -p "$BIN_DIR"
cp "$BINARY" "$VERSIONED"
chown git:git "$VERSIONED"
chmod 755 "$VERSIONED"

# 2. Update symlink
ln -sf "gitlab-task-hook-$VERSION" "$SYMLINK"

# 3. Create wrapper (idempotent)
mkdir -p "$PRE_RECV_DIR"
cat > "$WRAPPER" <<'WRAPPER_EOF'
#!/bin/sh
exec /var/opt/gitlab/gitaly/custom_hooks/bin/gitlab-task-hook
WRAPPER_EOF
chown git:git "$WRAPPER"
chmod 755 "$WRAPPER"

echo "Deployed: $VERSIONED"
echo "Symlink:  $SYMLINK -> gitlab-task-hook-$VERSION"
echo "Wrapper:  $WRAPPER"
echo ""
echo "Verify: $SYMLINK --version"
