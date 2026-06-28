package hook

import (
	"regexp"
	"strings"

	"gitlab-task-hook/internal/config"
)

// IsRoot returns true when username is "root" (case-insensitive).
// Kept for backward compatibility; rule engine uses IsInRootBypass with the
// YAML usernames list.
func IsRoot(username string) bool {
	return IsInRootBypass(username, []string{"root"})
}

// IsInRootBypass returns true when username (case-insensitive) appears in the
// configured root bypass usernames list.
func IsInRootBypass(username string, usernames []string) bool {
	lower := strings.ToLower(username)
	for _, u := range usernames {
		if strings.ToLower(u) == lower {
			return true
		}
	}
	return false
}

// IsDeleteRef returns true when the new value is the all-zeros SHA (branch deletion).
func IsDeleteRef(newValue string) bool {
	return newValue == config.ZeroSHA
}

// IsTagRef returns true when the ref is under refs/tags/.
func IsTagRef(refName string) bool {
	return strings.HasPrefix(refName, "refs/tags/")
}

// IsNewBranch returns true when the old value is the all-zeros SHA (new branch).
func IsNewBranch(oldValue string) bool {
	return oldValue == config.ZeroSHA
}

// IsFastForward returns true when mergeBase equals oldValue.
// An empty mergeBase means no common ancestor → not fast-forward.
func IsFastForward(mergeBase, oldValue string) bool {
	return mergeBase != "" && mergeBase == oldValue
}

// IsMergeCommit returns true when the commit has 2 or more parents.
func IsMergeCommit(parentCount int) bool {
	return parentCount >= 2
}

// MatchesBranchCheckScope returns true when refName matches re.
// Returns false when re is nil (task_id rule disabled or no regex configured).
func MatchesBranchCheckScope(refName string, re *regexp.Regexp) bool {
	return re != nil && re.MatchString(refName)
}

// MatchesBranchWhitelist returns true when refName matches re.
// Returns false when re is nil.
func MatchesBranchWhitelist(refName string, re *regexp.Regexp) bool {
	return re != nil && re.MatchString(refName)
}

// HasValidTaskID returns true when subject matches the configured task ID regex.
// Returns false when re is nil.
func HasValidTaskID(subject string, re *regexp.Regexp) bool {
	return re != nil && re.MatchString(subject)
}

// MatchesMessageWhitelist returns true when subject matches re.
// Returns false when re is nil (no exempt_message_regex configured).
func MatchesMessageWhitelist(subject string, re *regexp.Regexp) bool {
	return re != nil && re.MatchString(subject)
}

// MatchesPushDenyBranch returns true when refName matches re.
// Returns false when re is nil (deny_direct_push disabled or no regex).
func MatchesPushDenyBranch(refName string, re *regexp.Regexp) bool {
	return re != nil && re.MatchString(refName)
}

// IsInProtocolList returns true when protocol appears (case-insensitive) in the list.
func IsInProtocolList(protocol string, list []string) bool {
	lower := strings.ToLower(protocol)
	for _, p := range list {
		if strings.ToLower(p) == lower {
			return true
		}
	}
	return false
}

// IsInUserWhitelist returns true when username (case-insensitive) is in whitelist.
// Whitelist entries are compared case-insensitively.
func IsInUserWhitelist(username string, whitelist []string) bool {
	lower := strings.ToLower(username)
	for _, u := range whitelist {
		if strings.ToLower(u) == lower {
			return true
		}
	}
	return false
}

// IsInProjectWhitelist returns true when repoName (case-insensitive) is in whitelist.
func IsInProjectWhitelist(repoName string, whitelist []string) bool {
	lower := strings.ToLower(repoName)
	for _, p := range whitelist {
		if strings.ToLower(p) == lower {
			return true
		}
	}
	return false
}

// ExtractEmailPrefix returns the part of an email before the '@'.
func ExtractEmailPrefix(email string) string {
	if idx := strings.Index(email, "@"); idx >= 0 {
		return email[:idx]
	}
	return email
}

// CommitterMatchesPushUser returns true when the committer email prefix matches
// glUsername (case-insensitive).
func CommitterMatchesPushUser(committerEmail, glUsername string) bool {
	return strings.EqualFold(ExtractEmailPrefix(committerEmail), glUsername)
}

// ExtractRepoName returns the last path segment of a project path.
func ExtractRepoName(projectPath string) string {
	if idx := strings.LastIndex(projectPath, "/"); idx >= 0 {
		return projectPath[idx+1:]
	}
	return projectPath
}

// ExtractBranchName returns the short branch name from a full ref.
func ExtractBranchName(refName string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(refName, prefix) {
		return refName[len(prefix):]
	}
	return refName
}
