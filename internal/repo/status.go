package repo

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// RemoteState describes how the local branch relates to its upstream.
type RemoteState int

const (
	RemoteUnknown RemoteState = iota
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
func GetStatus(r Repo) (RepoStatus, error) {
	statusOut, err := gitOutput(r.AbsPath, "status", "-b", "--porcelain")
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

	stashOut, _ := gitOutput(r.AbsPath, "stash", "list")
	stashed := parseStashCount(stashOut) > 0

	logOut, _ := gitOutput(r.AbsPath, "log", "-1", "--format=%s")
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

func gitOutput(repoPath string, args ...string) ([]byte, error) {
	out, err := exec.Command("git", append([]string{"-C", repoPath}, args...)...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.Stderr, err
		}
		return nil, err
	}
	return out, nil
}

func parsePorcelainV1(stdout []byte) (dirty, staged, untracked bool) {
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(line) < 2 {
			continue
		}

		if bytes.HasPrefix(line, []byte("## ")) {
			continue
		}

		x, y := line[0], line[1]
		if x == '?' && y == '?' {
			untracked = true
		} else {
			if x == 'M' || x == 'A' || x == 'D' || x == 'R' || x == 'C' {
				staged = true
			}
			if y == 'M' || y == 'D' {
				dirty = true
			}
		}

		if dirty && staged && untracked {
			return
		}
	}
	return
}

func parseBranchLine(line string) (branch string, remote RemoteState) {
	line = strings.TrimPrefix(line, "## ")

	if strings.HasPrefix(line, "HEAD (no branch)") {
		return "HEAD", RemoteUnknown
	}

	if strings.HasPrefix(line, "No commits yet on ") {
		return line[len("No commits yet on "):], NoRemote
	}

	parts := strings.SplitN(line, "...", 2)
	branch = parts[0]
	if len(parts) == 1 {
		return branch, NoRemote
	}

	tracking := parts[1]
	ahead := strings.Contains(tracking, "ahead")
	behind := strings.Contains(tracking, "behind")

	switch {
	case ahead && behind:
		return branch, Diverged
	case ahead:
		return branch, LocalAhead
	case behind:
		return branch, RemoteAhead
	default:
		return branch, InSync
	}
}

func parseStashCount(stdout []byte) int {
	if len(stdout) == 0 {
		return 0
	}
	return bytes.Count(stdout, []byte("\n"))
}
