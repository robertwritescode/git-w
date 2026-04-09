package repo_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type RestoreSuite struct {
	testutil.CmdSuite
}

func (s *RestoreSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.SetupWorkspaceDir()
}

func TestRestoreSuite(t *testing.T) {
	s := new(RestoreSuite)
	s.InitRoot(repo.Register)
	testutil.RunSuite(t, s)
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
				s.RunGit("", "clone", url, dest)
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
		[]byte(fmt.Sprintf("[metarepo]\nname = \"testws\"\n\n[[repo]]\nname = \"myrepo\"\npath = \"myrepo\"\nclone_url = %q\n", url)),
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
		[]byte("[metarepo]\nname = \"testws\"\n"),
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
		"[metarepo]\nname = \"testws\"\n\n"+
			"[[repo]]\nname = \"validrepo\"\npath = \"validrepo\"\nclone_url = %q\n\n"+
			"[[repo]]\nname = \"badrepo\"\npath = \"badrepo\"\nclone_url = %q\n",
		validURL, invalidURL,
	)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))

	_, err := s.ExecuteCmd("restore")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "1 of 2")

	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "validrepo")))
}

func (s *RestoreSuite) TestRestore_WorktreeMaterializationPaths() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})

	bareRel := filepath.Join("infra", ".bare")
	devRel := filepath.Join("infra", "dev")
	testRel := filepath.Join("infra", "test")

	toml := fmt.Sprintf("[metarepo]\nname = \"testws\"\n\n[worktrees.infra]\nurl = %q\nbare_path = %q\n\n[worktrees.infra.branches]\ndev = %q\ntest = %q\n", remoteURL, bareRel, devRel, testRel)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))

	out, err := s.ExecuteCmd("restore")
	s.Require().NoError(err)
	s.Assert().Contains(out, "[infra]")

	s.Assert().DirExists(filepath.Join(wsDir, "infra", ".bare"))
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "dev")))
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "test")))

	_, err = os.Stat(filepath.Join(wsDir, ".gitignore"))
	s.Require().NoError(err)
}

func (s *RestoreSuite) TestRestore_WorktreeExistingBareMissingOneTree() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})

	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)

	devAbs := filepath.Join(wsDir, "infra", "dev")
	s.RunGit("", "-C", bareAbs, "worktree", "add", devAbs, "dev")

	// Create a marker file and commit it so the pull path has something new.
	s.Require().NoError(os.WriteFile(filepath.Join(devAbs, "marker.txt"), []byte("before-restore\n"), 0o644))
	s.RunGit(devAbs, "add", ".")
	s.RunGit(devAbs, "commit", "-m", "marker commit")

	toml := fmt.Sprintf("[metarepo]\nname = \"testws\"\n\n[worktrees.infra]\nurl = %q\nbare_path = %q\n\n[worktrees.infra.branches]\ndev = %q\ntest = %q\n", remoteURL, filepath.Join("infra", ".bare"), filepath.Join("infra", "dev"), filepath.Join("infra", "test"))
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))

	out, err := s.ExecuteCmd("restore")
	s.Require().NoError(err)
	s.Assert().Contains(out, "pulled 1", "existing dev worktree should have been pulled")
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "dev")))
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "test")))
}

func (s *RestoreSuite) TestRestore_MixedWorkspace() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	// Regular repo remote.
	repoSource := s.MakeGitRepo("")
	repoURL := "file://" + repoSource

	// Worktree remote with dev and test branches.
	wtURL := s.MakeRemoteWithBranches([]string{"dev", "test"})

	toml := fmt.Sprintf(
		"[metarepo]\nname = \"testws\"\n\n"+
			"[[repo]]\nname = \"myrepo\"\npath = \"myrepo\"\nclone_url = %q\n\n"+
			"[worktrees.infra]\nurl = %q\nbare_path = %q\n\n"+
			"[worktrees.infra.branches]\ndev = %q\ntest = %q\n",
		repoURL,
		wtURL,
		filepath.Join("infra", ".bare"),
		filepath.Join("infra", "dev"),
		filepath.Join("infra", "test"),
	)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))

	out, err := s.ExecuteCmd("restore")
	s.Require().NoError(err)

	// Each logical entry must appear exactly once.
	s.Assert().Equal(1, strings.Count(out, "[myrepo]"), "expected exactly one [myrepo] line in output")
	s.Assert().Equal(1, strings.Count(out, "[infra]"), "expected exactly one [infra] line in output")

	// Synthesized repo names must NOT appear as separate restore targets.
	s.Assert().False(strings.Contains(out, "[infra-dev]"), "output must not contain synthesized [infra-dev]")
	s.Assert().False(strings.Contains(out, "[infra-test]"), "output must not contain synthesized [infra-test]")

	// Regular repo cloned correctly.
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "myrepo")))

	// Worktree branches materialized correctly.
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "dev")))
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "test")))
}

func (s *RestoreSuite) TestRestore_WorktreeNoURL() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	toml := "[metarepo]\nname = \"testws\"\n\n" +
		"[worktrees.infra]\n" +
		"bare_path = \"infra/.bare\"\n\n" +
		"[worktrees.infra.branches]\n" +
		"dev = \"infra/dev\"\n"
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))

	out, err := s.ExecuteCmd("restore")
	s.Require().NoError(err)
	s.Assert().Contains(out, "skipped")
	s.Assert().Contains(out, "[infra]")
}

func (s *RestoreSuite) TestRestore_WorktreeSetsUpstreamTracking() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})

	toml := fmt.Sprintf("[metarepo]\nname = \"testws\"\n\n[worktrees.infra]\nurl = %q\nbare_path = %q\n\n[worktrees.infra.branches]\ndev = %q\ntest = %q\n",
		remoteURL,
		filepath.Join("infra", ".bare"),
		filepath.Join("infra", "dev"),
		filepath.Join("infra", "test"),
	)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))

	_, err := s.ExecuteCmd("restore")
	s.Require().NoError(err)

	for _, branch := range []string{"dev", "test"} {
		branchPath := filepath.Join(wsDir, "infra", branch)
		s.RunGit(branchPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	}
}

func buildRestoreConfig(url string, hasURL bool) string {
	if hasURL {
		return fmt.Sprintf("[metarepo]\nname = \"testws\"\n\n[[repo]]\nname = \"myrepo\"\npath = \"myrepo\"\nclone_url = %q\n", url)
	}

	return "[metarepo]\nname = \"testws\"\n\n[[repo]]\nname = \"myrepo\"\npath = \"myrepo\"\n"
}
