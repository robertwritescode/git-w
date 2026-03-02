package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type SafetySuite struct {
	testutil.CmdSuite
}

func TestSafetySuite(t *testing.T) {
	testutil.RunSuite(t, new(SafetySuite))
}

func (s *SafetySuite) TestSafetyViolations() {
	cases := []struct {
		name      string
		mutate    func(dir string)
		wantCount int
	}{
		{name: "clean", wantCount: 0},
		{
			name: "dirty",
			mutate: func(dir string) {
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0o644))
			},
			wantCount: 1,
		},
		{
			name: "local ahead",
			mutate: func(dir string) {
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "ahead.txt"), []byte("x"), 0o644))
				testutil.RunGit(s.T(), dir, "add", ".")
				testutil.RunGit(s.T(), dir, "commit", "-m", "ahead")
			},
			wantCount: 1,
		},
		{
			name: "diverged",
			mutate: func(dir string) {
				// save the base commit before any local changes
				cmd := exec.Command("git", "rev-parse", "HEAD")
				cmd.Dir = dir
				out, err := cmd.CombinedOutput()
				s.Require().NoError(err)
				base := strings.TrimSpace(string(out))

				// add a local commit on the current branch
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "local.txt"), []byte("local"), 0o644))
				testutil.RunGit(s.T(), dir, "add", ".")
				testutil.RunGit(s.T(), dir, "commit", "-m", "local commit")

				// create a fake-upstream branch at base with a diverging commit
				testutil.RunGit(s.T(), dir, "checkout", "-b", "fake-upstream", base)
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "remote.txt"), []byte("remote"), 0o644))
				testutil.RunGit(s.T(), dir, "add", ".")
				testutil.RunGit(s.T(), dir, "commit", "-m", "fake remote commit")

				// go back to the original branch and set fake-upstream as its upstream
				testutil.RunGit(s.T(), dir, "checkout", "-")
				testutil.RunGit(s.T(), dir, "branch", "--set-upstream-to=fake-upstream")
			},
			wantCount: 1,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			bareDir := s.T().TempDir()
			testutil.RunGit(s.T(), "", "init", "--bare", bareDir)

			repoDir := s.MakeGitRepo("file://" + bareDir)
			s.PushToRemote(repoDir)

			if tc.mutate != nil {
				tc.mutate(repoDir)
			}

			violations, err := safetyViolations(repo.Repo{Name: "x", AbsPath: repoDir})
			s.Require().NoError(err)
			s.Assert().Len(violations, tc.wantCount)
		})
	}
}
