package hook

import (
	"regexp"
	"testing"
)

// defaultTaskIDRe and defaultBranchCheckScopeRe mirror the built-in defaults.
var (
	defaultBranchCheckScopeRe = regexp.MustCompile(`(?i)^refs/heads/(feature|dev)(/|_|-|$)`)
	defaultTaskIDRe            = regexp.MustCompile(`\[#(TSK|DEF)-[^\[\]]+\]`)
	defaultBranchWhitelistRe   = regexp.MustCompile(`(?i)^refs/heads/(init/|migrate/|tmp/)`)
	defaultDenyDirectPushRe    = regexp.MustCompile(`^refs/heads/(master|main|release/.*)$`)
)

// ---- IsRoot / IsInRootBypass ----

func TestIsRoot(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"root", true},
		{"Root", true},
		{"ROOT", true},
		{"admin", false},
		{"zhangsan", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsRoot(c.input); got != c.want {
			t.Errorf("IsRoot(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestIsInRootBypass(t *testing.T) {
	list := []string{"root", "deploy-bot"}
	if !IsInRootBypass("root", list) {
		t.Error("root should be in bypass list")
	}
	if !IsInRootBypass("Deploy-Bot", list) {
		t.Error("Deploy-Bot (case-insensitive) should be in bypass list")
	}
	if IsInRootBypass("zhangsan", list) {
		t.Error("zhangsan should not be in bypass list")
	}
}

// ---- ExtractRepoName ----

func TestExtractRepoName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"group/demo", "demo"},
		{"group/sub/demo", "demo"},
		{"group/subgroup/demo-service", "demo-service"},
		{"demo", "demo"},
		{"", ""},
	}
	for _, c := range cases {
		if got := ExtractRepoName(c.input); got != c.want {
			t.Errorf("ExtractRepoName(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ---- IsInProjectWhitelist ----

func TestIsInProjectWhitelist(t *testing.T) {
	cases := []struct {
		repoName  string
		whitelist []string
		want      bool
	}{
		{"demo", []string{"demo", "app"}, true},
		{"demo", []string{"app", "tool"}, false},
		{"DEMO", []string{"demo"}, true},
		{"demo", []string{}, false},
		{"demo", nil, false},
		{"legacy-repo", []string{"legacy-repo", "migrate-tool"}, true},
	}
	for _, c := range cases {
		if got := IsInProjectWhitelist(c.repoName, c.whitelist); got != c.want {
			t.Errorf("IsInProjectWhitelist(%q, %v) = %v, want %v",
				c.repoName, c.whitelist, got, c.want)
		}
	}
}

// ---- IsInUserWhitelist ----

func TestIsInUserWhitelist(t *testing.T) {
	cases := []struct {
		username  string
		whitelist []string
		want      bool
	}{
		{"zhangsan", []string{"zhangsan"}, true},
		{"ZHANGSAN", []string{"zhangsan"}, true},
		{"lisi", []string{"zhangsan", "wangwu"}, false},
		{"zhangsan", []string{}, false},
		{"zhangsan", nil, false},
	}
	for _, c := range cases {
		if got := IsInUserWhitelist(c.username, c.whitelist); got != c.want {
			t.Errorf("IsInUserWhitelist(%q, %v) = %v, want %v",
				c.username, c.whitelist, got, c.want)
		}
	}
}

// ---- HasValidTaskID ----

func TestHasValidTaskID(t *testing.T) {
	re := defaultTaskIDRe
	valid := []string{
		"add login api [#TSK-1001]",
		"fix bug [#DEF-A20260001]",
		"[#TSK-anything] update service",
		"fix [#TSK-1]",
		"fix [#DEF-abc]",
	}
	for _, s := range valid {
		if !HasValidTaskID(s, re) {
			t.Errorf("HasValidTaskID(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"add login api",
		"add login api #TSK-1001",
		"add login api [TSK-1001]",
		"add login api [#BUG-1001]",
		"add login api [#TSK-]",
		"[#TSK-]",
	}
	for _, s := range invalid {
		if HasValidTaskID(s, re) {
			t.Errorf("HasValidTaskID(%q) = true, want false", s)
		}
	}

	// nil regex → always false
	if HasValidTaskID("fix [#TSK-1]", nil) {
		t.Error("HasValidTaskID with nil regex should return false")
	}
}

// ---- MatchesBranchCheckScope ----

func TestMatchesBranchCheckScope(t *testing.T) {
	re := defaultBranchCheckScopeRe
	inScope := []string{
		"refs/heads/dev",
		"refs/heads/dev/login",
		"refs/heads/dev-login",
		"refs/heads/dev_login",
		"refs/heads/feature",
		"refs/heads/feature/login",
		"refs/heads/feature-login",
	}
	for _, ref := range inScope {
		if !MatchesBranchCheckScope(ref, re) {
			t.Errorf("MatchesBranchCheckScope(%q) = false, want true", ref)
		}
	}

	outOfScope := []string{
		"refs/heads/develop",
		"refs/heads/master",
		"refs/heads/main",
		"refs/heads/SIT_A2026001",
		"refs/heads/release/1.0",
		"refs/tags/v1.0",
	}
	for _, ref := range outOfScope {
		if MatchesBranchCheckScope(ref, re) {
			t.Errorf("MatchesBranchCheckScope(%q) = true, want false", ref)
		}
	}

	// nil regex → always false
	if MatchesBranchCheckScope("refs/heads/dev", nil) {
		t.Error("MatchesBranchCheckScope with nil regex should return false")
	}
}

// ---- MatchesBranchWhitelist ----

func TestMatchesBranchWhitelist(t *testing.T) {
	re := defaultBranchWhitelistRe
	hit := []string{
		"refs/heads/init/base",
		"refs/heads/migrate/gitlab16",
		"refs/heads/tmp/test",
	}
	for _, ref := range hit {
		if !MatchesBranchWhitelist(ref, re) {
			t.Errorf("MatchesBranchWhitelist(%q) = false, want true", ref)
		}
	}

	miss := []string{
		"refs/heads/dev/test",
		"refs/heads/feature/init",
		"refs/heads/master",
	}
	for _, ref := range miss {
		if MatchesBranchWhitelist(ref, re) {
			t.Errorf("MatchesBranchWhitelist(%q) = true, want false", ref)
		}
	}
}

// ---- ExtractEmailPrefix ----

func TestExtractEmailPrefix(t *testing.T) {
	cases := []struct {
		email string
		want  string
	}{
		{"zhangsan@example.com", "zhangsan"},
		{"lisi@company.org", "lisi"},
		{"no-at-sign", "no-at-sign"},
		{"", ""},
	}
	for _, c := range cases {
		if got := ExtractEmailPrefix(c.email); got != c.want {
			t.Errorf("ExtractEmailPrefix(%q) = %q, want %q", c.email, got, c.want)
		}
	}
}

// ---- CommitterMatchesPushUser ----

func TestCommitterMatchesPushUser(t *testing.T) {
	cases := []struct {
		email    string
		username string
		want     bool
	}{
		{"zhangsan@example.com", "zhangsan", true},
		{"ZHANGSAN@example.com", "zhangsan", true},
		{"zhangsan@example.com", "ZHANGSAN", true},
		{"lisi@example.com", "zhangsan", false},
		{"", "zhangsan", false},
		{"zhangsan@example.com", "", false},
	}
	for _, c := range cases {
		if got := CommitterMatchesPushUser(c.email, c.username); got != c.want {
			t.Errorf("CommitterMatchesPushUser(%q, %q) = %v, want %v",
				c.email, c.username, got, c.want)
		}
	}
}

// ---- ExtractBranchName ----

func TestExtractBranchName(t *testing.T) {
	cases := []struct {
		ref  string
		want string
	}{
		{"refs/heads/dev/login", "dev/login"},
		{"refs/heads/feature", "feature"},
		{"refs/heads/master", "master"},
		{"refs/tags/v1.0", "refs/tags/v1.0"},
	}
	for _, c := range cases {
		if got := ExtractBranchName(c.ref); got != c.want {
			t.Errorf("ExtractBranchName(%q) = %q, want %q", c.ref, got, c.want)
		}
	}
}

// ---- IsDeleteRef ----

func TestIsDeleteRef(t *testing.T) {
	if !IsDeleteRef("0000000000000000000000000000000000000000") {
		t.Error("IsDeleteRef(ZeroSHA) = false, want true")
	}
	if IsDeleteRef("abcdef1234567890abcdef1234567890abcdef12") {
		t.Error("IsDeleteRef(non-zero) = true, want false")
	}
}

// ---- IsTagRef ----

func TestIsTagRef(t *testing.T) {
	if !IsTagRef("refs/tags/v1.0") {
		t.Error("IsTagRef(refs/tags/v1.0) = false, want true")
	}
	if IsTagRef("refs/heads/dev") {
		t.Error("IsTagRef(refs/heads/dev) = true, want false")
	}
}

// ---- IsFastForward ----

func TestIsFastForward(t *testing.T) {
	sha := "aabbccdd"
	if !IsFastForward(sha, sha) {
		t.Error("IsFastForward(same) = false, want true")
	}
	if IsFastForward("", sha) {
		t.Error("IsFastForward(empty mergeBase) = true, want false")
	}
	if IsFastForward("other", sha) {
		t.Error("IsFastForward(different) = true, want false")
	}
}

// ---- IsMergeCommit ----

func TestIsMergeCommit(t *testing.T) {
	if IsMergeCommit(1) {
		t.Error("IsMergeCommit(1) = true, want false")
	}
	if !IsMergeCommit(2) {
		t.Error("IsMergeCommit(2) = false, want true")
	}
	if !IsMergeCommit(3) {
		t.Error("IsMergeCommit(3) = false, want true")
	}
}

// ---- MatchesMessageWhitelist ----

func TestMatchesMessageWhitelist(t *testing.T) {
	// nil regex → false
	if MatchesMessageWhitelist("anything", nil) {
		t.Error("nil regex should return false")
	}
	re := regexp.MustCompile(`^auto:`)
	if !MatchesMessageWhitelist("auto: merge branch", re) {
		t.Error("matching pattern should return true")
	}
	if MatchesMessageWhitelist("manual commit", re) {
		t.Error("non-matching pattern should return false")
	}
}

// ---- MatchesPushDenyBranch ----

func TestMatchesPushDenyBranch(t *testing.T) {
	re := defaultDenyDirectPushRe

	// nil regex disables the feature
	if MatchesPushDenyBranch("refs/heads/master", nil) {
		t.Error("nil regex should return false")
	}

	denied := []string{
		"refs/heads/master",
		"refs/heads/main",
		"refs/heads/release/1.0",
		"refs/heads/release/2026-Q1",
	}
	for _, ref := range denied {
		if !MatchesPushDenyBranch(ref, re) {
			t.Errorf("MatchesPushDenyBranch(%q) = false, want true", ref)
		}
	}

	allowed := []string{
		"refs/heads/dev/login",
		"refs/heads/feature/auth",
		"refs/heads/hotfix/123",
	}
	for _, ref := range allowed {
		if MatchesPushDenyBranch(ref, re) {
			t.Errorf("MatchesPushDenyBranch(%q) = true, want false", ref)
		}
	}
}
