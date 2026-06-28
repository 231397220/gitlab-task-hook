#!/bin/sh
# Integration test script for gitlab-task-hook.
# Creates a temporary bare git repo, installs the hook binary, and tests all 12 scenarios.
# Usage:
#   ./scripts/integration_test.sh [path/to/gitlab-task-hook]
#
# If no binary path is given it assumes ./gitlab-task-hook in the current directory.

set -e

BINARY="${1:-./gitlab-task-hook}"
if [ ! -x "$BINARY" ]; then
  echo "ERROR: binary not found or not executable: $BINARY"
  exit 1
fi

PASS=0
FAIL=0

# ---- helpers ----

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

bare="$tmpdir/bare.git"
work="$tmpdir/work"

# Initialise a bare repo and a working clone.
git init --bare "$bare" >/dev/null 2>&1
git clone "$bare" "$work" >/dev/null 2>&1

# Configure a committer identity in the working copy.
git -C "$work" config user.name  "zhangsan"
git -C "$work" config user.email "zhangsan@example.com"

# Helper: make a commit in the working copy.
make_commit() {
  msg="$1"
  echo "$msg" >> "$work/file.txt"
  git -C "$work" add file.txt >/dev/null 2>&1
  git -C "$work" commit -m "$msg" >/dev/null 2>&1
}

# Helper: push from working copy to bare repo, injecting the hook.
# Returns the exit code of the hook.
run_hook() {
  ref="$1"
  old="$2"
  new="$3"
  shift 3
  # Remaining args are env vars: KEY=VALUE ...
  env_args=""
  for kv in "$@"; do
    env_args="$env_args $kv"
  done

  # Run the hook binary with stdin in the bare repo directory.
  # We must cd into the bare repo because git commands inside the hook need a repo context.
  (cd "$bare" && echo "$old $new $ref" | env $env_args "$BINARY")
  return $?
}

assert_exit() {
  label="$1"
  expected="$2"
  actual="$3"
  if [ "$actual" = "$expected" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: $label"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: $label (expected exit $expected, got $actual)"
  fi
}

# ---- setup: push an initial commit so the bare repo has history ----

make_commit "initial [#TSK-0000]"
git -C "$work" push origin master >/dev/null 2>&1 || git -C "$work" push origin main >/dev/null 2>&1 || true

# Determine the default branch name.
DEFAULT_BRANCH=$(git -C "$bare" symbolic-ref HEAD 2>/dev/null | sed 's|refs/heads/||')
ZERO="0000000000000000000000000000000000000000"
HEAD=$(git -C "$bare" rev-parse HEAD)

echo ""
echo "=== gitlab-task-hook integration tests ==="
echo ""

# ---- Scenario 1: compliant push succeeds ----
echo "Scenario 1: compliant push (enforce)"
make_commit "add login api [#TSK-1001]"
NEW1=$(git -C "$work" rev-parse HEAD)
RESULT=0
(cd "$bare" && echo "$HEAD $NEW1 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo-service GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "compliant push → exit 0" 0 $RESULT
git -C "$work" reset --hard HEAD~ >/dev/null 2>&1

# ---- Scenario 2: missing task ID, enforce mode ----
echo ""
echo "Scenario 2: missing task ID (enforce)"
make_commit "add login api without task id"
NEW2=$(git -C "$work" rev-parse HEAD)
RESULT=0
(cd "$bare" && echo "$HEAD $NEW2 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo-service GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "missing task ID in enforce → exit 1" 1 $RESULT
git -C "$work" reset --hard HEAD~ >/dev/null 2>&1

# ---- Scenario 3: missing task ID, warn mode ----
echo ""
echo "Scenario 3: missing task ID (warn mode)"
make_commit "add login api without task id"
NEW3=$(git -C "$work" rev-parse HEAD)
RESULT=0
(cd "$bare" && echo "$HEAD $NEW3 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo-service GL_PROTOCOL=http HOOK_MODE=warn \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "missing task ID in warn → exit 0" 0 $RESULT
git -C "$work" reset --hard HEAD~ >/dev/null 2>&1

# ---- Scenario 4: root user skips all checks ----
echo ""
echo "Scenario 4: root user skips all checks"
make_commit "no task id, root push"
NEW4=$(git -C "$work" rev-parse HEAD)
RESULT=0
(cd "$bare" && echo "$HEAD $NEW4 refs/heads/dev/login" | \
  GL_USERNAME=root GL_PROJECT_PATH=group/demo GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "root user → exit 0" 0 $RESULT
git -C "$work" reset --hard HEAD~ >/dev/null 2>&1

# ---- Scenario 5: tag ref is allowed ----
echo ""
echo "Scenario 5: tag ref passes"
RESULT=0
(cd "$bare" && echo "$ZERO $HEAD refs/tags/v1.0.0" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "tag ref → exit 0" 0 $RESULT

# ---- Scenario 6: delete branch is allowed ----
echo ""
echo "Scenario 6: delete branch passes"
RESULT=0
(cd "$bare" && echo "$HEAD $ZERO refs/heads/feature/old" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "delete branch → exit 0" 0 $RESULT

# ---- Scenario 7: non fast-forward is rejected ----
echo ""
echo "Scenario 7: non fast-forward rejected"
# Make a diverging commit in a separate branch
git -C "$work" checkout -b tmp-diverge >/dev/null 2>&1
make_commit "diverge [#TSK-9999]"
NEW7=$(git -C "$work" rev-parse HEAD)
git -C "$work" checkout - >/dev/null 2>&1
# Use HEAD as old (doesn't descend from NEW7), so merge-base ≠ HEAD
RESULT=0
(cd "$bare" && echo "$NEW7 $HEAD refs/heads/dev/x" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "non fast-forward → exit 1" 1 $RESULT
git -C "$work" branch -D tmp-diverge >/dev/null 2>&1

# ---- Scenario 8: GL_PROTOCOL=web skips push-class checks ----
echo ""
echo "Scenario 8: GL_PROTOCOL=web skips task check"
make_commit "no task id, web merge"
NEW8=$(git -C "$work" rev-parse HEAD)
RESULT=0
(cd "$bare" && echo "$HEAD $NEW8 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo GL_PROTOCOL=web HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "web protocol → exit 0" 0 $RESULT
git -C "$work" reset --hard HEAD~ >/dev/null 2>&1

# ---- Scenario 9: project whitelist skips task check ----
echo ""
echo "Scenario 9: project whitelist skips task check"
make_commit "no task id, whitelisted project"
NEW9=$(git -C "$work" rev-parse HEAD)
RESULT=0
(cd "$bare" && echo "$HEAD $NEW9 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/legacy-service GL_PROTOCOL=http HOOK_MODE=enforce \
  WHITELIST_PROJECT_NAMES=legacy-service \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "project whitelist → exit 0" 0 $RESULT
git -C "$work" reset --hard HEAD~ >/dev/null 2>&1

# ---- Scenario 10: committer mismatch is rejected ----
echo ""
echo "Scenario 10: committer mismatch rejected"
# Temporarily override committer identity
git -C "$work" config user.email "lisi@example.com"
make_commit "add something [#TSK-2001]"
git -C "$work" config user.email "zhangsan@example.com"
NEW10=$(git -C "$work" rev-parse HEAD)
RESULT=0
(cd "$bare" && echo "$HEAD $NEW10 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "committer mismatch → exit 1" 1 $RESULT
git -C "$work" reset --hard HEAD~ >/dev/null 2>&1

# ---- Scenario 11: merge commit is exempted ----
echo ""
echo "Scenario 11: merge commit exempted"
# Create a feature branch and merge it back (without fast-forward to force a merge commit)
git -C "$work" checkout -b feature-test >/dev/null 2>&1
make_commit "feature work [#TSK-3001]"
git -C "$work" checkout - >/dev/null 2>&1
make_commit "parallel work [#TSK-3002]"
git -C "$work" merge --no-ff feature-test -m "Merge feature-test" >/dev/null 2>&1
NEW11=$(git -C "$work" rev-parse HEAD)
RESULT=0
# We push the merge commit; it has no task id in its subject, but should be exempt.
(cd "$bare" && echo "$HEAD $NEW11 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY") 2>/dev/null; RESULT=$?
assert_exit "merge commit exempted → exit 0" 0 $RESULT
git -C "$work" branch -D feature-test >/dev/null 2>&1
git -C "$work" reset --hard HEAD~3 >/dev/null 2>&1

# ---- Scenario 12: multiple commits, only one violation message ----
echo ""
echo "Scenario 12: multiple commits, only first violation reported"
make_commit "good commit [#TSK-4001]"
make_commit "bad commit no task"
make_commit "another bad commit"
NEW12=$(git -C "$work" rev-parse HEAD)
OUTPUT=$(cd "$bare" && echo "$HEAD $NEW12 refs/heads/dev/login" | \
  GL_USERNAME=zhangsan GL_PROJECT_PATH=group/demo GL_PROTOCOL=http HOOK_MODE=enforce \
  "$BINARY" 2>&1) || true
COUNT=$(echo "$OUTPUT" | grep -c "提交信息不符合规范" || true)
if [ "$COUNT" -eq 1 ]; then
  PASS=$((PASS + 1))
  echo "  PASS: only one violation message (count=$COUNT)"
else
  FAIL=$((FAIL + 1))
  echo "  FAIL: expected exactly 1 violation message, got $COUNT"
fi
git -C "$work" reset --hard HEAD~3 >/dev/null 2>&1

# ---- Summary ----
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
