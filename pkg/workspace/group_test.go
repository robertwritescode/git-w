package workspace_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workspace"
)

type GroupSuite struct {
	testutil.CmdSuite
}

func TestGroupSuite(t *testing.T) {
	s := new(GroupSuite)
	s.InitRoot(workspace.Register)
	testutil.RunSuite(t, s)
}

func (s *GroupSuite) TestGroupAdd() {
	cases := []struct {
		name      string
		repos     []string
		extraTOML string
		cmdArgs   []string
		wantRepos []string
		wantPath  string
		wantErr   bool
	}{
		{
			name:      "create new group",
			repos:     []string{"frontend", "backend"},
			cmdArgs:   []string{"group", "add", "-n", "web", "frontend", "backend"},
			wantRepos: []string{"frontend", "backend"},
		},
		{
			name:      "add to existing group without duplicates",
			repos:     []string{"frontend", "backend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\n",
			cmdArgs:   []string{"group", "add", "-n", "web", "backend"},
			wantRepos: []string{"frontend", "backend"},
		},
		{
			name:      "adding already-present repo is idempotent",
			repos:     []string{"frontend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\n",
			cmdArgs:   []string{"group", "add", "-n", "web", "frontend"},
			wantRepos: []string{"frontend"},
		},
		{
			name:    "error on unknown repo",
			repos:   []string{"frontend"},
			cmdArgs: []string{"group", "add", "-n", "web", "nonexistent"},
			wantErr: true,
		},
		{
			name:      "create group with path",
			repos:     []string{"frontend"},
			cmdArgs:   []string{"group", "add", "-n", "web", "--path", "apps", "frontend"},
			wantRepos: []string{"frontend"},
			wantPath:  "apps",
		},
		{
			name:      "add repos without --path preserves existing path",
			repos:     []string{"frontend", "backend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\npath = \"apps\"\n",
			cmdArgs:   []string{"group", "add", "-n", "web", "backend"},
			wantRepos: []string{"frontend", "backend"},
			wantPath:  "apps",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.MakeWorkspaceWithRepoNames(tc.repos, tc.extraTOML)

			_, err := s.ExecuteCmd(tc.cmdArgs...)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(loadErr)

			s.Assert().Equal(tc.wantRepos, cfg.Groups["web"].Repos)
			s.Assert().Equal(tc.wantPath, cfg.Groups["web"].Path)
		})
	}
}

func (s *GroupSuite) TestGroupRm() {
	cases := []struct {
		name      string
		repos     []string
		extraTOML string
		cmdArgs   []string
		wantErr   bool
	}{
		{
			name:      "removes existing group",
			repos:     []string{"frontend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\n",
			cmdArgs:   []string{"group", "rm", "web"},
		},
		{
			name:    "error on not found",
			repos:   []string{"frontend"},
			cmdArgs: []string{"group", "rm", "nonexistent"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.MakeWorkspaceWithRepoNames(tc.repos, tc.extraTOML)

			_, err := s.ExecuteCmd(tc.cmdArgs...)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(loadErr)

			_, exists := cfg.Groups["web"]
			s.Assert().False(exists)
		})
	}
}

func (s *GroupSuite) TestGroupRename() {
	cases := []struct {
		name      string
		repos     []string
		extraTOML string
		cmdArgs   []string
		wantErr   bool
	}{
		{
			name:      "renames group",
			repos:     []string{"frontend", "backend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\", \"backend\"]\n",
			cmdArgs:   []string{"group", "rename", "web", "platform"},
		},
		{
			name:    "error if old not found",
			repos:   []string{"frontend"},
			cmdArgs: []string{"group", "rename", "nonexistent", "newname"},
			wantErr: true,
		},
		{
			name:      "error if new already exists",
			repos:     []string{"frontend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\n\n[groups.platform]\nrepos = [\"frontend\"]\n",
			cmdArgs:   []string{"group", "rename", "web", "platform"},
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.MakeWorkspaceWithRepoNames(tc.repos, tc.extraTOML)

			_, err := s.ExecuteCmd(tc.cmdArgs...)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(loadErr)

			_, oldExists := cfg.Groups["web"]
			s.Assert().False(oldExists)

			s.Assert().Equal([]string{"frontend", "backend"}, cfg.Groups["platform"].Repos)
		})
	}
}

func (s *GroupSuite) TestGroupRmrepo() {
	cases := []struct {
		name      string
		repos     []string
		extraTOML string
		cmdArgs   []string
		wantRepos []string
		wantErr   bool
	}{
		{
			name:      "removes one repo",
			repos:     []string{"frontend", "backend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\", \"backend\"]\n",
			cmdArgs:   []string{"group", "rmrepo", "-n", "web", "frontend"},
			wantRepos: []string{"backend"},
		},
		{
			name:      "silently skips repo not in group",
			repos:     []string{"frontend", "backend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\n",
			cmdArgs:   []string{"group", "rmrepo", "-n", "web", "backend"},
			wantRepos: []string{"frontend"},
		},
		{
			name:    "error if group not found",
			repos:   []string{"frontend"},
			cmdArgs: []string{"group", "rmrepo", "-n", "nonexistent", "frontend"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.MakeWorkspaceWithRepoNames(tc.repos, tc.extraTOML)

			_, err := s.ExecuteCmd(tc.cmdArgs...)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(loadErr)

			s.Assert().Equal(tc.wantRepos, cfg.Groups["web"].Repos)
		})
	}
}

func (s *GroupSuite) TestGroupList() {
	cases := []struct {
		name      string
		repos     []string
		extraTOML string
		wantLines []string
	}{
		{
			name:      "no groups gives empty output",
			repos:     []string{"frontend"},
			wantLines: nil,
		},
		{
			name:      "multiple groups sorted one per line",
			repos:     []string{"frontend", "backend", "infra"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\n\n[groups.api]\nrepos = [\"backend\"]\n",
			wantLines: []string{"api", "web"},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.MakeWorkspaceWithRepoNames(tc.repos, tc.extraTOML)

			out, err := s.ExecuteCmd("group", "list")
			s.Require().NoError(err)

			if len(tc.wantLines) == 0 {
				s.Assert().Equal("", strings.TrimSpace(out))
				return
			}

			lines := strings.Split(strings.TrimSpace(out), "\n")
			s.Assert().Equal(tc.wantLines, lines)
		})
	}
}

func (s *GroupSuite) TestGroupList_AliasWorks() {
	s.MakeWorkspaceWithRepoNames([]string{"frontend"}, "[groups.web]\nrepos = [\"frontend\"]\n")

	out1, err := s.ExecuteCmd("group", "list")
	s.Require().NoError(err)

	out2, err := s.ExecuteCmd("group", "ls")
	s.Require().NoError(err)

	s.Assert().Equal(out1, out2)
}

func (s *GroupSuite) TestGroupInfo() {
	cases := []struct {
		name      string
		repos     []string
		extraTOML string
		cmdArgs   []string
		wantOut   string
		wantErr   bool
	}{
		{
			name:      "all groups printed sorted",
			repos:     []string{"frontend", "backend"},
			extraTOML: "[groups.api]\nrepos = [\"backend\"]\n\n[groups.web]\nrepos = [\"frontend\"]\n",
			cmdArgs:   []string{"group", "info"},
			wantOut:   "api: backend\nweb: frontend\n",
		},
		{
			name:      "single group with arg",
			repos:     []string{"frontend"},
			extraTOML: "[groups.web]\nrepos = [\"frontend\"]\n",
			cmdArgs:   []string{"group", "info", "web"},
			wantOut:   "web: frontend\n",
		},
		{
			name:    "error on unknown group",
			repos:   []string{"frontend"},
			cmdArgs: []string{"group", "info", "nonexistent"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.MakeWorkspaceWithRepoNames(tc.repos, tc.extraTOML)

			out, err := s.ExecuteCmd(tc.cmdArgs...)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			s.Assert().Equal(tc.wantOut, out)
		})
	}
}

func (s *GroupSuite) TestGroupEdit() {
	cases := []struct {
		name      string
		extraTOML string
		cmdArgs   []string
		wantPath  string
		wantErr   bool
	}{
		{
			name:      "sets path with --path flag",
			extraTOML: "[groups.web]\nrepos = []\n",
			cmdArgs:   []string{"group", "edit", "web", "--path", "apps"},
			wantPath:  "apps",
		},
		{
			name:      "clears path with --clear-path",
			extraTOML: "[groups.web]\nrepos = []\npath = \"apps\"\n",
			cmdArgs:   []string{"group", "edit", "web", "--clear-path"},
			wantPath:  "",
		},
		{
			name:      "error when no flags given",
			extraTOML: "[groups.web]\nrepos = []\n",
			cmdArgs:   []string{"group", "edit", "web"},
			wantErr:   true,
		},
		{
			name:      "error when --path and --clear-path both given",
			extraTOML: "[groups.web]\nrepos = []\n",
			cmdArgs:   []string{"group", "edit", "web", "--path", "apps", "--clear-path"},
			wantErr:   true,
		},
		{
			name:    "error if group not found",
			cmdArgs: []string{"group", "edit", "nonexistent", "--path", "apps"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.MakeWorkspaceWithRepoNames(nil, tc.extraTOML)

			_, err := s.ExecuteCmd(tc.cmdArgs...)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(loadErr)

			s.Assert().Equal(tc.wantPath, cfg.Groups["web"].Path)
		})
	}
}

func (s *GroupSuite) TestGroupInfo_AliasWorks() {
	s.MakeWorkspaceWithRepoNames([]string{"frontend"}, "[groups.web]\nrepos = [\"frontend\"]\n")

	out1, err := s.ExecuteCmd("group", "info")
	s.Require().NoError(err)

	out2, err := s.ExecuteCmd("group", "ll")
	s.Require().NoError(err)

	s.Assert().Equal(out1, out2)
}
