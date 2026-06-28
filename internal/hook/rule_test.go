package hook

import (
	"strings"
	"testing"

	"gitlab-task-hook/internal/config"
	"gitlab-task-hook/internal/env"
)

// fakeRunner implements git.Runner for hook rule tests without touching disk.
type fakeRunner struct {
	mergeBase      string
	mergeBaseErr   error
	newCommits     []string
	newCommitsErr  error
	parentCount    int
	parentCountErr error
	subject        string
	subjectErr     error
	committerEmail string
	emailErr       error
}

func (f *fakeRunner) MergeBase(_, _ string) (string, error) {
	return f.mergeBase, f.mergeBaseErr
}
func (f *fakeRunner) NewCommits(_ string) ([]string, error) {
	return f.newCommits, f.newCommitsErr
}
func (f *fakeRunner) ParentCount(_ string) (int, error) {
	return f.parentCount, f.parentCountErr
}
func (f *fakeRunner) CommitSubject(_ string) (string, error) {
	return f.subject, f.subjectErr
}
func (f *fakeRunner) CommitterEmail(_ string) (string, error) {
	return f.committerEmail, f.emailErr
}

func defaultCompiledConfig() *config.CompiledConfig {
	cc, _ := config.Compile(config.DefaultConfig())
	return cc
}

func defaultEnv(username string) env.Config {
	return env.Config{
		GLUsername:    username,
		GLProjectPath: "group/test-repo",
		GLProtocol:    "ssh",
	}
}

const (
	zeroSHA    = "0000000000000000000000000000000000000000"
	nonZeroSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	prevSHA    = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestCheckRef_DeleteRef(t *testing.T) {
	ref := RefUpdate{OldValue: nonZeroSHA, NewValue: zeroSHA, RefName: "refs/heads/dev"}
	v, err := CheckRef(ref, defaultEnv("alice"), defaultCompiledConfig(), &fakeRunner{})
	if err != nil || v != nil {
		t.Errorf("delete ref should pass: v=%v err=%v", v, err)
	}
}

func TestCheckRef_TagRef(t *testing.T) {
	ref := RefUpdate{OldValue: zeroSHA, NewValue: nonZeroSHA, RefName: "refs/tags/v1.0"}
	v, err := CheckRef(ref, defaultEnv("alice"), defaultCompiledConfig(), &fakeRunner{})
	if err != nil || v != nil {
		t.Errorf("tag ref should pass: v=%v err=%v", v, err)
	}
}

func TestCheckRef_ForcePushBlocked(t *testing.T) {
	g := &fakeRunner{mergeBase: "other-sha"} // not equal to prevSHA → not fast-forward
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev"}
	v, err := CheckRef(ref, defaultEnv("alice"), defaultCompiledConfig(), g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil || v.Type != ViolationForcePush {
		t.Errorf("expected ForcePush violation, got %v", v)
	}
}

func TestCheckRef_DenyDirectPushBlocked(t *testing.T) {
	// master branch, ssh protocol → should be denied
	g := &fakeRunner{mergeBase: prevSHA} // fast-forward OK
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/master"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil || v.Type != ViolationProtectedBranchDirect {
		t.Errorf("expected ProtectedBranchDirect violation, got %v", v)
	}
}

func TestCheckRef_WebBypassesDenyDirectPush(t *testing.T) {
	// master branch, web protocol → should pass
	g := &fakeRunner{mergeBase: prevSHA}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/master"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "web"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil || v != nil {
		t.Errorf("web protocol on master should pass: v=%v err=%v", v, err)
	}
}

func TestCheckRef_CommitterMismatch(t *testing.T) {
	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "lisi@example.com", // mismatch with alice
		subject:        "fix bug [#TSK-123]",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil || v.Type != ViolationCommitterMismatch {
		t.Errorf("expected CommitterMismatch violation, got %v", v)
	}
}

func TestCheckRef_MissingTaskID(t *testing.T) {
	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "alice@example.com",
		subject:        "add feature without task id",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev/feature-x"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil || v.Type != ViolationMissingTaskID {
		t.Errorf("expected MissingTaskID violation, got %v", v)
	}
}

func TestCheckRef_ValidTaskID(t *testing.T) {
	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "alice@example.com",
		subject:        "add feature [#TSK-1001]",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev/feature-x"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil || v != nil {
		t.Errorf("valid commit should pass: v=%v err=%v", v, err)
	}
}

func TestCheckRef_ProjectWhitelistSkipsTaskID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Whitelist.Projects = []string{"test-repo"}
	cc, _ := config.Compile(cfg)

	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "alice@example.com",
		subject:        "no task id here",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev/x"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "group/test-repo", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, cc, g)
	if err != nil || v != nil {
		t.Errorf("whitelisted project should skip task id check: v=%v err=%v", v, err)
	}
}

func TestCheckRef_MergeCommitExemptFromTaskID(t *testing.T) {
	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    2, // merge commit
		committerEmail: "alice@example.com",
		subject:        "Merge branch 'feature/x' into dev",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil || v != nil {
		t.Errorf("merge commit should be exempt from task id: v=%v err=%v", v, err)
	}
}

func TestCheckRef_GlobalDisabled(t *testing.T) {
	// This test is handled at the hook subcommand level, not inside CheckRef.
	// Verify that CheckRef itself still checks rules (global enabled is checked outside).
	// Just confirm CheckRef returns no error when called normally.
	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "alice@example.com",
		subject:        "fix [#TSK-100]",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev"}
	_, err := CheckRef(ref, defaultEnv("alice"), defaultCompiledConfig(), g)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckRef_UserWhitelistSkipsTaskID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Whitelist.Users = []string{"alice"}
	cc, _ := config.Compile(cfg)

	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "alice@example.com",
		subject:        "no task id",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev/x"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, cc, g)
	if err != nil || v != nil {
		t.Errorf("whitelisted user should skip task id: v=%v err=%v", v, err)
	}
}

func TestCheckRef_NewBranchNoFastForwardCheck(t *testing.T) {
	// New branch (OldValue = zeroSHA) should skip the fast-forward check.
	g := &fakeRunner{
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "alice@example.com",
		subject:        "init [#TSK-1]",
	}
	ref := RefUpdate{OldValue: zeroSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev/new"}
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil || v != nil {
		t.Errorf("new branch should not trigger force-push check: v=%v err=%v", v, err)
	}
}

func TestCheckRef_BranchWhitelistSkipsTaskID(t *testing.T) {
	g := &fakeRunner{
		mergeBase:      prevSHA,
		newCommits:     []string{nonZeroSHA},
		parentCount:    1,
		committerEmail: "alice@example.com",
		subject:        "init repo",
	}
	ref := RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/init/base"}
	// init/* matches the whitelist but NOT the task-check branch scope → no task ID needed
	e := env.Config{GLUsername: "alice", GLProjectPath: "g/r", GLProtocol: "ssh"}
	v, err := CheckRef(ref, e, defaultCompiledConfig(), g)
	if err != nil || v != nil {
		t.Errorf("init/* branch should not need task id: v=%v err=%v", v, err)
	}
}

// Ensure violation messages are non-empty.
func TestViolationMessages_NonEmpty(t *testing.T) {
	tests := []struct {
		name string
		ref  RefUpdate
		env  env.Config
		g    *fakeRunner
		want ViolationType
	}{
		{
			name: "force push",
			ref:  RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev"},
			env:  defaultEnv("alice"),
			g:    &fakeRunner{mergeBase: "other"},
			want: ViolationForcePush,
		},
		{
			name: "missing task id",
			ref:  RefUpdate{OldValue: prevSHA, NewValue: nonZeroSHA, RefName: "refs/heads/dev/x"},
			env:  defaultEnv("alice"),
			g: &fakeRunner{
				mergeBase: prevSHA, newCommits: []string{nonZeroSHA},
				parentCount: 1, committerEmail: "alice@example.com",
				subject: "no task id",
			},
			want: ViolationMissingTaskID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := CheckRef(tt.ref, tt.env, defaultCompiledConfig(), tt.g)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v == nil {
				t.Fatal("expected violation")
			}
			if v.Type != tt.want {
				t.Errorf("violation type = %v, want %v", v.Type, tt.want)
			}
			if strings.TrimSpace(v.Message) == "" {
				t.Error("violation message should be non-empty")
			}
		})
	}
}
