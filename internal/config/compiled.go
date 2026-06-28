package config

import (
	"fmt"
	"regexp"
)

// CompiledConfig wraps HookConfig with pre-compiled regular expressions.
// All regexes are compiled once at load time; nil means the rule/field is
// disabled or has no pattern configured.
type CompiledConfig struct {
	*HookConfig
	DenyDirectPushRegex   *regexp.Regexp
	BranchCheckScopeRegex *regexp.Regexp
	TaskIDRegex           *regexp.Regexp
	BranchWhitelistRegex  *regexp.Regexp
	ExemptMessageRegex    *regexp.Regexp
}

// Compile validates and pre-compiles all regex fields in hcfg.
// Returns an error if any regex fails to compile (validation should have
// already caught these, but Compile is the authoritative compile step).
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

	return c, nil
}
