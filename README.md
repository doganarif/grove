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

- **Dashboard view** — All worktrees with dirty state, remote sync, age at a glance
- **Detail panel** — Changed files, last commit, notes without leaving the list
- **Fuzzy branch autocomplete** — Create worktrees fast with branch search
- **Smart cleanup** — Detect stale (remote deleted) and merged worktrees, prune in bulk
- **Notes** — Attach context to any worktree ("fixing prod incident", "PR #42")
- **Colors & icons** — Tag worktrees visually for quick identification
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

## How It Works

grove reads worktree data directly from `git worktree list --porcelain` and runs `git status` in parallel across all worktrees for fast loading. Metadata (notes, colors, icons) is stored in `.grove/worktrees.json` inside your repo.

```
.grove/
└── worktrees.json    # notes, colors, icons per worktree
```

## Roadmap

- [ ] CI/CD status (GitHub Actions, GitLab CI)
- [ ] AI agent detection (Claude Code, Codex, Cursor)
- [ ] tmux / zellij integration
- [ ] `.grove.toml` config file
- [ ] Lifecycle hooks (post-create, pre-delete)

## License

[MIT](LICENSE)
