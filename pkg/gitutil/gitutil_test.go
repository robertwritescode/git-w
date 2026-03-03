package gitutil_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	gitutil "github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type GitutilSuite struct {
	testutil.CmdSuite
}

type checkoutBranchCase struct {
	name     string
	branch   string
	checkout string
	wantErr  bool
}

type fetchOriginCase struct {
	name    string
	withURL bool
	wantErr bool
}

type branchExistsCase struct {
	name    string
	branch  string
	valid   bool
	want    bool
	wantErr bool
}

func TestGitutilSuite(t *testing.T) {
	testutil.RunSuite(t, new(GitutilSuite))
}

func (s *GitutilSuite) TestRemoteURL_NoRemote() {
	repoDir := s.MakeGitRepo("")
	got := gitutil.RemoteURL(repoDir)
	s.Assert().Equal("", got)
}

func (s *GitutilSuite) TestRemoteURL_WithRemote() {
	want := "file:///tmp/fake-origin"
	repoDir := s.MakeGitRepo(want)
	got := gitutil.RemoteURL(repoDir)
	s.Assert().Equal(want, got)
}

func (s *GitutilSuite) TestEnsureGitignore_CreatesMissing() {
	dir := s.T().TempDir()
	err := gitutil.EnsureGitignore(dir, ".workspace-cache")
	s.Require().NoError(err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	s.Require().NoError(err)
	s.Assert().Contains(string(data), ".workspace-cache")
}

func (s *GitutilSuite) TestEnsureGitignore_AlreadyPresent() {
	dir := s.T().TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	entry := ".workspace-cache"

	err := os.WriteFile(gitignorePath, []byte(entry+"\n"), 0o644)
	s.Require().NoError(err)

	err = gitutil.EnsureGitignore(dir, entry)
	s.Require().NoError(err)

	data, err := os.ReadFile(gitignorePath)
	s.Require().NoError(err)

	count := strings.Count(string(data), entry)
	s.Assert().Equal(1, count, "entry should appear exactly once, got:\n%s", string(data))
}

func (s *GitutilSuite) TestEnsureGitignore_AppendsWithNewline() {
	dir := s.T().TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")

	err := os.WriteFile(gitignorePath, []byte("node_modules\n"), 0o644)
	s.Require().NoError(err)

	err = gitutil.EnsureGitignore(dir, ".workspace-cache")
	s.Require().NoError(err)

	data, err := os.ReadFile(gitignorePath)
	s.Require().NoError(err)
	s.Assert().Equal("node_modules\n.workspace-cache\n", string(data))
}

func (s *GitutilSuite) TestEnsureGitignore_AppendsWithoutNewline() {
	dir := s.T().TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")

	// Intentionally omit trailing newline.
	err := os.WriteFile(gitignorePath, []byte("node_modules"), 0o644)
	s.Require().NoError(err)

	err = gitutil.EnsureGitignore(dir, ".workspace-cache")
	s.Require().NoError(err)

	data, err := os.ReadFile(gitignorePath)
	s.Require().NoError(err)
	s.Assert().Equal("node_modules\n.workspace-cache\n", string(data))
}

func (s *GitutilSuite) TestEnsureGitignore_ConcurrentSafe() {
	dir := s.T().TempDir()
	entry := ".workspace-cache"

	var wg sync.WaitGroup
	const goroutines = 20
	wg.Add(goroutines)

	for range make([]struct{}, goroutines) {
		go func() {
			defer wg.Done()
			err := gitutil.EnsureGitignore(dir, entry)
			s.Require().NoError(err)
		}()
	}

	wg.Wait()

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	s.Require().NoError(err)

	count := strings.Count(string(data), entry)
	s.Assert().Equal(1, count, "entry should appear exactly once after concurrent writes, got:\n%s", string(data))
}

func (s *GitutilSuite) TestClone_Success() {
	// Create a bare repo to serve as the remote source.
	bareDir := s.T().TempDir()
	s.InitBareGitRepo(bareDir)
	sourceURL := "file://" + bareDir

	// Clone into a new destination directory (must not exist yet).
	destDir := filepath.Join(s.T().TempDir(), "cloned-repo")

	err := gitutil.Clone(context.Background(), sourceURL, destDir)
	s.Require().NoError(err)

	// A successful clone leaves a .git directory at the destination.
	_, statErr := os.Stat(filepath.Join(destDir, ".git"))
	s.Assert().NoError(statErr, ".git directory should exist in cloned repo")
}

func (s *GitutilSuite) TestClone_Cancelled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before the clone starts

	bareDir := s.T().TempDir()
	s.InitBareGitRepo(bareDir)
	sourceURL := "file://" + bareDir

	destDir := filepath.Join(s.T().TempDir(), "cloned-repo")

	err := gitutil.Clone(ctx, sourceURL, destDir)
	s.Assert().Error(err, "clone with cancelled context should return an error")
}

func (s *GitutilSuite) TestCloneBare_AddRemoveWorktree() {
	remoteURL := s.MakeRemoteWithBranches([]string{"dev"})

	bareDir := filepath.Join(s.T().TempDir(), "infra-bare")
	s.Require().NoError(gitutil.CloneBare(context.Background(), remoteURL, bareDir))
	s.DirExists(bareDir)

	worktreeDir := filepath.Join(s.T().TempDir(), "infra-dev")
	s.Require().NoError(gitutil.AddWorktree(context.Background(), bareDir, worktreeDir, "dev"))
	s.True(repo.IsGitRepo(worktreeDir))

	s.Require().NoError(gitutil.RemoveWorktree(bareDir, worktreeDir))
	s.NoDirExists(worktreeDir)
}

func (s *GitutilSuite) TestFetchBare() {
	remoteURL := s.MakeRemoteWithBranches([]string{"dev"})
	bareDir := filepath.Join(s.T().TempDir(), "infra-bare")
	s.Require().NoError(gitutil.CloneBare(context.Background(), remoteURL, bareDir))

	localDir := s.T().TempDir()
	s.RunGit("", "clone", remoteURL, localDir)
	s.RunGit(localDir, "checkout", "dev")
	s.RunGit(localDir, "config", "user.email", "test@example.com")
	s.RunGit(localDir, "config", "user.name", "Test User")
	s.Require().NoError(os.WriteFile(filepath.Join(localDir, "fetch.txt"), []byte("x"), 0o644))
	s.RunGit(localDir, "add", ".")
	s.RunGit(localDir, "commit", "-m", "fetch-update")
	s.RunGit(localDir, "push", "origin", "dev")

	s.Require().NoError(gitutil.FetchBare(bareDir))
	out, err := exec.Command("git", "-C", bareDir, "show-ref").CombinedOutput()
	s.Require().NoError(err, string(out))
}

func (s *GitutilSuite) TestCheckoutBranch() {
	s.runCheckoutBranchCases(checkoutBranchCases())
}

func (s *GitutilSuite) TestFetchOrigin() {
	s.runFetchOriginCases(fetchOriginCases())
}

func (s *GitutilSuite) TestPullBranch() {
	repoDir, remoteURL := s.makeRemoteMainRepo()
	cloneDir := filepath.Join(s.T().TempDir(), "clone")
	s.RunGit("", "clone", remoteURL, cloneDir)

	s.addCommit(repoDir, "pull.txt", "pull-update")
	s.RunGit(repoDir, "push", "origin", "main")

	err := gitutil.PullBranch(context.Background(), cloneDir, "main")
	s.Require().NoError(err)

	msg := gitOutput(s.T(), cloneDir, "log", "-1", "--pretty=%s")
	s.Assert().Equal("pull-update", msg)
}

func (s *GitutilSuite) TestBranchExists() {
	s.runBranchExistsCases(branchExistsCases())
}

func (s *GitutilSuite) TestCreateBranch() {
	repoDir := s.MakeGitRepo("")

	err := gitutil.CreateBranch(context.Background(), repoDir, "feature", "HEAD")
	s.Require().NoError(err)

	got, err := gitutil.BranchExists(context.Background(), repoDir, "feature")
	s.Require().NoError(err)
	s.Assert().True(got)

	s.Assert().Error(gitutil.CreateBranch(context.Background(), repoDir, "feature", "HEAD"))
	s.Assert().Error(gitutil.CreateBranch(context.Background(), repoDir, "other", "missing"))
}

func (s *GitutilSuite) TestPushBranchUpstream() {
	repoDir, remoteURL := s.makeRemoteMainRepo()
	s.RunGit(repoDir, "checkout", "-b", "feature")

	err := gitutil.PushBranchUpstream(context.Background(), repoDir, "origin", "feature")
	s.Require().NoError(err)

	up := gitOutput(s.T(), repoDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	s.Assert().Equal("origin/feature", up)

	s.Assert().NotEmpty(remoteURL)
}

func (s *GitutilSuite) TestSetBranchUpstream() {
	repoDir, _ := s.makeRemoteMainRepo()
	s.RunGit(repoDir, "checkout", "-b", "feature")
	s.RunGit(repoDir, "push", "origin", "feature")

	err := gitutil.SetBranchUpstream(context.Background(), repoDir, "feature", "origin")
	s.Require().NoError(err)

	remote := gitOutput(s.T(), repoDir, "config", "branch.feature.remote")
	s.Assert().Equal("origin", remote)

	s.Assert().Error(gitutil.SetBranchUpstream(context.Background(), repoDir, "feature", "missing"))
}

func (s *GitutilSuite) makeRepoWithBranch(branch string) string {
	repoDir := s.MakeGitRepo("")
	if branch == "" {
		return repoDir
	}

	s.RunGit(repoDir, "checkout", "-b", branch)
	return repoDir
}

func (s *GitutilSuite) makeRemoteMainRepo() (string, string) {
	repoDir := s.MakeGitRepo("")
	_, remoteURL := s.CreateBareRepo()

	s.RunGit(repoDir, "branch", "-M", "main")
	s.RunGit(repoDir, "remote", "add", "origin", remoteURL)
	s.RunGit(repoDir, "push", "-u", "origin", "main")

	return repoDir, remoteURL
}

func (s *GitutilSuite) addCommit(repoDir, filename, message string) {
	path := filepath.Join(repoDir, filename)
	s.Require().NoError(os.WriteFile(path, []byte(message+"\n"), 0o644))

	s.RunGit(repoDir, "add", filename)
	s.RunGit(repoDir, "commit", "-m", message)
}

func (s *GitutilSuite) makeBranchExistsRepo(valid bool, branch string) string {
	if !valid {
		return s.T().TempDir()
	}

	if branch == "feature" {
		return s.makeRepoWithBranch(branch)
	}

	return s.MakeGitRepo("")
}

func (s *GitutilSuite) runCheckoutBranchCases(cases []checkoutBranchCase) {
	for _, tt := range cases {
		s.Run(tt.name, func() { s.assertCheckoutBranch(tt) })
	}
}

func (s *GitutilSuite) runFetchOriginCases(cases []fetchOriginCase) {
	for _, tt := range cases {
		s.Run(tt.name, func() { s.assertFetchOrigin(tt) })
	}
}

func (s *GitutilSuite) runBranchExistsCases(cases []branchExistsCase) {
	for _, tt := range cases {
		s.Run(tt.name, func() { s.assertBranchExists(tt) })
	}
}

func (s *GitutilSuite) assertCheckoutBranch(tt checkoutBranchCase) {
	repoDir := s.makeRepoWithBranch(tt.branch)

	err := gitutil.CheckoutBranch(context.Background(), repoDir, tt.checkout)
	if tt.wantErr {
		s.Assert().Error(err)
		return
	}

	s.Assert().NoError(err)
}

func (s *GitutilSuite) assertFetchOrigin(tt fetchOriginCase) {
	repoDir := s.MakeGitRepo("")
	if tt.withURL {
		_, remoteURL := s.CreateBareRepo()
		s.RunGit(repoDir, "remote", "add", "origin", remoteURL)
		s.RunGit(repoDir, "push", "-u", "origin", "HEAD")
	}

	err := gitutil.FetchOrigin(context.Background(), repoDir)
	if tt.wantErr {
		s.Assert().Error(err)
		return
	}

	s.Assert().NoError(err)
}

func (s *GitutilSuite) assertBranchExists(tt branchExistsCase) {
	repoDir := s.makeBranchExistsRepo(tt.valid, tt.branch)

	got, err := gitutil.BranchExists(context.Background(), repoDir, tt.branch)
	if tt.wantErr {
		s.Assert().Error(err)
		return
	}

	s.Assert().NoError(err)
	s.Assert().Equal(tt.want, got)
}

func checkoutBranchCases() []checkoutBranchCase {
	return []checkoutBranchCase{
		{name: "exists", branch: "feature", checkout: "feature", wantErr: false},
		{name: "missing", branch: "", checkout: "missing", wantErr: true},
	}
}

func fetchOriginCases() []fetchOriginCase {
	return []fetchOriginCase{
		{name: "has remote", withURL: true, wantErr: false},
		{name: "no remote", withURL: false, wantErr: true},
	}
}

func branchExistsCases() []branchExistsCase {
	return []branchExistsCase{
		{name: "exists", branch: "feature", valid: true, want: true, wantErr: false},
		{name: "missing", branch: "missing", valid: true, want: false, wantErr: false},
		{name: "invalid repo", branch: "main", valid: false, want: false, wantErr: true},
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	return strings.TrimSpace(string(output))
}
