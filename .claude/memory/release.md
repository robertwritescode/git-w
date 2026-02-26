# Build, Release & Distribution

## `git w` Short Alias

Git requires a `git-w` executable in `$PATH` for `git w` to work.
The Homebrew formula installs the binary and creates the symlink — no user config needed.
For non-Homebrew installs, the README documents a manual `ln -s`.

---

## Local Development (Mage)

Build targets live in `magefile.go` at repo root (excluded from normal builds via
`//go:build mage`). Run `mage -l` to list targets.

```go
//go:build mage

package main

import (
    "fmt"
    "strings"

    "github.com/magefile/mage/mg"
    "github.com/magefile/mage/sh"
)

var Default = All

func All() {
    mg.Deps(Lint, Test, Build)
}

func Build() error {
    version := gitVersion()
    ldflags := fmt.Sprintf("-X main.version=%s", version)
    return sh.RunV("go", "build", "-ldflags="+ldflags, "-o", "bin/git-w", ".")
}

func Install() error {
    return sh.RunV("go", "install", ".")
}

func Test() error {
    return sh.RunV("go", "test", "-race", "-count=1", "./...")
}

func Cover() error {
    if err := sh.RunV("go", "test", "-race", "-count=1", "-coverprofile=coverage.out", "./..."); err != nil {
        return err
    }
    return sh.RunV("go", "tool", "cover", "-html=coverage.out")
}

func Lint() error {
    if err := sh.RunV("golangci-lint", "fmt", "--diff", "./..."); err != nil {
        return err
    }
    return sh.RunV("golangci-lint", "run", "./...")
}

func LintFix() error {
    if err := sh.RunV("golangci-lint", "fmt", "./..."); err != nil {
        return err
    }
    return sh.RunV("golangci-lint", "run", "--fix", "./...")
}

func Fmt() error {
    return sh.RunV("golangci-lint", "fmt", "./...")
}

func gitVersion() string {
    out, err := sh.Output("git", "describe", "--tags", "--always", "--dirty")
    if err != nil {
        return "dev"
    }
    return strings.TrimSpace(out)
}
```

`magefile.go` is added to `go.mod` as a tool dependency:
```
require github.com/magefile/mage v1.x
```

---

## Versioning

Semver tags: `v0.1.0`, `v0.2.0`, etc. GoReleaser reads the tag; `main.version`
is injected via ldflags and exposed by `git w --version`.

`--version` is provided for free by Cobra when `rootCmd.Version` is set:
```go
var version = "dev"   // overridden at build time by ldflags

func main() {
    cmd.Execute(version)
}
```

---

## Release Tooling: GoReleaser

GoReleaser handles cross-compilation, archiving, checksums, GitHub Release creation,
and Homebrew tap updates from a single `.goreleaser.yaml`.

**`.goreleaser.yaml` key configuration:**

```yaml
project_name: git-w

builds:
  - main: .
    binary: git-w
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

release:
  # Find the draft release Release Please already created for this tag and
  # attach binaries to it rather than creating a second release.
  replace_existing_draft: true

changelog:
  # Release Please already wrote the changelog into the draft release body.
  # Skip GoReleaser's own changelog generation to avoid overwriting it.
  disable: true

brews:
  - name: git-w
    repository:
      owner: <github-user>
      name: homebrew-git-w
    homepage: https://github.com/<github-user>/git-w
    description: "Manage multiple git repos from a single workspace"
    install: |
      bin.install "git-w"
      bin.install_symlink bin/"git-w" => "git-w"
    test: |
      system "#{bin}/git-w", "--version"
```

---

## GitHub Actions Workflows

Three workflows in `.github/workflows/`:

---

**`ci.yml`** — runs on pushes, PRs, and workflow dispatch; gates merges:
```yaml
on:
  push:
  pull_request:
  workflow_dispatch:
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - uses: golangci/golangci-lint-action@v7
        with: { version: v2.10.1 }
      - run: golangci-lint fmt --diff ./...
  test-and-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - run: go test -race -count=1 ./...
      - run: go build ./...
```

---

**`release-please.yml`** — runs on every push to `main`; opens/updates a Release PR:
```yaml
on:
  push:
    branches: [main]
permissions:
  contents: write
  pull-requests: write
jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          config-file: .release-please-config.json
          manifest-file: .release-please-manifest.json
```

Release Please reads conventional commits since the last release, determines the
semver bump, updates `CHANGELOG.md`, and opens a PR titled e.g. "chore(main): release v0.2.0".
Merging that PR creates the tag and a **draft** GitHub Release with generated notes.

---

**`goreleaser.yml`** — runs when Release Please pushes a `v*` tag (renamed from `release.yml`):
```yaml
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }  # GoReleaser needs full tag history
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - name: Test
        run: go test -race -count=1 ./...
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

GoReleaser is configured with `release.replace_existing_draft: true` so it finds
the draft Release that Release Please created, attaches the compiled binaries and
archives, and publishes it — rather than creating a second release.

---

## Release Please Config Files

**`.release-please-config.json`** — committed to repo root:
```json
{
  "release-type": "go",
  "packages": {
    ".": {
      "changelog-path": "CHANGELOG.md"
    }
  },
  "bump-minor-pre-major": true,
  "bump-patch-for-minor-pre-major": true
}
```

**`.release-please-manifest.json`** — committed to repo root; tracks current version:
```json
{
  ".": "0.0.0"
}
```
Release Please updates this file automatically on each release.

---

## Developer Release Workflow

```
1. Work on branch with conventional commits:
     feat: add recursive add support
     fix: handle missing .git in status check
     feat!: rename `rm` to `unlink`  ← triggers major bump

2. Merge PR to main
   → ci.yml runs (lint + test + build)
   → release-please.yml runs; opens/updates Release PR

3. When ready to ship: merge the Release Please PR
   → Release Please pushes tag (e.g. v0.3.0)
   → Release Please creates draft GitHub Release with CHANGELOG entries

4. goreleaser.yml fires on the tag:
   → go test ./...  (if fails: release aborted)
   → GoReleaser: builds darwin/linux × amd64/arm64, creates archives + checksums,
     attaches to the draft release, publishes it, updates Homebrew tap formula
```

**Conventional commit → semver mapping:**
| Commit prefix | Bump |
|---|---|
| `feat:` | minor (0.x.0) |
| `fix:`, `perf:`, `refactor:` | patch (0.0.x) |
| `feat!:` or `BREAKING CHANGE:` footer | major (x.0.0) |
| `docs:`, `chore:`, `ci:` | no bump (excluded from changelog) |

---

## Homebrew Tap Repo

Separate repo: `github.com/<user>/homebrew-git-w`

```
homebrew-git-w/
└── Formula/
    └── git-w.rb    # auto-updated by GoReleaser on each release
```

Install:
```sh
brew tap <user>/git-w
brew install git-w
# installs git-w binary
# → both `git w <cmd>` and `git-w <cmd>` work
```
