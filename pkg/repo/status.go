package repo

import (
	"bytes"
	"context"
	"strings"

	"github.com/robertwritescode/git-w/pkg/gitutil"
)

// RemoteState describes how the local branch relates to its upstream.
type RemoteState int

const (
	Unknown  RemoteState = iota // zero value; not yet determined
	Detached                    // HEAD in detached state (no branch)
	InSync
	LocalAhead
	RemoteAhead
	Diverged
	NoRemote
)

// RepoStatus holds the current state of a repository.
type RepoStatus struct {
	Branch      string
	RemoteState RemoteState
	Dirty       bool   // unstaged changes in working tree
	Staged      bool   // changes staged for commit
	Untracked   bool   // untracked files present
	Stashed     bool   // one or more stash entries exist
	LastCommit  string // subject line of HEAD commit
}

// GetStatus returns the current status of r by running git subprocesses.
// Returns an error if the directory does not exist or git cannot run.
func GetStatus(ctx context.Context, r Repo) (RepoStatus, error) {
	statusOut, err := gitutil.Output(ctx, r.AbsPath, "status", "-b", "--porcelain")
	if err != nil {
		return RepoStatus{}, err
	}

	lines := bytes.SplitN(statusOut, []byte("\n"), 2)
	branchLine := ""
	if len(lines) > 0 {
		branchLine = string(lines[0])
	}

	branch, remoteState := parseBranchLine(branchLine)
	dirty, staged, untracked := parsePorcelainV1(statusOut)

	stashOut, _ := gitutil.Output(ctx, r.AbsPath, "stash", "list")
	stashed := parseStashCount(stashOut) > 0

	logOut, _ := gitutil.Output(ctx, r.AbsPath, "log", "-1", "--format=%s")
	lastCommit := strings.TrimSpace(string(logOut))

	return RepoStatus{
		Branch:      branch,
		RemoteState: remoteState,
		Dirty:       dirty,
		Staged:      staged,
		Untracked:   untracked,
		Stashed:     stashed,
		LastCommit:  lastCommit,
	}, nil
}

func parsePorcelainV1(stdout []byte) (dirty, staged, untracked bool) {
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if isPorcelainHeader(line) {
			continue
		}

		updateStatusFlags(line[0], line[1], &dirty, &staged, &untracked)
	}

	return
}

func isPorcelainHeader(line []byte) bool {
	return len(line) < 2 || bytes.HasPrefix(line, []byte("## "))
}

func updateStatusFlags(x, y byte, dirty, staged, untracked *bool) {
	if x == '?' && y == '?' {
		*untracked = true
		return
	}

	if isStagedCode(x) {
		*staged = true
	}

	if isDirtyCode(y) {
		*dirty = true
	}
}

func isStagedCode(c byte) bool {
	return c == 'M' || c == 'A' || c == 'D' || c == 'R' || c == 'C'
}

func isDirtyCode(c byte) bool {
	return c == 'M' || c == 'D'
}

func parseBranchLine(line string) (branch string, remote RemoteState) {
	line = strings.TrimPrefix(line, "## ")

	if strings.HasPrefix(line, "HEAD (no branch)") {
		return "HEAD", Detached
	}

	if strings.HasPrefix(line, "No commits yet on ") {
		return line[len("No commits yet on "):], NoRemote
	}

	parts := strings.SplitN(line, "...", 2)
	branch = parts[0]
	if len(parts) == 1 {
		return branch, NoRemote
	}

	return branch, parseTrackingState(parts[1])
}

func parseTrackingState(tracking string) RemoteState {
	ahead := strings.Contains(tracking, "ahead")
	behind := strings.Contains(tracking, "behind")

	switch {
	case ahead && behind:
		return Diverged
	case ahead:
		return LocalAhead
	case behind:
		return RemoteAhead
	default:
		return InSync
	}
}

func parseStashCount(stdout []byte) int {
	if len(stdout) == 0 {
		return 0
	}

	count := 0
	for _, line := range bytes.Split(bytes.TrimRight(stdout, "\n"), []byte("\n")) {
		if len(line) > 0 {
			count++
		}
	}

	return count
}
