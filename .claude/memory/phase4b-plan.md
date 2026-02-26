# Phase 4b: Group Path Management — Implementation Plan

**Status:** Complete
**Goal:** Make `context auto` usable without manual TOML editing by letting users set and update `path` on a group via CLI commands.

**Problem:** `GroupConfig.Path` is only consumed by `context auto` but is never set by any CLI command today. Users must hand-edit `.gitworkspace` to use auto-detection.

---

## Changes Required

Two files change:

| File | Change |
|------|--------|
| `cmd/group.go` | Add `--path` flag to `groupAddCmd`; add `groupSetPathCmd`; add `groupEditCmd` |
| `cmd/group_test.go` | Extend `TestGroupAdd` table; add `TestGroupSetPath`; add `TestGroupEdit` |

No changes needed in `internal/config/` — `GroupConfig.Path` already exists and `config.Save` already marshals it.

---

## cmd/group.go Changes

### 1. Package-level flag vars

Add alongside existing `var groupName string`:

```go
var groupAddPath    string // --path on group add
var groupEditPath   string // --path on group edit
var groupClearPath  bool   // --clear-path on group edit
```

### 2. New command declarations

```go
var groupSetPathCmd = &cobra.Command{
    Use:   "set-path <name> <path>",
    Short: "Set the filesystem path for a group (used by context auto)",
    Args:  cobra.ExactArgs(2),
    RunE:  runGroupSetPath,
}

var groupEditCmd = &cobra.Command{
    Use:   "edit <name>",
    Short: "Edit group attributes",
    Args:  cobra.ExactArgs(1),
    RunE:  runGroupEdit,
}
```

### 3. Update `init()`

Add to the `groupCmd.AddCommand(...)` call:

```go
groupCmd.AddCommand(..., groupSetPathCmd, groupEditCmd)
```

Add flag registrations:

```go
groupAddCmd.Flags().StringVar(&groupAddPath, "path", "", "filesystem path for auto-context detection")

groupEditCmd.Flags().StringVar(&groupEditPath, "path", "", "set group path")
groupEditCmd.Flags().BoolVar(&groupClearPath, "clear-path", false, "clear group path")
```

### 4. Update `runGroupAdd`

After `g.Repos = appendUnique(g.Repos, args)`, apply path only when the flag was explicitly provided:

```go
if cmd.Flags().Changed("path") {
    g.Path = groupAddPath
}
```

Use `Changed()` rather than comparing to `""` so that a bare `group add` on an existing group with a path does not accidentally clear it.

### 5. `runGroupSetPath`

```go
func runGroupSetPath(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil {
        return err
    }

    name, path := args[0], args[1]
    g, ok := cfg.Groups[name]
    if !ok {
        return fmt.Errorf("group %q not found", name)
    }

    g.Path = path
    cfg.Groups[name] = g

    if err := config.Save(cfgPath, cfg); err != nil {
        return err
    }

    fmt.Fprintf(cmd.OutOrStdout(), "Group %q path set to %q\n", name, path)
    return nil
}
```

Note: passing `""` as path is valid — it clears the path. `set-path` is the scripting-friendly option; `edit --clear-path` is the human-friendly option.

### 6. `runGroupEdit`

```go
func runGroupEdit(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil {
        return err
    }

    name := args[0]
    g, ok := cfg.Groups[name]
    if !ok {
        return fmt.Errorf("group %q not found", name)
    }

    pathChanged := cmd.Flags().Changed("path")
    if !pathChanged && !groupClearPath {
        return fmt.Errorf("at least one of --path or --clear-path must be provided")
    }
    if pathChanged && groupClearPath {
        return fmt.Errorf("--path and --clear-path are mutually exclusive")
    }

    if groupClearPath {
        g.Path = ""
    } else {
        g.Path = groupEditPath
    }

    cfg.Groups[name] = g

    if err := config.Save(cfgPath, cfg); err != nil {
        return err
    }

    fmt.Fprintf(cmd.OutOrStdout(), "Group %q updated\n", name)
    return nil
}
```

---

## cmd/group_test.go Changes

### 1. Extend `TestGroupAdd`

Add `wantPath string` field to the existing table struct and two new cases:

```go
{
    name:      "create group with path",
    repos:     []string{"frontend"},
    cmdArgs:   []string{"group", "add", "-n", "web", "--path", "apps", "frontend"},
    wantRepos: []string{"frontend"},
    wantPath:  "apps",
},
{
    name:      "add repos without --path preserves existing path",
    repos:     []string{"frontend", "backend"},
    extraTOML: "[groups.web]\nrepos = [\"frontend\"]\npath = \"apps\"\n",
    cmdArgs:   []string{"group", "add", "-n", "web", "backend"},
    wantRepos: []string{"frontend", "backend"},
    wantPath:  "apps",
},
```

In the assertion block, add:

```go
s.Assert().Equal(tc.wantPath, cfg.Groups[groupName].Path)
```

Existing cases with no `wantPath` field default to `""` — confirm the assertion holds for them too.

**Pflag caveat:** `--path apps` provided in one case will leave `groupAddPath == "apps"` for subsequent cases unless cobra resets it. Since each case calls `makeGroupWs` (which creates a fresh workspace), the cobra flag value itself is the risk. Existing cases do not pass `--path`, so `Changed("path")` returns false — path is not applied. No issue.

### 2. Add `TestGroupSetPath`

```go
func (s *GroupSuite) TestGroupSetPath() {
    cases := []struct {
        name      string
        extraTOML string
        cmdArgs   []string
        wantPath  string
        wantErr   bool
    }{
        {
            name:      "sets path on existing group",
            extraTOML: "[groups.web]\nrepos = []\n",
            cmdArgs:   []string{"group", "set-path", "web", "apps"},
            wantPath:  "apps",
        },
        {
            name:      "overwrites existing path",
            extraTOML: "[groups.web]\nrepos = []\npath = \"old\"\n",
            cmdArgs:   []string{"group", "set-path", "web", "new"},
            wantPath:  "new",
        },
        {
            name:    "error if group not found",
            cmdArgs: []string{"group", "set-path", "nonexistent", "apps"},
            wantErr: true,
        },
    }
    for _, tc := range cases {
        s.Run(tc.name, func() {
            wsDir := s.makeGroupWs(nil, tc.extraTOML)

            _, err := execCmd(s.T(), tc.cmdArgs...)

            if tc.wantErr {
                s.Require().Error(err)
                return
            }

            s.Require().NoError(err)
            cfg, loadErr := config.Load(filepath.Join(wsDir, ".gitworkspace"))
            s.Require().NoError(loadErr)
            s.Assert().Equal(tc.wantPath, cfg.Groups["web"].Path)
        })
    }
}
```

### 3. Add `TestGroupEdit`

```go
func (s *GroupSuite) TestGroupEdit() {
    cases := []struct {
        name      string
        extraTOML string
        cmdArgs   []string
        wantPath  string
        wantErr   bool
    }{
        {
            name:      "sets path with --path flag",
            extraTOML: "[groups.web]\nrepos = []\n",
            cmdArgs:   []string{"group", "edit", "web", "--path", "apps"},
            wantPath:  "apps",
        },
        {
            name:      "clears path with --clear-path",
            extraTOML: "[groups.web]\nrepos = []\npath = \"apps\"\n",
            cmdArgs:   []string{"group", "edit", "web", "--clear-path"},
            wantPath:  "",
        },
        {
            name:      "error when no flags given",
            extraTOML: "[groups.web]\nrepos = []\n",
            cmdArgs:   []string{"group", "edit", "web"},
            wantErr:   true,
        },
        {
            name:      "error when --path and --clear-path both given",
            extraTOML: "[groups.web]\nrepos = []\n",
            cmdArgs:   []string{"group", "edit", "web", "--path", "apps", "--clear-path"},
            wantErr:   true,
        },
        {
            name:    "error if group not found",
            cmdArgs: []string{"group", "edit", "nonexistent", "--path", "apps"},
            wantErr: true,
        },
    }
    for _, tc := range cases {
        s.Run(tc.name, func() {
            wsDir := s.makeGroupWs(nil, tc.extraTOML)

            _, err := execCmd(s.T(), tc.cmdArgs...)

            if tc.wantErr {
                s.Require().Error(err)
                return
            }

            s.Require().NoError(err)
            cfg, loadErr := config.Load(filepath.Join(wsDir, ".gitworkspace"))
            s.Require().NoError(loadErr)
            s.Assert().Equal(tc.wantPath, cfg.Groups["web"].Path)
        })
    }
}
```

---

## Exit Criteria

1. `group add -n web --path apps frontend` → group "web" created, `path = "apps"` in config
2. `group add -n web backend` (group already has path "apps") → path preserved, backend added
3. `group set-path web newpath` → `path = "newpath"` in config
4. `group set-path nonexistent apps` → error
5. `group edit web --path apps` → `path = "apps"` in config
6. `group edit web --clear-path` → `path` removed/empty in config
7. `group edit web` (no flags) → error
8. `group edit web --path apps --clear-path` → error
9. `group edit nonexistent --path apps` → error
10. After any of the above, `context auto` resolves correctly when CWD is under the group's path
11. `go test -race -count=1 ./...` passes

---

## Notes

- `config.Save` already handles `Path` correctly because `GroupConfig.Path` has `toml:"path,omitempty"` — empty path is omitted from the file entirely, which is the desired behavior for "no path set".
- `cmd.Flags().Changed("path")` is essential in `runGroupAdd` to distinguish "user didn't pass --path" from "user passed --path with value". Without it, calling `group add` on an existing group would silently clear its path.
- No changes to `internal/config/` are needed.
- Test setup uses direct TOML writes (not `execCmd("group set-path", ...)`) to avoid pflag state contamination across sub-tests.
