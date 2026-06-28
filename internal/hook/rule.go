package hook

import (
	"fmt"

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

// CheckRef evaluates all rules (priority 2-15) for a single ref update.
// Returns nil when the ref passes all checks.
// Returns a non-nil error only for unexpected internal failures (e.g. git command crash).
func CheckRef(ref RefUpdate, cfg env.Config, g git.Runner) (*Violation, error) {
	// Priority 2: delete ref
	if IsDeleteRef(ref.NewValue) {
		return nil, nil
	}

	// Priority 3: tag ref
	if IsTagRef(ref.RefName) {
		return nil, nil
	}

	branchName := ExtractBranchName(ref.RefName)
	repoName := ExtractRepoName(cfg.GLProjectPath)

	msgCtx := message.ViolationContext{
		ProjectPath: cfg.GLProjectPath,
		RepoName:    repoName,
		Username:    cfg.GLUsername,
		BranchName:  branchName,
	}

	// Priority 4: non fast-forward check (only when updating an existing branch)
	if !IsNewBranch(ref.OldValue) {
		mergeBase, err := g.MergeBase(ref.OldValue, ref.NewValue)
		if err != nil {
			return nil, fmt.Errorf("merge-base check for %s: %w", ref.RefName, err)
		}
		if !IsFastForward(mergeBase, ref.OldValue) {
			return &Violation{
				Type:    ViolationForcePush,
				Message: message.ForcePush(msgCtx),
			}, nil
		}
	}

	// Priority 5: web protocol — skip all subsequent push-type checks
	if IsWebProtocol(cfg.GLProtocol) {
		return nil, nil
	}

	// Priority 6: protected branch — deny direct push (only MR/web merge allowed)
	if MatchesPushDenyBranch(ref.RefName, cfg.PushDenyBranchRegex) {
		return &Violation{
			Type:    ViolationProtectedBranchDirect,
			Message: message.ProtectedBranchDirect(msgCtx),
		}, nil
	}

	// Priority 7: compute new commits
	commits, err := g.NewCommits(ref.NewValue)
	if err != nil {
		return nil, fmt.Errorf("rev-list for %s: %w", ref.RefName, err)
	}

	for _, commit := range commits {
		v, err := checkCommit(commit, ref.RefName, msgCtx, cfg, g)
		if err != nil {
			return nil, err
		}
		if v != nil {
			return v, nil
		}
	}

	return nil, nil
}

// checkCommit evaluates rules priority 8-15 for a single new commit.
func checkCommit(
	commit, refName string,
	msgCtx message.ViolationContext,
	cfg env.Config,
	g git.Runner,
) (*Violation, error) {
	// Priority 8: merge commit exemption
	if cfg.ExemptMergeCommit {
		parentCount, err := g.ParentCount(commit)
		if err != nil {
			return nil, fmt.Errorf("parent-count for %s: %w", commit, err)
		}
		if IsMergeCommit(parentCount) {
			return nil, nil
		}
	}

	// Priority 9: committer vs push user
	committerEmail, err := g.CommitterEmail(commit)
	if err != nil {
		return nil, fmt.Errorf("committer email for %s: %w", commit, err)
	}
	if !CommitterMatchesPushUser(committerEmail, cfg.GLUsername) {
		ctx := msgCtx
		ctx.CommitID = commit
		ctx.CommitterEmail = committerEmail
		return &Violation{
			Type:    ViolationCommitterMismatch,
			Message: message.CommitterMismatch(ctx),
		}, nil
	}

	// Priority 10: is branch in task-check scope?
	if !MatchesBranchCheckScope(refName) {
		return nil, nil
	}

	// Priority 11: user whitelist → skip task check
	if IsInUserWhitelist(cfg.GLUsername, cfg.WhitelistUsers) {
		return nil, nil
	}

	// Priority 12: branch whitelist → skip task check
	if MatchesBranchWhitelist(refName) {
		return nil, nil
	}

	// Priority 13: project whitelist → skip task check
	repoName := ExtractRepoName(cfg.GLProjectPath)
	if IsInProjectWhitelist(repoName, cfg.WhitelistProjectNames) {
		return nil, nil
	}

	// Priority 14: message whitelist → skip task check
	subject, err := g.CommitSubject(commit)
	if err != nil {
		return nil, fmt.Errorf("commit subject for %s: %w", commit, err)
	}
	if MatchesMessageWhitelist(subject, cfg.ExemptMessageRegex) {
		return nil, nil
	}

	// Priority 15: task ID check
	if !HasValidTaskID(subject) {
		ctx := msgCtx
		ctx.CommitID = commit
		ctx.CommitSubject = subject
		return &Violation{
			Type:    ViolationMissingTaskID,
			Message: message.MissingTaskID(ctx),
		}, nil
	}

	return nil, nil
}
