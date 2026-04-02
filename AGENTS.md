# git-w

Go CLI tool (`git-w`) that enables `git w <cmd>` via git's plugin system. Manages multiple git repos defined in a local `.gitw` TOML config.

**Language:** Go 1.26 | **Module:** `github.com/robertwritescode/git-w`

---

## Build & Test Commands

```bash
mage testfast    # Active development — uses cache, no race detector (~1s cached)
mage test        # Before marking work done — clears cache, race detector (~25s)
mage cover       # Coverage report (generates coverage.out, opens HTML)
mage lint        # golangci-lint fmt + run
mage lintfix     # golangci-lint fmt + run --fix
mage build       # Compile to bin/git-w with version ldflags
```

**Rule:** Always use `mage testfast` during iteration. Always run `mage test` before marking any task complete. Never run `go test ./...` directly.

---

## Codebase Map

Reference `.planning/codebase/` for authoritative analysis:
- `ARCHITECTURE.md` — layers, data flow, key abstractions
- `CONVENTIONS.md` — naming, imports, error handling, output patterns
- `STRUCTURE.md` — full file tree
- `TESTING.md` — framework, patterns, fixtures, per-package notes
- `STACK.md` — dependencies and toolchain
- `INTEGRATIONS.md` — CI/CD, GoReleaser, Homebrew tap
- `CONCERNS.md` — known issues and tech debt

---

## Coding Standards

Apply to every file written or touched. If you refactor or move code, evaluate it against all standards below.

### 1. Extract Business Logic

Public functions read as named steps; complex logic lives in private helpers. Extract any non-trivial logic that can stand alone as a named concept.

**Rules:**
- Keep cyclomatic complexity at or below 5 per function (gocyclo). Extract when it grows beyond that.
- If a function body exceeds ~20 lines, look for extraction opportunities.
- All non-test code that can be extracted to private functions should be.

```go
// Bad
func (c *Command) Run(...) error {
    f, err := os.Open(path)
    // ... 30 more lines
}

// Good
func (c *Command) Run(...) error {
    cfg, err := loadConfig(path)
    if err != nil { return err }
    return applyDefaults(cfg)
}
```

### 2. No Restate Comments

Never comment what the code already says. Only comment the *why*. Small, self-explanatory functions do not need comments at all.

```go
// Bad
// Create the file
f, err := os.Create(path)

// Good
// walk up until filesystem root; stop before infinite loop
for dir != filepath.Dir(dir) {
```

Exported types and functions get godoc comments. Unexported helpers do not need comments.

**Never use an em dash (--) in comments.** It is a marker of AI-generated text and must not appear in source code. All comments must be clear, concise, and as short as possible.

### 3. DRY

If two functions share more than 2-3 lines of identical logic, extract a shared helper. Check existing packages before writing new logic — it may already exist. Prefer table-driven tests over duplicated `TestXxx_CaseA / TestXxx_CaseB` functions.

### 4. Guard Clauses

Early return over nested conditionals.

```go
// Bad
func foo(x string) error {
    if x != "" {
        if isValid(x) { return process(x) }
        return ErrInvalid
    }
    return ErrEmpty
}

// Good
func foo(x string) error {
    if x == "" { return ErrEmpty }
    if !isValid(x) { return ErrInvalid }
    return process(x)
}
```

### 5. Whitespace

Use blank lines to separate logical blocks within a function. Code that belongs together reads together; code that is conceptually distinct gets a blank line between it.

```go
// Bad: wall of code
cfg, err := config.Load(cmd)
if err != nil { return err }
repos, err := repo.Filter(cfg, cfgPath, args)
if err != nil { return err }
results := RunParallel(repos, gitArgs, opts)
WriteResults(cmd.OutOrStdout(), results)
return ExecErrors(results)

// Good: setup / execution / output are distinct phases
cfg, err := config.Load(cmd)
if err != nil { return err }

repos, err := repo.Filter(cfg, cfgPath, args)
if err != nil { return err }

results := RunParallel(repos, gitArgs, opts)
WriteResults(cmd.OutOrStdout(), results)
return ExecErrors(results)
```

### 6. No Stuttering Names

Names must not repeat their package context.

```go
// Bad
config.LoadConfig()   // "config" repeated
hook.InstallHook()    // "hook" repeated
repo.RepoFilter()     // "repo" repeated

// Good
config.Load()
hook.Install()
repo.Filter()
```

Type names that share the package name are acceptable: `config.Config` is fine as a type.

### 7. Go Idioms

- Early returns over nested conditionals (see Guard Clauses)
- Named return values only when they genuinely aid clarity (e.g. deferred error capture); avoid as a shorthand for documentation
- Prefer composition over inheritance
- Errors as values: return `error`; never panic for expected failure paths
- `context.Context` as the first parameter on any function doing I/O or subprocess calls

### 8. Domain Package Convention

Each domain package (`pkg/workspace`, `pkg/repo`, `pkg/git`, `pkg/branch`, `pkg/worktree`, `pkg/workgroup`) exports only `Register(*cobra.Command)` in `register.go`.

```go
// pkg/<domain>/<name>.go
func register<Name>(root *cobra.Command) {
    root.AddCommand(&cobra.Command{
        Use:   "name",
        Short: "Description",
        RunE:  run<Name>,   // RunE, not Run
    })
}

func run<Name>(cmd *cobra.Command, args []string) error {
    cfg, cfgPath, err := config.Load(cmd)
    if err != nil { return err }
    // ... logic
}
```

**Adding a new command:** add `pkg/<domain>/<name>.go`, define `register<Name>` and `run<Name>`, wire `register<Name>(root)` into `pkg/<domain>/register.go`.

### 9. Output

Use `output.Writef(cmd.OutOrStdout(), ...)` for success/progress. Use `output.Writef(cmd.ErrOrStderr(), ...)` for warnings. Never `fmt.Fprintf` directly to stdout/stderr.

### 10. User Preferences

- No TUI framework (no bubbletea/lipgloss)
- Minimal dependencies — justify every new import
- Plain formatted output for info commands (`text/tabwriter`)
- Single compiled binary, no runtime deps

---

## Self-Review Checklist

Before marking any source file complete:

- [ ] No function exceeds ~20 lines without a clear reason
- [ ] No function has cyclomatic complexity above 5 (gocyclo); extract when exceeded
- [ ] No inline comments that restate what the code does
- [ ] No em dash (--) anywhere in comments
- [ ] No duplicated logic that could be a shared private helper
- [ ] No stuttering names (names do not repeat their package context)
- [ ] Exported symbols have godoc; unexported helpers do not need comments
- [ ] Logical blocks within functions are separated by blank lines
- [ ] Tests with shared setup/teardown use a testify suite; simple tests without shared lifecycle use plain `func TestXxx(t *testing.T)`
- [ ] Multi-case tests use table-driven `[]struct{ name, ... }` + `s.Run(tc.name, ...)` unless cases require excessive props or per-case functions
- [ ] `mage test` passes (race detector, no cache)

---

## Testing Standards

**Suites vs. plain tests:**
- Simple tests with no shared setup or teardown do not require a testify suite. A plain `func TestFoo(t *testing.T)` is fine.
- Tests that share setup or teardown logic must use a testify suite — embed `testutil.CmdSuite` (which embeds `suite.Suite`) and use `SetupTest` / `TeardownTest` for per-test lifecycle.
- Use `s.Require()` for setup/preconditions (fatal); `s.Assert()` for value checks (non-fatal).

**Table-driven tests:**
- Use table-driven tests to eliminate repetitive `TestFoo_CaseA`, `TestFoo_CaseB` variants.
- Do not use a table-driven test if it requires passing excessive props through each case struct, or passing functions into cases to customize the run. That complexity signals the cases are too different — write separate focused tests instead.

**Other patterns:**
- `s.T().TempDir()` for filesystem isolation
- `s.Run(tc.name, func() { ... })` to name sub-tests within table-driven loops

**Command integration tests:** `s.SetRoot(<domain>.Register)` + `s.ExecuteCmd(args...)`.
`repo` commands need `"repo"` prefix: `s.ExecuteCmd("repo", "add", ...)`. Exception: `restore` is on root.

### Known Pitfalls

**SetupTest does NOT run per `s.Run` sub-test.** Table-driven sub-tests that need isolated state must set up that state themselves inside the closure.

**pflag state persists between `ExecuteCmd` calls.** Never use `execCmd` to set up group/context state for downstream tests. Write `.gitw` and `.gitw.local` directly via helpers.

**Disable colors in display tests.** Set `color.NoColor = true` in `SetupTest` for reliable string assertions.

**Inline `&cobra.Command{...}` over package-level `var xxxCmd`.** Avoids pflag state bleed between test runs.

**`MarkFlagRequired` + pflag state.** Don't test "missing required flag" via `ExecuteCmd` — unreliable after any call that sets the flag.

**Multi-line output deduplication.** Don't assert `strings.Count(out, "[repo]") == 1` — use a baseline comparison instead.

---

## Release Workflow

- **Commits:** conventional commit format (`feat:`, `fix:`, `docs:` etc.)
- **Breaking Commits:** breaking commits (e.g `feat!:`) are only used for major version bumps, which are planned and implemented by the user directly; do not assume and make a breaking commit message.
- **CI:** `ci.yml` runs lint + test + build on every push
- **Release:** merge to `main` → Release Please opens release PR → merging PR triggers GoReleaser → publishes to GitHub Releases + Homebrew tap (`robertwritescode/homebrew-tap`)
- **Version:** injected via ldflags at build time; `git w --version` shows it
