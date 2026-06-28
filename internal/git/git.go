package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner defines the git operations needed by the hook rules.
type Runner interface {
	// MergeBase returns the merge-base of old and new, or ("", nil) when there is none.
	MergeBase(oldSHA, newSHA string) (string, error)
	// NewCommits returns commits reachable from newSHA but not from any existing ref.
	NewCommits(newSHA string) ([]string, error)
	// ParentCount returns the number of parents of a commit (2+ means merge commit).
	ParentCount(commit string) (int, error)
	// CommitSubject returns the first line (subject) of the commit message.
	CommitSubject(commit string) (string, error)
	// CommitterEmail returns the committer email of a commit.
	CommitterEmail(commit string) (string, error)
}

// realRunner executes real git commands in the current working directory.
type realRunner struct{}

// New returns a Runner that executes real git commands.
func New() Runner {
	return &realRunner{}
}

func (r *realRunner) MergeBase(oldSHA, newSHA string) (string, error) {
	out, err := runGit("merge-base", oldSHA, newSHA)
	if err != nil {
		// git merge-base exits 1 when there is no common ancestor; treat as empty.
		if isExitCode(err, 1) {
			return "", nil
		}
		return "", fmt.Errorf("git merge-base: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (r *realRunner) NewCommits(newSHA string) ([]string, error) {
	out, err := runGit("rev-list", newSHA, "--not", "--all")
	if err != nil {
		return nil, fmt.Errorf("git rev-list: %w", err)
	}
	return splitLines(out), nil
}

func (r *realRunner) ParentCount(commit string) (int, error) {
	// --parents prints: <commit> <parent1> <parent2> ...
	out, err := runGit("rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return 0, fmt.Errorf("git rev-list --parents: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) == 0 {
		return 0, nil
	}
	// fields[0] is the commit itself; remaining are parents.
	return len(fields) - 1, nil
}

func (r *realRunner) CommitSubject(commit string) (string, error) {
	out, err := runGit("log", "--pretty=format:%s", "-1", commit)
	if err != nil {
		return "", fmt.Errorf("git log (subject): %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (r *realRunner) CommitterEmail(commit string) (string, error) {
	out, err := runGit("log", "--pretty=format:%ce", "-1", commit)
	if err != nil {
		return "", fmt.Errorf("git log (committer email): %w", err)
	}
	return strings.TrimSpace(out), nil
}

// runGit executes a git command and returns combined stdout. stderr is discarded
// to avoid leaking internal git messages into the hook output.
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func isExitCode(err error, code int) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == code
	}
	return false
}

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
