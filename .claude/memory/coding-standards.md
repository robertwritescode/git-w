# Coding Standards

Apply these proactively to every file written in every phase.

---

## 1. Extract Business Logic to Private Functions

Public functions should read as a high-level sequence of named steps.
Complex logic, multi-step operations, and repeated patterns belong in private helpers.

**Rule of thumb:** if a function body exceeds ~20 lines, look for extraction opportunities.

```go
// Bad — all logic inline in one long function
func (c *Command) Run(...) error {
    f, err := os.Open(path)
    if err != nil { ... }
    b, err := io.ReadAll(f)
    if err != nil { ... }
    var cfg Config
    if err := toml.Unmarshal(b, &cfg); err != nil { ... }
    if cfg.Workspace.Name == "" {
        cfg.Workspace.Name = "default"
    }
    // ... 20 more lines
}

// Good — intent clear at a glance; details in private helpers
func (c *Command) Run(...) error {
    cfg, err := loadConfig(path)
    if err != nil { return err }
    return applyDefaults(cfg)
}
```

---

## 2. No Unnecessary Comments

Do not add comments that restate what the code already says.

**Bad** (AI code smell — remove these):
```go
// Create the file
f, err := os.Create(path)

// Return the error
return nil, err

// Loop over repos
for _, r := range repos {
```

**Good** — only comment the *why*, never the *what*:
```go
// walk up until we reach filesystem root; stop before infinite loop
for dir != filepath.Dir(dir) {
```

Godoc comments on **exported** types and functions are appropriate.
Inline comments should be rare — if you need one, the code may need a better name instead.

---

## 3. Don't Repeat Yourself (DRY)

If two functions share more than 2–3 lines of identical logic, extract a shared helper.

Common targets: error wrapping patterns, path construction, file I/O setup,
`t.TempDir()` scaffolding in tests, repeated assertion sequences.

Prefer table-driven tests over duplicated `func TestXxx_CaseA / TestXxx_CaseB` functions.

---

## 4. Complexity Budget

- Use guard clauses (check error → return early) instead of wrapping the happy path in nested conditionals
- Flat is better than nested; early return is better than a long else block
- Aim for cyclomatic complexity ≤ 5 per function; extract when it grows beyond that

```go
// Bad — nested
func foo(x string) error {
    if x != "" {
        if isValid(x) {
            return process(x)
        } else {
            return ErrInvalid
        }
    }
    return ErrEmpty
}

// Good — guard clauses
func foo(x string) error {
    if x == "" { return ErrEmpty }
    if !isValid(x) { return ErrInvalid }
    return process(x)
}
```

---

## 5. Self-Review Checklist

Before marking any source file as done, verify:

- [ ] No function exceeds ~20 lines without a clear reason
- [ ] No inline comments that restate what the code does
- [ ] No duplicated logic that could be a shared private helper
- [ ] Exported symbols have godoc; unexported helpers do not need comments
- [ ] Test file uses `testify/suite` — not bare `func TestXxx(t *testing.T)` functions
- [ ] Every multi-case test uses table-driven `[]struct{ name, ... }` + `s.Run(tc.name, ...)`

---

## 6. Domain Package Convention

Commands live in domain packages under `pkg/` (`workspace`, `repo`, or `git`).
Each domain package has a single exported `Register` in `register.go`, which calls private `register<Name>` functions defined in each command file.

```go
// pkg/workspace/mycommand.go  (or pkg/repo/, pkg/git/)
package workspace  // or repo, or git

import (
    "github.com/spf13/cobra"
)

func registerMyCommand(root *cobra.Command) {
    root.AddCommand(&cobra.Command{
        Use:   "mycommand",
        Short: "Description of the command",
        RunE:  runMyCommand,
    })
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    // workspace package: call LoadConfig directly (same package)
    cfg, cfgPath, err := LoadConfig(cmd)
    // repo/git packages: use workspace.LoadConfig(cmd) via pkg/workspace import
    if err != nil {
        return err
    }
    // ... command logic using cfg and cfgPath
}
```

```go
// pkg/workspace/register.go
package workspace

import "github.com/spf13/cobra"

func Register(root *cobra.Command) {
    registerInit(root)
    registerContext(root)
    registerGroup(root)
    registerMyCommand(root)  // add new commands here
}
```

**Key rules:**
- Only `Register(root *cobra.Command)` is exported per domain package — no other public symbols from command files
- Config loading: `LoadConfig(cmd)` (same-package in workspace) or `workspace.LoadConfig(cmd)` (in repo/git)
- `RunE` (not `Run`) — return errors to cobra for consistent error handling
- Prefer inline `&cobra.Command{...}` over package-level `var xxxCmd` for leaf commands (avoids pflag state bleed between test runs); use named vars only when needed for flag attachment or subcommand trees
- Flag variables prefixed with command name if generic (e.g., `addRecursive`, `groupName`)
- Test files use `testutil.CmdSuite` and `s.SetRoot(<domain>.Register)` + `s.ExecuteCmd(args...)`
- **Use `output.Writef()` for terminal output** — prefer `output.Writef(cmd.OutOrStdout(), ...)` over `fmt.Fprintf(...)` for consistency. The `output` package provides best-effort terminal writes that intentionally ignore errors.

**Note on `pkg/repo` subcommand structure:**
`repo.Register` creates a `repo` parent command (alias `r`) and adds most lifecycle commands under it. `restore` is added directly to root. When testing repo commands, `s.SetRoot(repo.Register)` registers everything including the `repo` subcommand — use `s.ExecuteCmd("repo", "add", ...)` accordingly.
