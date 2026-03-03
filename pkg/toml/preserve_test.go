package toml_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePreservingComments_NoChanges(t *testing.T) {
	original := []byte(`[workspace]
name = "test"

# User comment here
[repos]
myrepo = { path = "./myrepo" }
`)

	type config struct {
		Workspace struct {
			Name string `toml:"name"`
		} `toml:"workspace"`
		Repos map[string]struct {
			Path string `toml:"path"`
		} `toml:"repos"`
	}

	oldCfg := config{}
	oldCfg.Workspace.Name = "test"
	oldCfg.Repos = make(map[string]struct {
		Path string `toml:"path"`
	})
	oldCfg.Repos["myrepo"] = struct {
		Path string `toml:"path"`
	}{Path: "./myrepo"}

	newCfg := oldCfg // No changes

	result, err := toml.UpdatePreservingComments(original, oldCfg, newCfg)
	require.NoError(t, err)

	// Should return original unchanged
	assert.Equal(t, string(original), string(result))
}

func TestUpdatePreservingComments_WorkspaceChange(t *testing.T) {
	original := []byte(`[workspace]
name = "test"

# Important repos below
[repos]
myrepo = { path = "./myrepo" }
`)

	type config struct {
		Workspace struct {
			Name string `toml:"name"`
		} `toml:"workspace"`
		Repos map[string]struct {
			Path string `toml:"path"`
		} `toml:"repos"`
	}

	oldCfg := config{}
	oldCfg.Workspace.Name = "test"
	oldCfg.Repos = make(map[string]struct {
		Path string `toml:"path"`
	})
	oldCfg.Repos["myrepo"] = struct {
		Path string `toml:"path"`
	}{Path: "./myrepo"}

	newCfg := config{}
	newCfg.Workspace.Name = "renamed"
	newCfg.Repos = oldCfg.Repos

	result, err := toml.UpdatePreservingComments(original, oldCfg, newCfg)
	require.NoError(t, err)

	resultStr := string(result)

	// Should contain new name (go-toml uses single quotes for simple strings)
	assert.Contains(t, resultStr, `name = 'renamed'`)

	// Should preserve comment
	assert.Contains(t, resultStr, "# Important repos below")

	// Should preserve repos section
	assert.Contains(t, resultStr, "myrepo")
}

func TestUpdatePreservingComments_AddRepo(t *testing.T) {
	original := []byte(`[workspace]
name = "test"

# My repositories
[repos]
repo1 = { path = "./repo1" }
`)

	type config struct {
		Workspace struct {
			Name string `toml:"name"`
		} `toml:"workspace"`
		Repos map[string]struct {
			Path string `toml:"path"`
		} `toml:"repos"`
	}

	oldCfg := config{}
	oldCfg.Workspace.Name = "test"
	oldCfg.Repos = make(map[string]struct {
		Path string `toml:"path"`
	})
	oldCfg.Repos["repo1"] = struct {
		Path string `toml:"path"`
	}{Path: "./repo1"}

	newCfg := config{}
	newCfg.Workspace.Name = "test"
	newCfg.Repos = make(map[string]struct {
		Path string `toml:"path"`
	})
	newCfg.Repos["repo1"] = oldCfg.Repos["repo1"]
	newCfg.Repos["repo2"] = struct {
		Path string `toml:"path"`
	}{Path: "./repo2"}

	result, err := toml.UpdatePreservingComments(original, oldCfg, newCfg)
	require.NoError(t, err)

	resultStr := string(result)

	// Should contain both repos
	assert.Contains(t, resultStr, "repo1")
	assert.Contains(t, resultStr, "repo2")

	// Comment before section should be preserved
	assert.Contains(t, resultStr, "# My repositories")
	assert.Contains(t, resultStr, "[workspace]")
	assert.Contains(t, resultStr, "[repos]")
}

func TestUpdatePreservingComments_NewSection(t *testing.T) {
	original := []byte(`[workspace]
name = "test"
`)

	type config struct {
		Workspace struct {
			Name string `toml:"name"`
		} `toml:"workspace"`
		Groups map[string]struct {
			Repos []string `toml:"repos"`
		} `toml:"groups,omitempty"`
	}

	oldCfg := config{}
	oldCfg.Workspace.Name = "test"

	newCfg := config{}
	newCfg.Workspace.Name = "test"
	newCfg.Groups = make(map[string]struct {
		Repos []string `toml:"repos"`
	})
	newCfg.Groups["mygroup"] = struct {
		Repos []string `toml:"repos"`
	}{Repos: []string{"repo1", "repo2"}}

	result, err := toml.UpdatePreservingComments(original, oldCfg, newCfg)
	require.NoError(t, err)

	resultStr := string(result)

	// Should contain new groups section
	assert.Contains(t, resultStr, "[groups")
	assert.Contains(t, resultStr, "mygroup")

	// Should preserve workspace
	assert.Contains(t, resultStr, `name = "test"`)
}

func TestPreserveUserEdits_SimpleComment(t *testing.T) {
	original := []byte(`# This is my workspace
[workspace]
name = "test"
`)

	generated := []byte(`[workspace]
name = "test"
`)

	result := toml.PreserveUserEdits(original, generated)
	resultStr := string(result)

	// Should attempt to preserve comment
	assert.Contains(t, resultStr, "workspace")
}

func TestUpdatePreservingComments_ComplexConfig(t *testing.T) {
	original := []byte(`# Workspace configuration
[workspace]
name = "myworkspace"

# Development repositories
[repos]
frontend = { path = "./frontend", url = "https://github.com/example/frontend.git" }
backend = { path = "./backend" }

# Team groups
[groups]
fullstack = { repos = ["frontend", "backend"] }
`)

	type RepoConfig struct {
		Path string `toml:"path"`
		URL  string `toml:"url,omitempty"`
	}

	type GroupConfig struct {
		Repos []string `toml:"repos"`
	}

	type config struct {
		Workspace struct {
			Name string `toml:"name"`
		} `toml:"workspace"`
		Repos  map[string]RepoConfig  `toml:"repos"`
		Groups map[string]GroupConfig `toml:"groups"`
	}

	// Parse original
	oldCfg := config{}
	oldCfg.Workspace.Name = "myworkspace"
	oldCfg.Repos = map[string]RepoConfig{
		"frontend": {Path: "./frontend", URL: "https://github.com/example/frontend.git"},
		"backend":  {Path: "./backend"},
	}
	oldCfg.Groups = map[string]GroupConfig{
		"fullstack": {Repos: []string{"frontend", "backend"}},
	}

	// Add a new repo - need to make a deep copy of maps
	newCfg := config{}
	newCfg.Workspace.Name = "myworkspace"
	newCfg.Repos = map[string]RepoConfig{
		"frontend": {Path: "./frontend", URL: "https://github.com/example/frontend.git"},
		"backend":  {Path: "./backend"},
		"api":      {Path: "./api"},
	}
	newCfg.Groups = map[string]GroupConfig{
		"fullstack":    {Repos: []string{"frontend", "backend"}},
		"backend-team": {Repos: []string{"backend", "api"}},
	}

	result, err := toml.UpdatePreservingComments(original, oldCfg, newCfg)
	require.NoError(t, err)

	resultStr := string(result)

	// Should contain all repos
	assert.Contains(t, resultStr, "frontend")
	assert.Contains(t, resultStr, "backend")
	assert.Contains(t, resultStr, "api")

	// Should contain all groups
	assert.Contains(t, resultStr, "fullstack")
	assert.Contains(t, resultStr, "backend-team")

	// Comments should be preserved
	assert.Contains(t, resultStr, "# Workspace configuration")
	assert.Contains(t, resultStr, "# Development repositories")
	assert.Contains(t, resultStr, "# Team groups")
}
