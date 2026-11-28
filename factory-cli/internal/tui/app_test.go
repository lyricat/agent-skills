package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"factory-cli/internal/gitboard"
	"factory-cli/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func makeBoardForTUI(sha string, folder string) *model.Board {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	n1, n2 := 1, 2
	items := []model.Item{
		{
			Number:  &n1,
			Slug:    "search-box",
			Folder:  folder,
			File:    ".factory/" + folder + "/FT1-search-box.md",
			Name:    "FT1-search-box.md",
			ID:      "1",
			Label:   "FT1",
			Title:   "Search box",
			Changed: now.Add(-1 * time.Hour),
			Sections: []model.Section{
				{Who: "alex", Time: "2026-11-10 09:00", Body: "Add search"},
				{Who: "factory", Kind: "review", Time: "2026-11-10 10:00", Body: "verdict: fix\nfix:\n- [P1] fix bug", Verdict: "fix"},
			},
		},
		{
			Number:  &n2,
			Slug:    "auth-flow",
			Folder:  model.FolderNew,
			File:    ".factory/new/FT2-auth-flow.md",
			Name:    "FT2-auth-flow.md",
			ID:      "2",
			Label:   "FT2",
			Title:   "Auth flow",
			Changed: now.Add(-2 * time.Hour),
			Sections: []model.Section{
				{Who: "alex", Time: "2026-11-10 08:00", Body: "Add auth"},
			},
		},
	}
	return model.BuildBoard("demo", "", "acme/demo", "refs/remotes/origin/factory_board", "origin/factory_board", sha, model.DefaultConfig(), time.UTC, items, now)
}

func TestAppRefreshPreservesSelectionAndDegradesOffline(t *testing.T) {
	candidates := []gitboard.BranchCandidate{
		{Ref: "refs/remotes/origin/factory_board", ShortName: "origin/factory_board", Remote: "origin", Branch: "factory_board", IsRemote: true},
	}
	b1 := makeBoardForTUI("sha1111", model.FolderReview)
	b2 := makeBoardForTUI("sha2222", model.FolderReady)

	m := NewAppModel(Options{Interval: 30 * time.Second}, candidates, b1, nil, func(ctx context.Context, opts Options, c gitboard.BranchCandidate, prevSHA string, force bool) (RefreshPayload, error) {
		return RefreshPayload{Board: b2}, nil
	})
	m.Width = 140
	m.Height = 25

	// Select FT1 (in review) and open detail.
	// Note: FT2 (new) is sorted before FT1 (review) when viewing All because FT2 is older (8am vs 10am).
	for i, it := range m.filteredItems() {
		if it.Label == "FT1" {
			m.SelectedIdx = i
			m.recordSelection(m.filteredItems())
		}
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if m.Mode != ModeDetail {
		t.Fatalf("expected ModeDetail, got %v", m.Mode)
	}

	// Simulate background refresh where FT1 moved from review -> ready and local branch ff'd.
	m.RefreshSeq++
	updated, _ = m.Update(refreshMsg{
		board:       b2,
		diffs:       []string{"FT1 review → ready"},
		localState:  gitboard.LocalSyncFastFwd,
		localDetail: "local factory_board ff → sha2222",
		syncedAt:    time.Now(),
		seq:         m.RefreshSeq,
	})
	m = updated.(AppModel)
	sel := m.selectedItem()
	if sel == nil || sel.Label != "FT1" || sel.Folder != model.FolderReady {
		t.Fatalf("expected FT1 in ready to stay selected after move, got %+v", sel)
	}
	if !strings.Contains(m.View(), "FT1 review → ready") {
		t.Fatalf("expected diff toast in header:\n%s", m.View())
	}
	if !strings.Contains(m.View(), "local factory_board ff → sha2222") {
		t.Fatalf("expected local ff notice in header:\n%s", m.View())
	}

	// Simulate dirty worktree skip notice persisted in header.
	m.RefreshSeq++
	updated, _ = m.Update(refreshMsg{
		unchanged:   true,
		localState:  gitboard.LocalSyncDirty,
		localDetail: "local factory_board dirty (skipped pull)",
		syncedAt:    time.Now(),
		seq:         m.RefreshSeq,
	})
	m = updated.(AppModel)
	if !strings.Contains(m.View(), "local factory_board dirty (skipped pull)") {
		t.Fatalf("expected local dirty notice in header:\n%s", m.View())
	}

	// Simulate offline fetch error: existing board remains visible.
	m.RefreshSeq++
	updated, _ = m.Update(refreshMsg{
		fetchErr:  "could not resolve host",
		unchanged: true,
		syncedAt:  time.Now(),
		seq:       m.RefreshSeq,
	})
	m = updated.(AppModel)
	if m.Board == nil || !strings.Contains(m.View(), "offline:") {
		t.Fatalf("expected soft offline warning while keeping board:\n%s", m.View())
	}

	// Return to list and filter by mine-only.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)

	// Verify live per-item running indicator updates on unchanged: true refreshMsg.
	m.RefreshSeq++
	updated, _ = m.Update(refreshMsg{
		unchanged:  true,
		activeRuns: []gitboard.ActiveRun{{Kind: "build", Since: time.Now(), Items: []string{"FT2-auth-flow"}}},
		syncedAt:   time.Now(),
		seq:        m.RefreshSeq,
	})
	m = updated.(AppModel)
	if !strings.Contains(m.View(), "running: build (FT2)") || !strings.Contains(m.View(), "[● building]") {
		t.Fatalf("expected per-item running badge in list view:\n%s", m.View())
	}

	// Open FT2 detail and verify running banner, then clear lock on next tick.
	for i, it := range m.filteredItems() {
		if it.Label == "FT2" {
			m.SelectedIdx = i
			m.recordSelection(m.filteredItems())
		}
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if !strings.Contains(m.View(), "● Running: building") {
		t.Fatalf("expected running banner in detail view:\n%s", m.View())
	}
	m.RefreshSeq++
	updated, _ = m.Update(refreshMsg{
		unchanged:  true,
		activeRuns: nil,
		syncedAt:   time.Now(),
		seq:        m.RefreshSeq,
	})
	m = updated.(AppModel)
	if strings.Contains(m.View(), "● Running: building") {
		t.Fatalf("expected running banner to clear when lock disappears:\n%s", m.View())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(AppModel)
	items := m.filteredItems()
	if len(items) != 1 || items[0].Label != "FT1" {
		t.Fatalf("expected mine-only filter to keep only FT1, got %+v", items)
	}
}

func TestDetailHeaderStaysPinnedWhileScrolling(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	n1 := 1
	var sections []model.Section
	for i := 1; i <= 12; i++ {
		sections = append(sections, model.Section{
			Who:  "alex",
			Time: "2026-11-10 09:00",
			Body: strings.Repeat("Section "+string(rune('A'+i-1))+" details line\n", 2),
		})
	}
	board := model.BuildBoard("demo", "", "acme/demo", "refs/remotes/origin/factory_board", "origin/factory_board", "sha1111", model.DefaultConfig(), time.UTC, []model.Item{
		{
			Number:   &n1,
			Slug:     "search-box",
			Folder:   model.FolderReady,
			File:     ".factory/ready/FT1-search-box.md",
			Name:     "FT1-search-box.md",
			ID:       "1",
			Label:    "FT1",
			Title:    "Search box",
			Branch:   "factory/FT1-search-box",
			RawRound: "2",
			Changed:  now.Add(-1 * time.Hour),
			Sections: sections,
		},
	}, now)

	m := NewAppModel(Options{NoFetch: true}, nil, board, nil, nil)
	m.Width = 100
	m.Height = 20

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if m.Mode != ModeDetail {
		t.Fatalf("expected ModeDetail, got %v", m.Mode)
	}
	if m.DetailViewport.YOffset != 0 {
		t.Fatalf("expected initial YOffset=0, got %d", m.DetailViewport.YOffset)
	}

	// Scroll to bottom and verify pinned task header remains visible while early sections scroll away.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(AppModel)
	if m.DetailViewport.YOffset <= 0 {
		t.Fatalf("expected YOffset > 0 after scrolling to bottom, got %d", m.DetailViewport.YOffset)
	}
	view := m.View()
	if !strings.Contains(view, "[FT1 · ready]") || !strings.Contains(view, "Search box") || !strings.Contains(view, "Check, then ship") {
		t.Fatalf("expected pinned task header to stay visible while scrolled:\n%s", view)
	}
	if !strings.Contains(view, "Section L details line") {
		t.Fatalf("expected bottom section to be visible after GotoBottom:\n%s", view)
	}
	if strings.Contains(view, "Section A details line") {
		t.Fatalf("expected top section to have scrolled above viewport:\n%s", view)
	}
}
