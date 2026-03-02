package display

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type TableSuite struct {
	suite.Suite
}

func TestTableSuite(t *testing.T) {
	testutil.RunSuite(t, new(TableSuite))
}

func (s *TableSuite) SetupTest() {
	color.NoColor = true
}

func (s *TableSuite) TestRenderTable() {
	entries := []TableEntry{
		{Name: "frontend", Branch: "main", RemoteState: repo.InSync, Dirty: true, Staged: true},
		{Name: "backend", Branch: "feature/auth", RemoteState: repo.LocalAhead, Staged: true, LastCommit: "fix: token"},
		{Name: "infra", Branch: "main", RemoteState: repo.RemoteAhead, Untracked: true, LastCommit: "chore: bump"},
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
		{Name: "solo", Branch: "main", RemoteState: repo.NoRemote},
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
		{"detached", "HEAD", repo.Detached, "∅"},
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
		name     string
		entry    TableEntry
		expected string
	}{
		{"all false", TableEntry{}, ""},
		{"dirty only", TableEntry{Dirty: true}, "*"},
		{"staged only", TableEntry{Staged: true}, "+"},
		{"untracked only", TableEntry{Untracked: true}, "?"},
		{"stashed only", TableEntry{Stashed: true}, "$"},
		{"dirty+staged", TableEntry{Dirty: true, Staged: true}, "*+"},
		{"dirty+untracked", TableEntry{Dirty: true, Untracked: true}, "*?"},
		{"dirty+stashed", TableEntry{Dirty: true, Stashed: true}, "*$"},
		{"staged+untracked", TableEntry{Staged: true, Untracked: true}, "+?"},
		{"staged+stashed", TableEntry{Staged: true, Stashed: true}, "+$"},
		{"untracked+stashed", TableEntry{Untracked: true, Stashed: true}, "?$"},
		{"dirty+staged+untracked", TableEntry{Dirty: true, Staged: true, Untracked: true}, "*+?"},
		{"dirty+staged+stashed", TableEntry{Dirty: true, Staged: true, Stashed: true}, "*+$"},
		{"dirty+untracked+stashed", TableEntry{Dirty: true, Untracked: true, Stashed: true}, "*?$"},
		{"staged+untracked+stashed", TableEntry{Staged: true, Untracked: true, Stashed: true}, "+?$"},
		{"all true", TableEntry{Dirty: true, Staged: true, Untracked: true, Stashed: true}, "*+?$"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.expected, formatStatus(tt.entry))
		})
	}
}
