package env

import (
	"os"
	"testing"
)

func setenv(t *testing.T, key, val string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	os.Setenv(key, val)
	t.Cleanup(func() {
		if had {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	// Unset all relevant vars
	for _, k := range []string{"GL_USERNAME", "GL_PROJECT_PATH", "GL_PROTOCOL", "HOOK_MODE",
		"WHITELIST_USERS", "WHITELIST_PROJECT_NAMES", "EXEMPT_MESSAGE_REGEX", "EXEMPT_MERGE_COMMIT"} {
		old, had := os.LookupEnv(k)
		os.Unsetenv(k)
		k2 := k
		had2 := had
		old2 := old
		t.Cleanup(func() {
			if had2 {
				os.Setenv(k2, old2)
			}
		})
	}

	cfg := Load()
	if cfg.GLUsername != "" {
		t.Errorf("GLUsername = %q, want empty", cfg.GLUsername)
	}
	if !cfg.ExemptMergeCommit {
		t.Error("ExemptMergeCommit default should be true")
	}
	if len(cfg.WhitelistUsers) != 0 {
		t.Errorf("WhitelistUsers = %v, want empty", cfg.WhitelistUsers)
	}
}

func TestLoad_WhitelistUsers(t *testing.T) {
	setenv(t, "WHITELIST_USERS", "ZhangSan  lisi  WANGWU")
	cfg := Load()
	if len(cfg.WhitelistUsers) != 3 {
		t.Fatalf("WhitelistUsers len = %d, want 3", len(cfg.WhitelistUsers))
	}
	// All should be lowercased
	for _, u := range cfg.WhitelistUsers {
		for _, ch := range u {
			if ch >= 'A' && ch <= 'Z' {
				t.Errorf("whitelist user %q not lowercased", u)
			}
		}
	}
}

func TestLoad_WhitelistProjectNames(t *testing.T) {
	setenv(t, "WHITELIST_PROJECT_NAMES", "Demo-Service , legacy-repo , MigrateTool")
	cfg := Load()
	if len(cfg.WhitelistProjectNames) != 3 {
		t.Fatalf("WhitelistProjectNames len = %d, want 3", len(cfg.WhitelistProjectNames))
	}
	if cfg.WhitelistProjectNames[0] != "demo-service" {
		t.Errorf("first entry = %q, want %q", cfg.WhitelistProjectNames[0], "demo-service")
	}
}

func TestLoad_ExemptMergeCommitFalse(t *testing.T) {
	setenv(t, "EXEMPT_MERGE_COMMIT", "false")
	cfg := Load()
	if cfg.ExemptMergeCommit {
		t.Error("ExemptMergeCommit should be false when env=false")
	}
}

func TestIsWarnMode(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{"warn", true},
		{"WARN", true},
		{"enforce", false},
		{"", false},
		{"other", false},
	}
	for _, c := range cases {
		cfg := Config{HookMode: c.mode}
		if got := cfg.IsWarnMode(); got != c.want {
			t.Errorf("IsWarnMode(%q) = %v, want %v", c.mode, got, c.want)
		}
	}
}
