package repo

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workspace"
)

type RepoSuite struct {
	testutil.CmdSuite
}

func TestRepoSuite(t *testing.T) {
	testutil.RunSuite(t, new(RepoSuite))
}

func (s *RepoSuite) TestFromConfig() {
	const cfgPath = "/workspace/.gitw"
	const cfgDir = "/workspace"

	tests := []struct {
		name      string
		repos     map[string]workspace.RepoConfig
		wantNames []string
		wantPaths map[string]string
		wantFlags map[string][]string
	}{
		{
			name:      "empty config",
			repos:     map[string]workspace.RepoConfig{},
			wantNames: []string{},
		},
		{
			name: "single repo",
			repos: map[string]workspace.RepoConfig{
				"myrepo": {Path: "repos/myrepo"},
			},
			wantNames: []string{"myrepo"},
			wantPaths: map[string]string{"myrepo": cfgDir + "/repos/myrepo"},
		},
		{
			name: "multiple repos sorted by name",
			repos: map[string]workspace.RepoConfig{
				"zebra":  {Path: "z"},
				"alpha":  {Path: "a"},
				"middle": {Path: "m"},
			},
			wantNames: []string{"alpha", "middle", "zebra"},
		},
		{
			name: "with flags",
			repos: map[string]workspace.RepoConfig{
				"bare": {Path: "bare", Flags: []string{"--bare"}},
			},
			wantNames: []string{"bare"},
			wantFlags: map[string][]string{"bare": {"--bare"}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfg := &workspace.WorkspaceConfig{
				Repos:  tt.repos,
				Groups: map[string]workspace.GroupConfig{},
			}
			repos := FromConfig(cfg, cfgPath)

			names := make([]string, len(repos))
			for i, r := range repos {
				names[i] = r.Name
			}
			s.Assert().Equal(tt.wantNames, names)

			for _, r := range repos {
				if tt.wantPaths != nil {
					if want, ok := tt.wantPaths[r.Name]; ok {
						s.Assert().Equal(want, r.AbsPath)
					}
				}

				if tt.wantFlags != nil {
					if want, ok := tt.wantFlags[r.Name]; ok {
						s.Assert().Equal(want, r.Flags)
					}
				}
			}
		})
	}
}

func (s *RepoSuite) TestIsGitRepo() {
	tests := []struct {
		name string
		path func() string
		want bool
	}{
		{
			name: "valid git repo",
			path: func() string { return s.MakeGitRepo("") },
			want: true,
		},
		{
			name: "plain directory",
			path: func() string { return s.T().TempDir() },
			want: false,
		},
		{
			name: "nonexistent path",
			path: func() string { return "/nonexistent/path/that/does/not/exist" },
			want: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.want, IsGitRepo(tt.path()))
		})
	}
}
