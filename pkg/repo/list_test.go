package repo_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type ListSuite struct {
	testutil.CmdSuite
	wsDir string
}

func (s *ListSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.wsDir = s.SetupWorkspaceDir()
}

func TestListSuite(t *testing.T) {
	s := new(ListSuite)
	s.InitRoot(repo.Register)
	testutil.RunSuite(t, s)
}

func (s *ListSuite) TestListAll() {
	repo1Dir := s.MakeGitRepo("")
	repo2Dir := s.MakeGitRepo("")

	_, err := s.ExecuteCmd("repo", "add", repo1Dir)
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("repo", "add", repo2Dir)
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("repo", "list")
	s.Require().NoError(err)
	s.Assert().Contains(out, filepath.Base(repo1Dir))
	s.Assert().Contains(out, filepath.Base(repo2Dir))

	lines := strings.Split(strings.TrimSpace(out), "\n")
	s.Require().GreaterOrEqual(len(lines), 2)
	for i := 1; i < len(lines); i++ {
		s.Assert().LessOrEqual(lines[i-1], lines[i])
	}

	outAlias, err := s.ExecuteCmd("repo", "ls")
	s.Require().NoError(err)
	s.Assert().Equal(out, outAlias)
}

func (s *ListSuite) TestListEdgeCases() {
	tests := []struct {
		name      string
		addRepo   bool
		nameArg   string
		wantErr   bool
		wantEmpty bool
	}{
		{
			name:      "empty workspace returns nothing",
			wantEmpty: true,
		},
		{
			name:    "unknown name returns error",
			nameArg: "nonexistent",
			wantErr: true,
		},
		{
			name:    "known name returns absolute path",
			addRepo: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			s.Require().NoError(os.WriteFile(
				filepath.Join(dir, ".gitw"),
				[]byte("[metarepo]\nname = \"testws\"\n"), 0o644,
			))
			s.ChangeToDir(dir)

			args := []string{"repo", "list"}
			wantContain := ""

			if tt.addRepo {
				repoDir := s.MakeGitRepo("")
				_, err := s.ExecuteCmd("repo", "add", repoDir)
				s.Require().NoError(err)

				args = append(args, filepath.Base(repoDir))
				wantContain = repoDir
			} else if tt.nameArg != "" {
				args = append(args, tt.nameArg)
			}

			out, err := s.ExecuteCmd(args...)

			if tt.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			if tt.wantEmpty {
				s.Assert().Equal("", strings.TrimSpace(out))
			} else {
				s.Assert().Contains(strings.TrimSpace(out), wantContain)
			}
		})
	}
}
