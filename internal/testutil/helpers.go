package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// MakeGitRepo creates an initialized git repo with an initial commit in a new temp dir.
// If remoteURL is non-empty, it is added as the "origin" remote. Returns the absolute path.
func MakeGitRepo(t testing.TB, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test User")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "init")

	if remoteURL != "" {
		run("remote", "add", "origin", remoteURL)
	}

	return dir
}

// GitInitBare creates a bare git repository in dir.
func GitInitBare(t testing.TB, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git init --bare: %s", out)
}

// MakeWorkspace writes content to a .gitworkspace file in dir and returns
// the absolute path to that config file.
func MakeWorkspace(t *testing.T, dir, content string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, ".gitworkspace")
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))
	return cfgPath
}
