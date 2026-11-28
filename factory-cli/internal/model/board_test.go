package model

import (
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	cfg, loc, err := ParseConfig([]byte(`
base = "develop"
owners = ["alex", "sam"]
max_rounds = 3
prefix = "AB"
timezone = "UTC"
`))
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}
	if cfg.Base != "develop" || cfg.Prefix != "AB" || cfg.MaxRounds != 3 {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if loc.String() != "UTC" {
		t.Fatalf("loc=%v, want UTC", loc)
	}

	if _, _, err := ParseConfig([]byte(`prefix = "ft"`)); err == nil {
		t.Fatalf("expected error for lowercase prefix")
	}
}

func TestBuildBoardAndDiff(t *testing.T) {
	loc := time.UTC
	cfg := DefaultConfig()
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, loc)
	n1 := 1
	n2 := 2

	items1 := []Item{
		{
			Number:   &n1,
			Slug:     "alpha",
			Folder:   FolderReview,
			File:     ".factory/review/FT1-alpha.md",
			Name:     "FT1-alpha.md",
			ID:       "1",
			Label:    "FT1",
			Title:    "Alpha",
			RawRound: "1",
			Changed:  now.Add(-2 * time.Hour),
			Sections: []Section{
				{Who: "alex", Time: "2026-11-10 09:00", Body: "build alpha"},
				{Who: "factory", Kind: "review", Time: "2026-11-10 11:00", Body: "verdict: fix", Verdict: "fix"},
			},
			Steps: []Move{
				{Folder: FolderNew, At: now.Add(-5 * time.Hour)},
				{Folder: FolderReview, At: now.Add(-2 * time.Hour)},
			},
		},
		{
			Number:  &n2,
			Slug:    "beta",
			Folder:  FolderNew,
			File:    ".factory/new/FT2-beta.md",
			Name:    "FT2-beta.md",
			ID:      "2",
			Label:   "FT2",
			Title:   "Beta",
			Changed: now.Add(-1 * time.Hour),
			Sections: []Section{
				{Who: "alex", Time: "2026-11-10 11:00", Body: "build beta"},
			},
		},
	}

	b1 := BuildBoard("demo", "", "owner/demo", "refs/remotes/origin/factory_board", "origin/factory_board", "aaa1111", cfg, loc, items1, now)
	if b1.Yours != 1 || b1.Agents != 1 {
		t.Fatalf("counts yours=%d agents=%d, want 1/1", b1.Yours, b1.Agents)
	}
	if b1.Items[1].Folder != FolderReview || b1.Items[1].Ask != "Decide: fix or take it" {
		t.Fatalf("unexpected review item: %+v", b1.Items[1])
	}
	if avg := AverageStay(b1.Items, FolderNew); avg != 3*time.Hour {
		t.Fatalf("AverageStay(new)=%v, want 3h", avg)
	}

	items2 := append([]Item(nil), items1...)
	items2[0].Folder = FolderReady
	items2[0].File = ".factory/ready/FT1-alpha.md"
	items2[0].Sections = append(items2[0].Sections, Section{Who: "factory", Kind: "review", Time: "2026-11-10 11:30", Body: "verdict: pass", Verdict: "pass"})
	b2 := BuildBoard("demo", "", "owner/demo", "refs/remotes/origin/factory_board", "origin/factory_board", "bbb2222", cfg, loc, items2, now)

	diffs := DiffBoards(b1, b2)
	if len(diffs) != 1 || diffs[0] != "FT1 review → ready" {
		t.Fatalf("diffs=%v, want [FT1 review → ready]", diffs)
	}

	found, err := FindItem(b2, "ft1")
	if err != nil || found.Label != "FT1" {
		t.Fatalf("FindItem(ft1) got %+v, err=%v", found, err)
	}
}
