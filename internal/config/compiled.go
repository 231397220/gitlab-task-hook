package config

import (
	"fmt"
	"regexp"
	"text/template"
)

// CompiledConfig wraps HookConfig with pre-compiled regular expressions and
// message templates. All regexes and templates are compiled once at load time;
// nil means the rule/field is disabled or has no pattern configured.
type CompiledConfig struct {
	*HookConfig
	DenyDirectPushRegex   *regexp.Regexp
	BranchCheckScopeRegex *regexp.Regexp
	TaskIDRegex           *regexp.Regexp
	BranchWhitelistRegex  *regexp.Regexp
	ExemptMessageRegex    *regexp.Regexp

	// Message templates — nil means use the built-in default for that message.
	MsgNonFastForward    *template.Template
	MsgDirectPushDenied  *template.Template
	MsgCommitterMismatch *template.Template
	MsgTaskIDMissing     *template.Template
}

// Compile validates and pre-compiles all regex and template fields in hcfg.
// Returns an error if any regex or template fails to compile.
func Compile(hcfg *HookConfig) (*CompiledConfig, error) {
	c := &CompiledConfig{HookConfig: hcfg}

	mustCompile := func(label, pattern string, dst **regexp.Regexp) error {
		if pattern == "" {
			return nil
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		*dst = re
		return nil
	}

	if hcfg.Rules.DenyDirectPush.Enabled {
		if err := mustCompile("deny_direct_push.branch_regex",
			hcfg.Rules.DenyDirectPush.BranchRegex, &c.DenyDirectPushRegex); err != nil {
			return nil, err
		}
	}

	if hcfg.Rules.TaskID.Enabled {
		if err := mustCompile("task_id.branch_regex",
			hcfg.Rules.TaskID.BranchRegex, &c.BranchCheckScopeRegex); err != nil {
			return nil, err
		}
		if err := mustCompile("task_id.subject_regex",
			hcfg.Rules.TaskID.SubjectRegex, &c.TaskIDRegex); err != nil {
			return nil, err
		}
		if err := mustCompile("task_id.exempt_message_regex",
			hcfg.Rules.TaskID.ExemptMessageRegex, &c.ExemptMessageRegex); err != nil {
			return nil, err
		}
	}

	if err := mustCompile("whitelist.branch_regex",
		hcfg.Whitelist.BranchRegex, &c.BranchWhitelistRegex); err != nil {
		return nil, err
	}

	// Compile message template overrides. Empty string → nil (use built-in default).
	compileMsg := func(name, tplStr string, dst **template.Template) error {
		if tplStr == "" {
			return nil
		}
		t, err := template.New(name).Parse(tplStr)
		if err != nil {
			return fmt.Errorf("messages.templates.%s: %w", name, err)
		}
		*dst = t
		return nil
	}

	if err := compileMsg("non_fast_forward", hcfg.Messages.Templates.NonFastForward, &c.MsgNonFastForward); err != nil {
		return nil, err
	}
	if err := compileMsg("direct_push_denied", hcfg.Messages.Templates.DirectPushDenied, &c.MsgDirectPushDenied); err != nil {
		return nil, err
	}
	if err := compileMsg("committer_mismatch", hcfg.Messages.Templates.CommitterMismatch, &c.MsgCommitterMismatch); err != nil {
		return nil, err
	}
	if err := compileMsg("task_id_missing", hcfg.Messages.Templates.TaskIDMissing, &c.MsgTaskIDMissing); err != nil {
		return nil, err
	}

	return c, nil
}
