package config

// DefaultConfig returns the built-in minimal safe configuration used when
// the local cache file is absent. Only non_fast_forward is enabled so that
// the hook can start safely before Nacos provides a full config.
func DefaultConfig() *HookConfig {
	return &HookConfig{
		Version: "1.0",
		Enabled: true,
		Mode:    ModeConf{Default: "enforce"},
		Rules: Rules{
			RootBypass: RootBypassRule{
				Enabled:   true,
				Usernames: []string{"root"},
			},
			NonFastForward: NonFastForwardRule{Enabled: true},
			DenyDirectPush: DenyDirectPushRule{
				Enabled:        true,
				BranchRegex:    `(?i)^refs/heads/(master|sit_.*|uat_.*)$`,
				AllowProtocols: []string{"web"},
				DenyProtocols:  []string{"http", "ssh", ""},
			},
			WebBypassPushChecks: WebBypassPushChecksRule{
				Enabled:   true,
				Protocols: []string{"web"},
			},
			CommitterMatchPushUser: CommitterMatchPushUserRule{
				Enabled:         true,
				CompareStrategy: "email_prefix",
				SkipMergeCommit: true,
				CaseInsensitive: true,
			},
			TaskID: TaskIDRule{
				Enabled:           true,
				BranchRegex:       `(?i)^refs/heads/(feature|dev)(/|_|-|$)`,
				SubjectRegex:      `\[#(TSK|DEF)-[^\[\]]+\]`,
				CheckSubjectOnly:  true,
				ExemptMergeCommit: true,
			},
		},
		Whitelist: Whitelist{
			BranchRegex: `(?i)^refs/heads/(init/|migrate/|tmp/)`,
		},
		Messages: Messages{
			Language:     "zh-CN",
			ShowFixGuide: true,
		},
		Logging: LogConf{
			Level:  "info",
			Stderr: true,
		},
	}
}
