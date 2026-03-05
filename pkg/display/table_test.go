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

func (s *TableSuite) TestSplitBySet() {
	tests := []struct {
		name              string
		entries           []TableEntry
		sets              []WorktreeSet
		wantStandaloneLen int
		wantGroupedSets   map[string]int
	}{
		{
			name:              "no sets",
			entries:           []TableEntry{{Name: "a"}, {Name: "b"}},
			sets:              []WorktreeSet{},
			wantStandaloneLen: 2,
			wantGroupedSets:   map[string]int{},
		},
		{
			name:    "one set with matching entries",
			entries: []TableEntry{{Name: "infra-dev"}, {Name: "infra-prod"}, {Name: "service-a"}},
			sets: []WorktreeSet{
				{SetName: "infra", Branches: []string{"dev", "prod"}},
			},
			wantStandaloneLen: 1,
			wantGroupedSets:   map[string]int{"infra": 2},
		},
		{
			name:              "set with no matching entries",
			entries:           []TableEntry{{Name: "service-a"}},
			sets:              []WorktreeSet{{SetName: "infra", Branches: []string{"dev"}}},
			wantStandaloneLen: 1,
			wantGroupedSets:   map[string]int{},
		},
		{
			name:    "mixed entries and multiple sets",
			entries: []TableEntry{{Name: "platform-staging"}, {Name: "platform-release"}, {Name: "infra-dev"}, {Name: "service-a"}},
			sets: []WorktreeSet{
				{SetName: "platform", Branches: []string{"staging", "release"}},
				{SetName: "infra", Branches: []string{"dev"}},
			},
			wantStandaloneLen: 1,
			wantGroupedSets:   map[string]int{"platform": 2, "infra": 1},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			standalone, grouped := splitBySet(tt.entries, tt.sets)
			s.Assert().Len(standalone, tt.wantStandaloneLen)
			s.Assert().Len(grouped, len(tt.wantGroupedSets))
			for setName, count := range tt.wantGroupedSets {
				s.Assert().Len(grouped[setName], count)
			}
		})
	}
}

func (s *TableSuite) TestBuildGroupedRows() {
	tests := []struct {
		name               string
		standalone         []TableEntry
		grouped            map[string][]TableEntry
		sets               []WorktreeSet
		wantHeaderCount    int
		wantRowCount       int
		wantSetHeaders     []string
		wantBranchPrefixes []string
	}{
		{
			name:            "no sets, only standalone",
			standalone:      []TableEntry{{Name: "a"}, {Name: "b"}},
			grouped:         map[string][]TableEntry{},
			sets:            []WorktreeSet{},
			wantHeaderCount: 0,
			wantRowCount:    2,
		},
		{
			name:       "one set with two entries",
			standalone: []TableEntry{{Name: "service-a"}},
			grouped: map[string][]TableEntry{
				"infra": {{Name: "infra-dev"}, {Name: "infra-prod"}},
			},
			sets: []WorktreeSet{
				{SetName: "infra", Branches: []string{"dev", "prod"}},
			},
			wantHeaderCount:    1,
			wantRowCount:       4,
			wantSetHeaders:     []string{"infra"},
			wantBranchPrefixes: []string{"  └ dev", "  └ prod"},
		},
		{
			name:       "two sets with mixed entries",
			standalone: []TableEntry{{Name: "service-a"}},
			grouped: map[string][]TableEntry{
				"infra":    {{Name: "infra-dev"}},
				"platform": {{Name: "platform-staging"}, {Name: "platform-release"}},
			},
			sets: []WorktreeSet{
				{SetName: "infra", Branches: []string{"dev"}},
				{SetName: "platform", Branches: []string{"staging", "release"}},
			},
			wantHeaderCount: 2,
			wantRowCount:    6,
			wantSetHeaders:  []string{"infra", "platform"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rows := buildGroupedRows(tt.standalone, tt.grouped, tt.sets)
			s.Assert().Len(rows, tt.wantRowCount)

			headerCount := 0
			for _, r := range rows {
				if r.isSetHeader {
					headerCount++
				}
			}
			s.Assert().Equal(tt.wantHeaderCount, headerCount)

			if len(tt.wantSetHeaders) > 0 {
				found := 0
				for _, r := range rows {
					if r.isSetHeader {
						for _, wantHeader := range tt.wantSetHeaders {
							if r.name == wantHeader {
								found++
								break
							}
						}
					}
				}
				s.Assert().Equal(len(tt.wantSetHeaders), found)
			}

			if len(tt.wantBranchPrefixes) > 0 {
				branchPrefixCount := 0
				for _, r := range rows {
					for _, prefix := range tt.wantBranchPrefixes {
						if strings.HasPrefix(r.name, prefix) {
							branchPrefixCount++
							break
						}
					}
				}
				s.Assert().Equal(len(tt.wantBranchPrefixes), branchPrefixCount)
			}
		})
	}
}

func (s *TableSuite) TestRenderGroupedTable_NoSets() {
	tests := []struct {
		name    string
		entries []TableEntry
	}{
		{"empty", []TableEntry{}},
		{"single entry", []TableEntry{{Name: "a", Branch: "main", RemoteState: repo.InSync}}},
		{"multiple entries", []TableEntry{
			{Name: "a", Branch: "main", RemoteState: repo.InSync},
			{Name: "b", Branch: "dev", RemoteState: repo.LocalAhead},
		}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			buf1 := &bytes.Buffer{}
			RenderTable(buf1, tt.entries)
			flatOut := buf1.String()

			buf2 := &bytes.Buffer{}
			RenderGroupedTable(buf2, tt.entries, []WorktreeSet{})
			groupedOut := buf2.String()

			s.Assert().Equal(flatOut, groupedOut)
		})
	}
}

func (s *TableSuite) TestRenderGroupedTable_WithSets() {
	entries := []TableEntry{
		{Name: "service-a", Branch: "main", RemoteState: repo.InSync},
		{Name: "infra-dev", Branch: "dev", RemoteState: repo.InSync},
		{Name: "infra-prod", Branch: "prod", RemoteState: repo.InSync},
	}
	sets := []WorktreeSet{
		{SetName: "infra", Branches: []string{"dev", "prod"}},
	}

	buf := &bytes.Buffer{}
	RenderGroupedTable(buf, entries, sets)
	out := buf.String()

	s.Run("header present", func() {
		s.Assert().Contains(out, "REPO")
		s.Assert().Contains(out, "BRANCH")
	})

	s.Run("set header present", func() {
		s.Assert().Contains(out, "infra")
	})

	s.Run("standalone repo present", func() {
		s.Assert().Contains(out, "service-a")
	})

	s.Run("branch entries have tree character", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		hasTreeChar := false
		for _, line := range lines {
			if strings.Contains(line, "└") {
				hasTreeChar = true
				break
			}
		}
		s.Assert().True(hasTreeChar, "expected tree character (└) in output")
	})

	s.Run("flatten names do not appear", func() {
		s.Assert().NotContains(out, "infra-dev")
		s.Assert().NotContains(out, "infra-prod")
	})
}

func (s *TableSuite) TestRenderGroupedTable_MultipleSets() {
	entries := []TableEntry{
		{Name: "infra-dev", Branch: "dev", RemoteState: repo.InSync},
		{Name: "infra-prod", Branch: "prod", RemoteState: repo.InSync},
		{Name: "platform-staging", Branch: "staging", RemoteState: repo.InSync},
		{Name: "platform-release", Branch: "release", RemoteState: repo.InSync},
		{Name: "service-a", Branch: "main", RemoteState: repo.InSync},
	}
	sets := []WorktreeSet{
		{SetName: "infra", Branches: []string{"dev", "prod"}},
		{SetName: "platform", Branches: []string{"staging", "release"}},
	}

	buf := &bytes.Buffer{}
	RenderGroupedTable(buf, entries, sets)
	out := buf.String()

	s.Run("both set headers present", func() {
		s.Assert().Contains(out, "infra")
		s.Assert().Contains(out, "platform")
	})

	s.Run("no flattened names", func() {
		s.Assert().NotContains(out, "infra-dev")
		s.Assert().NotContains(out, "platform-staging")
	})

	s.Run("correct total lines", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		s.Assert().Greater(len(lines), 5)
	})
}

func (s *TableSuite) TestRenderGroupedTable_SetEntryOrder() {
	entries := []TableEntry{
		{Name: "infra-dev", Branch: "dev", RemoteState: repo.InSync},
		{Name: "infra-prod", Branch: "prod", RemoteState: repo.InSync},
	}
	sets := []WorktreeSet{
		{SetName: "infra", Branches: []string{"dev", "prod"}},
	}

	buf := &bytes.Buffer{}
	RenderGroupedTable(buf, entries, sets)
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	devIdx := -1
	prodIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "dev") && strings.Contains(line, "└") {
			devIdx = i
		}
		if strings.Contains(line, "prod") && strings.Contains(line, "└") {
			prodIdx = i
		}
	}

	s.Assert().Greater(devIdx, -1, "dev entry not found")
	s.Assert().Greater(prodIdx, -1, "prod entry not found")
	s.Assert().Less(devIdx, prodIdx, "dev should appear before prod")
}

func (s *TableSuite) TestRenderWorkgroupRows() {
	tests := []struct {
		name          string
		sections      []WorkgroupSection
		wantRowCount  int
		wantFirstName []string
	}{
		{
			name:          "single section with one entry",
			sections:      []WorkgroupSection{{Name: "fix-auth", Entries: []TableEntry{{Name: "service-a"}}}},
			wantRowCount:  1,
			wantFirstName: []string{"fix-auth"},
		},
		{
			name: "single section with multiple entries",
			sections: []WorkgroupSection{
				{Name: "fix-auth", Entries: []TableEntry{
					{Name: "service-a"},
					{Name: "service-b"},
				}},
			},
			wantRowCount:  2,
			wantFirstName: []string{"fix-auth"},
		},
		{
			name: "two sections",
			sections: []WorkgroupSection{
				{Name: "fix-auth", Entries: []TableEntry{{Name: "service-a"}}},
				{Name: "add-logging", Entries: []TableEntry{{Name: "service-b"}}},
			},
			wantRowCount:  2,
			wantFirstName: []string{"fix-auth", "add-logging"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rows := renderWorkgroupRows(tt.sections)
			s.Assert().Len(rows, tt.wantRowCount)

			firstNames := make([]string, 0)
			for i, row := range rows {
				if i == 0 || (i > 0 && rows[i-1].workgroup != row.workgroup && row.workgroup != "") {
					firstNames = append(firstNames, row.workgroup)
				}
			}
			s.Assert().Equal(tt.wantFirstName, firstNames)
		})
	}
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

func (s *TableSuite) TestRenderWorkgroupTable_Empty() {
	buf := &bytes.Buffer{}
	RenderWorkgroupTable(buf, []WorkgroupSection{})
	out := buf.String()
	s.Assert().Empty(out)
}

func (s *TableSuite) TestRenderWorkgroupTable_SingleWorkgroup() {
	sections := []WorkgroupSection{
		{
			Name: "fix-auth",
			Entries: []TableEntry{
				{Name: "service-a", Branch: "feature/auth", RemoteState: repo.InSync},
				{Name: "service-b", Branch: "feature/auth", RemoteState: repo.LocalAhead},
			},
		},
	}

	buf := &bytes.Buffer{}
	RenderWorkgroupTable(buf, sections)
	out := buf.String()

	s.Run("header present", func() {
		s.Assert().Contains(out, "WORKGROUP")
		s.Assert().Contains(out, "REPO")
		s.Assert().Contains(out, "BRANCH")
		s.Assert().Contains(out, "STATUS")
		s.Assert().Contains(out, "COMMIT")
	})

	s.Run("workgroup name on first row", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		s.Require().GreaterOrEqual(len(lines), 2)
		s.Assert().Contains(lines[1], "fix-auth")
	})

	s.Run("workgroup name not on second row", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		s.Require().GreaterOrEqual(len(lines), 3)
		s.Assert().NotContains(lines[2], "fix-auth")
	})

	s.Run("both repos present", func() {
		s.Assert().Contains(out, "service-a")
		s.Assert().Contains(out, "service-b")
	})
}

func (s *TableSuite) TestRenderWorkgroupTable_MultipleWorkgroups() {
	sections := []WorkgroupSection{
		{
			Name: "fix-auth",
			Entries: []TableEntry{
				{Name: "service-a", Branch: "fix-auth", RemoteState: repo.InSync},
			},
		},
		{
			Name: "add-logging",
			Entries: []TableEntry{
				{Name: "service-b", Branch: "add-logging", RemoteState: repo.InSync},
			},
		},
	}

	buf := &bytes.Buffer{}
	RenderWorkgroupTable(buf, sections)
	out := buf.String()

	s.Run("both workgroups present", func() {
		s.Assert().Contains(out, "fix-auth")
		s.Assert().Contains(out, "add-logging")
	})

	s.Run("workgroup names appear", func() {
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		s.Require().GreaterOrEqual(len(lines), 3)
		s.Assert().Contains(lines[1], "fix-auth")
		s.Assert().Contains(lines[2], "add-logging")
	})
}

func (s *TableSuite) TestRenderWorkgroupTable_ColumnAlignment() {
	sections := []WorkgroupSection{
		{
			Name: "fix-auth",
			Entries: []TableEntry{
				{Name: "service-a", Branch: "fix-auth-bug", RemoteState: repo.InSync},
				{Name: "service-b", Branch: "fb", RemoteState: repo.InSync},
			},
		},
	}

	buf := &bytes.Buffer{}
	RenderWorkgroupTable(buf, sections)
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	s.Require().GreaterOrEqual(len(lines), 3)

	for i := 1; i < len(lines); i++ {
		s.Assert().Greater(len(lines[i]), 0, "line %d is empty", i)
	}

	s.Assert().Contains(out, "  ")
}
