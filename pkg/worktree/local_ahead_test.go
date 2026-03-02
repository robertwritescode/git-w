package worktree_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func makeBranchLocalAhead(t *testing.T, repoDir, branch string) {
	t.Helper()

	filePath := filepath.Join(repoDir, fmt.Sprintf("ahead-%s.txt", branch))
	require.NoError(t, os.WriteFile(filePath, []byte("ahead\n"), 0o644))
	testutil.RunGit(t, repoDir, "add", ".")
	testutil.RunGit(t, repoDir, "commit", "-m", "local ahead")

	baseCommit := gitOutput(t, repoDir, "rev-parse", "HEAD~1")
	upstreamBranch := branch + "-upstream"
	testutil.RunGit(t, repoDir, "branch", "-f", upstreamBranch, baseCommit)
	testutil.RunGit(t, repoDir, "branch", "--set-upstream-to="+upstreamBranch, branch)

	status := gitOutput(t, repoDir, "status", "-sb")
	require.Contains(t, status, "ahead", "expected local-ahead status, got:\n%s", status)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)

	return strings.TrimSpace(string(out))
}
