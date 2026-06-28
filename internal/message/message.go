package message

import "fmt"

// ViolationContext carries the fields needed to render violation messages.
type ViolationContext struct {
	ProjectPath    string
	RepoName       string
	Username       string
	BranchName     string // short branch name, e.g. "dev/login"
	CommitID       string
	CommitSubject  string
	CommitterEmail string
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// ForcePush returns the force-push violation message (section 9.3).
func ForcePush(ctx ViolationContext) string {
	return fmt.Sprintf(`禁止强制推送（non fast-forward / rewrite history）

project: %s
repo: %s
user: %s
branch: %s`,
		orUnknown(ctx.ProjectPath),
		orUnknown(ctx.RepoName),
		orUnknown(ctx.Username),
		orUnknown(ctx.BranchName),
	)
}

// CommitterMismatch returns the committer-mismatch violation message (section 9.4).
func CommitterMismatch(ctx ViolationContext) string {
	return fmt.Sprintf(`提交人信息不符合规范：commit 提交人与 push 用户不一致

【规范要求】
commit 的 committer 用户必须与当前 push 的 GitLab 用户一致。

project: %s
repo: %s
push user: %s
committer email: %s
branch: %s

commitid: %s

【如何修改】
请确认本地 Git 用户信息是否与 GitLab 用户一致：

1) 查看当前 Git 用户信息
   git config user.name
   git config user.email

2) 修改当前仓库的 Git 用户信息
   git config user.name "<GitLab用户名>"
   git config user.email "<GitLab用户名>@<your-domain>"

3) 修正最近一次提交的 committer 信息
   git commit --amend --reset-author

4) 如果需要修改多个历史提交，请使用 interactive rebase：
   git rebase -i HEAD~5
   # 将需要修改的 commit 标记为 edit
   # 逐个执行：
   git commit --amend --reset-author
   git rebase --continue

5) 推送修正后的提交
   git push --force-with-lease`,
		orUnknown(ctx.ProjectPath),
		orUnknown(ctx.RepoName),
		orUnknown(ctx.Username),
		orUnknown(ctx.CommitterEmail),
		orUnknown(ctx.BranchName),
		orUnknown(ctx.CommitID),
	)
}

// MissingTaskID returns the missing-task-ID violation message (section 9.5).
func MissingTaskID(ctx ViolationContext) string {
	return fmt.Sprintf(`提交信息不符合规范：缺少任务ID（[#TSK-...] 或 [#DEF-...]）

【规范要求】commit subject 必须包含任务ID（二选一）
  [#TSK-xxx]  或  [#DEF-xxx]

project: %s
repo: %s
user: %s
branch: %s

commitid: %s
commit: %s

【如何修改】
以下操作可能涉及"重写历史（rewrite history）"。为避免代码丢失，请先确认：
1) 工作区干净（working tree clean）：git status
2) 建一个备份分支（backup branch）：git branch backup/<name> HEAD
   如误操作，可用 git reflog 找回提交

A) 只修改最近一次提交（amend）
   git commit --amend -m "<原提交信息> [#TSK-xxx]"
   # 或
   git commit --amend -m "<原提交信息> [#DEF-xxx]"
   git push --force-with-lease

B) 修改多次提交（interactive rebase / reword）
   方式1：按数量修改（示例：最近 5 个提交）
   git rebase -i HEAD~5
   # 将需要修改的提交从 pick 改为 reword，保存退出后按提示逐个修改
   git push --force-with-lease

   方式2：从分叉点开始（更稳，适合不知道数量）
   base=$(git merge-base origin/<target-branch> HEAD)
   git rebase -i "$base"
   git push --force-with-lease`,
		orUnknown(ctx.ProjectPath),
		orUnknown(ctx.RepoName),
		orUnknown(ctx.Username),
		orUnknown(ctx.BranchName),
		orUnknown(ctx.CommitID),
		orUnknown(ctx.CommitSubject),
	)
}
