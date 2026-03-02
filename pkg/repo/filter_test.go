package repo_test

import (
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type FilterSuite struct {
	suite.Suite
}

func TestFilter(t *testing.T) {
	testutil.RunSuite(t, new(FilterSuite))
}

// makeConfig builds a minimal WorkspaceConfig and returns it with a temp cfgPath.
func (s *FilterSuite) makeConfig(repoNames []string, groups map[string][]string, activeCtx string) (*config.WorkspaceConfig, string) {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, ".gitw")

	cfg := &config.WorkspaceConfig{
		Repos:   make(map[string]config.RepoConfig, len(repoNames)),
		Groups:  make(map[string]config.GroupConfig, len(groups)),
		Context: config.ContextConfig{Active: activeCtx},
	}

	for _, r := range repoNames {
		cfg.Repos[r] = config.RepoConfig{Path: r}
	}

	for g, members := range groups {
		cfg.Groups[g] = config.GroupConfig{Repos: members}
	}

	return cfg, cfgPath
}

func (s *FilterSuite) TestFilter_NoNames() {
	cases := []struct {
		name      string
		repos     []string
		groups    map[string][]string
		activeCtx string
		wantNames []string
		wantErr   bool
	}{
		{
			name:      "no context returns all repos",
			repos:     []string{"a", "b", "c"},
			wantNames: []string{"a", "b", "c"},
		},
		{
			name:      "active context filters to group",
			repos:     []string{"a", "b", "c"},
			groups:    map[string][]string{"web": {"a", "b"}},
			activeCtx: "web",
			wantNames: []string{"a", "b"},
		},
		{
			name:      "unknown active context returns error",
			repos:     []string{"a"},
			activeCtx: "missing",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			cfg, cfgPath := s.makeConfig(tc.repos, tc.groups, tc.activeCtx)
			repos, err := repo.Filter(cfg, cfgPath, nil)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			names := repoNames(repos)
			s.Assert().ElementsMatch(tc.wantNames, names)
		})
	}
}

func (s *FilterSuite) TestFilter_WithNames() {
	cases := []struct {
		name      string
		repos     []string
		groups    map[string][]string
		filterBy  []string
		wantNames []string
		wantErr   bool
	}{
		{
			name:      "single repo name",
			repos:     []string{"a", "b"},
			filterBy:  []string{"a"},
			wantNames: []string{"a"},
		},
		{
			name:      "group name expands to members",
			repos:     []string{"a", "b", "c"},
			groups:    map[string][]string{"web": {"a", "b"}},
			filterBy:  []string{"web"},
			wantNames: []string{"a", "b"},
		},
		{
			name:     "group with unknown member returns error",
			repos:    []string{"a", "b"},
			groups:   map[string][]string{"web": {"a", "missing"}},
			filterBy: []string{"web"},
			wantErr:  true,
		},
		{
			name:      "mixed repo and group deduplicated",
			repos:     []string{"a", "b", "c"},
			groups:    map[string][]string{"web": {"a", "b"}},
			filterBy:  []string{"web", "a"},
			wantNames: []string{"a", "b"},
		},
		{
			name:     "unknown name returns error",
			repos:    []string{"a"},
			filterBy: []string{"nope"},
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			cfg, cfgPath := s.makeConfig(tc.repos, tc.groups, "")
			repos, err := repo.Filter(cfg, cfgPath, tc.filterBy)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			s.Assert().ElementsMatch(tc.wantNames, repoNames(repos))
		})
	}
}

func (s *FilterSuite) TestForGroup() {
	cases := []struct {
		name      string
		groupName string
		wantNames []string
		wantErr   bool
	}{
		{
			name:      "returns group members",
			groupName: "web",
			wantNames: []string{"a", "b"},
		},
		{
			name:      "unknown group returns error",
			groupName: "missing",
			wantErr:   true,
		},
		{
			name:      "group with unknown repo returns error",
			groupName: "broken",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			cfg, cfgPath := s.makeConfig(
				[]string{"a", "b", "c"},
				map[string][]string{"web": {"a", "b"}, "broken": {"a", "missing"}},
				"",
			)
			repos, err := repo.ForGroup(cfg, cfgPath, tc.groupName)

			if tc.wantErr {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			s.Assert().ElementsMatch(tc.wantNames, repoNames(repos))
		})
	}
}

func repoNames(repos []repo.Repo) []string {
	names := make([]string, len(repos))

	for i, r := range repos {
		names[i] = r.Name
	}

	return names
}
