package repo_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type RestoreSuite struct {
	testutil.CmdSuite
}

func (s *RestoreSuite) SetupTest() {
	s.SetRoot(repo.Register)
	s.SetupWorkspaceDir()
}

func TestRestoreSuite(t *testing.T) {
	suite.Run(t, new(RestoreSuite))
}

func (s *RestoreSuite) TestRestore() {
	tests := []struct {
		name               string
		repoExists         bool
		hasURL             bool
		wantOutput         string
		wantCloned         bool
		wantGitignoreEntry string
	}{
		{
			name:               "clones missing repo",
			repoExists:         false,
			hasURL:             true,
			wantOutput:         "cloned",
			wantCloned:         true,
			wantGitignoreEntry: "",
		},
		{
			name:       "pulls existing repo",
			repoExists: true,
			hasURL:     true,
		},
		{
			name:       "skips no-URL repo",
			repoExists: false,
			hasURL:     false,
			wantOutput: "skipped",
		},
		{
			name:               "gitignore updated",
			repoExists:         false,
			hasURL:             true,
			wantGitignoreEntry: "myrepo",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			s.ChangeToDir(wsDir)

			var url string
			if tt.hasURL {
				sourceDir := s.MakeGitRepo("")
				url = "file://" + sourceDir
			}

			if tt.repoExists {
				dest := filepath.Join(wsDir, "myrepo")
				out, err := exec.Command("git", "clone", url, dest).CombinedOutput()
				s.Require().NoError(err, "pre-clone: %s", out)
			}

			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitw"),
				[]byte(buildRestoreConfig(url, tt.hasURL)),
				0o644,
			))

			out, err := s.ExecuteCmd("restore")
			s.Require().NoError(err)

			if tt.wantOutput != "" {
				s.Assert().Contains(out, tt.wantOutput)
			}

			if tt.wantCloned {
				s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "myrepo")))
			}

			if tt.wantGitignoreEntry != "" {
				data, readErr := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
				s.Require().NoError(readErr)
				s.Assert().Contains(string(data), tt.wantGitignoreEntry)
			}
		})
	}
}

func (s *RestoreSuite) TestRestoreIdempotent() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	sourceDir := s.MakeGitRepo("")
	url := "file://" + sourceDir

	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitw"),
		[]byte(fmt.Sprintf("[workspace]\nname = \"testws\"\n\n[repos.myrepo]\npath = \"myrepo\"\nurl = %q\n", url)),
		0o644,
	))

	_, err := s.ExecuteCmd("restore")
	s.Require().NoError(err)

	s.ChangeToDir(wsDir)
	_, err = s.ExecuteCmd("restore")
	s.Require().NoError(err)

	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "myrepo")))
}

func (s *RestoreSuite) TestRestoreEmpty() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitw"),
		[]byte("[workspace]\nname = \"testws\"\n"),
		0o644,
	))

	out, err := s.ExecuteCmd("restore")
	s.Require().NoError(err)
	s.Assert().Empty(out)
}

func (s *RestoreSuite) TestRestorePartialFailure() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	// Valid repo: bare remote that can be cloned.
	remoteDir := s.T().TempDir()
	s.InitBareGitRepo(remoteDir)
	validURL := "file://" + remoteDir

	// Invalid repo: nonexistent URL.
	invalidURL := "file:///nonexistent/path/repo.git"

	toml := fmt.Sprintf(
		"[workspace]\nname = \"testws\"\n\n"+
			"[repos.validrepo]\npath = \"validrepo\"\nurl = %q\n\n"+
			"[repos.badrepo]\npath = \"badrepo\"\nurl = %q\n",
		validURL, invalidURL,
	)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))

	_, err := s.ExecuteCmd("restore")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "1 of 2")

	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "validrepo")))
}

func buildRestoreConfig(url string, hasURL bool) string {
	if hasURL {
		return fmt.Sprintf("[workspace]\nname = \"testws\"\n\n[repos.myrepo]\npath = \"myrepo\"\nurl = %q\n", url)
	}

	return "[workspace]\nname = \"testws\"\n\n[repos.myrepo]\npath = \"myrepo\"\n"
}
