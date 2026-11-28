package gitboard

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"factory-cli/internal/model"
)

func runCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"LC_ALL=C",
		"GIT_AUTHOR_NAME=Tester",
		"GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester",
		"GIT_COMMITTER_EMAIL=tester@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s failed: %v\n%s", strings.Join(args, " "), dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverLoadAndFetchWithoutTouchingWorktree(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	remoteDir := filepath.Join(tmp, "remote.git")
	seedDir := filepath.Join(tmp, "seed")
	workDir := filepath.Join(tmp, "work")

	runCmd(t, tmp, "init", "--bare", remoteDir)
	runCmd(t, tmp, "clone", remoteDir, seedDir)
	runCmd(t, seedDir, "checkout", "-b", "master")
	writeTestFile(t, filepath.Join(seedDir, "README.md"), "hello\n")
	runCmd(t, seedDir, "add", "README.md")
	runCmd(t, seedDir, "commit", "-m", "Initial master")
	runCmd(t, seedDir, "push", "-u", "origin", "master")

	// Create orphan factory_board branch in seedDir.
	runCmd(t, seedDir, "switch", "--orphan", "factory_board")
	writeTestFile(t, filepath.Join(seedDir, ".factory", "factory.toml"), `
base = "master"
owners = ["alex"]
prefix = "FT"
timezone = "UTC"
`)
	writeTestFile(t, filepath.Join(seedDir, ".factory", "new", "search-box.md"), `# Search box

## alex · 2026-11-04 09:00
Add toolbar search.
`)
	runCmd(t, seedDir, "add", ".factory")
	runCmd(t, seedDir, "commit", "-m", "Start board and add unnumbered item")

	// Number and move item across folders to verify history reconstruction.
	runCmd(t, seedDir, "mv", ".factory/new/search-box.md", ".factory/new/FT1-search-box.md")
	runCmd(t, seedDir, "commit", "-m", "factory: number FT1 search-box")

	if err := os.MkdirAll(filepath.Join(seedDir, ".factory", "review"), 0o755); err != nil {
		t.Fatal(err)
	}
	runCmd(t, seedDir, "mv", ".factory/new/FT1-search-box.md", ".factory/review/FT1-search-box.md")
	writeTestFile(t, filepath.Join(seedDir, ".factory", "review", "FT1-search-box.md"), `---
branch: factory/FT1-search-box
round: 1
---
# Search box

## alex · 2026-11-04 09:00
Add toolbar search.

## factory · review · 2026-11-05 10:00
head: abc1234
verdict: fix
fix:
- [P1] search.go:10: handle escape key
`)
	runCmd(t, seedDir, "add", ".factory")
	runCmd(t, seedDir, "commit", "-m", "factory: FT1 new -> review")

	writeTestFile(t, filepath.Join(seedDir, ".factory", "backlog", "FT2-dark-mode.md"), `# Dark mode

## alex · 2026-11-05 11:00
Add theme toggle.
`)
	runCmd(t, seedDir, "add", ".factory")
	runCmd(t, seedDir, "commit", "-m", "Add FT2")
	runCmd(t, seedDir, "push", "-u", "origin", "factory_board")

	// Clone working repo checked out on master with dirty uncommitted changes.
	runCmd(t, tmp, "clone", "-b", "master", remoteDir, workDir)
	writeTestFile(t, filepath.Join(workDir, "dirty.txt"), "do not touch me\n")
	writeTestFile(t, filepath.Join(workDir, "README.md"), "modified locally\n")

	root, err := FindRepoRoot(ctx, workDir)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}
	if root != workDir {
		// On macOS /var is a symlink to /private/var; resolve both before comparing.
		evalRoot, _ := filepath.EvalSymlinks(root)
		evalWork, _ := filepath.EvalSymlinks(workDir)
		if evalRoot != evalWork {
			t.Fatalf("FindRepoRoot=%q, want %q", root, workDir)
		}
	}

	candidates, err := DiscoverCandidates(ctx, workDir, "origin", "")
	if err != nil {
		t.Fatalf("DiscoverCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].ShortName != "origin/factory_board" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}

	board, err := LoadSnapshot(ctx, workDir, candidates[0], time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if len(board.Items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(board.Items))
	}
	ft1, err := model.FindItem(board, "FT1")
	if err != nil {
		t.Fatalf("FindItem(FT1): %v", err)
	}
	if ft1.Folder != model.FolderReview || ft1.Turn != model.TurnYou {
		t.Fatalf("FT1 folder=%q turn=%q, want review/you", ft1.Folder, ft1.Turn)
	}
	if len(ft1.Steps) != 2 || ft1.Steps[0].Folder != model.FolderNew || ft1.Steps[1].Folder != model.FolderReview {
		t.Fatalf("FT1 steps=%+v, want new->review across numbering rename", ft1.Steps)
	}

	// Verify working tree stayed on master and dirty files remained intact.
	if head := runCmd(t, workDir, "rev-parse", "--abbrev-ref", "HEAD"); head != "master" {
		t.Fatalf("HEAD changed to %q", head)
	}
	if data, err := os.ReadFile(filepath.Join(workDir, "dirty.txt")); err != nil || string(data) != "do not touch me\n" {
		t.Fatalf("dirty file altered: %v %q", err, string(data))
	}

	// Push an update from seedDir and verify FetchBranch + LoadSnapshot picks it up.
	if err := os.MkdirAll(filepath.Join(seedDir, ".factory", "ready"), 0o755); err != nil {
		t.Fatal(err)
	}
	runCmd(t, seedDir, "mv", ".factory/review/FT1-search-box.md", ".factory/ready/FT1-search-box.md")
	runCmd(t, seedDir, "commit", "-m", "factory: FT1 review -> ready")
	runCmd(t, seedDir, "push", "origin", "factory_board")

	oldSHA := board.CommitSHA
	if err := FetchBranch(ctx, workDir, candidates[0].Remote, candidates[0].Branch); err != nil {
		t.Fatalf("FetchBranch: %v", err)
	}
	newSHA, err := ResolveSHA(ctx, workDir, candidates[0].Ref)
	if err != nil {
		t.Fatalf("ResolveSHA: %v", err)
	}
	if newSHA == oldSHA {
		t.Fatalf("expected SHA to advance after FetchBranch, still %s", newSHA)
	}

	board2, err := LoadSnapshot(ctx, workDir, candidates[0], time.Date(2026, 11, 6, 12, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("LoadSnapshot after fetch: %v", err)
	}
	diffs := model.DiffBoards(board, board2)
	if len(diffs) != 1 || diffs[0] != "FT1 review → ready" {
		t.Fatalf("diffs=%v, want [FT1 review → ready]", diffs)
	}
}

func TestDetectActiveRuns(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	lockDir := filepath.Join(root, "acme", "demo", "build", "lock")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpTaskDir := filepath.Join(root, "acme", "demo", "build", "tmp", "FT16-search-box", "factory")
	if err := os.MkdirAll(tmpTaskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runs := DetectActiveRuns("acme/demo", root, now)
	if len(runs) != 1 || runs[0].Kind != "build" {
		t.Fatalf("DetectActiveRuns=%+v, want [build]", runs)
	}
	if len(runs[0].Items) != 1 || runs[0].Items[0] != "FT16-search-box" {
		t.Fatalf("runs[0].Items=%v, want [FT16-search-box]", runs[0].Items)
	}
	if runs[0].Summary() != "build (FT16)" {
		t.Fatalf("Summary=%q, want build (FT16)", runs[0].Summary())
	}

	// Stale lock (> 10 minutes) should be ignored.
	runsStale := DetectActiveRuns("acme/demo", root, now.Add(15*time.Minute))
	if len(runsStale) != 0 {
		t.Fatalf("expected stale lock to be ignored, got %+v", runsStale)
	}
}

func TestDetectActiveRunsPerItemLockAndApply(t *testing.T) {
	root := t.TempDir()
	now := time.Now()

	// Build lock active with one recent worktree (FT16) and one stale old worktree (FT10).
	buildDir := filepath.Join(root, "acme", "demo", "build")
	if err := os.MkdirAll(filepath.Join(buildDir, "lock"), 0o755); err != nil {
		t.Fatal(err)
	}
	recentWT := filepath.Join(buildDir, "FT16-search-box")
	if err := os.MkdirAll(recentWT, 0o755); err != nil {
		t.Fatal(err)
	}
	oldWT := filepath.Join(buildDir, "FT10-old-item")
	if err := os.MkdirAll(oldWT, 0o755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(oldWT, "README.md")
	writeTestFile(t, oldFile, "old\n")
	staleTime := now.Add(-30 * time.Minute)
	if err := os.Chtimes(oldFile, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldWT, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	// Review lock active with legacy 4-digit review-0012 directory.
	reviewDir := filepath.Join(root, "acme", "demo", "review")
	if err := os.MkdirAll(filepath.Join(reviewDir, "lock"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(reviewDir, "review-0012"), 0o755); err != nil {
		t.Fatal(err)
	}

	runs := DetectActiveRuns("acme/demo", root, now)
	if len(runs) != 2 {
		t.Fatalf("expected 2 active runs, got %+v", runs)
	}

	n16, n10, n12 := 16, 10, 12
	board := model.BuildBoard("demo", "", "acme/demo", "refs/remotes/origin/factory_board", "origin/factory_board", "abc1234", model.DefaultConfig(), time.UTC, []model.Item{
		{Number: &n16, Slug: "search-box", Folder: model.FolderNew, File: ".factory/new/FT16-search-box.md", Name: "FT16-search-box.md", ID: "16", Label: "FT16", Title: "Search"},
		{Number: &n10, Slug: "old-item", Folder: model.FolderNew, File: ".factory/new/FT10-old-item.md", Name: "FT10-old-item.md", ID: "10", Label: "FT10", Title: "Old"},
		{Number: &n12, Slug: "legacy", Folder: model.FolderReview, File: ".factory/review/0012-legacy.md", Name: "0012-legacy.md", ID: "12", Label: "FT12", Title: "Legacy"},
	}, now)

	applied := ApplyActiveRuns(board, runs)
	byLabel := make(map[string]model.Item, len(applied.Items))
	for _, it := range applied.Items {
		byLabel[it.Label] = it
	}
	if byLabel["FT16"].Running != "build" {
		t.Fatalf("FT16.Running=%q, want build", byLabel["FT16"].Running)
	}
	if byLabel["FT10"].Running != "" {
		t.Fatalf("FT10.Running=%q, want empty (stale worktree)", byLabel["FT10"].Running)
	}
	if byLabel["FT12"].Running != "review" {
		t.Fatalf("FT12.Running=%q, want review", byLabel["FT12"].Running)
	}

	// Clearing locks clears Running field.
	cleared := ApplyActiveRuns(applied, nil)
	for _, it := range cleared.Items {
		if it.Running != "" {
			t.Fatalf("expected cleared Running on %s, got %q", it.Label, it.Running)
		}
	}
}

func setupSyncTestEnv(t *testing.T) (remoteDir, seedDir, workDir string) {
	t.Helper()
	tmp := t.TempDir()
	remoteDir = filepath.Join(tmp, "remote.git")
	seedDir = filepath.Join(tmp, "seed")
	workDir = filepath.Join(tmp, "work")

	runCmd(t, tmp, "init", "--bare", remoteDir)
	runCmd(t, tmp, "clone", remoteDir, seedDir)
	runCmd(t, seedDir, "checkout", "-b", "master")
	writeTestFile(t, filepath.Join(seedDir, "README.md"), "base\n")
	runCmd(t, seedDir, "add", "README.md")
	runCmd(t, seedDir, "commit", "-m", "Initial master")
	runCmd(t, seedDir, "push", "-u", "origin", "master")

	runCmd(t, seedDir, "switch", "--orphan", "factory_board")
	writeTestFile(t, filepath.Join(seedDir, ".factory", "factory.toml"), "base = \"master\"\nowners = [\"alex\"]\nprefix = \"FT\"\n")
	writeTestFile(t, filepath.Join(seedDir, ".factory", "new", "FT1-alpha.md"), "# Alpha\n\n## alex · 2026-11-10 09:00\nInit\n")
	runCmd(t, seedDir, "add", ".factory")
	runCmd(t, seedDir, "commit", "-m", "Board v1")
	runCmd(t, seedDir, "push", "-u", "origin", "factory_board")

	runCmd(t, tmp, "clone", "-b", "master", remoteDir, workDir)
	return remoteDir, seedDir, workDir
}

func pushBoardCommit(t *testing.T, seedDir, fileRel, content, msg string) string {
	t.Helper()
	writeTestFile(t, filepath.Join(seedDir, fileRel), content)
	runCmd(t, seedDir, "add", ".factory")
	runCmd(t, seedDir, "commit", "-m", msg)
	runCmd(t, seedDir, "push", "origin", "factory_board")
	return runCmd(t, seedDir, "rev-parse", "HEAD")
}

func TestSyncBranchAutoFastForwardUncheckedLocal(t *testing.T) {
	ctx := context.Background()
	_, seedDir, workDir := setupSyncTestEnv(t)

	// Create local factory_board at v1 without checking it out; keep master dirty.
	runCmd(t, workDir, "branch", "factory_board", "origin/factory_board")
	writeTestFile(t, filepath.Join(workDir, "README.md"), "dirty master\n")

	wantSHA := pushBoardCommit(t, seedDir, ".factory/new/FT2-beta.md", "# Beta\n\n## alex · 2026-11-10 10:00\nSecond\n", "Board v2")

	cand := BranchCandidate{
		Ref:       "refs/remotes/origin/factory_board",
		ShortName: "origin/factory_board",
		Remote:    "origin",
		Branch:    "factory_board",
		IsRemote:  true,
	}

	// Verify --no-pull skips advancing local ref.
	resNoPull := SyncBranch(ctx, workDir, cand, "origin", false, true)
	if resNoPull.FetchErr != nil || resNoPull.LocalState != LocalSyncNone {
		t.Fatalf("unexpected no-pull result: %+v", resNoPull)
	}
	if got := runCmd(t, workDir, "rev-parse", "refs/heads/factory_board"); got == wantSHA {
		t.Fatalf("expected --no-pull to leave local ref behind, got %s", got)
	}

	// Now sync with auto-pull enabled.
	res := SyncBranch(ctx, workDir, cand, "origin", false, false)
	if res.FetchErr != nil || res.LocalState != LocalSyncFastFwd {
		t.Fatalf("SyncBranch got %+v, want LocalSyncFastFwd", res)
	}
	if got := runCmd(t, workDir, "rev-parse", "refs/heads/factory_board"); got != wantSHA {
		t.Fatalf("local factory_board SHA=%s, want %s", got, wantSHA)
	}
	if data, err := os.ReadFile(filepath.Join(workDir, "README.md")); err != nil || string(data) != "dirty master\n" {
		t.Fatalf("dirty master file changed: %v %q", err, string(data))
	}
}

func TestSyncBranchAutoPullCleanWorktree(t *testing.T) {
	ctx := context.Background()
	_, seedDir, workDir := setupSyncTestEnv(t)

	boardWT := filepath.Join(filepath.Dir(workDir), "board-wt")
	runCmd(t, workDir, "worktree", "add", "-b", "factory_board", boardWT, "origin/factory_board")

	wantSHA := pushBoardCommit(t, seedDir, ".factory/new/FT2-beta.md", "# Beta\n\n## alex · 2026-11-10 10:00\nSecond\n", "Board v2")

	cand := BranchCandidate{
		Ref:       "refs/remotes/origin/factory_board",
		ShortName: "origin/factory_board",
		Remote:    "origin",
		Branch:    "factory_board",
		IsRemote:  true,
	}
	res := SyncBranch(ctx, workDir, cand, "origin", false, false)
	if res.FetchErr != nil || res.LocalState != LocalSyncFastFwd {
		t.Fatalf("SyncBranch got %+v, want LocalSyncFastFwd", res)
	}
	if got := runCmd(t, boardWT, "rev-parse", "HEAD"); got != wantSHA {
		t.Fatalf("worktree HEAD=%s, want %s", got, wantSHA)
	}
	if _, err := os.Stat(filepath.Join(boardWT, ".factory", "new", "FT2-beta.md")); err != nil {
		t.Fatalf("expected pulled file in clean worktree: %v", err)
	}
	if status := runCmd(t, boardWT, "status", "--porcelain"); status != "" {
		t.Fatalf("expected clean worktree after ff pull, got %q", status)
	}
}

func TestSyncBranchSkipsDirtyWorktree(t *testing.T) {
	ctx := context.Background()
	_, seedDir, workDir := setupSyncTestEnv(t)

	boardWT := filepath.Join(filepath.Dir(workDir), "board-wt-dirty")
	runCmd(t, workDir, "worktree", "add", "-b", "factory_board", boardWT, "origin/factory_board")
	oldSHA := runCmd(t, boardWT, "rev-parse", "HEAD")

	// Make board worktree dirty with an untracked/modified file.
	writeTestFile(t, filepath.Join(boardWT, ".factory", "new", "draft.md"), "uncommitted\n")
	pushBoardCommit(t, seedDir, ".factory/new/FT2-beta.md", "# Beta\n\n## alex · 2026-11-10 10:00\nSecond\n", "Board v2")

	cand := BranchCandidate{
		Ref:       "refs/remotes/origin/factory_board",
		ShortName: "origin/factory_board",
		Remote:    "origin",
		Branch:    "factory_board",
		IsRemote:  true,
	}
	res := SyncBranch(ctx, workDir, cand, "origin", false, false)
	if res.FetchErr != nil || res.LocalState != LocalSyncDirty {
		t.Fatalf("SyncBranch got %+v, want LocalSyncDirty", res)
	}
	if got := runCmd(t, boardWT, "rev-parse", "HEAD"); got != oldSHA {
		t.Fatalf("dirty worktree HEAD moved from %s to %s", oldSHA, got)
	}
	if data, err := os.ReadFile(filepath.Join(boardWT, ".factory", "new", "draft.md")); err != nil || string(data) != "uncommitted\n" {
		t.Fatalf("dirty file altered: %v %q", err, string(data))
	}
}

func TestSyncBranchSkipsDivergedLocal(t *testing.T) {
	ctx := context.Background()
	_, seedDir, workDir := setupSyncTestEnv(t)

	boardWT := filepath.Join(filepath.Dir(workDir), "board-wt-div")
	runCmd(t, workDir, "worktree", "add", "-b", "factory_board", boardWT, "origin/factory_board")
	writeTestFile(t, filepath.Join(boardWT, ".factory", "new", "FT99-local.md"), "# Local\n\n## alex · 2026-11-10 10:30\nLocal only\n")
	runCmd(t, boardWT, "add", ".factory")
	runCmd(t, boardWT, "commit", "-m", "Local diverged commit")
	localSHA := runCmd(t, boardWT, "rev-parse", "HEAD")

	pushBoardCommit(t, seedDir, ".factory/new/FT2-beta.md", "# Beta\n\n## alex · 2026-11-10 10:00\nRemote\n", "Board v2")

	cand := BranchCandidate{
		Ref:       "refs/remotes/origin/factory_board",
		ShortName: "origin/factory_board",
		Remote:    "origin",
		Branch:    "factory_board",
		IsRemote:  true,
	}
	res := SyncBranch(ctx, workDir, cand, "origin", false, false)
	if res.FetchErr != nil || res.LocalState != LocalSyncDiverged {
		t.Fatalf("SyncBranch got %+v, want LocalSyncDiverged", res)
	}
	if got := runCmd(t, boardWT, "rev-parse", "HEAD"); got != localSHA {
		t.Fatalf("diverged local HEAD moved from %s to %s", localSHA, got)
	}
}

func TestSyncBranchLocalCandidateWithUpstream(t *testing.T) {
	ctx := context.Background()
	_, seedDir, workDir := setupSyncTestEnv(t)

	runCmd(t, workDir, "branch", "--track", "factory_board", "origin/factory_board")
	wantSHA := pushBoardCommit(t, seedDir, ".factory/new/FT2-beta.md", "# Beta\n\n## alex · 2026-11-10 10:00\nSecond\n", "Board v2")

	localCand := BranchCandidate{
		Ref:       "refs/heads/factory_board",
		ShortName: "factory_board",
		Branch:    "factory_board",
		IsRemote:  false,
	}
	res := SyncBranch(ctx, workDir, localCand, "origin", false, false)
	if res.FetchErr != nil || res.LocalState != LocalSyncFastFwd {
		t.Fatalf("SyncBranch(local candidate) got %+v, want LocalSyncFastFwd", res)
	}
	if got := runCmd(t, workDir, "rev-parse", "refs/heads/factory_board"); got != wantSHA {
		t.Fatalf("local factory_board SHA=%s, want %s", got, wantSHA)
	}
}
