package message

import (
	"strings"
	"text/template"
)

// ViolationContext carries the fields needed to render violation messages.
type ViolationContext struct {
	ProjectPath    string
	RepoName       string
	Username       string
	BranchName     string
	Protocol       string // GL_PROTOCOL, used in direct-push-denied message
	CommitID       string
	CommitSubject  string
	CommitterEmail string
}

// Default template strings — used when no override is configured in YAML.
const (
	DefaultNonFastForwardTemplate = `禁止强制推送（non fast-forward / rewrite history）

project: {{.ProjectPath}}
repo:    {{.RepoName}}
user:    {{.Username}}
branch:  {{.BranchName}}

【如何处理】
您的推送被拒绝，因为本地历史与远端不一致（可能是执行了 rebase 或 reset）。

1) 拉取远端最新代码并将本地提交追加在后面：
   git pull --rebase origin {{.BranchName}}

2) 如果出现冲突，解决冲突后继续：
   # 编辑冲突文件，解决后执行：
   git add <冲突文件>
   git rebase --continue

3) 冲突全部解决后，正常推送：
   git push origin {{.BranchName}}

注意：请勿使用 git push --force，这会覆盖其他人的提交。
如您确认需要强制推送（例如整理个人开发分支），请联系管理员处理。`

	DefaultDirectPushDeniedTemplate = `当前分支禁止直接 push，请通过 Merge Request 合并代码。

【规范要求】
该分支属于受保护分支，只允许通过 GitLab Merge Request 合并，
不允许本地直接 git push。

project:  {{.ProjectPath}}
repo:     {{.RepoName}}
user:     {{.Username}}
branch:   {{.BranchName}}
protocol: {{.Protocol}}

【如何处理】
1) 从目标分支创建新的功能分支：
   git checkout -b feature/<你的功能名> origin/{{.BranchName}}

2) 在新分支上完成开发，然后推送功能分支：
   git push origin feature/<你的功能名>

3) 登录 GitLab，发起 Merge Request，目标分支选择：
   {{.BranchName}}

4) 等待审核通过后，由 GitLab 自动合并。`

	DefaultCommitterMismatchTemplate = `提交人信息不符合规范：commit 提交人与 push 用户不一致

【规范要求】
commit 的 committer 必须与当前 push 的 GitLab 用户一致。

project:         {{.ProjectPath}}
repo:            {{.RepoName}}
push user:       {{.Username}}
committer email: {{.CommitterEmail}}
branch:          {{.BranchName}}
commitid:        {{.CommitID}}

【如何修改】
请确认本地 Git 用户信息是否与 GitLab 用户一致：

1) 查看当前本地 Git 用户配置：
   git config user.name
   git config user.email

2) 将本地用户改为与 GitLab 一致：
   git config user.name "<GitLab用户名>"
   git config user.email "<GitLab用户名>@<公司域名>"

3) 修正最近一次提交的 committer：
   git commit --amend --reset-author --no-edit

4) 如需修改多次历史提交：
   git rebase -i HEAD~5
   # 将需要修改的 commit 标记为 edit，逐个执行：
   git commit --amend --reset-author --no-edit
   git rebase --continue

5) 推送修正后的提交：
   git push --force-with-lease`

	DefaultTaskIDMissingTemplate = `提交信息不符合规范：缺少任务ID（[#TSK-...] 或 [#DEF-...]）

【规范要求】commit subject 必须包含任务ID（二选一）
  [#TSK-xxx]  或  [#DEF-xxx]

project:  {{.ProjectPath}}
repo:     {{.RepoName}}
user:     {{.Username}}
branch:   {{.BranchName}}
commitid: {{.CommitID}}
commit:   {{.CommitSubject}}

【如何修改】
以下操作会重写提交历史，请先做好保护：
1) 确认工作区干净：
   git status
2) 创建备份分支（出错可恢复）：
   git branch backup/before-fix HEAD

A) 只修改最近一次提交：
   git commit --amend -m "<原提交信息> [#TSK-xxx]"
   git push --force-with-lease

B) 修改多次历史提交（rebase 方式）：
   方式1：修改最近 N 次提交（把 5 替换为实际数量）
   git rebase -i HEAD~5
   # 将需要修改的行从 pick 改为 reword，保存后按提示逐个修改

   方式2：从分叉点开始（推荐，不用数数量）
   base=$(git merge-base origin/{{.BranchName}} HEAD)
   git rebase -i "$base"

   完成后推送：
   git push --force-with-lease`
)

// Pre-compiled default templates — parsed once at startup.
var (
	defaultNonFastForwardTmpl    = template.Must(template.New("non_fast_forward").Parse(DefaultNonFastForwardTemplate))
	defaultDirectPushDeniedTmpl  = template.Must(template.New("direct_push_denied").Parse(DefaultDirectPushDeniedTemplate))
	defaultCommitterMismatchTmpl = template.Must(template.New("committer_mismatch").Parse(DefaultCommitterMismatchTemplate))
	defaultTaskIDMissingTmpl     = template.Must(template.New("task_id_missing").Parse(DefaultTaskIDMissingTemplate))
)

// NonFastForwardMessage renders the force-push violation message.
// If tmpl is nil the built-in default template is used.
func NonFastForwardMessage(tmpl *template.Template, ctx ViolationContext) string {
	return render(tmpl, defaultNonFastForwardTmpl, ctx)
}

// DirectPushDeniedMessage renders the protected-branch direct-push violation message.
func DirectPushDeniedMessage(tmpl *template.Template, ctx ViolationContext) string {
	return render(tmpl, defaultDirectPushDeniedTmpl, ctx)
}

// CommitterMismatchMessage renders the committer-vs-push-user violation message.
func CommitterMismatchMessage(tmpl *template.Template, ctx ViolationContext) string {
	return render(tmpl, defaultCommitterMismatchTmpl, ctx)
}

// TaskIDMissingMessage renders the missing-task-ID violation message.
func TaskIDMissingMessage(tmpl *template.Template, ctx ViolationContext) string {
	return render(tmpl, defaultTaskIDMissingTmpl, ctx)
}

// render executes tmpl (or fallback if tmpl is nil). On execution error it
// retries with the fallback so a broken custom template never panics.
func render(tmpl, fallback *template.Template, ctx ViolationContext) string {
	ctx = fillUnknown(ctx)
	t := tmpl
	if t == nil {
		t = fallback
	}
	var buf strings.Builder
	if err := t.Execute(&buf, ctx); err != nil {
		buf.Reset()
		_ = fallback.Execute(&buf, ctx)
	}
	return buf.String()
}

func fillUnknown(ctx ViolationContext) ViolationContext {
	ctx.ProjectPath = orUnknown(ctx.ProjectPath)
	ctx.RepoName = orUnknown(ctx.RepoName)
	ctx.Username = orUnknown(ctx.Username)
	ctx.BranchName = orUnknown(ctx.BranchName)
	ctx.Protocol = orUnknown(ctx.Protocol)
	ctx.CommitID = orUnknown(ctx.CommitID)
	ctx.CommitSubject = orUnknown(ctx.CommitSubject)
	ctx.CommitterEmail = orUnknown(ctx.CommitterEmail)
	return ctx
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
