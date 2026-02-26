package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WorkspaceSuite is a base suite that creates a .gitworkspace in a temp dir
// and changes into it before each test.
type WorkspaceSuite struct {
	suite.Suite
	wsDir string
}

func (s *WorkspaceSuite) SetupTest() {
	s.wsDir = s.T().TempDir()
	cfgPath := filepath.Join(s.wsDir, ".gitworkspace")
	s.Require().NoError(os.WriteFile(cfgPath, []byte("[workspace]\nname = \"testws\"\n"), 0o644))
	changeToDir(s.T(), s.wsDir)
}

type AddSuite struct {
	WorkspaceSuite
}

func TestAddSuite(t *testing.T) {
	suite.Run(t, new(AddSuite))
}

func (s *AddSuite) TestAdd() {
	tests := []struct {
		name           string
		addFlags       []string
		wantGroup      string
		checkGitignore bool
	}{
		{
			name: "registers repo",
		},
		{
			name:      "with group membership",
			addFlags:  []string{"-g", "mygroup"},
			wantGroup: "mygroup",
		},
		{
			name:           "updates gitignore",
			checkGitignore: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitworkspace"),
				[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
			))
			changeToDir(s.T(), wsDir)

			repoDir := testutil.MakeGitRepo(s.T(), "")
			name := filepath.Base(repoDir)

			cmdArgs := append(append([]string{"add"}, tt.addFlags...), repoDir)
			_, err := execCmd(s.T(), cmdArgs...)
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(wsDir, ".gitworkspace"))
			s.Require().NoError(err)

			_, exists := cfg.Repos[name]
			s.Assert().True(exists)

			if tt.wantGroup != "" {
				s.Assert().Contains(cfg.Groups[tt.wantGroup].Repos, name)
			}
			if tt.checkGitignore {
				data, err := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
				s.Require().NoError(err)
				s.Assert().Contains(string(data), cfg.Repos[name].Path)
			}
		})
	}
}

func (s *AddSuite) TestErrorNotGitRepo() {
	notARepo := s.T().TempDir()
	_, err := execCmd(s.T(), "add", notARepo)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not a git repository")
}

func (s *AddSuite) TestErrorAlreadyRegistered() {
	repoDir := testutil.MakeGitRepo(s.T(), "")

	_, err := execCmd(s.T(), "add", repoDir)
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "add", repoDir)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "already registered")
}

func (s *AddSuite) TestDetectsRemoteURL() {
	fakeURL := "file:///tmp/fake-origin.git"
	repoDir := testutil.MakeGitRepo(s.T(), fakeURL)

	_, err := execCmd(s.T(), "add", repoDir)
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
	s.Require().NoError(err)

	name := filepath.Base(repoDir)
	s.Assert().NotEmpty(cfg.Repos[name].URL)
	s.Assert().Contains(cfg.Repos[name].URL, "fake-origin")
}

func (s *AddSuite) TestRecursiveAdd() {
	tests := []struct {
		name        string
		setup       func(t *testing.T, wsDir string) []string
		wantNames   []string
		wantGroups  map[string][]string
		wantSkipped []string
	}{
		{
			name: "finds single repo",
			setup: func(t *testing.T, wsDir string) []string {
				makeRepoAt(t, wsDir, "repos", "myrepo")
				return []string{"add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames: []string{"myrepo"},
		},
		{
			name: "auto-groups repos by parent dir",
			setup: func(t *testing.T, wsDir string) []string {
				makeRepoAt(t, wsDir, "apps", "frontend")
				makeRepoAt(t, wsDir, "apps", "backend")
				return []string{"add", "-r", wsDir}
			},
			wantNames:  []string{"frontend", "backend"},
			wantGroups: map[string][]string{"apps": {"frontend", "backend"}},
		},
		{
			name: "skips non-git directories",
			setup: func(t *testing.T, wsDir string) []string {
				makeRepoAt(t, wsDir, "repos", "realrepo")
				require.NoError(t, os.MkdirAll(filepath.Join(wsDir, "repos", "notarepo"), 0o755))
				return []string{"add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames:   []string{"realrepo"},
			wantSkipped: []string{"notarepo"},
		},
		{
			name: "skips already-registered repos without error",
			setup: func(t *testing.T, wsDir string) []string {
				repoDir := makeRepoAt(t, wsDir, "repos", "myrepo")
				relPath, _ := filepath.Rel(wsDir, repoDir)
				toml := fmt.Sprintf("[workspace]\nname = \"testws\"\n\n[repos.myrepo]\npath = %q\n", relPath)
				require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".gitworkspace"), []byte(toml), 0o644))
				return []string{"add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames: []string{"myrepo"},
		},
		{
			name: "non-nesting: inner git repo not registered",
			setup: func(t *testing.T, wsDir string) []string {
				outerDir := makeRepoAt(t, wsDir, "repos", "outer")
				makeRepoAt(t, outerDir, "", "inner")
				return []string{"add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames:   []string{"outer"},
			wantSkipped: []string{"inner"},
		},
		{
			name: "uses CWD when -r given without value",
			setup: func(t *testing.T, wsDir string) []string {
				makeRepoAt(t, wsDir, "repos", "somerepo")
				changeToDir(t, filepath.Join(wsDir, "repos"))
				return []string{"add", "-r"}
			},
			wantNames: []string{"somerepo"},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitworkspace"),
				[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
			))
			changeToDir(s.T(), wsDir)

			args := tt.setup(s.T(), wsDir)
			_, err := execCmd(s.T(), args...)
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(wsDir, ".gitworkspace"))
			s.Require().NoError(err)

			for _, name := range tt.wantNames {
				s.Assert().Contains(cfg.Repos, name, "expected repo %q in config", name)
			}
			for group, repos := range tt.wantGroups {
				for _, repo := range repos {
					s.Assert().Contains(cfg.Groups[group].Repos, repo)
				}
			}
			for _, name := range tt.wantSkipped {
				s.Assert().NotContains(cfg.Repos, name)
			}
		})
	}
}

// makeRepoAt creates a git repo at base/sub/name (or base/name if sub is empty).
// Returns the absolute path of the created repo.
func makeRepoAt(t *testing.T, base, sub, name string) string {
	t.Helper()
	parent := base
	if sub != "" {
		parent = filepath.Join(base, sub)
	}
	require.NoError(t, os.MkdirAll(parent, 0o755))
	repoDir := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(repoDir, 0o755))

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# test\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "init")
	return repoDir
}
