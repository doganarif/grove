```
                           
  ┌──────────────────────┐
  │  grove               │
  │  git worktree manager│
  └──────────────────────┘

```

# grove

A keyboard-driven TUI for managing git worktrees. See all your worktrees at a glance — status, branches, notes, and more. Create, switch, and clean up without memorizing `git worktree` commands.

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

<!-- ![grove demo](assets/demo.gif) -->

## Features

- **Dashboard view** — All worktrees with dirty state, remote sync, CI status, and active agents at a glance
- **Detail panel** — Changed files, last commit, CI jobs, agent status, notes
- **Fuzzy branch autocomplete** — Create worktrees fast with branch search
- **Smart cleanup** — Detect stale (remote deleted) and merged worktrees, prune in bulk
- **CI/CD status** — GitHub Actions and GitLab CI per branch, auto-detected from remote
- **AI agent detection** — See which worktrees have Claude Code, Codex, Cursor, or Aider active
- **tmux / zellij integration** — Open worktrees in new windows or splits directly from grove
- **Notes** — Attach context to any worktree ("fixing prod incident", "PR #42")
- **Colors & icons** — Tag worktrees visually for quick identification
- **Config file** — `.grove.toml` for path patterns, hooks, CI provider, tmux behavior
- **Lifecycle hooks** — Run commands on worktree create/delete (e.g. `npm install`, `cp .env`)
- **Bare repo first-class** — Works in bare repo + worktree layouts out of the box
- **Shell integration** — Select a worktree and `cd` into it

## Install

### From source

```sh
go install github.com/doganarif/grove@latest
```

### Homebrew

```sh
brew install doganarif/tap/grove
```

## Quick Start

```sh
# Run inside any git repo
grove

# Or from a bare repo root
cd ~/projects/myrepo.git
grove
```

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move cursor up/down |
| `g` / `G` | Jump to top/bottom |
| `l` / `Tab` | Toggle detail panel |
| `h` | Close detail panel |
| `/` | Filter worktrees |
| `s` | Cycle sort column |

### Actions

| Key | Action |
|-----|--------|
| `a` | Add worktree (with branch autocomplete) |
| `d` | Delete worktree |
| `D` | Delete worktree + branch |
| `n` | Edit note |
| `c` | Color / icon picker |
| `p` | Prune stale & merged worktrees |
| `t` | tmux / zellij menu |
| `w` | Open CI run in browser |
| `r` | Refresh |
| `Enter` | Open (cd into worktree) |
| `?` | Help |
| `q` | Quit |

## Shell Integration

`grove` prints the selected worktree path on exit. Add this to your `.zshrc` or `.bashrc` to `cd` on select:

```sh
grove() {
  local dir
  dir=$(command grove "$@")
  if [ -n "$dir" ] && [ -d "$dir" ]; then
    cd "$dir"
  fi
}
```

## Configuration

Create `.grove.toml` in your repo root (or `~/.config/grove/config.toml` for global defaults):

```toml
[core]
base_branch = "main"
path_pattern = "../{branch_slug}"    # {branch_slug}, {branch}, {name}

[tmux]
enabled = true
auto_open = "none"                   # "window", "hsplit", "vsplit", "none"
session_prefix = "grove"
shell_command = ""                   # run after opening (e.g. "nvim .")

[ci]
provider = "auto"                    # "github", "gitlab", "auto", "none"

[agent]
detect = ["claude", "codex", "cursor", "aider"]

[hooks]
post_create = [
  "cp .env.example .env",
  "npm install",
]
pre_delete = ["git stash"]
post_delete = []

[appearance]
default_color = "none"
default_icon = ""
```

All fields are optional — grove works without any config file.

## How It Works

grove reads worktree data directly from `git worktree list --porcelain` and runs `git status` in parallel across all worktrees for fast loading. CI status loads asynchronously via `gh` / `glab` CLI. Agent detection checks for known marker directories (`.claude/`, `.codex/`, etc.).

```
.grove/
├── worktrees.json    # notes, colors, icons per worktree
.grove.toml           # optional config (repo root)
```

## License

[MIT](LICENSE)
