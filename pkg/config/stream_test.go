package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeStreamFixture(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".gitw-stream")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadStreamParsesFullManifest(t *testing.T) {
	manifest := loadStreamFixture(t, `
name        = "INFRA-42"
description = "Add RDS config to all environments"
workspace   = "platform-infra"
status      = "active"
created     = "2026-03-15"

[[worktree]]
repo   = "infra-dev"
branch = "feat/INFRA-42-new-rds"
path   = "infra-dev"

[[worktree]]
repo   = "infra-test"
branch = "feat/INFRA-42-new-rds"

[ship]
pr_urls             = ["https://github.com/org/repo/pull/1"]
shipped_at          = "2026-01-01"

[ship.pre_ship_branches]
dev = "backup-dev"

[context]
summary       = "summary text"
key_decisions = ["dec1", "dec2"]
`)

	if manifest.Name != "INFRA-42" {
		t.Errorf("Name = %q, want INFRA-42", manifest.Name)
	}

	if manifest.Description != "Add RDS config to all environments" {
		t.Errorf("Description = %q", manifest.Description)
	}

	if manifest.Workspace != "platform-infra" {
		t.Errorf("Workspace = %q", manifest.Workspace)
	}

	if manifest.Status != StatusActive {
		t.Errorf("Status = %q, want active", manifest.Status)
	}

	if manifest.Created != "2026-03-15" {
		t.Errorf("Created = %q", manifest.Created)
	}

	if len(manifest.Worktrees) != 2 {
		t.Fatalf("len(Worktrees) = %d, want 2", len(manifest.Worktrees))
	}

	if manifest.Worktrees[0].Repo != "infra-dev" {
		t.Errorf("Worktrees[0].Repo = %q", manifest.Worktrees[0].Repo)
	}

	if manifest.Ship.ShippedAt != "2026-01-01" {
		t.Errorf("Ship.ShippedAt = %q", manifest.Ship.ShippedAt)
	}

	if manifest.Context.Summary != "summary text" {
		t.Errorf("Context.Summary = %q", manifest.Context.Summary)
	}

	if len(manifest.Context.KeyDecisions) != 2 || manifest.Context.KeyDecisions[0] != "dec1" {
		t.Errorf("Context.KeyDecisions = %v", manifest.Context.KeyDecisions)
	}
}

func TestLoadStreamDefaultsWorktreeNameFromRepo(t *testing.T) {
	manifest := loadStreamFixture(t, `
name = "ws"
[[worktree]]
repo   = "infra-dev"
branch = "feat/foo"
`)

	if manifest.Worktrees[0].Name != "infra-dev" {
		t.Errorf("Name = %q, want infra-dev", manifest.Worktrees[0].Name)
	}
}

func TestLoadStreamDefaultsWorktreePathFromName(t *testing.T) {
	manifest := loadStreamFixture(t, `
name = "ws"
[[worktree]]
repo   = "some-repo"
name   = "dev"
branch = "feat/foo"
`)

	if manifest.Worktrees[0].Path != "dev" {
		t.Errorf("Path = %q, want dev", manifest.Worktrees[0].Path)
	}
}

func TestLoadStreamDefaultsWorktreeNameAndPathFromRepo(t *testing.T) {
	manifest := loadStreamFixture(t, `
name = "ws"
[[worktree]]
repo = "infra-dev"
`)

	if manifest.Worktrees[0].Name != "infra-dev" {
		t.Errorf("Name = %q, want infra-dev", manifest.Worktrees[0].Name)
	}

	if manifest.Worktrees[0].Path != "infra-dev" {
		t.Errorf("Path = %q, want infra-dev", manifest.Worktrees[0].Path)
	}
}

func TestLoadStreamPreservesExplicitWorktreePath(t *testing.T) {
	manifest := loadStreamFixture(t, `
name = "ws"
[[worktree]]
repo   = "infra-dev"
name   = "dev"
path   = "custom/path"
`)

	if manifest.Worktrees[0].Path != "custom/path" {
		t.Errorf("Path = %q, want custom/path", manifest.Worktrees[0].Path)
	}
}

func TestLoadStreamReturnsNotExistForMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".gitw-stream")

	_, err := LoadStream(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("errors.Is(err, os.ErrNotExist) = false, err = %v", err)
	}
}

func TestLoadStreamRejectsDuplicateWorktreeName(t *testing.T) {
	path := writeStreamFixture(t, t.TempDir(), `
name = "ws"
[[worktree]]
repo = "repo-a"
name = "dev"

[[worktree]]
repo = "repo-b"
name = "dev"
`)

	_, err := LoadStream(path)
	assertErrorContains(t, err, "dev")
}

func TestLoadStreamRejectsDuplicateWorktreePath(t *testing.T) {
	path := writeStreamFixture(t, t.TempDir(), `
name = "ws"
[[worktree]]
repo = "repo-a"
name = "alpha"
path = "shared-path"

[[worktree]]
repo = "repo-b"
name = "beta"
path = "shared-path"
`)

	_, err := LoadStream(path)
	assertErrorContains(t, err, "shared-path")
}

func TestLoadStreamRejectsMissingNameForMultiOccurrenceRepo(t *testing.T) {
	path := writeStreamFixture(t, t.TempDir(), `
name = "ws"
[[worktree]]
repo = "consolidated-infra"
name = "dev"

[[worktree]]
repo = "consolidated-infra"
`)

	_, err := LoadStream(path)
	assertErrorContains(t, err, "consolidated-infra")
}

func TestLoadStreamPreservesScopeField(t *testing.T) {
	manifest := loadStreamFixture(t, `
name = "ws"
[[worktree]]
repo  = "infra-dev"
scope = "environments/dev"
`)

	if manifest.Worktrees[0].Scope != "environments/dev" {
		t.Errorf("Scope = %q, want environments/dev", manifest.Worktrees[0].Scope)
	}
}

func TestLoadStreamPreservesPreShipBranches(t *testing.T) {
	manifest := loadStreamFixture(t, `
name = "ws"

[ship.pre_ship_branches]
dev  = "backup-dev"
prod = "backup-prod"
`)

	if manifest.Ship.PreShipBranches["dev"] != "backup-dev" {
		t.Errorf("PreShipBranches[dev] = %q", manifest.Ship.PreShipBranches["dev"])
	}

	if manifest.Ship.PreShipBranches["prod"] != "backup-prod" {
		t.Errorf("PreShipBranches[prod] = %q", manifest.Ship.PreShipBranches["prod"])
	}
}

func TestLoadStreamPreservesKeyDecisions(t *testing.T) {
	manifest := loadStreamFixture(t, `
name = "ws"

[context]
key_decisions = ["decision one", "decision two", "decision three"]
	`)

	want := []string{"decision one", "decision two", "decision three"}
	if len(manifest.Context.KeyDecisions) != len(want) {
		t.Fatalf("KeyDecisions = %v, want %v", manifest.Context.KeyDecisions, want)
	}

	for i, value := range want {
		if manifest.Context.KeyDecisions[i] != value {
			t.Errorf("KeyDecisions[%d] = %q, want %q", i, manifest.Context.KeyDecisions[i], value)
		}
	}
}

func loadStreamFixture(t *testing.T, content string) *WorkstreamManifest {
	t.Helper()

	path := writeStreamFixture(t, t.TempDir(), content)
	manifest, err := LoadStream(path)
	if err != nil {
		t.Fatalf("LoadStream() error = %v", err)
	}

	return manifest
}

func assertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error %q does not contain %q", err.Error(), substr)
	}
}

func TestApplyStreamDefaults(t *testing.T) {
	tests := []struct {
		name      string
		worktrees []WorktreeEntry
		want      []WorktreeEntry
	}{
		{
			name: "single-occurrence repo: name and path default to repo",
			worktrees: []WorktreeEntry{
				{Repo: "infra-dev"},
			},
			want: []WorktreeEntry{
				{Repo: "infra-dev", Name: "infra-dev", Path: "infra-dev"},
			},
		},
		{
			name: "explicit name: path defaults to name",
			worktrees: []WorktreeEntry{
				{Repo: "some-repo", Name: "dev"},
			},
			want: []WorktreeEntry{
				{Repo: "some-repo", Name: "dev", Path: "dev"},
			},
		},
		{
			name: "explicit path: path not overwritten",
			worktrees: []WorktreeEntry{
				{Repo: "some-repo", Name: "dev", Path: "custom/path"},
			},
			want: []WorktreeEntry{
				{Repo: "some-repo", Name: "dev", Path: "custom/path"},
			},
		},
		{
			name: "multi-occurrence repo: name not defaulted",
			worktrees: []WorktreeEntry{
				{Repo: "consolidated-infra", Name: "dev"},
				{Repo: "consolidated-infra"},
			},
			want: []WorktreeEntry{
				{Repo: "consolidated-infra", Name: "dev", Path: "dev"},
				{Repo: "consolidated-infra", Name: "", Path: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &WorkstreamManifest{Worktrees: tt.worktrees}
			applyStreamDefaults(m)
			if len(m.Worktrees) != len(tt.want) {
				t.Fatalf("len(Worktrees) = %d, want %d", len(m.Worktrees), len(tt.want))
			}
			for i, got := range m.Worktrees {
				w := tt.want[i]
				if got.Name != w.Name {
					t.Errorf("[%d] Name = %q, want %q", i, got.Name, w.Name)
				}
				if got.Path != w.Path {
					t.Errorf("[%d] Path = %q, want %q", i, got.Path, w.Path)
				}
			}
		})
	}
}

func TestValidateStream(t *testing.T) {
	tests := []struct {
		name      string
		worktrees []WorktreeEntry
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid: all names and paths unique",
			worktrees: []WorktreeEntry{
				{Repo: "repo-a", Name: "alpha", Path: "alpha"},
				{Repo: "repo-b", Name: "beta", Path: "beta"},
			},
			wantErr: false,
		},
		{
			name: "invalid: duplicate name",
			worktrees: []WorktreeEntry{
				{Repo: "repo-a", Name: "dev", Path: "path-a"},
				{Repo: "repo-b", Name: "dev", Path: "path-b"},
			},
			wantErr:   true,
			errSubstr: "dev",
		},
		{
			name: "invalid: duplicate path",
			worktrees: []WorktreeEntry{
				{Repo: "repo-a", Name: "alpha", Path: "shared-path"},
				{Repo: "repo-b", Name: "beta", Path: "shared-path"},
			},
			wantErr:   true,
			errSubstr: "shared-path",
		},
		{
			name: "invalid: multi-occurrence repo missing name",
			worktrees: []WorktreeEntry{
				{Repo: "consolidated-infra", Name: "dev", Path: "dev"},
				{Repo: "consolidated-infra", Name: "", Path: ""},
			},
			wantErr:   true,
			errSubstr: "consolidated-infra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &WorkstreamManifest{Worktrees: tt.worktrees}
			err := validateStream(m)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !containsStr(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
