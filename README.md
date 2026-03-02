[![Go Version](https://img.shields.io/github/go-mod/go-version/robertwritescode/git-w)](go.mod)
[![Release](https://img.shields.io/github/v/release/robertwritescode/git-w)](https://github.com/robertwritescode/git-w/releases)
[![CI](https://github.com/robertwritescode/git-w/actions/workflows/ci.yml/badge.svg)](https://github.com/robertwritescode/git-w/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/robertwritescode/git-w)](https://goreportcard.com/report/github.com/robertwritescode/git-w)
[![License](https://img.shields.io/github/license/robertwritescode/git-w)](LICENSE)

```
 ________  ___  _________            ___       __      
|\   ____\|\  \|\___   ___\         |\  \     |\  \    
\ \  \___|\ \  \|___ \  \_|_________\ \  \    \ \  \   
 \ \  \  __\ \  \   \ \  \|\_________\ \  \  __\ \  \  
  \ \  \|\  \ \  \   \ \  \|_________|\ \  \|\__\_\  \ 
   \ \_______\ \__\   \ \__\           \ \____________\
    \|_______|\|__|    \|__|            \|____________|
                                                          
```

# git-w: A Git plugin for managing meta-repo workspaces

A meta-repo is a working folder containing multiple Git repositories, each organized into child folders. Meta-repos are a common pattern for working with microservices locally and managing development that spans across backend, frontend, and packages at the same time.

`git-w` makes it easy to set up, share, manage, and run operations across a meta-repo.

Invoke it as `git w <cmd>` via Git's plugin system. As long as `git-w` is in your `$PATH`, Git will find it automatically.

It uses a config file that can be committed to version control to share your meta-repo configurations between teams.

## Features

- Declare and track multiple repos in a single `.gitw` config file (TOML)
- Clone missing repos and pull existing ones with one `restore` command
- Run `fetch`, `pull`, `push`, and `status` across all repos (or a filtered subset) in parallel
- Execute any arbitrary git command across repos with `exec`
- Organize repos into named groups
- Set an active context to scope all commands to a group without specifying it each time
- Local overrides (active context) stored in `.gitw.local`, which is kept out of version control automatically

## Installation

### From source

Requires Go 1.26+.

```sh
go install github.com/robertwritescode/git-w@latest
```

Or clone, build, and install to `$GOPATH/bin` with [Mage](https://magefile.org):

```sh
git clone https://github.com/robertwritescode/git-w.git
cd git-w
mage install
```

Make sure `$GOPATH/bin` (or `$GOBIN`) is in your `$PATH` so that Git can find the plugin.

## Quick start

```sh
# Create a workspace config in the current directory
git w init my-workspace

# Clone a repo and register it
git w clone https://github.com/org/repo-a

# Register an existing local repo
git w add ../repo-b

# Recursively register all git repos under a directory
git w add -r ./projects/

# Fetch across all repos
git w fetch

# Show a status table for all repos
git w info
```

## Configuration

`git-w` uses two TOML files in the same directory:

| File | Purpose | Version control |
|---|---|---|
| `.gitw` | Workspace definition (repos, groups, settings) | Commit this |
| `.gitw.local` | Local state (active context) | Auto-added to `.gitignore` |

Example `.gitw`:

```toml
[workspace]
name = "my-workspace"
# auto_gitignore = true  # default: automatically adds cloned repo paths to .gitignore

[repos.repo-a]
path = "repos/repo-a"
url  = "https://github.com/org/repo-a"

[repos.repo-b]
path = "repos/repo-b"
url  = "https://github.com/org/repo-b"

[repos.repo-c]
path = "repos/repo-c"

[groups.backend]
repos = ["repo-a", "repo-b"]

[groups.frontend]
repos = ["repo-c"]
path  = "/absolute/path/used/for/auto-context"
```

## Commands

### Workspace setup

| Command | Description |
|---|---|
| `git w init [name]` | Create a `.gitw` in the current directory. Defaults to the directory name. |
| `git w restore` | Materialize all repos: clone missing ones, pull existing ones (runs in parallel). |

### Managing repos

| Command | Description |
|---|---|
| `git w add [<path>]` | Register an existing local git repo. |
| `git w add -r [<dir>]` | Recursively find and register all git repos under a directory. |
| `git w clone <url> [<path>]` | Clone a remote repo and register it in the workspace. |
| `git w remove <name>...` | Unregister one or more repos (also removes them from all groups). Alias: `rm` |
| `git w rename <old> <new>` | Rename a repo in the config. |
| `git w list [name]` | List all registered repo names. With a name, prints the absolute path to that repo. Alias: `ls` |

Both `add` and `clone` accept `-g <group>` / `--group <group>` to also add the repo to a group.

### Running git commands

All git commands accept an optional list of repo names to filter targets. With no filter, the command runs against all repos (or the active context group, if one is set).

| Command | Description |
|---|---|
| `git w fetch [repos...]` | Run `git fetch` in repos. Alias: `f` |
| `git w pull [repos...]` | Run `git pull` in repos. Alias: `pl` |
| `git w push [repos...]` | Run `git push` in repos. Alias: `ps` |
| `git w status [repos...]` | Run `git status -sb` in repos. Alias: `st` |
| `git w exec [repos...] -- <git-args>` | Run any git command across repos concurrently. Output is prefixed with `[repo-name]`. Aliases: `x`, `run` |

**Examples:**

```sh
# Fetch all repos
git w fetch

# Pull only two specific repos
git w pull repo-a repo-b

# Run an arbitrary git command across all repos
git w exec -- log --oneline -5

# Run a command on specific repos
git w exec repo-a repo-c -- diff HEAD~1
```

### Groups

Groups let you organize repos into named sets. Many commands accept repo names as filters; groups serve as a logical layer on top of that.

```sh
# Create a group and add repos to it
git w group add repo-a repo-b --name backend

# Add repos to an existing group
git w group add repo-c --name backend

# List all groups
git w group list

# Show repos in a group
git w group info backend

# Show repos in all groups
git w group info

# Remove repos from a group
git w group remove-repo repo-c --name backend

# Rename a group
git w group rename backend services

# Delete a group
git w group remove backend
```

### Context

The active context scopes all git commands to a specific group without needing to specify repos each time. The active context is stored in `.gitw.local` and is local to your machine.

```sh
# Show the active context
git w context

# Set the active context to a group
git w context backend

# Auto-detect the active context based on CWD (uses the group's path attribute)
git w context auto

# Clear the active context
git w context none
```

With an active context set, `fetch`, `pull`, `push`, `status`, `exec`, and `info` all operate on that group's repos by default.

### Status overview

```sh
# Show a status table for all repos (branch, remote state, last commit)
git w info

# Show status table for a specific group
git w info backend
```

### Shell completion

```sh
# Generate shell completion script (bash, zsh, fish, powershell)
git w completion bash
git w completion zsh
```

## Global flags

| Flag | Description |
|---|---|
| `--config <path>` | Path to `.gitw` config. Defaults to the nearest `.gitw` found by walking up from the current directory. |

## Development

This project uses [Mage](https://magefile.org) as its build tool.

```sh
# Build binary to bin/git-w
mage build

# Install to $GOPATH/bin
mage install

# Run tests (with race detector)
mage test

# Lint (golangci-lint)
mage lint

# Lint + test + build (default)
mage

# Generate coverage report
mage cover
```

## Release pipeline (maintainers)

Releases are automated from `.github/workflows/release.yml` using Release Please + GoReleaser on pushes to `main`.

Required repository secret:

- `TAP_GITHUB_TOKEN` — a PAT with access to `robertwritescode/homebrew-tap` (at minimum, Contents: Read and write) so GoReleaser can update the tap cask.

Required repository setting:

- In GitHub, go to **Settings → Actions → General** and enable **Allow GitHub Actions to create and approve pull requests** (required for Release Please PR automation).

Release trigger:

- To publish a release, merge the open Release Please PR (for example, `chore(main): release 1.2.3`) into `main`; this is what creates the tag and runs the GoReleaser job.
- To postpone/cancel a pending release, close that Release Please PR without merging and remove its `autorelease: pending` label; Release Please will open or update a new one after subsequent conventional commits land on `main`.

Troubleshooting:

- If the release job fails with `resource not accessible by integration`, the token used for the target repo is missing required permissions; verify `TAP_GITHUB_TOKEN` can write contents to `robertwritescode/homebrew-tap`.
- The release workflow runs `brew style --fix` on `Casks/git-w.rb` in `robertwritescode/homebrew-tap` after GoReleaser publishes the cask. This is intentional: current GoReleaser output can fail Homebrew's latest cask style cops, so CI auto-normalizes stanza ordering/format before final push.

## Similar projects & inspiration

`git-w` was directly inspired by [gita](https://github.com/nosarthur/gita), a Python tool for managing multiple git repos side-by-side. `gita` introduced the core ideas of grouping repos and running git commands across them in parallel — `git-w` extends these ideas with a portable, single-binary implementation and a richer feature set.

Other projects in this space worth knowing about:

| Project | Language | Notes |
|---|---|---|
| [gita](https://github.com/nosarthur/gita) | Python | Original inspiration. Groups, parallel commands, shell delegating. |
| [myrepos (mr)](https://myrepos.branchable.com/) | Perl | Classic multi-repo tool; highly configurable via `.mrconfig`. |
| [meta](https://github.com/mateodelnorte/meta) | Node.js | Manages repos as a monorepo alternative using a `.meta` file. |
| [mu-repo](https://fabioz.github.io/mu-repo/) | Python | Runs git commands across multiple repos; supports grouping. |
| [repo](https://gerrit.googlesource.com/git-repo) | Python | Google's tool for Android development; XML manifest-based. |
| [git-xargs](https://github.com/gruntwork-io/git-xargs) | Go | Run arbitrary commands across many GitHub repos via the API. |
| [git-workspace](https://github.com/orf/git-workspace) | Rust | Sync and fetch repos from git providers into a structured local workspace. |

## License

[MIT](LICENSE)
