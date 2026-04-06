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

func TestLoadStream(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantErr   bool
		wantErrIs error
		check     func(t *testing.T, m *WorkstreamManifest)
	}{
		{
			name: "full manifest parses all fields",
			content: `
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
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				if m.Name != "INFRA-42" {
					t.Errorf("Name = %q, want INFRA-42", m.Name)
				}
				if m.Description != "Add RDS config to all environments" {
					t.Errorf("Description = %q", m.Description)
				}
				if m.Workspace != "platform-infra" {
					t.Errorf("Workspace = %q", m.Workspace)
				}
				if m.Status != StatusActive {
					t.Errorf("Status = %q, want active", m.Status)
				}
				if m.Created != "2026-03-15" {
					t.Errorf("Created = %q", m.Created)
				}
				if len(m.Worktrees) != 2 {
					t.Fatalf("len(Worktrees) = %d, want 2", len(m.Worktrees))
				}
				if m.Worktrees[0].Repo != "infra-dev" {
					t.Errorf("Worktrees[0].Repo = %q", m.Worktrees[0].Repo)
				}
				if m.Ship.ShippedAt != "2026-01-01" {
					t.Errorf("Ship.ShippedAt = %q", m.Ship.ShippedAt)
				}
				if m.Context.Summary != "summary text" {
					t.Errorf("Context.Summary = %q", m.Context.Summary)
				}
				if len(m.Context.KeyDecisions) != 2 || m.Context.KeyDecisions[0] != "dec1" {
					t.Errorf("Context.KeyDecisions = %v", m.Context.KeyDecisions)
				}
			},
		},
		{
			name: "name defaults to repo for single-occurrence repo with no name field",
			content: `
name = "ws"
[[worktree]]
repo   = "infra-dev"
branch = "feat/foo"
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				if m.Worktrees[0].Name != "infra-dev" {
					t.Errorf("Name = %q, want infra-dev", m.Worktrees[0].Name)
				}
			},
		},
		{
			name: "path defaults to name when name is set and path is omitted",
			content: `
name = "ws"
[[worktree]]
repo   = "some-repo"
name   = "dev"
branch = "feat/foo"
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				if m.Worktrees[0].Path != "dev" {
					t.Errorf("Path = %q, want dev", m.Worktrees[0].Path)
				}
			},
		},
		{
			name: "name and path both default from repo when both omitted (single-occurrence)",
			content: `
name = "ws"
[[worktree]]
repo = "infra-dev"
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				if m.Worktrees[0].Name != "infra-dev" {
					t.Errorf("Name = %q, want infra-dev", m.Worktrees[0].Name)
				}
				if m.Worktrees[0].Path != "infra-dev" {
					t.Errorf("Path = %q, want infra-dev", m.Worktrees[0].Path)
				}
			},
		},
		{
			name: "path stays as-is when explicitly set",
			content: `
name = "ws"
[[worktree]]
repo   = "infra-dev"
name   = "dev"
path   = "custom/path"
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				if m.Worktrees[0].Path != "custom/path" {
					t.Errorf("Path = %q, want custom/path", m.Worktrees[0].Path)
				}
			},
		},
		{
			name:      "missing file returns os.ErrNotExist",
			wantErr:   true,
			wantErrIs: os.ErrNotExist,
		},
		{
			name: "duplicate name values produce error containing the duplicate name",
			content: `
name = "ws"
[[worktree]]
repo = "repo-a"
name = "dev"

[[worktree]]
repo = "repo-b"
name = "dev"
`,
			wantErr: true,
			check: func(t *testing.T, m *WorkstreamManifest) {
				// m should be nil
			},
		},
		{
			name: "duplicate path values produce error",
			content: `
name = "ws"
[[worktree]]
repo = "repo-a"
name = "alpha"
path = "shared-path"

[[worktree]]
repo = "repo-b"
name = "beta"
path = "shared-path"
`,
			wantErr: true,
		},
		{
			name: "multi-occurrence repo with missing name produces error containing repo name",
			content: `
name = "ws"
[[worktree]]
repo = "consolidated-infra"
name = "dev"

[[worktree]]
repo = "consolidated-infra"
`,
			wantErr: true,
		},
		{
			name: "scope field preserved verbatim",
			content: `
name = "ws"
[[worktree]]
repo  = "infra-dev"
scope = "environments/dev"
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				if m.Worktrees[0].Scope != "environments/dev" {
					t.Errorf("Scope = %q, want environments/dev", m.Worktrees[0].Scope)
				}
			},
		},
		{
			name: "ShipState.PreShipBranches round-trips as map[string]string",
			content: `
name = "ws"

[ship.pre_ship_branches]
dev  = "backup-dev"
prod = "backup-prod"
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				if m.Ship.PreShipBranches["dev"] != "backup-dev" {
					t.Errorf("PreShipBranches[dev] = %q", m.Ship.PreShipBranches["dev"])
				}
				if m.Ship.PreShipBranches["prod"] != "backup-prod" {
					t.Errorf("PreShipBranches[prod] = %q", m.Ship.PreShipBranches["prod"])
				}
			},
		},
		{
			name: "StreamContext.KeyDecisions round-trips as []string",
			content: `
name = "ws"

[context]
key_decisions = ["decision one", "decision two", "decision three"]
`,
			check: func(t *testing.T, m *WorkstreamManifest) {
				want := []string{"decision one", "decision two", "decision three"}
				if len(m.Context.KeyDecisions) != len(want) {
					t.Fatalf("KeyDecisions = %v, want %v", m.Context.KeyDecisions, want)
				}
				for i, v := range want {
					if m.Context.KeyDecisions[i] != v {
						t.Errorf("KeyDecisions[%d] = %q, want %q", i, m.Context.KeyDecisions[i], v)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.content != "" {
				path = writeStreamFixture(t, t.TempDir(), tt.content)
			} else {
				path = filepath.Join(t.TempDir(), ".gitw-stream")
			}

			m, err := LoadStream(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("errors.Is(err, %v) = false, err = %v", tt.wantErrIs, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, m)
			}
		})
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
