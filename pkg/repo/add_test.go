package repo_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type AddSuite struct {
	testutil.CmdSuite
	wsDir string
}

func (s *AddSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.wsDir = s.SetupWorkspaceDir()
}

func TestAddSuite(t *testing.T) {
	s := new(AddSuite)
	s.InitRoot(repo.Register)
	testutil.RunSuite(t, s)
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
			wsDir := s.SetupWorkspaceDir()

			repoDir := s.MakeGitRepo("")
			name := filepath.Base(repoDir)

			cmdArgs := append(append([]string{"repo", "add"}, tt.addFlags...), repoDir)
			_, err := s.ExecuteCmd(cmdArgs...)
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(wsDir, ".gitw"))
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
	_, err := s.ExecuteCmd("repo", "add", notARepo)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not a git repository")
}

func (s *AddSuite) TestErrorAlreadyRegistered() {
	repoDir := s.MakeGitRepo("")

	_, err := s.ExecuteCmd("repo", "add", repoDir)
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("repo", "add", repoDir)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "already registered")
}

func (s *AddSuite) TestDetectsRemoteURL() {
	fakeURL := "file:///tmp/fake-origin.git"
	repoDir := s.MakeGitRepo(fakeURL)

	_, err := s.ExecuteCmd("repo", "add", repoDir)
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitw"))
	s.Require().NoError(err)

	name := filepath.Base(repoDir)
	s.Assert().NotEmpty(cfg.Repos[name].URL)
	s.Assert().Contains(cfg.Repos[name].URL, "fake-origin")
}

func (s *AddSuite) TestRecursiveAdd() {
	tests := []struct {
		name        string
		setup       func(wsDir string) []string
		wantNames   []string
		wantGroups  map[string][]string
		wantSkipped []string
	}{
		{
			name: "finds single repo",
			setup: func(wsDir string) []string {
				s.MakeGitRepoAt(wsDir, "repos", "myrepo")
				return []string{"repo", "add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames: []string{"myrepo"},
		},
		{
			name: "auto-groups repos by parent dir",
			setup: func(wsDir string) []string {
				s.MakeGitRepoAt(wsDir, "apps", "frontend")
				s.MakeGitRepoAt(wsDir, "apps", "backend")
				return []string{"repo", "add", "-r", wsDir}
			},
			wantNames:  []string{"frontend", "backend"},
			wantGroups: map[string][]string{"apps": {"frontend", "backend"}},
		},
		{
			name: "skips non-git directories",
			setup: func(wsDir string) []string {
				s.MakeGitRepoAt(wsDir, "repos", "realrepo")
				s.Require().NoError(os.MkdirAll(filepath.Join(wsDir, "repos", "notarepo"), 0o755))
				return []string{"repo", "add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames:   []string{"realrepo"},
			wantSkipped: []string{"notarepo"},
		},
		{
			name: "skips already-registered repos without error",
			setup: func(wsDir string) []string {
				repoDir := s.MakeGitRepoAt(wsDir, "repos", "myrepo")
				relPath, _ := filepath.Rel(wsDir, repoDir)
				toml := fmt.Sprintf("[metarepo]\nname = \"testws\"\n\n[repos.myrepo]\npath = %q\n", relPath)
				s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(toml), 0o644))
				return []string{"repo", "add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames: []string{"myrepo"},
		},
		{
			name: "non-nesting: inner git repo not registered",
			setup: func(wsDir string) []string {
				outerDir := s.MakeGitRepoAt(wsDir, "repos", "outer")
				s.MakeGitRepoAt(outerDir, "", "inner")
				return []string{"repo", "add", "-r", filepath.Join(wsDir, "repos")}
			},
			wantNames:   []string{"outer"},
			wantSkipped: []string{"inner"},
		},
		{
			name: "uses CWD when -r given without value",
			setup: func(wsDir string) []string {
				s.MakeGitRepoAt(wsDir, "repos", "somerepo")
				s.ChangeToDir(filepath.Join(wsDir, "repos"))
				return []string{"repo", "add", "-r"}
			},
			wantNames: []string{"somerepo"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.SetupWorkspaceDir()

			args := tt.setup(wsDir)
			_, err := s.ExecuteCmd(args...)
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(err)

			for _, name := range tt.wantNames {
				s.Assert().Contains(cfg.Repos, name, "expected repo %q in config", name)
			}

			for group, repos := range tt.wantGroups {
				for _, r := range repos {
					s.Assert().Contains(cfg.Groups[group].Repos, r)
				}
			}

			for _, name := range tt.wantSkipped {
				s.Assert().NotContains(cfg.Repos, name)
			}
		})
	}
}
