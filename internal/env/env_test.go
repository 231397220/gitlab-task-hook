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
	for _, k := range []string{"GL_USERNAME", "GL_PROJECT_PATH", "GL_PROTOCOL", "HOOK_MODE"} {
		old, had := os.LookupEnv(k)
		os.Unsetenv(k)
		k2, had2, old2 := k, had, old
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
	if cfg.GLProjectPath != "" {
		t.Errorf("GLProjectPath = %q, want empty", cfg.GLProjectPath)
	}
	if cfg.HookMode != "" {
		t.Errorf("HookMode = %q, want empty", cfg.HookMode)
	}
}

func TestLoad_GLFields(t *testing.T) {
	setenv(t, "GL_USERNAME", "zhangsan")
	setenv(t, "GL_PROJECT_PATH", "group/repo")
	setenv(t, "GL_PROTOCOL", "ssh")

	cfg := Load()
	if cfg.GLUsername != "zhangsan" {
		t.Errorf("GLUsername = %q, want zhangsan", cfg.GLUsername)
	}
	if cfg.GLProjectPath != "group/repo" {
		t.Errorf("GLProjectPath = %q, want group/repo", cfg.GLProjectPath)
	}
	if cfg.GLProtocol != "ssh" {
		t.Errorf("GLProtocol = %q, want ssh", cfg.GLProtocol)
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
