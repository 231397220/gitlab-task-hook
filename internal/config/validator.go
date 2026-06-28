package config

import (
	"fmt"
	"regexp"
	"text/template"
)

// ValidationError describes a single configuration validation failure.
type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Reason)
}

var supportedVersions = map[string]bool{"1.0": true}

// Validate checks hcfg for structural and semantic correctness.
// Returns a slice of all violations found (never nil when empty).
func Validate(hcfg *HookConfig) []ValidationError {
	var errs []ValidationError

	add := func(field, reason string) {
		errs = append(errs, ValidationError{Field: field, Reason: reason})
	}
	compileField := func(field, pattern string) {
		if pattern != "" {
			if _, err := regexp.Compile(pattern); err != nil {
				add(field, err.Error())
			}
		}
	}

	if !supportedVersions[hcfg.Version] {
		add("version", fmt.Sprintf("unsupported version %q (supported: 1.0)", hcfg.Version))
	}

	if hcfg.Mode.Default != "enforce" && hcfg.Mode.Default != "warn" {
		add("mode.default", fmt.Sprintf("must be enforce or warn, got %q", hcfg.Mode.Default))
	}

	if hcfg.Rules.DenyDirectPush.Enabled {
		compileField("rules.deny_direct_push.branch_regex", hcfg.Rules.DenyDirectPush.BranchRegex)
		if len(hcfg.Rules.DenyDirectPush.AllowProtocols) == 0 &&
			len(hcfg.Rules.DenyDirectPush.DenyProtocols) == 0 {
			add("rules.deny_direct_push", "allow_protocols and deny_protocols cannot both be empty when enabled")
		}
	}

	if hcfg.Rules.CommitterMatchPushUser.Enabled {
		if hcfg.Rules.CommitterMatchPushUser.CompareStrategy != "email_prefix" {
			add("rules.committer_match_push_user.compare_strategy",
				fmt.Sprintf("only email_prefix is supported, got %q", hcfg.Rules.CommitterMatchPushUser.CompareStrategy))
		}
	}

	if hcfg.Rules.TaskID.Enabled {
		compileField("rules.task_id.branch_regex", hcfg.Rules.TaskID.BranchRegex)
		compileField("rules.task_id.subject_regex", hcfg.Rules.TaskID.SubjectRegex)
		compileField("rules.task_id.exempt_message_regex", hcfg.Rules.TaskID.ExemptMessageRegex)
	}

	compileField("whitelist.branch_regex", hcfg.Whitelist.BranchRegex)

	// Validate message template overrides (Go text/template syntax).
	validateTemplate := func(field, tplStr string) {
		if tplStr != "" {
			if _, err := template.New("").Parse(tplStr); err != nil {
				add(field, "invalid Go template syntax: "+err.Error())
			}
		}
	}
	validateTemplate("messages.templates.non_fast_forward", hcfg.Messages.Templates.NonFastForward)
	validateTemplate("messages.templates.direct_push_denied", hcfg.Messages.Templates.DirectPushDenied)
	validateTemplate("messages.templates.committer_mismatch", hcfg.Messages.Templates.CommitterMismatch)
	validateTemplate("messages.templates.task_id_missing", hcfg.Messages.Templates.TaskIDMissing)

	return errs
}
