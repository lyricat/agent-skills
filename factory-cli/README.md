# factory-cli

Read-only Go CLI and interactive TUI for inspecting a repository's `factory_board` branch without touching or switching the active git working tree.

## Features

1. **Automatic factory branch discovery**
   - Scans remote and local refs for `.factory/factory.toml` via `git for-each-ref` and `git cat-file --batch-check`.
   - Prioritizes `<remote>/factory_board` (default `origin/factory_board`), falling back to local `factory_board` or any branch containing `.factory/factory.toml`.
   - Reads board config, item Markdown files, and move history directly from git objects (`git ls-tree`, `git cat-file --batch`, `git log`).
2. **Task status & full detail inspection**
   - Aligns parsing, numbering, turn detection (`you` / `agent`), waiting age, stage trail, average stay, and terminal control-character sanitization with `../factory/scripts/board.py`.
   - Detects active `build` and `review` locks under `<ROOT>/<OWNER>/<REPO>/` (within the 10-minute lease) and displays per-item running indicators (`[● building]` / `[● reviewing]`) in list, detail, and CLI outputs.
   - Interactive TUI with stage tabs, gate highlights (`[!]`), `mine` filter, `stop/` toggle, live search (`/`), and full thread viewer with review verdict / `[P0]`–`[P3]` highlighting.
   - Non-interactive `status`, `show <ID>`, and `branches` commands (with `--json`).
3. **Automatic branch refresh & safe local fast-forward**
   - Background `git fetch <remote> <branch>` on a configurable interval (default `30s`) with commit-SHA change detection, cursor preservation across folder moves, and offline fallback warnings.
   - When a matching local factory branch exists and is strictly behind remote:
     - Not checked out in any worktree: atomically fast-forwards `refs/heads/<branch>` via `git update-ref`.
     - Checked out in a clean worktree: runs `git merge --ff-only` in that worktree.
     - Dirty, busy, ahead, or diverged local branches are never modified and show a status notice (pass `--no-pull` to disable auto fast-forward).

## Build & Install

```bash
cd factory-cli
go install ./cmd/factory-cli
```

Or build a standalone binary:

```bash
cd factory-cli
go build -o factory-cli ./cmd/factory-cli
```

## Usage

```bash
# Launch interactive TUI (auto-discovers factory branch and refreshes every 30s)
factory-cli

# Track a specific repo, branch, or custom refresh interval
factory-cli -C /path/to/repo -i 15s
factory-cli -b custom_board --no-fetch

# Non-interactive status and item detail
factory-cli status
factory-cli status --mine --stage review
factory-cli status --json
factory-cli show FT16
factory-cli branches
```

## TUI Keybindings

| Key | Action |
| --- | --- |
| `j` / `k` / `↑` / `↓` | Move cursor or scroll detail |
| `h` / `l` / `Tab` / `1`-`6` | Switch stage tab |
| `Enter` | Open selected item detail |
| `n` / `p` | Jump to next / previous section in detail |
| `/` | Search by ID, title, slug, or branch |
| `m` | Toggle filter for items waiting on you |
| `s` | Toggle `stop/` stage visibility |
| `b` | Cycle discovered factory branch candidates |
| `r` | Fetch and refresh immediately |
| `?` | Toggle help |
| `Esc` / `Backspace` / `q` | Back / quit |
