package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"factory-cli/internal/cli"
	"factory-cli/internal/gitboard"
	"factory-cli/internal/model"
	"factory-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.1.0"

type globalOpts struct {
	repoDir  string
	branch   string
	remote   string
	interval time.Duration
	noFetch  bool
	noPull   bool
	rootDir  string
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "factory-cli: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	opts := globalOpts{
		repoDir:  ".",
		remote:   "origin",
		interval: 30 * time.Second,
	}

	subcmd, rest, err := splitSubcommandAndGlobalFlags(args, &opts)
	if err != nil {
		return err
	}

	switch subcmd {
	case "", "tui", "watch":
		return runTUI(opts, rest)
	case "status", "ls":
		return runStatus(opts, rest, stdout)
	case "show":
		return runShow(opts, rest, stdout)
	case "branches":
		return runBranches(opts, rest, stdout)
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "factory-cli %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q (run 'factory-cli help' for usage)", subcmd)
	}
}

// splitSubcommandAndGlobalFlags lets global flags appear before or after the subcommand.
func splitSubcommandAndGlobalFlags(args []string, opts *globalOpts) (string, []string, error) {
	var subcmd string
	var rest []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if subcmd == "" && !strings.HasPrefix(a, "-") {
			subcmd = a
			continue
		}

		// Handle global flags regardless of whether they appear before or after subcmd.
		if name, val, hasEq := strings.Cut(a, "="); strings.HasPrefix(name, "-") {
			switch name {
			case "-C", "--repo":
				v, next, err := takeFlagVal(args, i, val, hasEq, name)
				if err != nil {
					return "", nil, err
				}
				opts.repoDir = v
				i = next
				continue
			case "-b", "--branch":
				v, next, err := takeFlagVal(args, i, val, hasEq, name)
				if err != nil {
					return "", nil, err
				}
				opts.branch = v
				i = next
				continue
			case "-r", "--remote":
				v, next, err := takeFlagVal(args, i, val, hasEq, name)
				if err != nil {
					return "", nil, err
				}
				opts.remote = v
				i = next
				continue
			case "-i", "--interval":
				v, next, err := takeFlagVal(args, i, val, hasEq, name)
				if err != nil {
					return "", nil, err
				}
				d, err := time.ParseDuration(v)
				if err != nil {
					return "", nil, fmt.Errorf("invalid %s value %q: %w", name, v, err)
				}
				opts.interval = d
				i = next
				continue
			case "--no-fetch":
				opts.noFetch = true
				continue
			case "--no-pull":
				opts.noPull = true
				continue
			case "--root":
				v, next, err := takeFlagVal(args, i, val, hasEq, name)
				if err != nil {
					return "", nil, err
				}
				opts.rootDir = v
				i = next
				continue
			}
		}
		rest = append(rest, a)
	}
	return subcmd, rest, nil
}

func takeFlagVal(args []string, idx int, val string, hasEq bool, name string) (string, int, error) {
	if hasEq {
		return val, idx, nil
	}
	if idx+1 >= len(args) {
		return "", idx, fmt.Errorf("flag %s requires a value", name)
	}
	return args[idx+1], idx + 1, nil
}

func resolveRepoAndCandidates(ctx context.Context, opts globalOpts) (string, []gitboard.BranchCandidate, error) {
	repoRoot, err := gitboard.FindRepoRoot(ctx, opts.repoDir)
	if err != nil {
		return "", nil, err
	}
	candidates, err := gitboard.DiscoverCandidates(ctx, repoRoot, opts.remote, opts.branch)
	if err != nil {
		return "", nil, err
	}
	return repoRoot, candidates, nil
}

func runTUI(opts globalOpts, rest []string) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(rest); err != nil {
		return err
	}

	ctx := context.Background()
	repoRoot, candidates, err := resolveRepoAndCandidates(ctx, opts)
	if err != nil {
		return err
	}
	now := time.Now()
	initialBoard, err := gitboard.LoadSnapshot(ctx, repoRoot, candidates[0], now)
	if err != nil {
		return err
	}
	initialRuns := gitboard.DetectActiveRuns(initialBoard.GitHubSlug, opts.rootDir, now)

	tuiOpts := tui.Options{
		RepoDir:         repoRoot,
		PreferredRemote: opts.remote,
		PreferredBranch: opts.branch,
		RootDir:         opts.rootDir,
		Interval:        opts.interval,
		NoFetch:         opts.noFetch,
		NoPull:          opts.noPull,
	}
	m := tui.NewAppModel(tuiOpts, candidates, initialBoard, initialRuns, nil)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func runStatus(opts globalOpts, rest []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stage := fs.String("stage", "", "filter by stage folder (backlog, new, review, ready, done, stop)")
	mine := fs.Bool("mine", false, "show only items waiting on you")
	stop := fs.Bool("stop", false, "include stop/ folder items")
	asJSON := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *stage != "" && !isValidFolder(*stage) {
		return fmt.Errorf("invalid stage %q (expected one of: %s)", *stage, strings.Join(model.AllFolders, ", "))
	}

	ctx := context.Background()
	repoRoot, candidates, err := resolveRepoAndCandidates(ctx, opts)
	if err != nil {
		return err
	}
	_ = gitboard.SyncBranch(ctx, repoRoot, candidates[0], opts.remote, opts.noFetch, opts.noPull)
	now := time.Now()
	board, err := gitboard.LoadSnapshot(ctx, repoRoot, candidates[0], now)
	if err != nil {
		return err
	}
	runs := gitboard.DetectActiveRuns(board.GitHubSlug, opts.rootDir, now)
	board = gitboard.ApplyActiveRuns(board, runs)

	return cli.RenderStatus(stdout, board, cli.StatusOptions{
		Stage:       *stage,
		MineOnly:    *mine,
		IncludeStop: *stop,
		JSON:        *asJSON,
		ActiveRuns:  runs,
	})
}

func runShow(opts globalOpts, rest []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	asJSON := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: factory-cli show <ID> [--json]")
	}
	query := fs.Arg(0)

	ctx := context.Background()
	repoRoot, candidates, err := resolveRepoAndCandidates(ctx, opts)
	if err != nil {
		return err
	}
	_ = gitboard.SyncBranch(ctx, repoRoot, candidates[0], opts.remote, opts.noFetch, opts.noPull)
	now := time.Now()
	board, err := gitboard.LoadSnapshot(ctx, repoRoot, candidates[0], now)
	if err != nil {
		return err
	}
	runs := gitboard.DetectActiveRuns(board.GitHubSlug, opts.rootDir, now)
	board = gitboard.ApplyActiveRuns(board, runs)
	item, err := model.FindItem(board, query)
	if err != nil {
		return err
	}
	return cli.RenderItemDetail(stdout, board, item, *asJSON)
}

func runBranches(opts globalOpts, rest []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("branches", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	asJSON := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(rest); err != nil {
		return err
	}

	ctx := context.Background()
	_, candidates, err := resolveRepoAndCandidates(ctx, opts)
	if err != nil {
		return err
	}
	return cli.RenderBranches(stdout, candidates, *asJSON)
}

func isValidFolder(folder string) bool {
	for _, f := range model.AllFolders {
		if f == folder {
			return true
		}
	}
	return false
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `factory-cli - Read-only CLI & TUI for the factory skill board

Usage:
  factory-cli [flags] [tui|watch]         Launch interactive board TUI (default)
  factory-cli [flags] status [options]    Print stage summary and items (alias: ls)
  factory-cli [flags] show <ID> [--json]  Print item metadata, trail, and thread
  factory-cli [flags] branches [--json]   List discovered factory branches
  factory-cli version                     Print version

Global Flags:
  -C, --repo <path>        Repository path (default: .)
  -b, --branch <name>      Explicit factory branch ref/name to track
  -r, --remote <name>      Preferred remote name (default: origin)
  -i, --interval <dur>     Auto-refresh interval for TUI (default: 30s, 0 disables)
      --no-fetch           Read local git refs only without running git fetch
      --no-pull            Fetch remote without auto fast-forwarding local branch
      --root <path>        Override factory runtime root for active lock detection

Status Options:
      --stage <folder>     Filter by folder (backlog, new, review, ready, done, stop)
      --mine               Show only items waiting on you
      --stop               Include parked items in stop/
      --json               Output JSON
`)
}
