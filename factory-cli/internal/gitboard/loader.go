package gitboard

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"factory-cli/internal/model"
)

type treeEntry struct {
	OID    string
	Path   string
	Folder string
	Name   string
}

var (
	githubRemoteRe = regexp.MustCompile(`^(?:git@github\.com:|https://github\.com/|ssh://git@github\.com/)([\w.-]+/[\w.-]+?)(?:\.git)?/?$`)
	repoNameRe     = regexp.MustCompile(`([^/:]+?)(?:\.git)?/?$`)
)

// LoadSnapshot reads .factory/factory.toml, item files, and move history directly from git objects.
func LoadSnapshot(ctx context.Context, repoDir string, candidate BranchCandidate, now time.Time) (*model.Board, error) {
	sha, err := ResolveSHA(ctx, repoDir, candidate.Ref)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", candidate.Ref, err)
	}

	entries, tomlOID, err := readFactoryTree(ctx, repoDir, sha)
	if err != nil {
		return nil, err
	}
	if tomlOID == "" {
		return nil, fmt.Errorf("%s has no .factory/factory.toml", candidate.ShortName)
	}

	oids := make([]string, 0, len(entries)+1)
	oids = append(oids, tomlOID)
	for _, e := range entries {
		oids = append(oids, e.OID)
	}
	blobs, err := readBlobsBatch(ctx, repoDir, oids)
	if err != nil {
		return nil, err
	}

	cfg, loc, err := model.ParseConfig(blobs[tomlOID])
	if err != nil {
		return nil, err
	}

	remoteName := candidate.Remote
	if remoteName == "" {
		remoteName = cfg.Remote
	}
	repoName, ghURL, ghSlug := readRemoteMetadata(ctx, repoDir, remoteName)

	history, err := readMoveHistory(ctx, repoDir, sha, cfg.Prefix)
	if err != nil {
		return nil, err
	}

	signers := cfg.SignerSet()
	folderSet := make(map[string]bool, len(model.AllFolders))
	for _, f := range model.AllFolders {
		folderSet[f] = true
	}

	var items []model.Item
	for _, e := range entries {
		if !folderSet[e.Folder] {
			continue
		}
		ident, ok := model.ParseFilename(e.Folder, e.Name, cfg.Prefix)
		if !ok {
			continue
		}
		parsed := model.ParseItemContent(string(blobs[e.OID]), signers)
		steps := history[e.Name]
		var changed time.Time
		if len(steps) > 0 {
			changed = steps[len(steps)-1].At
		}
		branch := parsed.Header["branch"]
		link := ""
		if ghURL != "" && branch != "" {
			link = fmt.Sprintf("%s/compare/%s...%s", ghURL, cfg.Base, branch)
		}
		items = append(items, model.Item{
			Number:   ident.Number,
			Slug:     ident.Slug,
			Folder:   e.Folder,
			File:     e.Path,
			Name:     e.Name,
			ID:       ident.ID,
			Label:    ident.Label,
			Title:    parsed.Title,
			Source:   parsed.Header["source"],
			Branch:   branch,
			RawRound: parsed.Header["round"],
			Sections: parsed.Sections,
			Problems: parsed.Problems,
			Steps:    steps,
			Changed:  changed,
			Link:     link,
		})
	}

	return model.BuildBoard(repoName, ghURL, ghSlug, candidate.Ref, candidate.ShortName, sha, cfg, loc, items, now), nil
}

func readFactoryTree(ctx context.Context, repoDir, sha string) ([]treeEntry, string, error) {
	out, _, err := RunGit(ctx, repoDir, "ls-tree", "-r", "-z", sha, "--", ".factory")
	if err != nil {
		return nil, "", err
	}
	var entries []treeEntry
	var tomlOID string
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		meta, filePath, ok := bytes.Cut(rec, []byte{'\t'})
		if !ok {
			continue
		}
		fields := strings.Fields(string(meta))
		if len(fields) < 3 || fields[1] != "blob" {
			continue
		}
		oid := fields[2]
		p := string(filePath)
		if p == ".factory/factory.toml" {
			tomlOID = oid
			continue
		}
		parts := strings.Split(p, "/")
		if len(parts) != 3 || parts[0] != ".factory" {
			continue
		}
		entries = append(entries, treeEntry{
			OID:    oid,
			Path:   p,
			Folder: parts[1],
			Name:   parts[2],
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries, tomlOID, nil
}

func readBlobsBatch(ctx context.Context, repoDir string, oids []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(oids))
	if len(oids) == 0 {
		return result, nil
	}
	var stdin bytes.Buffer
	seen := make(map[string]bool, len(oids))
	for _, oid := range oids {
		if oid == "" || seen[oid] {
			continue
		}
		seen[oid] = true
		stdin.WriteString(oid)
		stdin.WriteByte('\n')
	}

	out, _, err := RunGitWithStdin(ctx, repoDir, stdin.Bytes(), "cat-file", "--batch")
	if err != nil {
		return nil, err
	}

	pos := 0
	for pos < len(out) {
		nl := bytes.IndexByte(out[pos:], '\n')
		if nl < 0 {
			return nil, fmt.Errorf("malformed cat-file --batch header at offset %d", pos)
		}
		header := string(out[pos : pos+nl])
		pos += nl + 1

		parts := strings.Fields(header)
		if len(parts) >= 2 && parts[1] == "missing" {
			continue
		}
		if len(parts) != 3 {
			return nil, fmt.Errorf("unexpected cat-file --batch header %q", header)
		}
		oid := parts[0]
		size, err := strconv.Atoi(parts[2])
		if err != nil || size < 0 {
			return nil, fmt.Errorf("invalid blob size in header %q", header)
		}
		if pos+size > len(out) {
			return nil, fmt.Errorf("truncated blob data for %s: need %d bytes, have %d", oid, size, len(out)-pos)
		}
		content := make([]byte, size)
		copy(content, out[pos:pos+size])
		result[oid] = content
		pos += size
		if pos < len(out) && out[pos] == '\n' {
			pos++
		}
	}
	return result, nil
}

func readMoveHistory(ctx context.Context, repoDir, sha, prefix string) (map[string][]model.Move, error) {
	out, _, err := RunGit(ctx, repoDir, "log", sha, "--reverse", "--no-renames", "--name-status", "--format=%x00%ct", "--", ".factory")
	if err != nil {
		return nil, err
	}
	history := make(map[string][]model.Move)
	chunks := bytes.Split(out, []byte{0})
	for _, chunk := range chunks[1:] {
		trimmed := strings.TrimSpace(string(chunk))
		if trimmed == "" {
			continue
		}
		lines := strings.Split(trimmed, "\n")
		unixSec, err := strconv.ParseInt(strings.TrimSpace(lines[0]), 10, 64)
		if err != nil {
			continue
		}
		ts := time.Unix(unixSec, 0)

		type addedEntry struct {
			folder string
			name   string
		}
		var added []addedEntry
		deleted := make(map[string]bool)

		for _, line := range lines[1:] {
			status, filePath, ok := strings.Cut(line, "\t")
			if !ok {
				continue
			}
			parts := strings.Split(strings.TrimSpace(filePath), "/")
			if len(parts) != 3 || parts[0] != ".factory" {
				continue
			}
			folder, name := parts[1], parts[2]
			if status == "D" {
				deleted[name] = true
			} else if status == "A" || history[name] == nil {
				added = append(added, addedEntry{folder: folder, name: name})
			}
		}

		for _, entry := range added {
			bare := entry.name
			if ident, ok := model.ParseFilename(entry.folder, entry.name, prefix); ok {
				bare = ident.Bare
			}
			if history[entry.name] == nil && bare != entry.name && deleted[bare] && history[bare] != nil {
				history[entry.name] = history[bare]
				delete(history, bare)
			}
			steps := history[entry.name]
			if len(steps) == 0 || steps[len(steps)-1].Folder != entry.folder {
				history[entry.name] = append(steps, model.Move{Folder: entry.folder, At: ts})
			}
		}
	}
	return history, nil
}

func readRemoteMetadata(ctx context.Context, repoDir, remote string) (repoName, ghURL, ghSlug string) {
	candidates := []string{remote}
	if remote != "origin" {
		candidates = append(candidates, "origin")
	}
	for _, rem := range candidates {
		if strings.TrimSpace(rem) == "" {
			continue
		}
		out, _, err := RunGit(ctx, repoDir, "remote", "get-url", rem)
		if err != nil {
			continue
		}
		rawURL := strings.TrimSpace(string(out))
		if rawURL == "" {
			continue
		}
		if m := githubRemoteRe.FindStringSubmatch(rawURL); m != nil {
			ghSlug = m[1]
			ghURL = "https://github.com/" + ghSlug
			repoName = path.Base(ghSlug)
			return repoName, ghURL, ghSlug
		}
		if m := repoNameRe.FindStringSubmatch(rawURL); m != nil {
			repoName = m[1]
			return repoName, "", ""
		}
	}
	abs, err := filepath.Abs(repoDir)
	if err != nil {
		return filepath.Base(repoDir), "", ""
	}
	return filepath.Base(abs), "", ""
}
