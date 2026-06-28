package config

// HookConfig is the top-level structure for the YAML config file.
type HookConfig struct {
	Version   string    `yaml:"version"`
	Enabled   bool      `yaml:"enabled"`
	Mode      ModeConf  `yaml:"mode"`
	Rules     Rules     `yaml:"rules"`
	Whitelist Whitelist `yaml:"whitelist"`
	Messages  Messages  `yaml:"messages"`
	Logging   LogConf   `yaml:"logging"`
}

type ModeConf struct {
	Default string `yaml:"default"` // "enforce" | "warn"
}

type Rules struct {
	RootBypass             RootBypassRule             `yaml:"root_bypass"`
	NonFastForward         NonFastForwardRule         `yaml:"non_fast_forward"`
	DenyDirectPush         DenyDirectPushRule         `yaml:"deny_direct_push"`
	WebBypassPushChecks    WebBypassPushChecksRule    `yaml:"web_bypass_push_checks"`
	CommitterMatchPushUser CommitterMatchPushUserRule `yaml:"committer_match_push_user"`
	TaskID                 TaskIDRule                 `yaml:"task_id"`
}

type RootBypassRule struct {
	Enabled   bool     `yaml:"enabled"`
	Usernames []string `yaml:"usernames"`
}

type NonFastForwardRule struct {
	Enabled bool `yaml:"enabled"`
}

type DenyDirectPushRule struct {
	Enabled        bool     `yaml:"enabled"`
	BranchRegex    string   `yaml:"branch_regex"`
	AllowProtocols []string `yaml:"allow_protocols"`
	DenyProtocols  []string `yaml:"deny_protocols"`
}

type WebBypassPushChecksRule struct {
	Enabled   bool     `yaml:"enabled"`
	Protocols []string `yaml:"protocols"`
}

type CommitterMatchPushUserRule struct {
	Enabled         bool   `yaml:"enabled"`
	CompareStrategy string `yaml:"compare_strategy"` // "email_prefix"
	SkipMergeCommit bool   `yaml:"skip_merge_commit"`
	CaseInsensitive bool   `yaml:"case_insensitive"`
}

type TaskIDRule struct {
	Enabled            bool   `yaml:"enabled"`
	BranchRegex        string `yaml:"branch_regex"`
	SubjectRegex       string `yaml:"subject_regex"`
	CheckSubjectOnly   bool   `yaml:"check_subject_only"`
	ExemptMergeCommit  bool   `yaml:"exempt_merge_commit"`
	ExemptMessageRegex string `yaml:"exempt_message_regex"`
}

type Whitelist struct {
	Users       []string `yaml:"users"`
	BranchRegex string   `yaml:"branch_regex"`
	Projects    []string `yaml:"projects"`
}

type Messages struct {
	Language     string `yaml:"language"`
	ShowFixGuide bool   `yaml:"show_fix_guide"`
}

type LogConf struct {
	Level  string `yaml:"level"`
	Stderr bool   `yaml:"stderr"`
}
