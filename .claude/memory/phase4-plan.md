# Phase 4: Groups + Context — Implementation Plan

**Status:** Pending
**Goal:** Full group management and context-scoped command execution.

---

## Parallel Work Streams

Three streams are fully independent and can be worked simultaneously:

| Stream | Files | Depends On |
|--------|-------|------------|
| A | `cmd/group.go`, `cmd/group_test.go` | nothing |
| B | `cmd/context.go`, `cmd/context_test.go` | nothing |
| C | `cmd/exec.go` (update), `cmd/exec_test.go` (update), `cmd/info.go` (update) | nothing (existing filterRepos is in exec.go) |

All three streams touch different files with no shared mutable state. Assign each to a separate worker.

---

## Stream A: `cmd/group.go` + `cmd/group_test.go`

### A1: `cmd/group.go`

**Command tree:**

```
group (alias: g)
├── add <repos...> -n <name>
├── rm <name>
├── rename <old> <new>
├── rmrepo <repos...> -n <name>
├── list (alias: ls)
└── info [name] (alias: ll)
```

**Package-level vars:**

```go
var groupName string  // shared by group add -n and group rmrepo -n

var groupCmd = &cobra.Command{
    Use:     "group",
    Aliases: []string{"g"},
    Short:   "Manage repo groups",
}

var groupAddCmd = &cobra.Command{
    Use:   "add <repos...>",
    Short: "Create a group or add repos to an existing group",
    Args:  cobra.MinimumNArgs(1),
    RunE:  runGroupAdd,
}
var groupRmCmd = &cobra.Command{
    Use:   "rm <name>",
    Short: "Delete a group",
    Args:  cobra.ExactArgs(1),
    RunE:  runGroupRm,
}
var groupRenameCmd = &cobra.Command{
    Use:   "rename <old> <new>",
    Short: "Rename a group",
    Args:  cobra.ExactArgs(2),
    RunE:  runGroupRename,
}
var groupRmrepoCmd = &cobra.Command{
    Use:   "rmrepo <repos...>",
    Short: "Remove repos from a group",
    Args:  cobra.MinimumNArgs(1),
    RunE:  runGroupRmrepo,
}
var groupListCmd = &cobra.Command{
    Use:     "list",
    Aliases: []string{"ls"},
    Short:   "List group names",
    Args:    cobra.NoArgs,
    RunE:    runGroupList,
}
var groupInfoCmd = &cobra.Command{
    Use:     "info [name]",
    Aliases: []string{"ll"},
    Short:   "List groups with their repos",
    Args:    cobra.MaximumNArgs(1),
    RunE:    runGroupInfo,
}
```

**`init()`:**

```go
func init() {
    rootCmd.AddCommand(groupCmd)
    groupCmd.AddCommand(groupAddCmd, groupRmCmd, groupRenameCmd, groupRmrepoCmd, groupListCmd, groupInfoCmd)
    groupAddCmd.Flags().StringVarP(&groupName, "name", "n", "", "group name (required)")
    _ = groupAddCmd.MarkFlagRequired("name")
    groupRmrepoCmd.Flags().StringVarP(&groupName, "name", "n", "", "group name (required)")
    _ = groupRmrepoCmd.MarkFlagRequired("name")
}
```

**`runGroupAdd`** — create/append, skip duplicates, error if repo not registered:

```go
func runGroupAdd(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil { return err }
    for _, name := range args {
        if _, ok := cfg.Repos[name]; !ok {
            return fmt.Errorf("repo %q not found", name)
        }
    }
    g := cfg.Groups[groupName]
    g.Repos = appendUnique(g.Repos, args)
    cfg.Groups[groupName] = g
    if err := config.Save(cfgPath, cfg); err != nil { return err }
    fmt.Fprintf(cmd.OutOrStdout(), "Group %q updated\n", groupName)
    return nil
}
```

Private helper (extract to avoid duplication):

```go
// appendUnique appends items to slice, skipping any already present.
func appendUnique(existing, items []string) []string {
    set := make(map[string]bool, len(existing))
    for _, s := range existing { set[s] = true }
    result := existing
    for _, s := range items {
        if !set[s] {
            set[s] = true
            result = append(result, s)
        }
    }
    return result
}
```

**`runGroupRm`:**

```go
func runGroupRm(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil { return err }
    name := args[0]
    if _, ok := cfg.Groups[name]; !ok {
        return fmt.Errorf("group %q not found", name)
    }
    delete(cfg.Groups, name)
    if err := config.Save(cfgPath, cfg); err != nil { return err }
    fmt.Fprintf(cmd.OutOrStdout(), "Group %q removed\n", name)
    return nil
}
```

**`runGroupRename`:**

```go
func runGroupRename(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil { return err }
    old, newName := args[0], args[1]
    if _, ok := cfg.Groups[old]; !ok {
        return fmt.Errorf("group %q not found", old)
    }
    if _, ok := cfg.Groups[newName]; ok {
        return fmt.Errorf("group %q already exists", newName)
    }
    cfg.Groups[newName] = cfg.Groups[old]
    delete(cfg.Groups, old)
    if err := config.Save(cfgPath, cfg); err != nil { return err }
    fmt.Fprintf(cmd.OutOrStdout(), "Renamed group %q to %q\n", old, newName)
    return nil
}
```

**`runGroupRmrepo`** — silently skip repos not in group (idempotent):

```go
func runGroupRmrepo(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil { return err }
    g, ok := cfg.Groups[groupName]
    if !ok {
        return fmt.Errorf("group %q not found", groupName)
    }
    g.Repos = removeItems(g.Repos, args)
    cfg.Groups[groupName] = g
    if err := config.Save(cfgPath, cfg); err != nil { return err }
    fmt.Fprintf(cmd.OutOrStdout(), "Updated group %q\n", groupName)
    return nil
}
```

Private helper:

```go
// removeItems returns slice with all items in remove deleted.
func removeItems(slice, remove []string) []string {
    drop := make(map[string]bool, len(remove))
    for _, s := range remove { drop[s] = true }
    result := slice[:0:0]
    for _, s := range slice {
        if !drop[s] {
            result = append(result, s)
        }
    }
    return result
}
```

**`runGroupList`** — sorted names:

```go
func runGroupList(cmd *cobra.Command, args []string) error {
    cfg, _, err := loadConfig()
    if err != nil { return err }
    names := sortedKeys(cfg.Groups)
    for _, n := range names {
        fmt.Fprintln(cmd.OutOrStdout(), n)
    }
    return nil
}
```

Private helper (also used by group info and possibly context):

```go
func sortedKeys[M ~map[string]V, V any](m M) []string {
    keys := make([]string, 0, len(m))
    for k := range m { keys = append(keys, k) }
    sort.Strings(keys)
    return keys
}
```

Note: Go 1.21+ supports generic `sortedKeys`. Project is on Go 1.26.

**`runGroupInfo`** — format: `<name>: repo1, repo2, ...`:

```go
func runGroupInfo(cmd *cobra.Command, args []string) error {
    cfg, _, err := loadConfig()
    if err != nil { return err }
    if len(args) == 1 {
        return printGroupInfo(cmd.OutOrStdout(), cfg, args[0])
    }
    for _, name := range sortedKeys(cfg.Groups) {
        if err := printGroupInfo(cmd.OutOrStdout(), cfg, name); err != nil { return err }
    }
    return nil
}

func printGroupInfo(w io.Writer, cfg *config.WorkspaceConfig, name string) error {
    g, ok := cfg.Groups[name]
    if !ok {
        return fmt.Errorf("group %q not found", name)
    }
    fmt.Fprintf(w, "%s: %s\n", name, strings.Join(g.Repos, ", "))
    return nil
}
```

---

### A2: `cmd/group_test.go`

**Suite declaration:**

```go
package cmd

import (
    "os"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/suite"
)

type GroupSuite struct {
    suite.Suite
}

func TestGroup(t *testing.T) { suite.Run(t, new(GroupSuite)) }
```

**Workspace helper** — writes .gitworkspace with fake repos (no git init needed; group commands only check cfg.Repos):

```go
// makeGroupWs creates a workspace TOML with the given repo names (dirs not needed)
// and optional additional TOML appended. Returns wsDir.
func (s *GroupSuite) makeGroupWs(repoNames []string, extraTOML string) string {
    wsDir := s.T().TempDir()
    var sb strings.Builder
    sb.WriteString("[workspace]\nname = \"test\"\n")
    for _, name := range repoNames {
        fmt.Fprintf(&sb, "[repos.%s]\npath = %q\n", name, name)
    }
    sb.WriteString(extraTOML)
    s.Require().NoError(os.WriteFile(
        filepath.Join(wsDir, ".gitworkspace"),
        []byte(sb.String()), 0o644,
    ))
    changeToDir(s.T(), wsDir)
    return wsDir
}
```

**`TestGroupAdd`** — table-driven success + error paths:

```go
func (s *GroupSuite) TestGroupAdd() {
    cases := []struct {
        name     string
        repos    []string  // pre-existing repo names in config
        addArgs  []string  // repos passed to group add
        groupArg string    // -n value
        wantErr  bool
        wantRepos []string // expected group.Repos after op
    }{
        {
            name: "create new group",
            repos: []string{"frontend", "backend"},
            addArgs: []string{"frontend", "backend"}, groupArg: "web",
            wantRepos: []string{"frontend", "backend"},
        },
        {
            name: "add to existing group (no duplicates)",
            repos: []string{"frontend", "backend", "infra"},
            // pre-create group with frontend via extraTOML, then add backend
            addArgs: []string{"backend"}, groupArg: "web",
            wantRepos: []string{"frontend", "backend"}, // frontend pre-existing
        },
        {
            name: "unknown repo", repos: []string{"frontend"},
            addArgs: []string{"notexist"}, groupArg: "web",
            wantErr: true,
        },
    }
    for _, tc := range cases {
        s.Run(tc.name, func() {
            // setup inside closure: each sub-test needs isolated workspace
            // (SetupTest does not re-run per s.Run)
            // ... write config, call execCmd, assert
        })
    }
}
```

Note: the "add to existing group" case should write the initial group state in `extraTOML`.

**`TestGroupRm`** — table-driven:

```go
cases := []struct{name, groupArg string; wantErr bool}{
    {"existing group", "web", false},
    {"nonexistent group", "nope", true},
}
```

**`TestGroupRename`** — table-driven:

```go
cases := []struct{name, old, new string; wantErr bool}{
    {"success", "web", "frontend", false},
    {"old not found", "nope", "newname", true},
    {"new already exists", "web", "ops", true},  // pre-create both web and ops
}
```

**`TestGroupRmrepo`** — table-driven:

```go
cases := []struct{name string; removeRepos []string; wantRepos []string; wantErr bool}{
    {"remove one", []string{"frontend"}, []string{"backend"}, false},
    {"remove not-in-group (no error)", []string{"nothere"}, []string{"frontend","backend"}, false},
    {"group not found", []string{"any"}, nil, true},
}
```

**`TestGroupList`:**
- Empty groups → empty output
- Multiple groups → sorted names, one per line

**`TestGroupList_AliasEquivalent`:**
- Run `group list` and `group ls`; assert outputs equal

**`TestGroupInfo`:**
- Table-driven: all groups (sorted); single group; group not found (error)

**Key pitfall reminder:** Each `s.Run` closure must set up its own isolated workspace. Do not rely on outer setup.

---

## Stream B: `cmd/context.go` + `cmd/context_test.go`

### B1: `cmd/context.go`

**Command:**

```go
var contextCmd = &cobra.Command{
    Use:   "context [group|auto|none]",
    Short: "Get or set the active repo group context",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runContext,
}

func init() {
    rootCmd.AddCommand(contextCmd)
}
```

**Dispatch:**

```go
func runContext(cmd *cobra.Command, args []string) error {
    if len(args) == 0 {
        return runContextShow(cmd)
    }
    switch args[0] {
    case "none":
        return runContextClear(cmd)
    case "auto":
        return runContextAuto(cmd)
    default:
        return runContextSet(cmd, args[0])
    }
}
```

**`runContextShow`:**

```go
func runContextShow(cmd *cobra.Command) error {
    cfg, _, err := loadConfig()
    if err != nil { return err }
    if cfg.Context.Active == "" {
        fmt.Fprintln(cmd.OutOrStdout(), "(none)")
    } else {
        fmt.Fprintln(cmd.OutOrStdout(), cfg.Context.Active)
    }
    return nil
}
```

**`runContextSet`** — validates group exists, then writes .local:

```go
func runContextSet(cmd *cobra.Command, group string) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil { return err }
    if _, ok := cfg.Groups[group]; !ok {
        return fmt.Errorf("group %q not found", group)
    }
    if err := config.SaveLocal(cfgPath, config.ContextConfig{Active: group}); err != nil {
        return err
    }
    fmt.Fprintf(cmd.OutOrStdout(), "Active context: %q\n", group)
    return nil
}
```

**`runContextClear`:**

```go
func runContextClear(cmd *cobra.Command) error {
    _, cfgPath, err := loadConfig()
    if err != nil { return err }
    if err := config.SaveLocal(cfgPath, config.ContextConfig{}); err != nil {
        return err
    }
    fmt.Fprintln(cmd.OutOrStdout(), "Context cleared")
    return nil
}
```

**`runContextAuto`:**

```go
func runContextAuto(cmd *cobra.Command) error {
    cfg, cfgPath, err := loadConfig()
    if err != nil { return err }
    group, err := detectContextFromCWD(cfg, config.ConfigDir(cfgPath))
    if err != nil { return err }
    if err := config.SaveLocal(cfgPath, config.ContextConfig{Active: group}); err != nil {
        return err
    }
    fmt.Fprintf(cmd.OutOrStdout(), "Active context: %q\n", group)
    return nil
}
```

**`detectContextFromCWD`** — finds deepest group whose path contains CWD:

```go
func detectContextFromCWD(cfg *config.WorkspaceConfig, cfgRoot string) (string, error) {
    cwd, err := os.Getwd()
    if err != nil {
        return "", fmt.Errorf("getting working directory: %w", err)
    }
    best, bestDepth := "", -1
    for name, g := range cfg.Groups {
        if g.Path == "" {
            continue
        }
        absPath := filepath.Join(cfgRoot, g.Path)
        rel, err := filepath.Rel(absPath, cwd)
        if err != nil {
            continue
        }
        if strings.HasPrefix(rel, "..") {
            continue
        }
        depth := strings.Count(filepath.Clean(absPath), string(os.PathSeparator))
        if depth > bestDepth {
            bestDepth = depth
            best = name
        }
    }
    if best == "" {
        return "", fmt.Errorf("no group with a matching path found for current directory")
    }
    return best, nil
}
```

---

### B2: `cmd/context_test.go`

**Suite:**

```go
package cmd

type ContextSuite struct {
    suite.Suite
}

func TestContext(t *testing.T) { suite.Run(t, new(ContextSuite)) }
```

**Workspace helper** — all context tests use static TOML (no real git repos):

```go
// makeContextWs writes a .gitworkspace with given groups (name → path).
// Groups with a path can be used for auto-detection.
func (s *ContextSuite) makeContextWs(repos []string, groups map[string]string) string {
    wsDir := s.T().TempDir()
    var sb strings.Builder
    sb.WriteString("[workspace]\nname = \"test\"\n")
    for _, r := range repos {
        fmt.Fprintf(&sb, "[repos.%s]\npath = %q\n", r, r)
    }
    for gName, gPath := range groups {
        fmt.Fprintf(&sb, "[groups.%s]\nrepos = []\npath = %q\n", gName, gPath)
    }
    s.Require().NoError(os.WriteFile(
        filepath.Join(wsDir, ".gitworkspace"), []byte(sb.String()), 0o644,
    ))
    changeToDir(s.T(), wsDir)
    return wsDir
}
```

**`TestContextShow`** — table-driven:

```go
cases := []struct{name, localTOML, wantOut string}{
    {"no context set",      "",                              "(none)\n"},
    {"context set to web",  "[context]\nactive = \"web\"\n", "web\n"},
}
```

For the "context set" case, write `.gitworkspace.local` directly.

**`TestContextSet`** — table-driven:

```go
cases := []struct{name, group string; wantErr bool}{
    {"valid group",   "web",  false},
    {"unknown group", "nope", true},
}
```

Verify after success: load `.gitworkspace.local`, assert `Context.Active == "web"`.

**`TestContextClear`:**
- Set context in `.local` first, then `context none`, assert `.local` has empty `Active`.

**`TestContextAuto`** — table-driven:

```go
cases := []struct{
    name      string
    groups    map[string]string // name → path (relative to wsDir)
    cwdSubdir string            // subdirectory of wsDir to cd into
    wantGroup string
    wantErr   bool
}{
    {
        name: "CWD under group path",
        groups: map[string]string{"web": "apps"},
        cwdSubdir: "apps/frontend",
        wantGroup: "web",
    },
    {
        name: "CWD not under any group path",
        groups: map[string]string{"web": "apps"},
        cwdSubdir: "services",
        wantErr: true,
    },
    {
        name: "multiple groups, picks deepest",
        groups: map[string]string{"outer": "apps", "inner": "apps/sub"},
        cwdSubdir: "apps/sub/thing",
        wantGroup: "inner",
    },
    {
        name: "group with no path is skipped",
        groups: map[string]string{}, // no path groups
        cwdSubdir: ".",
        wantErr: true,
    },
}
```

Each sub-test: create directories inside wsDir (e.g. `apps/frontend`), `changeToDir` to that subdirectory, run `context auto`, assert output or error.

**`TestContext_WritesLocal`:**
- Run `context set web` → assert `.gitworkspace.local` exists and contains `[context]\nactive = "web"`.
- Separate from TestContextSet to isolate filesystem assertion logic.

---

## Stream C: Update `cmd/exec.go` + Tests

### C1: Update `cmd/exec.go` — context-aware + group-name filtering

Replace the current `filterRepos` with three focused functions:

```go
// filterRepos resolves names as repos and/or groups.
// With no names: falls back to active context, or all repos.
// With names: expands group names to their repos; errors on unknown names.
func filterRepos(cfg *config.WorkspaceConfig, cfgPath string, names []string) ([]repo.Repo, error) {
    if len(names) == 0 {
        return reposForContext(cfg, cfgPath)
    }
    return resolveTargets(cfg, cfgPath, names)
}

// reposForContext returns the active context's repos, or all repos if no context is set.
func reposForContext(cfg *config.WorkspaceConfig, cfgPath string) ([]repo.Repo, error) {
    active := cfg.Context.Active
    if active == "" {
        return repo.FromConfig(cfg, cfgPath), nil
    }
    g, ok := cfg.Groups[active]
    if !ok {
        return nil, fmt.Errorf("active context group %q not found", active)
    }
    return groupRepos(cfg, cfgPath, g), nil
}

// resolveTargets expands each name as a repo or group name, deduplicating by repo name.
func resolveTargets(cfg *config.WorkspaceConfig, cfgPath string, names []string) ([]repo.Repo, error) {
    all := repo.FromConfig(cfg, cfgPath)
    byRepo := repoIndex(all)
    seen := make(map[string]bool)
    var result []repo.Repo
    for _, name := range names {
        if r, ok := byRepo[name]; ok {
            if !seen[r.Name] {
                seen[r.Name] = true
                result = append(result, r)
            }
            continue
        }
        if g, ok := cfg.Groups[name]; ok {
            for _, r := range groupRepos(cfg, cfgPath, g) {
                if !seen[r.Name] {
                    seen[r.Name] = true
                    result = append(result, r)
                }
            }
            continue
        }
        return nil, fmt.Errorf("%q is not a registered repo or group", name)
    }
    return result, nil
}

// groupRepos builds the Repo slice for repos listed in a GroupConfig.
func groupRepos(cfg *config.WorkspaceConfig, cfgPath string, g config.GroupConfig) []repo.Repo {
    sub := &config.WorkspaceConfig{
        Repos:  make(map[string]config.RepoConfig, len(g.Repos)),
        Groups: make(map[string]config.GroupConfig),
    }
    for _, name := range g.Repos {
        if rc, ok := cfg.Repos[name]; ok {
            sub.Repos[name] = rc
        }
    }
    return repo.FromConfig(sub, cfgPath)
}
```

Also check `cmd/info.go` — if it currently calls `repo.FromConfig` directly (bypassing `filterRepos`), update it to use `filterRepos` with the active context when no group arg is supplied, or document why it doesn't need to.

### C2: Update `cmd/exec_test.go`

Add to `ExecSuite` (new methods, table-driven where applicable):

**`TestExec_ActiveContext_Scopes`:**
- 2 repos in workspace; set `cfg.Groups["web"] = {Repos: [name0]}`; write `.local` with `context.active = "web"`.
- Run `exec -- status` with no filter args.
- Assert `[name0]` in output, `[name1]` not in output.

**`TestExec_FilterByGroupName`:**
- 2 repos; `cfg.Groups["web"] = {Repos: [name0]}`.
- Run `exec web -- status`.
- Assert `[name0]` in output, `[name1]` not in output.

**`TestExec_MixedRepoAndGroupFilter`:**
- 3 repos; `cfg.Groups["web"] = {Repos: [name0]}`.
- Run `exec web name1 -- status`.
- Assert `[name0]` and `[name1]` in output, `[name2]` not.

**`TestExec_GroupFilter_Deduplication`:**
- 2 repos; `cfg.Groups["web"] = {Repos: [name0]}`.
- Run `exec web name0 -- status` (name0 is in group AND listed explicitly).
- Assert `[name0]` appears exactly once in output.

**`TestExec_UnknownNameNotRepoOrGroup`:**
- Supersedes or extends existing `TestExec_UnknownRepo_Error`.
- Error message should contain "not a registered repo or group".

All group-state setup should be done by writing the .gitworkspace config directly (not via `group add` subcommand) to avoid pflag state contamination between calls.

### C3: Update `cmd/git_cmds_test.go`

Add one table-driven test to `GitCmdsSuite`:

**`TestGitCmd_ActiveContext_Scopes`:**

```go
cases := []struct{name, cmd string}{
    {"fetch", "fetch"},
    {"pull",  "pull"},
    {"status","status"},
}
```

For each: create 2 repos with remotes; set `cfg.Groups["web"] = {Repos: [name0]}`; write `.local`; run the command; assert only `name0` targeted. (push is omitted since it requires upstream tracking setup that makes the test brittle.)

---

## Config: Verify `config.SaveLocal` Exists

Before starting, confirm `config.SaveLocal(cfgPath string, ctx config.ContextConfig) error` exists in `internal/config/loader.go`. This function must:
- Write only the `[context]` section to `<cfgPath>.local`
- Use the same atomic write pattern (`write to .tmp → rename`)

If it doesn't exist, implement it as part of Stream B setup before `cmd/context.go`.

---

## Checklist: Coding Standards

Apply to every file before marking done:

- [ ] No function exceeds ~20 lines — `runGroupAdd`, `resolveTargets`, etc. are already extracted
- [ ] No inline comments that restate what the code does
- [ ] `appendUnique`, `removeItems`, `groupRepos`, `sortedKeys` are private helpers, not duplicated
- [ ] Exported symbols have godoc; private helpers do not
- [ ] Every test file embeds `suite.Suite`, uses `s.Require()` / `s.Assert()`
- [ ] Every multi-case test uses `[]struct{ name, ... }` + `s.Run(tc.name, ...)`
- [ ] Each `s.Run` closure creates its own isolated workspace (does NOT rely on SetupTest)
- [ ] Group/context state for test setup is written via direct TOML, not via `execCmd("group add", ...)` (avoids pflag state leakage)

---

## Exit Criteria

Phase 4 is complete when all of the following hold:

1. `git workspace group add frontend backend -n web` → group "web" created with both repos
2. `git workspace group add frontend -n web` again → no duplicate; still just frontend
3. `git workspace group ls` → lists group names, sorted, one per line
4. `git workspace group rm web` → group deleted; error if not found
5. `git workspace group rename web frontend` → group renamed; error if old missing or new exists
6. `git workspace group rmrepo frontend -n web` → frontend removed from web; repos not in group silently skipped
7. `git workspace group ll` → each group on one line: `name: repo1, repo2`
8. `git workspace group ll web` → single group info
9. `git workspace context web` → writes web to `.gitworkspace.local`; error if group not found
10. `git workspace context` → prints active group or "(none)"
11. `git workspace context none` → clears active context in `.local`
12. `git workspace context auto` → detects nearest group by CWD; errors if no group path matches
13. `git workspace fetch` with active context "web" → only fetches repos in group web
14. `git workspace exec web -- status` → runs in web's repos (group name used as filter arg)
15. `git workspace exec web frontend -- status` → group expanded + explicit repo, deduped
16. `go test -race -count=1 ./...` passes

---

## Notes for Implementers

- **pflag state leakage**: avoid `execCmd("group add", ...)` for test setup; write TOML directly via `os.WriteFile`.
- **`sortedKeys` is generic**: requires Go 1.21+; project is on 1.26, so safe.
- **`context auto` path matching**: use `filepath.Rel(absGroupPath, cwd)` and check if result starts with `..` — if not, CWD is under the group path.
- **Deduplication in `resolveTargets`**: use a `seen map[string]bool` keyed on repo name.
- **`groupRepos` ordering**: builds a sub-config and passes to `repo.FromConfig`, which sorts alphabetically. This is consistent with the all-repos behavior.
- **`group rmrepo` is idempotent**: repos not in the group are silently skipped (no error).
- **`group add` validates repos exist**: errors if any arg is not a key in `cfg.Repos`.
