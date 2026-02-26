package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robertwritescode/git-workspace/internal/repo"
	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type ParallelSuite struct {
	suite.Suite
	repos []repo.Repo
}

func TestParallel(t *testing.T) {
	suite.Run(t, new(ParallelSuite))
}

func (s *ParallelSuite) SetupTest() {
	s.repos = make([]repo.Repo, 3)
	for i := range s.repos {
		dir := testutil.MakeGitRepo(s.T(), "")
		s.repos[i] = repo.Repo{Name: fmt.Sprintf("repo%d", i), AbsPath: dir}
	}
}

func (s *ParallelSuite) TestRunParallel_Modes() {
	tests := []struct {
		name       string
		repos      []repo.Repo
		args       []string
		opts       ExecOptions
		assertEach func(r ExecResult)
	}{
		{
			name:  "multi-repo collects all with exit 0",
			repos: s.repos,
			args:  []string{"status"},
			opts:  ExecOptions{Async: true},
			assertEach: func(r ExecResult) {
				s.Assert().Equal(0, r.ExitCode, "repo %s: %s", r.RepoName, r.Stderr)
			},
		},
		{
			name:  "multi-repo prefixes output",
			repos: s.repos,
			args:  []string{"status", "-sb"},
			opts:  ExecOptions{Async: true},
			assertEach: func(r ExecResult) {
				s.Assert().Contains(string(r.Stdout)+string(r.Stderr), "["+r.RepoName+"]")
			},
		},
		{
			name:  "single-repo no prefix",
			repos: s.repos[:1],
			args:  []string{"status"},
			opts:  ExecOptions{},
			assertEach: func(r ExecResult) {
				s.Assert().NotContains(string(r.Stdout), "[")
			},
		},
		{
			name:  "non-zero exit propagated",
			repos: s.repos,
			args:  []string{"invalid-subcommand"},
			opts:  ExecOptions{Async: true},
			assertEach: func(r ExecResult) {
				s.Assert().NotEqual(0, r.ExitCode, "repo %s should have non-zero exit", r.RepoName)
			},
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			results := RunParallel(tc.repos, tc.args, tc.opts)
			s.Assert().Len(results, len(tc.repos))
			for _, r := range results {
				tc.assertEach(r)
			}
		})
	}
}

func (s *ParallelSuite) TestRunParallel_ConcurrencyLimit() {
	repos := make([]repo.Repo, 4)
	for i := range repos {
		dir := testutil.MakeGitRepo(s.T(), "")
		repos[i] = repo.Repo{Name: fmt.Sprintf("r%d", i), AbsPath: dir}
	}

	// Fake git binary: atomically tracks concurrent worker count using mkdir as a lock.
	binDir := s.T().TempDir()
	stateDir := s.T().TempDir()
	lockDir := filepath.Join(stateDir, "lock")
	activeFile := filepath.Join(stateDir, "active")
	peakFile := filepath.Join(stateDir, "peak")

	os.WriteFile(activeFile, []byte("0"), 0o644)
	os.WriteFile(peakFile, []byte("0"), 0o644)

	script := fmt.Sprintf(`#!/bin/sh
acquire() { while ! mkdir %q 2>/dev/null; do :; done; }
release() { rmdir %q; }

acquire
n=$(cat %q 2>/dev/null || echo 0)
n=$((n+1))
echo "$n" > %q
p=$(cat %q 2>/dev/null || echo 0)
if [ "$n" -gt "$p" ]; then echo "$n" > %q; fi
release
sleep 0.06
acquire
n=$(cat %q 2>/dev/null || echo 0)
n=$((n-1))
echo "$n" > %q
release
`, lockDir, lockDir, activeFile, activeFile, peakFile, peakFile, activeFile, activeFile)

	gitPath := filepath.Join(binDir, "git")
	s.Require().NoError(os.WriteFile(gitPath, []byte(script), 0o755))
	s.T().Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	RunParallel(repos, []string{"noop"}, ExecOptions{Async: true, MaxConcurrency: 2})

	peakData, err := os.ReadFile(peakFile)
	s.Require().NoError(err)
	peak := strings.TrimSpace(string(peakData))
	s.Assert().Equal("2", peak, "peak concurrent workers should be 2, got %s", peak)
}

func (s *ParallelSuite) TestRunParallel_Timeout() {
	repos := make([]repo.Repo, 2)
	for i := range repos {
		dir := testutil.MakeGitRepo(s.T(), "")
		repos[i] = repo.Repo{Name: fmt.Sprintf("slow%d", i), AbsPath: dir}
	}

	// Fake git that sleeps longer than the timeout.
	binDir := s.T().TempDir()
	// exec replaces the shell with sleep so SIGKILL from context cancellation terminates it.
	script := "#!/bin/sh\nexec sleep 10\n"
	gitPath := filepath.Join(binDir, "git")
	s.Require().NoError(os.WriteFile(gitPath, []byte(script), 0o755))
	s.T().Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	start := time.Now()
	results := RunParallel(repos, []string{"noop"}, ExecOptions{
		Async:   true,
		Timeout: 150 * time.Millisecond,
	})
	elapsed := time.Since(start)

	s.Assert().Less(elapsed, 2*time.Second, "timeout should cancel commands quickly")
	for _, r := range results {
		s.Assert().True(r.Err != nil || r.ExitCode != 0,
			"repo %s should have error or non-zero exit after timeout", r.RepoName)
	}
}

