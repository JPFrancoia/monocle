package tui

import (
	"strings"
	"testing"
)

func TestStatusBarShowsAgentLabelWithUnknownFallback(t *testing.T) {
	m := newStatusBarModel(DefaultTheme())
	m.width = 120
	m.socketStarted = true
	m.subscriberCount = 1
	m.agentName = ""

	view := m.View()
	if !strings.Contains(view, "● Connected") {
		t.Fatalf("expected connected status, got %q", view)
	}
	if !strings.Contains(view, "Agent: unknown") {
		t.Fatalf("expected unknown agent label, got %q", view)
	}
	if strings.Contains(view, "Connected claude") {
		t.Fatalf("status should not append agent name to connection label: %q", view)
	}
}

func TestStatusBarShowsIdentifiedAgentLabel(t *testing.T) {
	m := newStatusBarModel(DefaultTheme())
	m.width = 120
	m.socketStarted = true
	m.connectionMode = "queue"
	m.agentName = "opencode"
	m.agentIdentified = true

	view := m.View()
	if !strings.Contains(view, "● Connected") {
		t.Fatalf("expected connected status, got %q", view)
	}
	if !strings.Contains(view, "Agent: opencode") {
		t.Fatalf("expected identified agent label, got %q", view)
	}
}

func TestStatusBarTreatsUnidentifiedSessionAgentAsUnknown(t *testing.T) {
	m := newStatusBarModel(DefaultTheme())
	m.width = 120
	m.socketStarted = true
	m.subscriberCount = 1
	m.agentName = "opencode"

	view := m.View()
	if !strings.Contains(view, "Agent: unknown") {
		t.Fatalf("expected unidentified session agent to display as unknown, got %q", view)
	}
	if strings.Contains(view, "Agent: opencode") {
		t.Fatalf("should not display persisted session agent as live agent: %q", view)
	}
}

func TestStatusBarShowsLiveClaudeAgentWhenIdentified(t *testing.T) {
	m := newStatusBarModel(DefaultTheme())
	m.width = 120
	m.socketStarted = true
	m.subscriberCount = 1
	m.agentName = "claude"
	m.agentIdentified = true

	view := m.View()
	if !strings.Contains(view, "Agent: claude") {
		t.Fatalf("expected live identified claude agent label, got %q", view)
	}
}
