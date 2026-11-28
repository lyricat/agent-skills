package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"factory-cli/internal/gitboard"
	"factory-cli/internal/model"
)

func testBoard() *model.Board {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	n1 := 1
	n2 := 2
	items := []model.Item{
		{
			Number:  &n1,
			Slug:    "search-box",
			Folder:  model.FolderReady,
			File:    ".factory/ready/FT1-search-box.md",
			Name:    "FT1-search-box.md",
			ID:      "1",
			Label:   "FT1",
			Title:   "Search box",
			Branch:  "factory/FT1-search-box",
			Changed: now.Add(-2 * time.Hour),
			Sections: []model.Section{
				{Who: "alex", Time: "2026-11-10 09:00", Body: "Need search"},
				{Who: "factory", Kind: "review", Time: "2026-11-10 10:00", Body: "verdict: pass", Verdict: "pass"},
			},
			Steps: []model.Move{
				{Folder: model.FolderNew, At: now.Add(-4 * time.Hour)},
				{Folder: model.FolderReady, At: now.Add(-2 * time.Hour)},
			},
		},
		{
			Number:  &n2,
			Slug:    "parked-idea",
			Folder:  model.FolderStop,
			File:    ".factory/stop/FT2-parked-idea.md",
			Name:    "FT2-parked-idea.md",
			ID:      "2",
			Label:   "FT2",
			Title:   "Parked",
			Changed: now.Add(-5 * time.Hour),
			Sections: []model.Section{
				{Who: "alex", Time: "2026-11-10 07:00", Body: "Wait"},
			},
		},
	}
	return model.BuildBoard("demo", "https://github.com/acme/demo", "acme/demo", "refs/remotes/origin/factory_board", "origin/factory_board", "abcdef1234567", model.DefaultConfig(), time.UTC, items, now)
}

func TestRenderStatusAndDetail(t *testing.T) {
	board := testBoard()
	runs := []gitboard.ActiveRun{{Kind: "build", Since: board.FetchedAt, Items: []string{"FT1-search-box"}}}
	board = gitboard.ApplyActiveRuns(board, runs)

	var buf bytes.Buffer
	err := RenderStatus(&buf, board, StatusOptions{
		MineOnly:   true,
		ActiveRuns: runs,
	})
	if err != nil {
		t.Fatalf("RenderStatus: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "FT1") || !strings.Contains(out, "Check, then ship") || !strings.Contains(out, "running: build (FT1)") || !strings.Contains(out, "[building]") {
		t.Fatalf("unexpected status output:\n%s", out)
	}
	if strings.Contains(out, "FT2") {
		t.Fatalf("stop item FT2 should be hidden by default:\n%s", out)
	}

	buf.Reset()
	if err := RenderStatus(&buf, board, StatusOptions{JSON: true, ActiveRuns: runs}); err != nil {
		t.Fatalf("RenderStatus JSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"running": "build"`) {
		t.Fatalf("expected running field in JSON:\n%s", buf.String())
	}
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	buf.Reset()
	if err := RenderItemDetail(&buf, board, &board.Items[0], false); err != nil {
		t.Fatalf("RenderItemDetail: %v", err)
	}
	if !strings.Contains(buf.String(), "Running: building") || !strings.Contains(buf.String(), "Trail: new (2h) → ready (2h)") {
		t.Fatalf("unexpected detail output:\n%s", buf.String())
	}
}
