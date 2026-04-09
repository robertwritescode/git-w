package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func withTestGitIdentity(cmd *exec.Cmd) {
	env := os.Environ()
	env = append(env,
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	cmd.Env = env
}

// makeGitRepo creates an initialized git repo with an initial commit in a new temp dir.
// If remoteURL is non-empty, it is added as the "origin" remote. Returns the absolute path.
// When a .gitw file is found in the current working directory, the repo is created inside
// the workspace under repos/ to follow the v2 path convention.
func makeGitRepo(t testing.TB, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()

	if cwd, err := os.Getwd(); err == nil {
		if _, statErr := os.Stat(filepath.Join(cwd, ".gitw")); statErr == nil {
			reposDir := filepath.Join(cwd, "repos")
			if mkErr := os.MkdirAll(reposDir, 0o755); mkErr == nil {
				for i := range 16 {
					candidate := filepath.Join(reposDir, fmt.Sprintf("repo-%d", time.Now().UnixNano()+int64(i)))
					if mkErr := os.MkdirAll(candidate, 0o755); mkErr == nil {
						dir = candidate
						t.Cleanup(func() { _ = os.RemoveAll(candidate) })
						break
					}
				}
			}
		}
	}

	initGitRepo(t, dir)

	if remoteURL != "" {
		addOriginRemote(t, dir, remoteURL)
	}

	return dir
}

// makeGitRepoAt creates an initialized git repo with an initial commit at
// base/sub/name (or base/name when sub is empty). Returns the absolute path.
func makeGitRepoAt(t *testing.T, base, sub, name string) string {
	t.Helper()
	parent := base

	if sub != "" {
		parent = filepath.Join(base, sub)
	}
	require.NoError(t, os.MkdirAll(parent, 0o755))

	repoDir := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(repoDir, 0o755))

	initGitRepo(t, repoDir)

	return repoDir
}

// initGitRepo runs git init, configures user identity, writes a README, and
// creates an initial commit in dir.
func initGitRepo(t testing.TB, dir string) {
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

	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "init")
}

// initBareGitRepo creates a bare git repository in dir.
func initBareGitRepo(t testing.TB, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git init --bare: %s", out)
}

// makeWorkspace writes content to a .gitw file in dir and returns
// the absolute path to that config file.
func makeWorkspace(t *testing.T, dir, content string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, ".gitw")
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))
	return cfgPath
}

// changeToDir changes the working directory to dir and restores the original
// directory when the test completes.
func changeToDir(t testing.TB, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// setupWorkspaceDir creates a temp dir, writes a minimal .gitw config
// (name = "testws"), and changes the working directory into it.
// Returns the absolute path to the temp dir.
func setupWorkspaceDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	makeWorkspace(t, dir, "[metarepo]\nname = \"testws\"\n")
	changeToDir(t, dir)
	return dir
}

// appendGroup appends a [groups.<groupName>] section referencing repoName to
// the .gitw file in wsDir.
func appendGroup(t *testing.T, wsDir, groupName, repoName string) {
	t.Helper()
	cfgPath := filepath.Join(wsDir, ".gitw")
	cfgData, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	groupTOML := fmt.Sprintf("\n[groups.%s]\nrepos = [%q]\n", groupName, repoName)
	require.NoError(t, os.WriteFile(cfgPath, append(cfgData, []byte(groupTOML)...), 0o644))
}

// setActiveContext writes a .gitw.local file in wsDir that sets the
// active context to ctxName.
func setActiveContext(t *testing.T, wsDir, ctxName string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(wsDir, ".gitw.local"),
		[]byte("[context]\nactive = \""+ctxName+"\"\n"),
		0o644,
	))
}

// createBareRepo initialises a bare git repository in a new temp dir and
// returns the absolute path and a file:// URL.
func createBareRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	initBareGitRepo(t, dir)
	return dir, "file://" + dir
}

// pushToRemote runs "git push -u origin HEAD" in repoDir.
func pushToRemote(t *testing.T, repoDir string) {
	t.Helper()
	cmd := exec.Command("git", "push", "-u", "origin", "HEAD")
	cmd.Dir = repoDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("git push: %s", out)
		t.Fatal(err)
	}
}

// makeBareGitRepo clones a bare repository from sourceURL into a new temp dir
// and returns the absolute bare repo path.
func makeBareGitRepo(t testing.TB, sourceURL string) string {
	t.Helper()
	dir := t.TempDir()
	b := exec.Command("git", "clone", "--bare", sourceURL, dir)

	out, err := b.CombinedOutput()
	require.NoError(t, err, "git clone --bare: %s", out)

	return dir
}

// addWorktreeToRepo runs `git -C barePath worktree add treePath branch`.
func addWorktreeToRepo(t testing.TB, barePath, treePath, branch string) {
	t.Helper()
	cmd := exec.Command("git", "-C", barePath, "worktree", "add", treePath, branch)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git worktree add: %s", out)
}

// RunGit executes `git <args...>` in dir and fails the test on error.
// Pass an empty dir to run without changing the command directory.
func RunGit(t testing.TB, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	withTestGitIdentity(cmd)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

// makeRemoteWithBranches creates a bare remote and pushes HEAD plus branches.
// Returns the remote URL as file://<path>.
func makeRemoteWithBranches(t testing.TB, branches []string) string {
	t.Helper()

	bareDir := t.TempDir()
	RunGit(t, "", "init", "--bare", bareDir)

	localDir := t.TempDir()
	RunGit(t, localDir, "init")
	RunGit(t, localDir, "config", "user.email", "test@example.com")
	RunGit(t, localDir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(localDir, "README.md"), []byte("# test\n"), 0o644))
	RunGit(t, localDir, "add", ".")
	RunGit(t, localDir, "commit", "-m", "init")
	RunGit(t, localDir, "remote", "add", "origin", "file://"+bareDir)
	RunGit(t, localDir, "push", "-u", "origin", "HEAD")

	for _, branch := range branches {
		RunGit(t, localDir, "checkout", "-b", branch)
		RunGit(t, localDir, "push", "-u", "origin", branch)
	}

	return "file://" + bareDir
}

// relPath returns target relative to base and fails the test on error.
func relPath(t testing.TB, base, target string) string {
	t.Helper()
	rel, err := filepath.Rel(base, target)
	require.NoError(t, err)
	return rel
}
