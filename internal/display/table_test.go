package display

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/robertwritescode/git-workspace/internal/repo"
	"github.com/stretchr/testify/suite"
)

type TableSuite struct {
	suite.Suite
}

func TestTableSuite(t *testing.T) {
	suite.Run(t, new(TableSuite))
}

func (s *TableSuite) SetupTest() {
	color.NoColor = true
}

func (s *TableSuite) TestRenderTable() {
	entries := []TableEntry{
		{Name: "frontend", Status: repo.RepoStatus{Branch: "main", RemoteState: repo.InSync, Dirty: true, Staged: true}},
		{Name: "backend", Status: repo.RepoStatus{Branch: "feature/auth", RemoteState: repo.LocalAhead, Staged: true, LastCommit: "fix: token"}},
		{Name: "infra", Status: repo.RepoStatus{Branch: "main", RemoteState: repo.RemoteAhead, Untracked: true, LastCommit: "chore: bump"}},
	}

	buf := &bytes.Buffer{}
	RenderTable(buf, entries)
	out := buf.String()

	s.Run("header present", func() {
		s.Assert().Contains(out, "REPO")
		s.Assert().Contains(out, "BRANCH")
		s.Assert().Contains(out, "STATUS")
		s.Assert().Contains(out, "COMMIT")
	})

	s.Run("repo names appear", func() {
		s.Assert().Contains(out, "frontend")
		s.Assert().Contains(out, "backend")
		s.Assert().Contains(out, "infra")
	})

	s.Run("status symbols", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		s.Require().Len(lines, 4)
		s.Assert().Contains(lines[1], "*+") // frontend: dirty+staged
		s.Assert().Contains(lines[2], "+")  // backend: staged
		s.Assert().Contains(lines[3], "?")  // infra: untracked
	})

	s.Run("sync symbols", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		s.Require().Len(lines, 4)
		s.Assert().Contains(lines[1], "✓") // frontend: InSync
		s.Assert().Contains(lines[2], "↑") // backend: LocalAhead
		s.Assert().Contains(lines[3], "↓") // infra: RemoteAhead
	})

	s.Run("column alignment", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		s.Require().Len(lines, 4)
		for _, line := range lines {
			s.Assert().Contains(line, "  ")
		}
	})
}

func (s *TableSuite) TestRenderTable_Empty() {
	buf := &bytes.Buffer{}
	RenderTable(buf, []TableEntry{})
	out := buf.String()
	s.Assert().Contains(out, "REPO")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	s.Assert().Len(lines, 1)
}

func (s *TableSuite) TestRenderTable_SingleEntry() {
	entries := []TableEntry{
		{Name: "solo", Status: repo.RepoStatus{Branch: "main", RemoteState: repo.NoRemote}},
	}
	buf := &bytes.Buffer{}
	RenderTable(buf, entries)
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	s.Assert().Len(lines, 2)
	s.Assert().Contains(lines[0], "REPO")
	s.Assert().Contains(lines[1], "solo")
}

func (s *TableSuite) TestFormatBranch() {
	tests := []struct {
		name    string
		branch  string
		state   repo.RemoteState
		wantSym string
	}{
		{"in sync", "main", repo.InSync, "✓"},
		{"local ahead", "main", repo.LocalAhead, "↑"},
		{"remote ahead", "main", repo.RemoteAhead, "↓"},
		{"diverged", "main", repo.Diverged, "⇕"},
		{"no remote", "main", repo.NoRemote, "∅"},
		{"unknown", "HEAD", repo.RemoteUnknown, "∅"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := formatBranch(tt.branch, tt.state)
			s.Assert().Contains(result, tt.wantSym)
			s.Assert().Contains(result, tt.branch)
		})
	}
}

func (s *TableSuite) TestFormatStatus() {
	tests := []struct {
		name      string
		status    repo.RepoStatus
		expected  string
	}{
		{"all false", repo.RepoStatus{}, ""},
		{"dirty only", repo.RepoStatus{Dirty: true}, "*"},
		{"staged only", repo.RepoStatus{Staged: true}, "+"},
		{"untracked only", repo.RepoStatus{Untracked: true}, "?"},
		{"stashed only", repo.RepoStatus{Stashed: true}, "$"},
		{"dirty+staged", repo.RepoStatus{Dirty: true, Staged: true}, "*+"},
		{"dirty+untracked", repo.RepoStatus{Dirty: true, Untracked: true}, "*?"},
		{"dirty+stashed", repo.RepoStatus{Dirty: true, Stashed: true}, "*$"},
		{"staged+untracked", repo.RepoStatus{Staged: true, Untracked: true}, "+?"},
		{"staged+stashed", repo.RepoStatus{Staged: true, Stashed: true}, "+$"},
		{"untracked+stashed", repo.RepoStatus{Untracked: true, Stashed: true}, "?$"},
		{"dirty+staged+untracked", repo.RepoStatus{Dirty: true, Staged: true, Untracked: true}, "*+?"},
		{"dirty+staged+stashed", repo.RepoStatus{Dirty: true, Staged: true, Stashed: true}, "*+$"},
		{"dirty+untracked+stashed", repo.RepoStatus{Dirty: true, Untracked: true, Stashed: true}, "*?$"},
		{"staged+untracked+stashed", repo.RepoStatus{Staged: true, Untracked: true, Stashed: true}, "+?$"},
		{"all true", repo.RepoStatus{Dirty: true, Staged: true, Untracked: true, Stashed: true}, "*+?$"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, formatStatus(tt.status))
		})
	}
}
