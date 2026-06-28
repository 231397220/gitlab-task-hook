package hook

import (
	"regexp"
	"strings"

	"gitlab-task-hook/internal/config"
)

var (
	reBranchCheckScope = regexp.MustCompile(config.BranchCheckScopeRegex)
	reTaskID           = regexp.MustCompile(config.TaskIDRegex)
	reBranchWhitelist  = regexp.MustCompile(config.BranchWhitelistRegex)
)

// IsRoot returns true when the username equals "root" (case-insensitive).
func IsRoot(username string) bool {
	return strings.EqualFold(username, "root")
}

// IsDeleteRef returns true when the new value is the all-zeros SHA (branch deletion).
func IsDeleteRef(newValue string) bool {
	return newValue == config.ZeroSHA
}

// IsTagRef returns true when the ref is under refs/tags/.
func IsTagRef(refName string) bool {
	return strings.HasPrefix(refName, "refs/tags/")
}

// IsNewBranch returns true when the old value is the all-zeros SHA (new branch creation).
func IsNewBranch(oldValue string) bool {
	return oldValue == config.ZeroSHA
}

// IsFastForward returns true when mergeBase equals oldValue (i.e. the push is a fast-forward).
// An empty mergeBase means no common ancestor → not fast-forward.
func IsFastForward(mergeBase, oldValue string) bool {
	return mergeBase != "" && mergeBase == oldValue
}

// IsWebProtocol returns true when GL_PROTOCOL is "web".
func IsWebProtocol(protocol string) bool {
	return strings.EqualFold(protocol, "web")
}

// IsMergeCommit returns true when the commit has 2 or more parents.
func IsMergeCommit(parentCount int) bool {
	return parentCount >= 2
}

// MatchesBranchCheckScope returns true when the ref falls within the task-ID check scope.
func MatchesBranchCheckScope(refName string) bool {
	return reBranchCheckScope.MatchString(refName)
}

// MatchesBranchWhitelist returns true when the ref matches the branch whitelist.
func MatchesBranchWhitelist(refName string) bool {
	return reBranchWhitelist.MatchString(refName)
}

// HasValidTaskID returns true when the commit subject contains a valid task ID.
func HasValidTaskID(subject string) bool {
	return reTaskID.MatchString(subject)
}

// IsInUserWhitelist returns true when username (case-insensitive) is found in whitelist.
// whitelist entries are expected to already be lowercased.
func IsInUserWhitelist(username string, whitelist []string) bool {
	lower := strings.ToLower(username)
	for _, u := range whitelist {
		if u == lower {
			return true
		}
	}
	return false
}

// IsInProjectWhitelist returns true when repoName (case-insensitive) is found in whitelist.
// whitelist entries are expected to already be lowercased.
func IsInProjectWhitelist(repoName string, whitelist []string) bool {
	lower := strings.ToLower(repoName)
	for _, p := range whitelist {
		if p == lower {
			return true
		}
	}
	return false
}

// MatchesMessageWhitelist returns true when subject matches the given regex pattern.
// Returns false (no exemption) when pattern is empty or invalid.
func MatchesMessageWhitelist(subject, pattern string) bool {
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(subject)
}

// ExtractEmailPrefix returns the part of an email before the '@'.
// Returns the full string if there is no '@'.
func ExtractEmailPrefix(email string) string {
	if idx := strings.Index(email, "@"); idx >= 0 {
		return email[:idx]
	}
	return email
}

// CommitterMatchesPushUser returns true when the committer email prefix matches glUsername.
// Comparison is case-insensitive.
func CommitterMatchesPushUser(committerEmail, glUsername string) bool {
	prefix := ExtractEmailPrefix(committerEmail)
	return strings.EqualFold(prefix, glUsername)
}

// ExtractRepoName returns the last path segment of a project path.
// e.g. "group/subgroup/demo-service" → "demo-service"
func ExtractRepoName(projectPath string) string {
	if idx := strings.LastIndex(projectPath, "/"); idx >= 0 {
		return projectPath[idx+1:]
	}
	return projectPath
}

// ExtractBranchName returns the short branch name from a full ref.
// e.g. "refs/heads/dev/login" → "dev/login"
func ExtractBranchName(refName string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(refName, prefix) {
		return refName[len(prefix):]
	}
	return refName
}
