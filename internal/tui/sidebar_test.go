package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/josephschmitt/monocle/internal/types"
)

func TestDefaultReviewedKeyIncludesSpace(t *testing.T) {
	keys := DefaultKeyMap()

	if !Matches("r", keys.Reviewed) {
		t.Fatal("expected reviewed action to match r")
	}
	if !Matches("space", keys.Reviewed) {
		t.Fatal("expected reviewed action to match Space")
	}
	if !Matches(" ", keys.Reviewed) {
		t.Fatal("expected reviewed action to match literal Space")
	}
	if got := Label(keys.Reviewed); got != "r/Space" {
		t.Fatalf("expected reviewed label r/Space, got %q", got)
	}
}

func TestCycleReviewFilter_TrackingEnabled(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = true

	if m.reviewFilter != "" {
		t.Fatalf("expected empty initial filter, got %q", m.reviewFilter)
	}
	m.cycleReviewFilter()
	if m.reviewFilter != "unreviewed" {
		t.Errorf("expected 'unreviewed', got %q", m.reviewFilter)
	}
	m.cycleReviewFilter()
	if m.reviewFilter != "reviewed" {
		t.Errorf("expected 'reviewed', got %q", m.reviewFilter)
	}
	m.cycleReviewFilter()
	if m.reviewFilter != "" {
		t.Errorf("expected empty after full cycle, got %q", m.reviewFilter)
	}
}

func TestCycleReviewFilter_TrackingDisabled(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = false

	m.cycleReviewFilter()
	if m.reviewFilter != "" {
		t.Errorf("expected filter to stay empty when tracking disabled, got %q", m.reviewFilter)
	}
}

func TestApplyReviewedFilter_UnreviewedOnly(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = true
	m.reviewFilter = "unreviewed"

	m.files = []types.ChangedFile{
		{Path: "a.go", Reviewed: false},
		{Path: "b.go", Reviewed: true},
		{Path: "c.go", Reviewed: false},
	}
	m.contentItems = []types.ContentItem{
		{ID: "p1", Reviewed: false},
		{ID: "p2", Reviewed: true},
	}
	m.additionalFiles = []types.AdditionalFile{
		{Path: "extra.go", Reviewed: true},
	}

	m.applyReviewedFilter()

	if len(m.files) != 2 {
		t.Errorf("expected 2 unreviewed files, got %d", len(m.files))
	}
	if len(m.contentItems) != 1 {
		t.Errorf("expected 1 unreviewed content item, got %d", len(m.contentItems))
	}
	if len(m.additionalFiles) != 0 {
		t.Errorf("expected 0 unreviewed additional files, got %d", len(m.additionalFiles))
	}
}

func TestApplyReviewedFilter_ReviewedOnly(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = true
	m.reviewFilter = "reviewed"

	m.files = []types.ChangedFile{
		{Path: "a.go", Reviewed: false},
		{Path: "b.go", Reviewed: true},
	}

	m.applyReviewedFilter()

	if len(m.files) != 1 {
		t.Errorf("expected 1 reviewed file, got %d", len(m.files))
	}
	if m.files[0].Path != "b.go" {
		t.Errorf("expected b.go, got %s", m.files[0].Path)
	}
}

func TestApplyReviewedFilter_NoFilter(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = true
	m.reviewFilter = ""

	m.files = []types.ChangedFile{
		{Path: "a.go", Reviewed: false},
		{Path: "b.go", Reviewed: true},
	}

	m.applyReviewedFilter()

	if len(m.files) != 2 {
		t.Errorf("expected all 2 files with no filter, got %d", len(m.files))
	}
}

func TestNextUnreviewed_FindsNext(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = true
	m.width = 80
	m.height = 40

	m.files = []types.ChangedFile{
		{Path: "a.go", Reviewed: true},
		{Path: "b.go", Reviewed: false},
		{Path: "c.go", Reviewed: true},
	}
	m.cursor = 0

	cmd := m.nextUnreviewed()
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}
}

func TestNextUnreviewed_NoneLeft(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = true

	m.files = []types.ChangedFile{
		{Path: "a.go", Reviewed: true},
		{Path: "b.go", Reviewed: true},
	}
	m.cursor = 0

	cmd := m.nextUnreviewed()
	if cmd != nil {
		t.Error("expected nil when all files are reviewed")
	}
}

func TestNextUnreviewed_TrackingDisabled(t *testing.T) {
	keys := DefaultKeyMap()
	m := newSidebarModel(&keys)
	m.reviewTracking = false

	m.files = []types.ChangedFile{
		{Path: "a.go", Reviewed: false},
		{Path: "b.go", Reviewed: false},
	}
	m.cursor = 0

	cmd := m.nextUnreviewed()
	if cmd != nil {
		t.Error("expected nil when tracking is disabled")
	}
}

func TestReviewedKeyNoop_TrackingDisabled(t *testing.T) {
	engine := &stubEngine{
		cfg: &types.Config{ReviewTracking: false},
		session: &types.ReviewSession{
			ID: "test",
			ChangedFiles: []types.ChangedFile{
				{Path: "a.go", Reviewed: false},
			},
		},
	}
	m := NewApp(engine)
	m.width = 120
	m.height = 40
	m.sidebar.files = engine.session.ChangedFiles

	// The r key should be a no-op
	result, _ := m.Update(keyPress('r'))
	app := result.(appModel)

	// File should remain unreviewed
	if len(app.sidebar.files) > 0 && app.sidebar.files[0].Reviewed {
		t.Error("expected file to remain unreviewed when tracking is disabled")
	}
}

func TestSpaceReviewedKeyWorksInSidebar(t *testing.T) {
	engine := &stubEngine{
		cfg: &types.Config{ReviewTracking: true},
		session: &types.ReviewSession{
			ID: "test",
			ChangedFiles: []types.ChangedFile{
				{Path: "a.go", Reviewed: false},
			},
		},
	}
	m := NewApp(engine)
	m.width = 120
	m.height = 40
	m.sidebar.files = engine.session.ChangedFiles

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd == nil {
		t.Fatal("expected Space to toggle reviewed state from the sidebar")
	}
}

func TestSpaceReviewedKeyFallsThroughInMainPane(t *testing.T) {
	engine := &stubEngine{
		cfg: &types.Config{ReviewTracking: true},
		session: &types.ReviewSession{
			ID: "test",
			ChangedFiles: []types.ChangedFile{
				{Path: "a.go", Reviewed: false},
			},
		},
	}
	m := NewApp(engine)
	m.width = 120
	m.height = 40
	m.sidebar.files = engine.session.ChangedFiles
	m.diffView.path = "a.go"
	m.focus = focusMain
	m.sidebar.focused = false
	m.diffView.focused = true

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd != nil {
		t.Fatal("expected Space to fall through to the main pane instead of toggling reviewed state")
	}
}

func TestFilterKeyNoop_TrackingDisabled(t *testing.T) {
	engine := &stubEngine{
		cfg: &types.Config{ReviewTracking: false},
	}
	m := NewApp(engine)
	m.width = 120
	m.height = 40

	// The / key should be a no-op
	result, _ := m.Update(keyPress('/'))
	app := result.(appModel)

	if app.sidebar.reviewFilter != "" {
		t.Errorf("expected filter to stay empty, got %q", app.sidebar.reviewFilter)
	}
}

func TestReviewedKeyWorks_TrackingEnabled(t *testing.T) {
	engine := &stubEngine{
		cfg: &types.Config{ReviewTracking: true},
		session: &types.ReviewSession{
			ID:           "test",
			FileStatuses: make(map[string]bool),
			ChangedFiles: []types.ChangedFile{
				{Path: "a.go", Reviewed: false},
			},
		},
	}
	m := NewApp(engine)
	m.width = 120
	m.height = 40
	m.sidebar.files = engine.session.ChangedFiles

	// The / key should cycle the filter
	result, _ := m.Update(keyPress('/'))
	app := result.(appModel)

	if app.sidebar.reviewFilter != "unreviewed" {
		t.Errorf("expected filter to be 'unreviewed', got %q", app.sidebar.reviewFilter)
	}
}
