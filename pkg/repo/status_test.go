package repo

import (
	"context"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
)

type StatusSuite struct {
	testutil.CmdSuite
}

func TestStatusSuite(t *testing.T) {
	testutil.RunSuite(t, new(StatusSuite))
}

func (s *StatusSuite) TestParsePorcelainV1() {
	tests := []struct {
		name      string
		input     string
		dirty     bool
		staged    bool
		untracked bool
	}{
		{"clean", "", false, false, false},
		{"dirty only", " M file.go\n", true, false, false},
		{"staged only", "M  file.go\n", false, true, false},
		{"untracked only", "?? newfile.go\n", false, false, true},
		{"staged and dirty", "MM file.go\n", true, true, false},
		{"all three", "MM a.go\n?? b.go\n", true, true, true},
		{"new file staged", "A  newfile.go\n", false, true, false},
		{"deleted unstaged", " D file.go\n", true, false, false},
		{"renamed staged", "R  old.go -> new.go\n", false, true, false},
		{"branch line skipped", "## main\n M file.go\n", true, false, false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dirty, staged, untracked := parsePorcelainV1([]byte(tt.input))
			s.Assert().Equal(tt.dirty, dirty, "dirty")
			s.Assert().Equal(tt.staged, staged, "staged")
			s.Assert().Equal(tt.untracked, untracked, "untracked")
		})
	}
}

func (s *StatusSuite) TestParseBranchLine() {
	tests := []struct {
		name       string
		input      string
		wantBranch string
		wantRemote RemoteState
	}{
		{"in sync", "## main...origin/main", "main", InSync},
		{"local ahead", "## main...origin/main [ahead 2]", "main", LocalAhead},
		{"remote ahead", "## main...origin/main [behind 3]", "main", RemoteAhead},
		{"diverged", "## main...origin/main [ahead 1, behind 1]", "main", Diverged},
		{"no remote", "## main", "main", NoRemote},
		{"feature branch", "## feature/auth...origin/feature/auth", "feature/auth", InSync},
		{"detached HEAD", "## HEAD (no branch)", "HEAD", Detached},
		{"fresh repo", "## No commits yet on main", "main", NoRemote},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			branch, remote := parseBranchLine(tt.input)
			s.Assert().Equal(tt.wantBranch, branch, "branch")
			s.Assert().Equal(tt.wantRemote, remote, "remote")
		})
	}
}

func (s *StatusSuite) TestParseStashCount() {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"one entry", "stash@{0}: WIP on main: abc\n", 1},
		{"three entries", "stash@{0}: a\nstash@{1}: b\nstash@{2}: c\n", 3},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.want, parseStashCount([]byte(tt.input)))
		})
	}
}

func (s *StatusSuite) TestGetStatus_Smoke() {
	dir := s.MakeGitRepo("")
	status, err := GetStatus(context.Background(), Repo{Name: "x", AbsPath: dir})
	s.Require().NoError(err)

	s.Assert().NotEmpty(status.Branch)
	s.Assert().Equal(NoRemote, status.RemoteState)
	s.Assert().False(status.Dirty)
	s.Assert().False(status.Staged)
	s.Assert().False(status.Untracked)
	s.Assert().Contains(status.LastCommit, "init")
}
