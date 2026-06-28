package message

import (
	"strings"
	"testing"
	"text/template"
)

var fullCtx = ViolationContext{
	ProjectPath:    "group/my-repo",
	RepoName:       "my-repo",
	Username:       "zhangsan",
	BranchName:     "master",
	Protocol:       "ssh",
	CommitID:       "abc1234",
	CommitSubject:  "add feature",
	CommitterEmail: "lisi@example.com",
}

// ---- nil template → default built-in content ----

func TestNonFastForwardMessage_Default(t *testing.T) {
	msg := NonFastForwardMessage(nil, fullCtx)
	assertContains(t, msg, "禁止强制推送")
	assertContains(t, msg, "如何处理")
	assertContains(t, msg, "git pull --rebase origin master")
	assertContains(t, msg, "zhangsan")
	assertContains(t, msg, "master")
}

func TestDirectPushDeniedMessage_Default(t *testing.T) {
	msg := DirectPushDeniedMessage(nil, fullCtx)
	assertContains(t, msg, "禁止直接 push")
	assertContains(t, msg, "如何处理")
	assertContains(t, msg, "protocol: ssh")
	assertContains(t, msg, "origin/master")
}

func TestCommitterMismatchMessage_Default(t *testing.T) {
	msg := CommitterMismatchMessage(nil, fullCtx)
	assertContains(t, msg, "提交人信息不符合规范")
	assertContains(t, msg, "如何修改")
	assertContains(t, msg, "lisi@example.com")
	assertContains(t, msg, "abc1234")
}

func TestTaskIDMissingMessage_Default(t *testing.T) {
	msg := TaskIDMissingMessage(nil, fullCtx)
	assertContains(t, msg, "缺少任务ID")
	assertContains(t, msg, "如何修改")
	assertContains(t, msg, "add feature")
	assertContains(t, msg, "abc1234")
	assertContains(t, msg, "origin/master")
}

// ---- empty context → "unknown" substitution ----

func TestMessages_EmptyContext_UsesUnknown(t *testing.T) {
	ctx := ViolationContext{}
	for _, msg := range []string{
		NonFastForwardMessage(nil, ctx),
		DirectPushDeniedMessage(nil, ctx),
		CommitterMismatchMessage(nil, ctx),
		TaskIDMissingMessage(nil, ctx),
	} {
		if strings.Contains(msg, "{{") {
			t.Errorf("unrendered template placeholder found in message: %s", msg)
		}
		assertContains(t, msg, "unknown")
	}
}

// ---- custom template override ----

func TestNonFastForwardMessage_CustomTemplate(t *testing.T) {
	tmpl := template.Must(template.New("custom").Parse("CUSTOM: branch={{.BranchName}} user={{.Username}}"))
	msg := NonFastForwardMessage(tmpl, fullCtx)
	if msg != "CUSTOM: branch=master user=zhangsan" {
		t.Errorf("unexpected custom template output: %q", msg)
	}
}

func TestDirectPushDeniedMessage_CustomTemplate(t *testing.T) {
	tmpl := template.Must(template.New("custom").Parse("project={{.ProjectPath}}"))
	msg := DirectPushDeniedMessage(tmpl, fullCtx)
	if msg != "project=group/my-repo" {
		t.Errorf("unexpected output: %q", msg)
	}
}

// ---- custom template execution error → fallback to default ----

func TestRender_CustomTemplateExecError_FallsBackToDefault(t *testing.T) {
	// A template that accesses a non-existent field in strict mode won't error
	// with text/template (it renders <no value>). Instead we test by providing
	// a template parsed with Option("missingkey=error") to force an error.
	tmpl := template.Must(
		template.New("bad").Option("missingkey=error").Parse("{{.NonExistentField}}"),
	)
	msg := NonFastForwardMessage(tmpl, fullCtx)
	// Should fall back to the default, which contains the standard content.
	assertContains(t, msg, "禁止强制推送")
}

func assertContains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("expected message to contain %q\nfull message:\n%s", sub, s)
	}
}
