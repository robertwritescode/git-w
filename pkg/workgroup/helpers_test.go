package workgroup_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/robertwritescode/git-w/pkg/testutil"
)

// makeWorkspaceWithLocalRepos creates n local repos and writes the config with
// default_branch set to the actual initial branch of the first repo.
func makeWorkspaceWithLocalRepos(s *testutil.CmdSuite, n int) (string, []string) {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(n)
	branch := currentBranchAt(s.T(), filepath.Join(wsDir, names[0]))
	rewriteConfigWithDefaultBranch(s, wsDir, names, branch)
	return wsDir, names
}

// makeWorkspaceWithRemoteRepos creates n remote-backed repos and writes the
// config with default_branch set to the actual initial branch.
func makeWorkspaceWithRemoteRepos(s *testutil.CmdSuite, n int) (string, []string) {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(n)
	branch := currentBranchAt(s.T(), filepath.Join(wsDir, names[0]))
	rewriteConfigWithDefaultBranch(s, wsDir, names, branch)
	return wsDir, names
}

func rewriteConfigWithDefaultBranch(s *testutil.CmdSuite, wsDir string, names []string, branch string) {
	s.T().Helper()
	sb := fmt.Sprintf("[metarepo]\nname = \"test\"\ndefault_branch = %q\n\n", branch)
	for _, name := range names {
		sb += fmt.Sprintf("[[repo]]\nname = %q\npath = %q\n\n", name, name)
	}
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(sb), 0o644))
}
