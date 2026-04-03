package git_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	gitpkg "github.com/robertwritescode/git-w/pkg/git"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type CommitSuite struct {
	testutil.CmdSuite
}

func TestCommitSuite(t *testing.T) {
	s := new(CommitSuite)
	s.InitRoot(gitpkg.Register)
	testutil.RunSuite(t, s)
}

func stageFile(t *testing.T, repoDir, filename, content string) {
	t.Helper()
	path := filepath.Join(repoDir, filename)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	testutil.RunGit(t, repoDir, "add", filename)
}

func commitCount(t *testing.T, repoDir string) int {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	require.NoError(t, err)
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	require.NoError(t, err)
	return n
}

func installFailHook(t *testing.T, repoDir string) {
	t.Helper()
	hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-commit")
	require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o755))
	require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 1\n"), 0o755))
}

func hasStagedChanges(t *testing.T, repoDir string) bool {
	t.Helper()
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = repoDir
	err := cmd.Run()
	return err != nil
}

func setupWorkgroup(t *testing.T, wsDir, wgName string, repoNames []string) {
	t.Helper()
	wgDir := workgroupDir(wsDir, wgName)
	require.NoError(t, os.MkdirAll(wgDir, 0o755))

	for _, name := range repoNames {
		treePath := filepath.Join(wgDir, name)
		testutil.RunGit(t, filepath.Join(wsDir, "repos", name), "worktree", "add", treePath, "-b", wgName)
	}

	appendWorkgroupConfig(t, wsDir, wgName, repoNames)
}

func workgroupDir(wsDir, wgName string) string {
	return filepath.Join(wsDir, ".workgroup", wgName)
}

func appendWorkgroupConfig(t *testing.T, wsDir, wgName string, repoNames []string) {
	t.Helper()
	localPath := filepath.Join(wsDir, ".gitw.local")
	f, err := os.OpenFile(localPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	quoted := make([]string, len(repoNames))
	for i, name := range repoNames {
		quoted[i] = fmt.Sprintf("%q", name)
	}

	entry := fmt.Sprintf("\n[workgroup.%s]\nrepos = [%s]\nbranch = %q\n", wgName, strings.Join(quoted, ", "), wgName)
	_, err = f.WriteString(entry)
	require.NoError(t, err)
}

func (s *CommitSuite) TestCommit_RequiresMessage() {
	wsDir, _ := s.MakeWorkspaceWithNLocalRepos(1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("commit")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "message")
}

func (s *CommitSuite) TestCommit_NothingToCommit() {
	tests := []struct {
		name   string
		nRepos int
	}{
		{"single repo", 1},
		{"multiple repos", 3},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			wsDir, _ := s.MakeWorkspaceWithNLocalRepos(tc.nRepos)
			s.ChangeToDir(wsDir)

			_, err := s.ExecuteCmd("commit", "-m", "test")
			s.Require().Error(err)
			s.Assert().Contains(err.Error(), "nothing to commit")
		})
	}
}

func (s *CommitSuite) TestCommit_CommitsStaged() {
	tests := []struct {
		name    string
		nRepos  int
		nStaged int
	}{
		{"single repo", 1, 1},
		{"two repos", 2, 2},
		{"three repos all staged", 3, 3},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			wsDir, names := s.MakeWorkspaceWithNLocalRepos(tc.nRepos)
			s.ChangeToDir(wsDir)

			repoPaths := make([]string, tc.nStaged)
			for i := 0; i < tc.nStaged; i++ {
				repoPaths[i] = filepath.Join(wsDir, "repos", names[i])
				stageFile(s.T(), repoPaths[i], "change.txt", "content")
			}

			initialCounts := make([]int, tc.nStaged)
			for i, p := range repoPaths {
				initialCounts[i] = commitCount(s.T(), p)
			}

			_, err := s.ExecuteCmd("commit", "-m", "atomic commit")
			s.Require().NoError(err)

			for i, p := range repoPaths {
				s.Assert().Equal(initialCounts[i]+1, commitCount(s.T(), p))
			}
		})
	}
}

func (s *CommitSuite) TestCommit_SkipsUnstaged() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(3)
	s.ChangeToDir(wsDir)

	stagedPath := filepath.Join(wsDir, "repos", names[0])
	stageFile(s.T(), stagedPath, "f.txt", "hello")
	initialStaged := commitCount(s.T(), stagedPath)
	initialOther := commitCount(s.T(), filepath.Join(wsDir, "repos", names[1]))

	out, err := s.ExecuteCmd("commit", "-m", "partial")
	s.Require().NoError(err)

	s.Assert().Equal(initialStaged+1, commitCount(s.T(), stagedPath))
	s.Assert().Equal(initialOther, commitCount(s.T(), filepath.Join(wsDir, "repos", names[1])))
	s.Assert().Equal(initialOther, commitCount(s.T(), filepath.Join(wsDir, "repos", names[2])))

	s.Assert().Contains(out, "skipped")
}

func (s *CommitSuite) TestCommit_DryRun() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	s.ChangeToDir(wsDir)

	repo1Path := filepath.Join(wsDir, "repos", names[0])
	stageFile(s.T(), repo1Path, "f.txt", "hello")
	initialCount := commitCount(s.T(), repo1Path)

	out, err := s.ExecuteCmd("commit", "--dry-run", "-m", "would commit")
	s.Require().NoError(err)

	s.Assert().Contains(out, names[0])
	s.Assert().Contains(out, "dry run")
	s.Assert().Equal(initialCount, commitCount(s.T(), repo1Path))
}

func (s *CommitSuite) TestCommit_Rollback_OnFailure() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	s.ChangeToDir(wsDir)

	goodPath := filepath.Join(wsDir, "repos", names[0])
	badPath := filepath.Join(wsDir, "repos", names[1])

	stageFile(s.T(), goodPath, "f.txt", "hello")
	stageFile(s.T(), badPath, "f.txt", "hello")
	installFailHook(s.T(), badPath)

	initialGood := commitCount(s.T(), goodPath)
	initialBad := commitCount(s.T(), badPath)

	out, err := s.ExecuteCmd("commit", "-m", "should rollback")
	s.Require().Error(err)

	s.Assert().Equal(initialGood, commitCount(s.T(), goodPath), "good repo must be rolled back")
	s.Assert().Equal(initialBad, commitCount(s.T(), badPath))

	s.Assert().True(hasStagedChanges(s.T(), goodPath), "good repo must still have staged changes")
	s.Assert().Contains(out, "rolling back")
	s.Assert().Contains(out, "rolled back")
}

func (s *CommitSuite) TestCommit_NoVerify_BypassesHook() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	s.ChangeToDir(wsDir)

	for _, name := range names {
		p := filepath.Join(wsDir, "repos", name)
		stageFile(s.T(), p, "f.txt", "hello")
		installFailHook(s.T(), p)
	}

	_, err := s.ExecuteCmd("commit", "--no-verify", "-m", "skip hooks")
	s.Require().NoError(err)

	for _, name := range names {
		s.Assert().Equal(2, commitCount(s.T(), filepath.Join(wsDir, "repos", name)))
	}
}

func (s *CommitSuite) TestCommit_Alias() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(1)
	s.ChangeToDir(wsDir)

	stageFile(s.T(), filepath.Join(wsDir, "repos", names[0]), "f.txt", "hello")

	_, err := s.ExecuteCmd("ci", "-m", "via alias")
	s.Require().NoError(err)
	s.Assert().Equal(2, commitCount(s.T(), filepath.Join(wsDir, "repos", names[0])))
}

func (s *CommitSuite) TestCommit_RespectsRepoFilter() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(3)
	s.ChangeToDir(wsDir)

	for _, name := range names {
		stageFile(s.T(), filepath.Join(wsDir, "repos", name), "f.txt", "hello")
	}

	_, err := s.ExecuteCmd("commit", "-m", "filtered", names[0])
	s.Require().NoError(err)

	s.Assert().Equal(2, commitCount(s.T(), filepath.Join(wsDir, "repos", names[0])))
	s.Assert().Equal(1, commitCount(s.T(), filepath.Join(wsDir, "repos", names[1])))
	s.Assert().Equal(1, commitCount(s.T(), filepath.Join(wsDir, "repos", names[2])))
}

func (s *CommitSuite) TestCommit_ActiveContext_Scopes() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)

	s.AppendGroup(wsDir, "web", names[0])
	s.SetActiveContext(wsDir, "web")
	s.ChangeToDir(wsDir)

	for _, name := range names {
		stageFile(s.T(), filepath.Join(wsDir, "repos", name), "f.txt", "hello")
	}

	_, err := s.ExecuteCmd("commit", "-m", "context scoped")
	s.Require().NoError(err)

	s.Assert().Equal(2, commitCount(s.T(), filepath.Join(wsDir, "repos", names[0])))
	s.Assert().Equal(1, commitCount(s.T(), filepath.Join(wsDir, "repos", names[1])))
}

func (s *CommitSuite) TestCommit_DryRun_NothingToCommit() {
	wsDir, _ := s.MakeWorkspaceWithNLocalRepos(1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("commit", "--dry-run", "-m", "nothing")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "nothing to commit")
}

func (s *CommitSuite) TestCommit_CommitMessage_Preserved() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(1)
	s.ChangeToDir(wsDir)

	repoPath := filepath.Join(wsDir, "repos", names[0])
	stageFile(s.T(), repoPath, "f.txt", "content")

	_, err := s.ExecuteCmd("commit", "-m", "my unique message")
	s.Require().NoError(err)

	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	s.Require().NoError(err)
	s.Assert().Equal("my unique message", strings.TrimSpace(string(out)))
}

func (s *CommitSuite) TestCommit_Workgroup_CommitsWorktrees() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	wgName := "feature-x"
	setupWorkgroup(s.T(), wsDir, wgName, names)
	s.ChangeToDir(wsDir)

	wgDir := workgroupDir(wsDir, wgName)
	for _, name := range names {
		stageFile(s.T(), filepath.Join(wgDir, name), "f.txt", "hello")
	}

	initialCounts := make([]int, len(names))
	for i, name := range names {
		initialCounts[i] = commitCount(s.T(), filepath.Join(wgDir, name))
	}

	_, err := s.ExecuteCmd("commit", "-m", "wg commit", "--workgroup", wgName)
	s.Require().NoError(err)

	for i, name := range names {
		s.Assert().Equal(initialCounts[i]+1, commitCount(s.T(), filepath.Join(wgDir, name)))
	}
}

func (s *CommitSuite) TestCommit_Workgroup_NotFound() {
	wsDir, _ := s.MakeWorkspaceWithNLocalRepos(1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("commit", "-m", "test", "--workgroup", "no-such-wg")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not found")
}

func (s *CommitSuite) TestCommit_Workgroup_MutuallyExclusiveWithArgs() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	wgName := "feature-x"
	setupWorkgroup(s.T(), wsDir, wgName, names)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("commit", "-m", "test", "--workgroup", wgName, names[0])
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "mutually exclusive")
}

func (s *CommitSuite) TestCommit_Workgroup_ShortFlag() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(1)
	wgName := "feature-x"
	setupWorkgroup(s.T(), wsDir, wgName, names)
	s.ChangeToDir(wsDir)

	wgDir := workgroupDir(wsDir, wgName)
	stageFile(s.T(), filepath.Join(wgDir, names[0]), "f.txt", "hello")

	_, err := s.ExecuteCmd("commit", "-m", "short flag", "-W", wgName)
	s.Require().NoError(err)
}

func (s *CommitSuite) TestCommit_Workgroup_DryRun() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	wgName := "feature-x"
	setupWorkgroup(s.T(), wsDir, wgName, names)
	s.ChangeToDir(wsDir)

	wgDir := workgroupDir(wsDir, wgName)
	stageFile(s.T(), filepath.Join(wgDir, names[0]), "f.txt", "hello")
	initialCount := commitCount(s.T(), filepath.Join(wgDir, names[0]))

	out, err := s.ExecuteCmd("commit", "--dry-run", "-m", "wg dry", "--workgroup", wgName)
	s.Require().NoError(err)

	s.Assert().Contains(out, "dry run")
	s.Assert().Contains(out, names[0])
	s.Assert().Equal(initialCount, commitCount(s.T(), filepath.Join(wgDir, names[0])))
}

func (s *CommitSuite) TestCommit_Workgroup_SkipsMissingWorktrees() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	wgName := "feature-x"
	setupWorkgroup(s.T(), wsDir, wgName, names)
	s.ChangeToDir(wsDir)

	wgDir := workgroupDir(wsDir, wgName)

	// Stage a file in the first worktree only.
	stageFile(s.T(), filepath.Join(wgDir, names[0]), "f.txt", "hello")

	// Remove the second worktree directory to simulate a missing path.
	missingPath := filepath.Join(wgDir, names[1])
	testutil.RunGit(s.T(), filepath.Join(wsDir, "repos", names[1]), "worktree", "remove", missingPath)

	out, err := s.ExecuteCmd("commit", "-m", "skip missing", "--workgroup", wgName)
	s.Require().NoError(err)

	// The first repo should have been committed.
	s.Assert().Equal(2, commitCount(s.T(), filepath.Join(wgDir, names[0])))

	// The missing worktree should NOT appear as "skipped: no staged changes".
	s.Assert().NotContains(out, names[1])
}
