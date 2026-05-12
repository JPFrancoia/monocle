package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestDefaultNavigationKeysIncludeNormalKeys verifies that common terminal
// navigation keys map to monocle's existing navigation actions.
func TestDefaultNavigationKeysIncludeNormalKeys(t *testing.T) {
	keys := DefaultKeyMap()
	tests := []struct {
		name     string
		msg      tea.KeyPressMsg
		bindings []string
	}{
		{name: "home jumps to top", msg: tea.KeyPressMsg{Code: tea.KeyHome}, bindings: keys.Top},
		{name: "end jumps to bottom", msg: tea.KeyPressMsg{Code: tea.KeyEnd}, bindings: keys.Bottom},
		{name: "page up scrolls up", msg: tea.KeyPressMsg{Code: tea.KeyPgUp}, bindings: keys.HalfUp},
		{name: "page down scrolls down", msg: tea.KeyPressMsg{Code: tea.KeyPgDown}, bindings: keys.HalfDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !Matches(tt.msg.String(), tt.bindings) {
				t.Fatalf("expected %q to match %v", tt.msg.String(), tt.bindings)
			}
		})
	}
}

// TestLabelIncludesAllUniqueBindings verifies help labels show every alias
// without repeating keys that render to the same display label.
func TestLabelIncludesAllUniqueBindings(t *testing.T) {
	got := Label([]string{"ctrl+d", "pgdown", "space", " "})
	want := "ctrl+d/PageDown/Space"
	if got != want {
		t.Fatalf("Label() = %q, want %q", got, want)
	}
}
