package config

const (
	ZeroSHA = "0000000000000000000000000000000000000000"

	// BranchCheckScopeRegex matches refs/heads/feature[/_ -...] and refs/heads/dev[/_ -...]
	// The $ alternative handles bare "dev" or "feature" without suffix.
	BranchCheckScopeRegex = `^refs/heads/(feature|dev)(/|_|-|$)`

	// TaskIDRegex matches [#TSK-xxx] or [#DEF-xxx] where xxx is non-empty and contains no brackets.
	TaskIDRegex = `\[#(TSK|DEF)-[^\[\]]+\]`

	// BranchWhitelistRegex matches init/ migrate/ tmp/ prefix branches.
	BranchWhitelistRegex = `^refs/heads/(init/|migrate/|tmp/)`

	DefaultExemptMergeCommit = true
)
