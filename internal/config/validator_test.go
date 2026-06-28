package config

import (
	"strings"
	"testing"
)

func validConfig() *HookConfig {
	return DefaultConfig()
}

func TestValidate_ValidDefault(t *testing.T) {
	errs := Validate(validConfig())
	if len(errs) != 0 {
		t.Errorf("default config should be valid, got: %v", errs)
	}
}

func TestValidate_UnsupportedVersion(t *testing.T) {
	cfg := validConfig()
	cfg.Version = "99.0"
	errs := Validate(cfg)
	if !hasFieldError(errs, "version") {
		t.Error("expected version error")
	}
}

func TestValidate_BadMode(t *testing.T) {
	cfg := validConfig()
	cfg.Mode.Default = "unknown"
	errs := Validate(cfg)
	if !hasFieldError(errs, "mode.default") {
		t.Error("expected mode.default error")
	}
}

func TestValidate_BadDenyDirectPushRegex(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.DenyDirectPush.BranchRegex = "[invalid"
	errs := Validate(cfg)
	if !hasFieldError(errs, "rules.deny_direct_push.branch_regex") {
		t.Error("expected deny_direct_push.branch_regex error")
	}
}

func TestValidate_DenyDirectPushBothProtocolsEmpty(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.DenyDirectPush.AllowProtocols = nil
	cfg.Rules.DenyDirectPush.DenyProtocols = nil
	errs := Validate(cfg)
	if !hasFieldError(errs, "rules.deny_direct_push") {
		t.Error("expected deny_direct_push protocols error")
	}
}

func TestValidate_BadCompareStrategy(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.CommitterMatchPushUser.CompareStrategy = "full_email"
	errs := Validate(cfg)
	if !hasFieldError(errs, "rules.committer_match_push_user.compare_strategy") {
		t.Error("expected compare_strategy error")
	}
}

func TestValidate_BadTaskIDBranchRegex(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.TaskID.BranchRegex = "[bad"
	errs := Validate(cfg)
	if !hasFieldError(errs, "rules.task_id.branch_regex") {
		t.Error("expected task_id.branch_regex error")
	}
}

func TestValidate_BadTaskIDSubjectRegex(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.TaskID.SubjectRegex = "[bad"
	errs := Validate(cfg)
	if !hasFieldError(errs, "rules.task_id.subject_regex") {
		t.Error("expected task_id.subject_regex error")
	}
}

func TestValidate_BadWhitelistBranchRegex(t *testing.T) {
	cfg := validConfig()
	cfg.Whitelist.BranchRegex = "[bad"
	errs := Validate(cfg)
	if !hasFieldError(errs, "whitelist.branch_regex") {
		t.Error("expected whitelist.branch_regex error")
	}
}

func TestValidate_DisabledDenyDirectPushSkipsProtocolCheck(t *testing.T) {
	cfg := validConfig()
	cfg.Rules.DenyDirectPush.Enabled = false
	cfg.Rules.DenyDirectPush.AllowProtocols = nil
	cfg.Rules.DenyDirectPush.DenyProtocols = nil
	errs := Validate(cfg)
	if hasFieldError(errs, "rules.deny_direct_push") {
		t.Error("disabled deny_direct_push should skip protocol validation")
	}
}

func TestValidate_BadMessageTemplate(t *testing.T) {
	cfg := validConfig()
	cfg.Messages.Templates.NonFastForward = "{{.Unclosed"
	errs := Validate(cfg)
	if !hasFieldError(errs, "messages.templates.non_fast_forward") {
		t.Error("expected template syntax error for non_fast_forward")
	}
}

func TestValidate_ValidMessageTemplate(t *testing.T) {
	cfg := validConfig()
	cfg.Messages.Templates.NonFastForward = "branch: {{.BranchName}}"
	errs := Validate(cfg)
	if hasFieldError(errs, "messages.templates.non_fast_forward") {
		t.Error("valid template should not produce an error")
	}
}

func TestValidate_EmptyMessageTemplates(t *testing.T) {
	// Default config has no templates set — should always pass.
	errs := Validate(validConfig())
	for _, e := range errs {
		if strings.HasPrefix(e.Field, "messages.templates") {
			t.Errorf("default config should not have template errors, got: %v", e)
		}
	}
}

func hasFieldError(errs []ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}
