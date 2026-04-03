package testutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// newCmdRoot returns a cobra root pre-configured like the real CLI:
// Use "git-w", SilenceUsage true, and a --config persistent flag.
// Callers should Register their sub-commands onto the returned root.
func newCmdRoot() *cobra.Command {
	root := &cobra.Command{Use: "git-w", SilenceUsage: true}
	root.PersistentFlags().String("config", "", "path to .gitw config")
	return root
}

// newCmdRootWith returns a fresh cobra root with the given Register function
// applied. This is the standard helper for testing_test.go files: pass the
// package-level Register func to get a root that has only that command tree.
func newCmdRootWith(register func(*cobra.Command)) *cobra.Command {
	root := newCmdRoot()
	register(root)
	return root
}

// executeCmd runs root with args, captures stdout+stderr combined, and returns
// the output and any error. Flags are reset to defaults before running.
func executeCmd(t testing.TB, root *cobra.Command, args ...string) (string, error) {
	t.Helper()
	resetFlags(t, root)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err := root.Execute()
	return buf.String(), err
}

// resetFlags recursively resets all flags in cmd and its sub-commands to their
// default values, reporting any errors via t.
func resetFlags(t testing.TB, cmd *cobra.Command) {
	t.Helper()
	reset := func(f *pflag.Flag) {
		if err := f.Value.Set(f.DefValue); err != nil {
			t.Errorf("resetting flag %q to default %q: %v", f.Name, f.DefValue, err)
		}
	}

	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)

	for _, child := range cmd.Commands() {
		resetFlags(t, child)
	}
}

// makeWorkspaceWithRepos creates a temp dir containing a .gitw config
// and one real git repo per entry in repos (key=name). The map value is ignored;
// only the key (repo name) is used. Real repos are created in temp dirs and the
// config stores paths relative to the workspace dir.
// Returns the workspace config path and a map of name→absolute repo path.
func makeWorkspaceWithRepos(t *testing.T, repos map[string]string) (string, map[string]string) {
	t.Helper()
	wsDir := t.TempDir()
	sb := newWorkspaceTOML("testws")

	repoPaths := make(map[string]string, len(repos))
	for name := range repos {
		absPath := makeGitRepoAt(t, wsDir, "", name)
		repoPaths[name] = absPath
		appendRepoTOML(t, sb, wsDir, name, absPath)
	}

	cfgPath := makeWorkspace(t, wsDir, sb.String())
	return cfgPath, repoPaths
}

// makeWorkspaceFromPaths creates a workspace whose repos are given as
// name → absolute path. Paths are stored relative to the workspace dir.
// Changes CWD into the workspace dir. Returns the workspace dir path.
func makeWorkspaceFromPaths(t *testing.T, repos map[string]string) string {
	t.Helper()
	wsDir := t.TempDir()
	sb := newWorkspaceTOML("test")

	for name, absPath := range repos {
		appendRepoTOML(t, sb, wsDir, name, absPath)
	}

	return finalizeWorkspace(t, wsDir, sb)
}

// makeWorkspaceWithNLocalRepos creates n local git repos, registers them in a
// fresh workspace, and changes into the workspace dir.
// Returns wsDir and the slice of repo names.
func makeWorkspaceWithNLocalRepos(t *testing.T, n int) (string, []string) {
	t.Helper()
	wsDir := t.TempDir()
	sb := newWorkspaceTOML("test")
	names := make([]string, n)

	for i := range names {
		name := fmt.Sprintf("%03d", i+1)
		dir := makeGitRepoAt(t, wsDir, "", name)
		names[i] = name
		appendRepoTOML(t, sb, wsDir, name, dir)
	}

	return finalizeWorkspace(t, wsDir, sb), names
}

// makeWorkspaceWithNRemoteRepos creates n repos each backed by a bare remote
// (so fetch/pull succeed), registers them in a fresh workspace, and changes
// into the workspace dir. Returns wsDir and the slice of repo names.
func makeWorkspaceWithNRemoteRepos(t *testing.T, n int) (string, []string) {
	t.Helper()
	wsDir := t.TempDir()
	sb := newWorkspaceTOML("test")
	names := make([]string, n)

	for i := range names {
		remoteDir := t.TempDir()
		initBareGitRepo(t, remoteDir)
		name := fmt.Sprintf("%03d", i+1)
		repoDir := makeGitRepoAt(t, wsDir, "", name)
		addOriginRemote(t, repoDir, "file://"+remoteDir)
		pushToRemote(t, repoDir)
		names[i] = name
		appendRepoTOML(t, sb, wsDir, name, repoDir)
	}

	return finalizeWorkspace(t, wsDir, sb), names
}

// makeWorkspaceWithRepoNames creates a workspace listing the given repo names
// (no actual git repos are created on disk). extraTOML is appended verbatim
// after the repo entries. Changes CWD into the workspace dir. Returns wsDir.
func makeWorkspaceWithRepoNames(t *testing.T, repoNames []string, extraTOML string) string {
	t.Helper()
	wsDir := t.TempDir()
	sb := newWorkspaceTOML("test")

	for _, name := range repoNames {
		fmt.Fprintf(sb, "[[repo]]\nname = %q\npath = %q\n\n", name, name)
	}

	if extraTOML != "" {
		sb.WriteString(extraTOML)
	}

	return finalizeWorkspace(t, wsDir, sb)
}

// newWorkspaceTOML returns a strings.Builder pre-populated with the standard
// [metarepo] TOML header using the given workspace name.
func newWorkspaceTOML(name string) *strings.Builder {
	sb := new(strings.Builder)
	fmt.Fprintf(sb, "[metarepo]\nname = %q\n\n", name)
	return sb
}

// appendRepoTOML computes the path of absPath relative to wsDir and appends a
// [[repo]] TOML entry to sb.
func appendRepoTOML(t testing.TB, sb *strings.Builder, wsDir, name, absPath string) {
	t.Helper()
	relPath, err := filepath.Rel(wsDir, absPath)
	require.NoError(t, err)
	fmt.Fprintf(sb, "[[repo]]\nname = %q\npath = %q\n\n", name, relPath)
}

// finalizeWorkspace writes the .gitw config from sb into wsDir, changes the
// working directory into wsDir, and returns wsDir.
func finalizeWorkspace(t *testing.T, wsDir string, sb *strings.Builder) string {
	t.Helper()
	makeWorkspace(t, wsDir, sb.String())
	changeToDir(t, wsDir)
	return wsDir
}

func addOriginRemote(t testing.TB, repoDir, remoteURL string) {
	t.Helper()
	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = repoDir

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git remote add origin: %s", out)
}
