package hook

import (
	"fmt"

	"gitlab-task-hook/internal/config"
	"gitlab-task-hook/internal/env"
	"gitlab-task-hook/internal/git"
	"gitlab-task-hook/internal/message"
)

// ViolationType identifies which rule was violated.
type ViolationType int

const (
	ViolationForcePush ViolationType = iota + 1
	ViolationProtectedBranchDirect
	ViolationCommitterMismatch
	ViolationMissingTaskID
)

// Violation holds a detected rule violation and its rendered message.
type Violation struct {
	Type    ViolationType
	Message string
}

// CheckRef evaluates rules for a single ref update against the YAML-based config.
// Rule execution order matches the fixed order defined in the requirements (§12).
// Returns nil when the ref passes all enabled checks.
// Returns a non-nil error only for unexpected internal failures (e.g. git command crash).
func CheckRef(ref RefUpdate, e env.Config, hcfg *config.CompiledConfig, g git.Runner) (*Violation, error) {
	// Rule 1: delete ref → pass
	if IsDeleteRef(ref.NewValue) {
		return nil, nil
	}

	// Rule 2: tag ref → pass
	if IsTagRef(ref.RefName) {
		return nil, nil
	}

	branchName := ExtractBranchName(ref.RefName)
	repoName := ExtractRepoName(e.GLProjectPath)

	msgCtx := message.ViolationContext{
		ProjectPath: e.GLProjectPath,
		RepoName:    repoName,
		Username:    e.GLUsername,
		BranchName:  branchName,
		Protocol:    e.GLProtocol,
	}

	// Rule 3: non fast-forward (only when updating an existing branch)
	if hcfg.Rules.NonFastForward.Enabled && !IsNewBranch(ref.OldValue) {
		mergeBase, err := g.MergeBase(ref.OldValue, ref.NewValue)
		if err != nil {
			return nil, fmt.Errorf("merge-base check for %s: %w", ref.RefName, err)
		}
		if !IsFastForward(mergeBase, ref.OldValue) {
			return &Violation{
				Type:    ViolationForcePush,
				Message: message.NonFastForwardMessage(hcfg.MsgNonFastForward, msgCtx),
			}, nil
		}
	}

	// Rule 4: deny direct push to protected branches
	if hcfg.Rules.DenyDirectPush.Enabled &&
		MatchesPushDenyBranch(ref.RefName, hcfg.DenyDirectPushRegex) &&
		IsInProtocolList(e.GLProtocol, hcfg.Rules.DenyDirectPush.DenyProtocols) {
		return &Violation{
			Type:    ViolationProtectedBranchDirect,
			Message: message.DirectPushDeniedMessage(hcfg.MsgDirectPushDenied, msgCtx),
		}, nil
	}

	// Rule 5: web bypass — skip all subsequent push-type checks
	if hcfg.Rules.WebBypassPushChecks.Enabled &&
		IsInProtocolList(e.GLProtocol, hcfg.Rules.WebBypassPushChecks.Protocols) {
		return nil, nil
	}

	// Rule 6: compute new commits
	commits, err := g.NewCommits(ref.NewValue)
	if err != nil {
		return nil, fmt.Errorf("rev-list for %s: %w", ref.RefName, err)
	}

	for _, commit := range commits {
		v, err := checkCommit(commit, ref.RefName, msgCtx, e, hcfg, g)
		if err != nil {
			return nil, err
		}
		if v != nil {
			return v, nil
		}
	}

	return nil, nil
}

// checkCommit evaluates per-commit rules (committer match + task ID).
func checkCommit(
	commit, refName string,
	msgCtx message.ViolationContext,
	e env.Config,
	hcfg *config.CompiledConfig,
	g git.Runner,
) (*Violation, error) {
	// Rule 7: committer vs push user
	if hcfg.Rules.CommitterMatchPushUser.Enabled {
		isMerge := false
		if hcfg.Rules.CommitterMatchPushUser.SkipMergeCommit {
			parentCount, err := g.ParentCount(commit)
			if err != nil {
				return nil, fmt.Errorf("parent-count for %s: %w", commit, err)
			}
			isMerge = IsMergeCommit(parentCount)
		}
		if !isMerge {
			committerEmail, err := g.CommitterEmail(commit)
			if err != nil {
				return nil, fmt.Errorf("committer email for %s: %w", commit, err)
			}
			if !CommitterMatchesPushUser(committerEmail, e.GLUsername) {
				ctx := msgCtx
				ctx.CommitID = commit
				ctx.CommitterEmail = committerEmail
				return &Violation{
					Type:    ViolationCommitterMismatch,
					Message: message.CommitterMismatchMessage(hcfg.MsgCommitterMismatch, ctx),
				}, nil
			}
		}
	}

	// Rule 8: is branch in task-check scope?
	if !hcfg.Rules.TaskID.Enabled || !MatchesBranchCheckScope(refName, hcfg.BranchCheckScopeRegex) {
		return nil, nil
	}

	// Rule 9: user whitelist → skip task check
	if IsInUserWhitelist(e.GLUsername, hcfg.Whitelist.Users) {
		return nil, nil
	}

	// Rule 10: branch whitelist → skip task check
	if MatchesBranchWhitelist(refName, hcfg.BranchWhitelistRegex) {
		return nil, nil
	}

	// Rule 11: project whitelist → skip task check
	if IsInProjectWhitelist(ExtractRepoName(e.GLProjectPath), hcfg.Whitelist.Projects) {
		return nil, nil
	}

	// Rule 12: task_id exempt_merge_commit
	if hcfg.Rules.TaskID.ExemptMergeCommit {
		parentCount, err := g.ParentCount(commit)
		if err != nil {
			return nil, fmt.Errorf("parent-count for %s: %w", commit, err)
		}
		if IsMergeCommit(parentCount) {
			return nil, nil
		}
	}

	// Rule 13: message whitelist → skip task check
	subject, err := g.CommitSubject(commit)
	if err != nil {
		return nil, fmt.Errorf("commit subject for %s: %w", commit, err)
	}
	if MatchesMessageWhitelist(subject, hcfg.ExemptMessageRegex) {
		return nil, nil
	}

	// Rule 14: task ID check
	if !HasValidTaskID(subject, hcfg.TaskIDRegex) {
		ctx := msgCtx
		ctx.CommitID = commit
		ctx.CommitSubject = subject
		return &Violation{
			Type:    ViolationMissingTaskID,
			Message: message.TaskIDMissingMessage(hcfg.MsgTaskIDMissing, ctx),
		}, nil
	}

	return nil, nil
}
