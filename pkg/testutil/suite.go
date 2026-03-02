package testutil

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type CmdSuite struct {
	suite.Suite
	Root       *cobra.Command
	registerFn func(*cobra.Command)
}

// RunSuite runs a testify suite in a consistent way.
func RunSuite(t *testing.T, testSuite suite.TestingSuite) {
	t.Helper()
	suite.Run(t, testSuite)
}

// SetRoot initialises s.Root from the given Register function.
func (s *CmdSuite) SetRoot(register func(*cobra.Command)) {
	s.Root = newCmdRootWith(register)
}

// InitRoot stores the package Register function used by SetupTest.
func (s *CmdSuite) InitRoot(register func(*cobra.Command)) {
	s.registerFn = register
}

// SetupTest rebuilds the root command when InitRoot has been configured.
func (s *CmdSuite) SetupTest() {
	if s.registerFn == nil {
		return
	}

	s.Root = newCmdRootWith(s.registerFn)
}

// ExecuteCmd runs s.Root with args and returns the combined output and any error.
func (s *CmdSuite) ExecuteCmd(args ...string) (string, error) {
	return executeCmd(s.T(), s.Root, args...)
}

// MakeGitRepo creates an initialized git repo in a new temp dir, optionally
// with remoteURL set as "origin". Returns the absolute path.
func (s *CmdSuite) MakeGitRepo(remoteURL string) string {
	return makeGitRepo(s.T(), remoteURL)
}

// MakeGitRepoAt creates an initialized git repo at base/sub/name (or base/name
// when sub is empty). Returns the absolute path.
func (s *CmdSuite) MakeGitRepoAt(base, sub, name string) string {
	return makeGitRepoAt(s.T(), base, sub, name)
}

// InitBareGitRepo creates a bare git repository in dir.
func (s *CmdSuite) InitBareGitRepo(dir string) {
	initBareGitRepo(s.T(), dir)
}

// MakeWorkspace writes content to a .gitw file in dir and returns the
// absolute path to that config file.
func (s *CmdSuite) MakeWorkspace(dir, content string) string {
	return makeWorkspace(s.T(), dir, content)
}

// ChangeToDir changes the working directory to dir and restores the original
// when the test completes.
func (s *CmdSuite) ChangeToDir(dir string) {
	changeToDir(s.T(), dir)
}

// SetupWorkspaceDir creates a temp dir with a minimal .gitw config and
// changes the working directory into it. Returns the absolute path.
func (s *CmdSuite) SetupWorkspaceDir() string {
	return setupWorkspaceDir(s.T())
}

// AppendGroup appends a [groups.<groupName>] section referencing repoName to
// the .gitw file in wsDir.
func (s *CmdSuite) AppendGroup(wsDir, groupName, repoName string) {
	appendGroup(s.T(), wsDir, groupName, repoName)
}

// SetActiveContext writes a .gitw.local file in wsDir that sets the
// active context to ctxName.
func (s *CmdSuite) SetActiveContext(wsDir, ctxName string) {
	setActiveContext(s.T(), wsDir, ctxName)
}

// CreateBareRepo initialises a bare git repository in a new temp dir and
// returns the absolute path and a file:// URL.
func (s *CmdSuite) CreateBareRepo() (string, string) {
	return createBareRepo(s.T())
}

// PushToRemote runs "git push -u origin HEAD" in repoDir.
func (s *CmdSuite) PushToRemote(repoDir string) {
	pushToRemote(s.T(), repoDir)
}

// MakeBareGitRepo clones a bare repository from sourceURL into a new temp dir.
func (s *CmdSuite) MakeBareGitRepo(sourceURL string) string {
	return makeBareGitRepo(s.T(), sourceURL)
}

// AddWorktreeToRepo runs `git -C barePath worktree add treePath branch`.
func (s *CmdSuite) AddWorktreeToRepo(barePath, treePath, branch string) {
	addWorktreeToRepo(s.T(), barePath, treePath, branch)
}

// RunGit executes `git <args...>` in dir and fails the test on error.
func (s *CmdSuite) RunGit(dir string, args ...string) {
	RunGit(s.T(), dir, args...)
}

// MakeRemoteWithBranches creates a bare remote and pushes HEAD plus branches.
func (s *CmdSuite) MakeRemoteWithBranches(branches []string) string {
	return makeRemoteWithBranches(s.T(), branches)
}

// RelPath returns target relative to base and fails the test on error.
func (s *CmdSuite) RelPath(base, target string) string {
	return relPath(s.T(), base, target)
}

// MakeWorkspaceWithRepos creates a temp dir with a .gitw config and one
// real git repo per entry in repos. Returns the config path and a name→abs path map.
func (s *CmdSuite) MakeWorkspaceWithRepos(repos map[string]string) (string, map[string]string) {
	return makeWorkspaceWithRepos(s.T(), repos)
}

// MakeWorkspaceFromPaths creates a workspace whose repos are given as name →
// absolute path. Changes CWD into the workspace dir. Returns wsDir.
func (s *CmdSuite) MakeWorkspaceFromPaths(repos map[string]string) string {
	return makeWorkspaceFromPaths(s.T(), repos)
}

// MakeWorkspaceWithNLocalRepos creates n local git repos, registers them in a
// fresh workspace, and changes into the workspace dir. Returns wsDir and repo names.
func (s *CmdSuite) MakeWorkspaceWithNLocalRepos(n int) (string, []string) {
	return makeWorkspaceWithNLocalRepos(s.T(), n)
}

// MakeWorkspaceWithNRemoteRepos creates n repos each backed by a bare remote,
// registers them in a fresh workspace, and changes into the workspace dir.
// Returns wsDir and repo names.
func (s *CmdSuite) MakeWorkspaceWithNRemoteRepos(n int) (string, []string) {
	return makeWorkspaceWithNRemoteRepos(s.T(), n)
}

// MakeWorkspaceWithRepoNames creates a workspace listing the given repo names.
// extraTOML is appended verbatim. Changes CWD into the workspace dir. Returns wsDir.
func (s *CmdSuite) MakeWorkspaceWithRepoNames(repoNames []string, extraTOML string) string {
	return makeWorkspaceWithRepoNames(s.T(), repoNames, extraTOML)
}
