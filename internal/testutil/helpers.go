package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// MakeGitRepo creates an initialized git repo with an initial commit in dir.
// dir must already exist (e.g. a subdirectory of t.TempDir()).
// Returns dir for convenience.
func MakeGitRepo(t *testing.T, dir string) string {
	t.Helper()

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

	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# test\n"), 0o644))

	run("add", ".")
	run("commit", "-m", "init")

	return dir
}

// MakeWorkspace writes content to a .gitworkspace file in dir and returns
// the absolute path to that config file.
func MakeWorkspace(t *testing.T, dir, content string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, ".gitworkspace")
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))
	return cfgPath
}
