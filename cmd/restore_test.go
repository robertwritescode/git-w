package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type RestoreSuite struct {
	WorkspaceSuite
}

func TestRestoreSuite(t *testing.T) {
	suite.Run(t, new(RestoreSuite))
}

func (s *RestoreSuite) TestRestore() {
	tests := []struct {
		name       string
		repoExists bool
		hasURL     bool
		wantOutput string
	}{
		{name: "clones missing repo", repoExists: false, hasURL: true, wantOutput: "cloned"},
		{name: "pulls existing repo", repoExists: true, hasURL: true, wantOutput: "up to date"},
		{name: "skips no-URL repo", repoExists: false, hasURL: false, wantOutput: "skipped"},
		{name: "gitignore updated", repoExists: false, hasURL: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			changeToDir(s.T(), wsDir)

			var url string
			if tt.hasURL {
				// MakeGitRepo creates a non-bare repo with an initial commit,
				// so git pull on a clone will succeed with "Already up to date."
				sourceDir := testutil.MakeGitRepo(s.T(), "")
				url = "file://" + sourceDir
			}

			if tt.repoExists {
				dest := filepath.Join(wsDir, "myrepo")
				out, err := exec.Command("git", "clone", url, dest).CombinedOutput()
				s.Require().NoError(err, "pre-clone: %s", out)
			}

			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitworkspace"),
				[]byte(buildRestoreConfig(url, tt.hasURL)),
				0o644,
			))

			out, err := execCmd(s.T(), "restore")
			s.Require().NoError(err)

			if tt.wantOutput != "" {
				s.Assert().Contains(out, tt.wantOutput)
			}

			if tt.name == "clones missing repo" {
				s.Assert().True(isGitRepo(filepath.Join(wsDir, "myrepo")))
			}

			if tt.name == "gitignore updated" {
				data, err := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
				s.Require().NoError(err)
				s.Assert().Contains(string(data), "myrepo")
			}
		})
	}
}

func buildRestoreConfig(url string, hasURL bool) string {
	if hasURL {
		return fmt.Sprintf("[workspace]\nname = \"testws\"\n\n[repos.myrepo]\npath = \"myrepo\"\nurl = %q\n", url)
	}
	return "[workspace]\nname = \"testws\"\n\n[repos.myrepo]\npath = \"myrepo\"\n"
}

func (s *RestoreSuite) TestRestoreIdempotent() {
	wsDir := s.T().TempDir()
	changeToDir(s.T(), wsDir)

	sourceDir := testutil.MakeGitRepo(s.T(), "")
	url := "file://" + sourceDir

	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		[]byte(fmt.Sprintf("[workspace]\nname = \"testws\"\n\n[repos.myrepo]\npath = \"myrepo\"\nurl = %q\n", url)),
		0o644,
	))

	_, err := execCmd(s.T(), "restore")
	s.Require().NoError(err)

	changeToDir(s.T(), wsDir)
	_, err = execCmd(s.T(), "restore")
	s.Require().NoError(err)

	s.Assert().True(isGitRepo(filepath.Join(wsDir, "myrepo")))
}

func (s *RestoreSuite) TestRestoreEmpty() {
	wsDir := s.T().TempDir()
	changeToDir(s.T(), wsDir)

	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		[]byte("[workspace]\nname = \"testws\"\n"),
		0o644,
	))

	out, err := execCmd(s.T(), "restore")
	s.Require().NoError(err)
	s.Assert().Empty(out)
}
