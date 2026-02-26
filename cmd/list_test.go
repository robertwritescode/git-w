package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type ListSuite struct {
	WorkspaceSuite
}

func TestListSuite(t *testing.T) {
	suite.Run(t, new(ListSuite))
}

func (s *ListSuite) TestListAll() {
	repo1Dir := testutil.MakeGitRepo(s.T(), "")
	repo2Dir := testutil.MakeGitRepo(s.T(), "")

	_, err := execCmd(s.T(), "add", repo1Dir)
	s.Require().NoError(err)
	_, err = execCmd(s.T(), "add", repo2Dir)
	s.Require().NoError(err)

	out, err := execCmd(s.T(), "list")
	s.Require().NoError(err)
	s.Assert().Contains(out, filepath.Base(repo1Dir))
	s.Assert().Contains(out, filepath.Base(repo2Dir))

	lines := strings.Split(strings.TrimSpace(out), "\n")
	s.Require().GreaterOrEqual(len(lines), 2)
	for i := 1; i < len(lines); i++ {
		s.Assert().LessOrEqual(lines[i-1], lines[i])
	}

	outAlias, err := execCmd(s.T(), "ls")
	s.Require().NoError(err)
	s.Assert().Equal(out, outAlias)
}

func (s *ListSuite) TestListEdgeCases() {
	tests := []struct {
		name      string
		addRepo   bool   // create and register a real repo; appends its name as the list arg
		nameArg   string // explicit name argument to the list command
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
			// fresh workspace per subtest to prevent state leaking between cases
			dir := s.T().TempDir()
			s.Require().NoError(os.WriteFile(
				filepath.Join(dir, ".gitworkspace"),
				[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
			))
			changeToDir(s.T(), dir)

			args := []string{"list"}
			wantContain := ""

			if tt.addRepo {
				repoDir := testutil.MakeGitRepo(s.T(), "")
				_, err := execCmd(s.T(), "add", repoDir)
				s.Require().NoError(err)
				args = append(args, filepath.Base(repoDir))
				wantContain = repoDir
			} else if tt.nameArg != "" {
				args = append(args, tt.nameArg)
			}

			out, err := execCmd(s.T(), args...)
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
