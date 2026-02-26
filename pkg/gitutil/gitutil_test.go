package gitutil_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	gitutil "github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type GitutilSuite struct {
	testutil.CmdSuite
}

func TestGitutilSuite(t *testing.T) {
	suite.Run(t, new(GitutilSuite))
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
