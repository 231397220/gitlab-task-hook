package hook

import (
	"testing"
)

// ---- IsRoot ----

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
		// whitelist entries must be lowercased (env.Load does this)
		wl := make([]string, len(c.whitelist))
		for i, v := range c.whitelist {
			wl[i] = v
		}
		if got := IsInProjectWhitelist(c.repoName, wl); got != c.want {
			t.Errorf("IsInProjectWhitelist(%q, %v) = %v, want %v", c.repoName, c.whitelist, got, c.want)
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
			t.Errorf("IsInUserWhitelist(%q, %v) = %v, want %v", c.username, c.whitelist, got, c.want)
		}
	}
}

// ---- HasValidTaskID ----

func TestHasValidTaskID(t *testing.T) {
	valid := []string{
		"add login api [#TSK-1001]",
		"fix bug [#DEF-A20260001]",
		"[#TSK-anything] update service",
		"fix [#TSK-1]",
		"fix [#DEF-abc]",
	}
	for _, s := range valid {
		if !HasValidTaskID(s) {
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
		if HasValidTaskID(s) {
			t.Errorf("HasValidTaskID(%q) = true, want false", s)
		}
	}
}

// ---- MatchesBranchCheckScope ----

func TestMatchesBranchCheckScope(t *testing.T) {
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
		if !MatchesBranchCheckScope(ref) {
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
		if MatchesBranchCheckScope(ref) {
			t.Errorf("MatchesBranchCheckScope(%q) = true, want false", ref)
		}
	}
}

// ---- MatchesBranchWhitelist ----

func TestMatchesBranchWhitelist(t *testing.T) {
	hit := []string{
		"refs/heads/init/base",
		"refs/heads/migrate/gitlab16",
		"refs/heads/tmp/test",
	}
	for _, ref := range hit {
		if !MatchesBranchWhitelist(ref) {
			t.Errorf("MatchesBranchWhitelist(%q) = false, want true", ref)
		}
	}

	miss := []string{
		"refs/heads/dev/test",
		"refs/heads/feature/init",
		"refs/heads/master",
	}
	for _, ref := range miss {
		if MatchesBranchWhitelist(ref) {
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
			t.Errorf("CommitterMatchesPushUser(%q, %q) = %v, want %v", c.email, c.username, got, c.want)
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
		{"refs/tags/v1.0", "refs/tags/v1.0"}, // not a branch ref, returned as-is
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
	if MatchesMessageWhitelist("anything", "") {
		t.Error("empty pattern should return false")
	}
	if !MatchesMessageWhitelist("auto: merge branch", "^auto:") {
		t.Error("matching pattern should return true")
	}
	if MatchesMessageWhitelist("manual commit", "^auto:") {
		t.Error("non-matching pattern should return false")
	}
	if MatchesMessageWhitelist("test", "[invalid") {
		t.Error("invalid regex should return false (no panic)")
	}
}

// ---- MatchesPushDenyBranch ----

func TestMatchesPushDenyBranch(t *testing.T) {
	pattern := `^refs/heads/(master|main|release/.*)$`

	// empty pattern disables the feature
	if MatchesPushDenyBranch("refs/heads/master", "") {
		t.Error("empty pattern should return false")
	}

	// branches that should be denied
	denied := []string{
		"refs/heads/master",
		"refs/heads/main",
		"refs/heads/release/1.0",
		"refs/heads/release/2026-Q1",
	}
	for _, ref := range denied {
		if !MatchesPushDenyBranch(ref, pattern) {
			t.Errorf("MatchesPushDenyBranch(%q) = false, want true", ref)
		}
	}

	// branches that should not be denied
	allowed := []string{
		"refs/heads/dev/login",
		"refs/heads/feature/auth",
		"refs/heads/hotfix/123",
	}
	for _, ref := range allowed {
		if MatchesPushDenyBranch(ref, pattern) {
			t.Errorf("MatchesPushDenyBranch(%q) = true, want false", ref)
		}
	}

	// invalid regex → false, no panic
	if MatchesPushDenyBranch("refs/heads/master", "[invalid") {
		t.Error("invalid regex should return false (no panic)")
	}
}
