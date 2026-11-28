package gitboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const DefaultFactoryBranch = "factory_board"

// BranchCandidate represents a git branch ref containing .factory/factory.toml.
type BranchCandidate struct {
	Ref       string `json:"ref"`
	ShortName string `json:"short_name"`
	Remote    string `json:"remote,omitempty"`
	Branch    string `json:"branch"`
	CommitSHA string `json:"commit_sha"`
	IsRemote  bool   `json:"is_remote"`
}

// FindRepoRoot resolves the top-level git working tree or git dir from startDir.
func FindRepoRoot(ctx context.Context, startDir string) (string, error) {
	out, _, err := RunGit(ctx, startDir, "rev-parse", "--show-toplevel")
	if err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root, nil
		}
	}
	// Bare repository fallback.
	out, _, bareErr := RunGit(ctx, startDir, "rev-parse", "--git-dir")
	if bareErr != nil {
		if err != nil {
			return "", fmt.Errorf("not a git repository (%s): %w", startDir, err)
		}
		return "", fmt.Errorf("not a git repository (%s): %w", startDir, bareErr)
	}
	return startDir, nil
}

// DiscoverCandidates locates all refs containing .factory/factory.toml sorted by priority.
func DiscoverCandidates(ctx context.Context, repoDir, preferredRemote, preferredBranch string) ([]BranchCandidate, error) {
	if preferredRemote == "" {
		preferredRemote = "origin"
	}

	var raw []BranchCandidate
	if strings.TrimSpace(preferredBranch) != "" {
		raw = probeExplicitRefs(ctx, repoDir, preferredRemote, strings.TrimSpace(preferredBranch))
	} else {
		var err error
		raw, err = listAllRefs(ctx, repoDir)
		if err != nil {
			return nil, err
		}
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("no factory branch found (expected a branch with .factory/factory.toml)")
	}

	matched, err := filterHasFactoryToml(ctx, repoDir, raw)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		if preferredBranch != "" {
			return nil, fmt.Errorf("branch %q does not contain .factory/factory.toml", preferredBranch)
		}
		return nil, fmt.Errorf("no factory branch found (none of %d branches contain .factory/factory.toml)", len(raw))
	}

	sortCandidates(matched, preferredRemote)
	return matched, nil
}

func probeExplicitRefs(ctx context.Context, repoDir, preferredRemote, branch string) []BranchCandidate {
	type spec struct {
		ref    string
		remote string
		branch string
		isRem  bool
	}
	var specs []spec
	switch {
	case strings.HasPrefix(branch, "refs/remotes/"):
		rest := strings.TrimPrefix(branch, "refs/remotes/")
		rem, br, ok := strings.Cut(rest, "/")
		if ok {
			specs = append(specs, spec{ref: branch, remote: rem, branch: br, isRem: true})
		}
	case strings.HasPrefix(branch, "refs/heads/"):
		br := strings.TrimPrefix(branch, "refs/heads/")
		specs = append(specs, spec{ref: branch, branch: br})
	default:
		specs = append(specs,
			spec{ref: "refs/remotes/" + preferredRemote + "/" + branch, remote: preferredRemote, branch: branch, isRem: true},
			spec{ref: "refs/heads/" + branch, branch: branch},
		)
		if rem, br, ok := strings.Cut(branch, "/"); ok {
			specs = append(specs, spec{ref: "refs/remotes/" + rem + "/" + br, remote: rem, branch: br, isRem: true})
		}
	}

	seen := make(map[string]bool)
	var out []BranchCandidate
	for _, sp := range specs {
		if seen[sp.ref] {
			continue
		}
		seen[sp.ref] = true
		sha, err := ResolveSHA(ctx, repoDir, sp.ref)
		if err != nil {
			continue
		}
		shortName := sp.branch
		if sp.isRem {
			shortName = sp.remote + "/" + sp.branch
		}
		out = append(out, BranchCandidate{
			Ref:       sp.ref,
			ShortName: shortName,
			Remote:    sp.remote,
			Branch:    sp.branch,
			CommitSHA: sha,
			IsRemote:  sp.isRem,
		})
	}
	return out
}

func listAllRefs(ctx context.Context, repoDir string) ([]BranchCandidate, error) {
	out, _, err := RunGit(ctx, repoDir, "for-each-ref", "--format=%(refname)\t%(objectname)\t%(symref)", "refs/remotes", "refs/heads")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	candidates := make([]BranchCandidate, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		ref, sha := parts[0], parts[1]
		if len(parts) >= 3 && parts[2] != "" {
			continue // Skip symbolic refs like origin/HEAD.
		}
		if strings.HasSuffix(ref, "/HEAD") {
			continue
		}
		switch {
		case strings.HasPrefix(ref, "refs/remotes/"):
			short := strings.TrimPrefix(ref, "refs/remotes/")
			rem, br, ok := strings.Cut(short, "/")
			if !ok {
				continue
			}
			candidates = append(candidates, BranchCandidate{
				Ref:       ref,
				ShortName: short,
				Remote:    rem,
				Branch:    br,
				CommitSHA: sha,
				IsRemote:  true,
			})
		case strings.HasPrefix(ref, "refs/heads/"):
			br := strings.TrimPrefix(ref, "refs/heads/")
			candidates = append(candidates, BranchCandidate{
				Ref:       ref,
				ShortName: br,
				Branch:    br,
				CommitSHA: sha,
				IsRemote:  false,
			})
		}
	}
	return candidates, nil
}

func filterHasFactoryToml(ctx context.Context, repoDir string, candidates []BranchCandidate) ([]BranchCandidate, error) {
	var stdin bytes.Buffer
	for _, c := range candidates {
		stdin.WriteString(c.Ref)
		stdin.WriteString(":.factory/factory.toml\n")
	}
	out, _, err := RunGitWithStdin(ctx, repoDir, stdin.Bytes(), "cat-file", "--batch-check=%(objecttype)")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if len(lines) != len(candidates) {
		return nil, fmt.Errorf("unexpected cat-file --batch-check line count: got %d, want %d", len(lines), len(candidates))
	}
	var matched []BranchCandidate
	for i, line := range lines {
		if strings.TrimSpace(line) == "blob" {
			matched = append(matched, candidates[i])
		}
	}
	return matched, nil
}

func sortCandidates(candidates []BranchCandidate, preferredRemote string) {
	rank := func(c BranchCandidate) int {
		isDefaultName := c.Branch == DefaultFactoryBranch
		switch {
		case c.IsRemote && c.Remote == preferredRemote && isDefaultName:
			return 0
		case c.IsRemote && isDefaultName:
			return 1
		case !c.IsRemote && isDefaultName:
			return 2
		case c.IsRemote && c.Remote == preferredRemote:
			return 3
		case c.IsRemote:
			return 4
		default:
			return 5
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		ri, rj := rank(candidates[i]), rank(candidates[j])
		if ri != rj {
			return ri < rj
		}
		return candidates[i].ShortName < candidates[j].ShortName
	})
}

// ResolveSHA resolves a ref or revision string to a full commit SHA.
func ResolveSHA(ctx context.Context, repoDir, ref string) (string, error) {
	out, _, err := RunGit(ctx, repoDir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// FetchBranch runs a targeted non-interactive fetch for a single factory branch.
func FetchBranch(ctx context.Context, repoDir, remote, branch string) error {
	if strings.TrimSpace(remote) == "" || strings.TrimSpace(branch) == "" {
		return nil
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	refspec := fmt.Sprintf("refs/heads/%s:refs/remotes/%s/%s", branch, remote, branch)
	_, _, err := RunGit(fetchCtx, repoDir, "fetch", "--no-tags", "--quiet", remote, refspec)
	return err
}

// LocalSyncState describes the outcome of auto fast-forwarding the matching local branch.
type LocalSyncState int

const (
	LocalSyncNone LocalSyncState = iota
	LocalSyncUpToDate
	LocalSyncFastFwd
	LocalSyncDirty
	LocalSyncDiverged
	LocalSyncError
)

// SyncResult captures the result of fetching a factory branch and fast-forwarding any matching local branch.
type SyncResult struct {
	FetchErr    error
	LocalState  LocalSyncState
	LocalDetail string
}

// SyncBranch fetches the remote tracking branch (if configured and !noFetch) and, unless noPull is set,
// safely fast-forwards a matching local branch when it is strictly behind.
func SyncBranch(ctx context.Context, repoDir string, candidate BranchCandidate, preferredRemote string, noFetch, noPull bool) SyncResult {
	remote, remoteBranch, remoteRef := resolveRemoteTracking(ctx, repoDir, candidate, preferredRemote)
	if remote == "" || noFetch {
		return SyncResult{}
	}
	if err := FetchBranch(ctx, repoDir, remote, remoteBranch); err != nil {
		return SyncResult{FetchErr: err}
	}
	if noPull {
		return SyncResult{}
	}
	state, detail := FastForwardLocal(ctx, repoDir, candidate.Branch, remoteRef)
	return SyncResult{
		LocalState:  state,
		LocalDetail: detail,
	}
}

func resolveRemoteTracking(ctx context.Context, repoDir string, candidate BranchCandidate, preferredRemote string) (remote, remoteBranch, remoteRef string) {
	if candidate.IsRemote {
		return candidate.Remote, candidate.Branch, candidate.Ref
	}
	if strings.TrimSpace(candidate.Branch) == "" {
		return "", "", ""
	}
	if remOut, _, err := RunGit(ctx, repoDir, "config", "--get", "branch."+candidate.Branch+".remote"); err == nil {
		rem := strings.TrimSpace(string(remOut))
		if rem != "" && rem != "." {
			if mergeOut, _, err := RunGit(ctx, repoDir, "config", "--get", "branch."+candidate.Branch+".merge"); err == nil {
				mergeBranch := strings.TrimPrefix(strings.TrimSpace(string(mergeOut)), "refs/heads/")
				if mergeBranch != "" {
					return rem, mergeBranch, "refs/remotes/" + rem + "/" + mergeBranch
				}
			}
		}
	}
	if strings.TrimSpace(preferredRemote) == "" {
		preferredRemote = "origin"
	}
	fallbackRef := "refs/remotes/" + preferredRemote + "/" + candidate.Branch
	if _, err := ResolveSHA(ctx, repoDir, fallbackRef); err == nil {
		return preferredRemote, candidate.Branch, fallbackRef
	}
	return "", "", ""
}

// FastForwardLocal safely advances refs/heads/<localBranch> to remoteRef if it already exists locally
// and is strictly behind remoteRef. Checked-out clean worktrees are updated with merge --ff-only;
// dirty, busy, ahead, or diverged local branches are left untouched.
func FastForwardLocal(ctx context.Context, repoDir, localBranch, remoteRef string) (LocalSyncState, string) {
	if strings.TrimSpace(localBranch) == "" || strings.TrimSpace(remoteRef) == "" {
		return LocalSyncNone, ""
	}
	localRef := "refs/heads/" + localBranch
	localSHA, err := ResolveSHA(ctx, repoDir, localRef)
	if err != nil {
		return LocalSyncNone, ""
	}
	remoteSHA, err := ResolveSHA(ctx, repoDir, remoteRef)
	if err != nil {
		return LocalSyncNone, ""
	}
	if localSHA == remoteSHA {
		return LocalSyncUpToDate, ""
	}

	anc, err := isAncestor(ctx, repoDir, localSHA, remoteSHA)
	if err != nil {
		return LocalSyncError, fmt.Sprintf("local %s check failed", localBranch)
	}
	if !anc {
		revAnc, revErr := isAncestor(ctx, repoDir, remoteSHA, localSHA)
		if revErr == nil && revAnc {
			return LocalSyncDiverged, fmt.Sprintf("local %s ahead", localBranch)
		}
		return LocalSyncDiverged, fmt.Sprintf("local %s diverged", localBranch)
	}

	wtPath, err := findBranchWorktree(ctx, repoDir, localRef)
	if err != nil {
		return LocalSyncError, fmt.Sprintf("local %s worktree check failed", localBranch)
	}
	if wtPath == "" {
		if _, _, err := RunGit(ctx, repoDir, "update-ref", localRef, remoteSHA, localSHA); err != nil {
			return LocalSyncError, fmt.Sprintf("local %s ff failed", localBranch)
		}
		return LocalSyncFastFwd, fmt.Sprintf("local %s ff → %s", localBranch, shortSHA(remoteSHA))
	}

	busy, err := worktreeBusy(ctx, wtPath)
	if err != nil || busy {
		return LocalSyncDirty, fmt.Sprintf("local %s busy (skipped pull)", localBranch)
	}
	statusOut, _, err := RunGit(ctx, wtPath, "status", "--porcelain", "--untracked-files=normal")
	if err != nil || strings.TrimSpace(string(statusOut)) != "" {
		return LocalSyncDirty, fmt.Sprintf("local %s dirty (skipped pull)", localBranch)
	}
	if _, _, err := RunGit(ctx, wtPath, "merge", "--ff-only", "--quiet", remoteSHA); err != nil {
		return LocalSyncDirty, fmt.Sprintf("local %s pull skipped", localBranch)
	}
	return LocalSyncFastFwd, fmt.Sprintf("local %s pulled → %s", localBranch, shortSHA(remoteSHA))
}

func isAncestor(ctx context.Context, repoDir, ancestorSHA, descendantSHA string) (bool, error) {
	_, _, err := RunGit(ctx, repoDir, "merge-base", "--is-ancestor", ancestorSHA, descendantSHA)
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func findBranchWorktree(ctx context.Context, repoDir, fullBranchRef string) (string, error) {
	out, _, err := RunGit(ctx, repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	var curPath string
	var curBranch string
	var isBare bool
	flush := func() string {
		if !isBare && curPath != "" && curBranch == fullBranchRef {
			return curPath
		}
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			if p := flush(); p != "" {
				return p, nil
			}
			curPath, curBranch, isBare = "", "", false
			continue
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			curPath = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			curBranch = strings.TrimPrefix(line, "branch ")
		case line == "bare":
			isBare = true
		}
	}
	return flush(), nil
}

func worktreeBusy(ctx context.Context, wtPath string) (bool, error) {
	out, _, err := RunGit(ctx, wtPath, "rev-parse", "--git-dir")
	if err != nil {
		return false, err
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(wtPath, gitDir)
	}
	markers := []string{
		"MERGE_HEAD",
		"REBASE_HEAD",
		"CHERRY_PICK_HEAD",
		"REVERT_HEAD",
		"BISECT_LOG",
		"rebase-merge",
		"rebase-apply",
	}
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(gitDir, m)); err == nil {
			return true, nil
		}
	}
	return false, nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
