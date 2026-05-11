package tui

import "testing"

func TestHandlePasteCommentEditor(t *testing.T) {
	m := NewApp(nil)
	m.overlay = overlayComment
	m.commentEditor.active = true
	m.commentEditor.body = "ab"
	m.commentEditor.cursor = 1

	model, _ := m.handlePaste("XYZ")
	app := model.(appModel)

	if app.commentEditor.body != "aXYZb" {
		t.Fatalf("comment body = %q, want %q", app.commentEditor.body, "aXYZb")
	}
	if app.commentEditor.cursor != 4 {
		t.Fatalf("cursor = %d, want 4", app.commentEditor.cursor)
	}
}

func TestHandlePasteReviewSummary(t *testing.T) {
	m := NewApp(nil)
	m.overlay = overlayReview
	m.reviewSummary.active = true
	m.reviewSummary.body = "review"

	model, _ := m.handlePaste(" body")
	app := model.(appModel)

	if app.reviewSummary.body != "review body" {
		t.Fatalf("review body = %q, want %q", app.reviewSummary.body, "review body")
	}
}

func TestHandlePasteCommandMode(t *testing.T) {
	m := NewApp(nil)
	m.commandMode = true
	m.commandBuffer = "sub"
	m.statusBar.commandBuffer = "sub"

	model, _ := m.handlePaste("mit")
	app := model.(appModel)

	if app.commandBuffer != "submit" {
		t.Fatalf("commandBuffer = %q, want %q", app.commandBuffer, "submit")
	}
	if app.statusBar.commandBuffer != "submit" {
		t.Fatalf("status commandBuffer = %q, want %q", app.statusBar.commandBuffer, "submit")
	}
}
