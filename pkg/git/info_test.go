package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	gitpkg "github.com/robertwritescode/git-w/pkg/git"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type InfoSuite struct {
	testutil.CmdSuite
	wsDir string
}

func (s *InfoSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.wsDir = s.SetupWorkspaceDir()
}

func TestInfoSuite(t *testing.T) {
	s := new(InfoSuite)
	s.InitRoot(gitpkg.Register)
	testutil.RunSuite(t, s)
}

func (s *InfoSuite) TestInfo_Output() {
	tests := []struct {
		name     string
		numRepos int
		cmd      string
	}{
		{"all repos via info", 2, "info"},
		{"all repos via ll alias", 1, "ll"},
		{"empty workspace", 0, "info"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			cfgPath := filepath.Join(wsDir, ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte("[workspace]\nname = \"testws\"\n"), 0o644))
			s.ChangeToDir(wsDir)

			// Register repos directly in config rather than using add command.
			dirs := make([]string, tt.numRepos)
			if tt.numRepos > 0 {
				cfg, err := config.Load(cfgPath)
				s.Require().NoError(err)
				for i := range dirs {
					dirs[i] = s.MakeGitRepo("")
					relPath, relErr := config.RelPath(cfgPath, dirs[i])
					s.Require().NoError(relErr)
					cfg.Repos[filepath.Base(dirs[i])] = config.RepoConfig{Path: relPath}
				}
				s.Require().NoError(config.Save(cfgPath, cfg))
			}

			out, err := s.ExecuteCmd(tt.cmd)
			s.Require().NoError(err)
			s.Assert().Contains(out, "REPO")
			for _, d := range dirs {
				s.Assert().Contains(out, filepath.Base(d))
			}
		})
	}
}

func (s *InfoSuite) TestInfo_ByGroup() {
	repoDir1 := s.MakeGitRepo("")
	repoDir2 := s.MakeGitRepo("")

	cfgPath := filepath.Join(s.wsDir, ".gitw")
	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	rel1, err := config.RelPath(cfgPath, repoDir1)
	s.Require().NoError(err)
	rel2, err := config.RelPath(cfgPath, repoDir2)
	s.Require().NoError(err)
	cfg.Repos[filepath.Base(repoDir1)] = config.RepoConfig{Path: rel1}
	cfg.Repos[filepath.Base(repoDir2)] = config.RepoConfig{Path: rel2}
	cfg.Groups["mygroup"] = config.GroupConfig{Repos: []string{filepath.Base(repoDir1)}}
	s.Require().NoError(config.Save(cfgPath, cfg))

	out, err := s.ExecuteCmd("info", "mygroup")
	s.Require().NoError(err)
	s.Assert().Contains(out, filepath.Base(repoDir1))
	s.Assert().NotContains(out, filepath.Base(repoDir2))
}

func (s *InfoSuite) TestInfo_Errors() {
	tests := []struct {
		name    string
		setup   func()
		args    []string
		wantErr string
	}{
		{
			name:    "group not found",
			setup:   func() {},
			args:    []string{"info", "nonexistent"},
			wantErr: "not found",
		},
		{
			name:    "missing config",
			setup:   func() { s.ChangeToDir(s.T().TempDir()) },
			args:    []string{"info"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setup()
			_, err := s.ExecuteCmd(tt.args...)
			s.Require().Error(err)
			if tt.wantErr != "" {
				s.Assert().Contains(err.Error(), tt.wantErr)
			}
		})
	}
}

func (s *InfoSuite) TestInfo_WorktreeSetCollapsing() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "prod"})
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "dev"), "dev")
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "prod"), "prod")

	serviceDir := filepath.Join(wsDir, "service-a")
	s.RunGit("", "init", serviceDir)
	s.RunGit(serviceDir, "config", "user.email", "test@example.com")
	s.RunGit(serviceDir, "config", "user.name", "Test User")
	s.RunGit(serviceDir, "commit", "--allow-empty", "-m", "init")

	cfgContent := `[workspace]
name = "test"

[worktrees.infra]
url = "` + remoteURL + `"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev = "infra/dev"
prod = "infra/prod"

[repos.service-a]
path = "service-a"
`

	cfgPath := filepath.Join(wsDir, ".gitw")
	s.Require().NoError(os.WriteFile(cfgPath, []byte(cfgContent), 0o644))

	out, err := s.ExecuteCmd("info")
	s.Require().NoError(err)
	s.Assert().Contains(out, "infra")
	s.Assert().Contains(out, "└")
	s.Assert().NotContains(out, "infra-dev")
	s.Assert().NotContains(out, "infra-prod")
	s.Assert().Contains(out, "service-a")
}

func (s *InfoSuite) TestInfo_NoWorktreeSets() {
	out, err := s.ExecuteCmd("info")
	s.Require().NoError(err)
	s.Assert().Contains(out, "REPO")
	s.Assert().NotContains(out, "└")
}

func (s *InfoSuite) TestInfo_WorkgroupSection() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	service1Dir := filepath.Join(wsDir, ".workgroup", "fix-auth", "service-a")
	service2Dir := filepath.Join(wsDir, ".workgroup", "fix-auth", "service-b")

	s.RunGit("", "init", service1Dir)
	s.RunGit(service1Dir, "config", "user.email", "test@example.com")
	s.RunGit(service1Dir, "config", "user.name", "Test User")
	s.RunGit(service1Dir, "commit", "--allow-empty", "-m", "init")

	s.RunGit("", "init", service2Dir)
	s.RunGit(service2Dir, "config", "user.email", "test@example.com")
	s.RunGit(service2Dir, "config", "user.name", "Test User")
	s.RunGit(service2Dir, "commit", "--allow-empty", "-m", "init")

	regularDir := filepath.Join(wsDir, "regular-repo")
	s.RunGit("", "init", regularDir)
	s.RunGit(regularDir, "config", "user.email", "test@example.com")
	s.RunGit(regularDir, "config", "user.name", "Test User")
	s.RunGit(regularDir, "commit", "--allow-empty", "-m", "init")

	cfgContent := `[workspace]
name = "test"

[repos.regular-repo]
path = "regular-repo"

[workgroup.fix-auth]
branch = "fix-auth"
repos = ["service-a", "service-b"]
`

	cfgPath := filepath.Join(wsDir, ".gitw")
	s.Require().NoError(os.WriteFile(cfgPath, []byte(cfgContent), 0o644))

	localContent := `[workgroup.fix-auth]
branch = "fix-auth"
repos = ["service-a", "service-b"]
`

	localPath := filepath.Join(wsDir, ".gitw.local")
	s.Require().NoError(os.WriteFile(localPath, []byte(localContent), 0o644))

	out, err := s.ExecuteCmd("info")
	s.Require().NoError(err)
	s.Assert().Contains(out, "WORKGROUP")
	s.Assert().Contains(out, "fix-auth")
	s.Assert().Contains(out, "service-a")
	s.Assert().Contains(out, "service-b")
}

func (s *InfoSuite) TestInfo_NoWorkgroups() {
	out, err := s.ExecuteCmd("info")
	s.Require().NoError(err)
	s.Assert().Contains(out, "REPO")
	s.Assert().NotContains(out, "WORKGROUP")
}

func (s *InfoSuite) TestInfo_WorkgroupWithMissingWorktree() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	service1Dir := filepath.Join(wsDir, ".workgroup", "fix-auth", "service-a")
	s.RunGit("", "init", service1Dir)
	s.RunGit(service1Dir, "config", "user.email", "test@example.com")
	s.RunGit(service1Dir, "config", "user.name", "Test User")
	s.RunGit(service1Dir, "commit", "--allow-empty", "-m", "init")

	cfgContent := `[workspace]
name = "test"

[workgroup.fix-auth]
branch = "fix-auth"
repos = ["service-a", "service-b"]
`

	cfgPath := filepath.Join(wsDir, ".gitw")
	s.Require().NoError(os.WriteFile(cfgPath, []byte(cfgContent), 0o644))

	localContent := `[workgroup.fix-auth]
branch = "fix-auth"
repos = ["service-a", "service-b"]
`

	localPath := filepath.Join(wsDir, ".gitw.local")
	s.Require().NoError(os.WriteFile(localPath, []byte(localContent), 0o644))

	out, err := s.ExecuteCmd("info")
	s.Require().NoError(err)
	s.Assert().Contains(out, "fix-auth")
	s.Assert().Contains(out, "service-a")
	s.Assert().NotContains(out, "service-b")
}

func (s *InfoSuite) TestInfo_BothFeatures() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "prod"})
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "dev"), "dev")
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "prod"), "prod")

	workgroupDir := filepath.Join(wsDir, ".workgroup", "fix-auth", "service-a")
	s.RunGit("", "init", workgroupDir)
	s.RunGit(workgroupDir, "config", "user.email", "test@example.com")
	s.RunGit(workgroupDir, "config", "user.name", "Test User")
	s.RunGit(workgroupDir, "commit", "--allow-empty", "-m", "init")

	regularDir := filepath.Join(wsDir, "regular-repo")
	s.RunGit("", "init", regularDir)
	s.RunGit(regularDir, "config", "user.email", "test@example.com")
	s.RunGit(regularDir, "config", "user.name", "Test User")
	s.RunGit(regularDir, "commit", "--allow-empty", "-m", "init")

	cfgContent := `[workspace]
name = "test"

[worktrees.infra]
url = "` + remoteURL + `"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev = "infra/dev"
prod = "infra/prod"

[repos.regular-repo]
path = "regular-repo"

[workgroup.fix-auth]
branch = "fix-auth"
repos = ["service-a"]
`

	cfgPath := filepath.Join(wsDir, ".gitw")
	s.Require().NoError(os.WriteFile(cfgPath, []byte(cfgContent), 0o644))

	localContent := `[workgroup.fix-auth]
branch = "fix-auth"
repos = ["service-a"]
`

	localPath := filepath.Join(wsDir, ".gitw.local")
	s.Require().NoError(os.WriteFile(localPath, []byte(localContent), 0o644))

	out, err := s.ExecuteCmd("info")
	s.Require().NoError(err)

	lines := strings.Split(out, "\n")
	var blankLineIdx int
	for i, line := range lines {
		if strings.TrimSpace(line) == "" && i > 2 {
			blankLineIdx = i
			break
		}
	}

	s.Assert().Greater(blankLineIdx, 0, "expected blank line separator")

	beforeBlank := strings.Join(lines[:blankLineIdx], "\n")
	afterBlank := strings.Join(lines[blankLineIdx+1:], "\n")

	s.Assert().Contains(beforeBlank, "infra")
	s.Assert().Contains(beforeBlank, "└")
	s.Assert().Contains(beforeBlank, "regular-repo")

	s.Assert().Contains(afterBlank, "WORKGROUP")
	s.Assert().Contains(afterBlank, "fix-auth")
}
