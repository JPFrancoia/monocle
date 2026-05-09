package tui

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/josephschmitt/monocle/internal/types"
	"github.com/josephschmitt/monocle/internal/wakatime"
)

type tuiWakaRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (r *tuiWakaRunner) run(_ context.Context, _ string, args []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, append([]string(nil), args...))
	return nil
}

func (r *tuiWakaRunner) call(i int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls[i]...)
}

func TestWakaTimeTargetForSidebarSelect(t *testing.T) {
	m := NewApp(nil, AppOptions{RepoRoot: "/repo"})
	m.sidebar.contentItems = []types.ContentItem{{ID: "plan-1", Title: "Review Plan"}}

	tests := []struct {
		name string
		msg  sidebarSelectMsg
		want wakatime.Target
	}{
		{
			name: "repo file",
			msg:  sidebarSelectMsg{path: "internal/app.go"},
			want: wakatime.FileTarget("/repo/internal/app.go"),
		},
		{
			name: "additional file",
			msg:  sidebarSelectMsg{path: "/tmp/notes.md", isAdditionalFile: true},
			want: wakatime.FileTarget("/tmp/notes.md"),
		},
		{
			name: "artifact",
			msg:  sidebarSelectMsg{contentID: "plan-1", isContent: true},
			want: wakatime.AppTarget("monocle artifact: Review Plan"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.wakaTimeTargetForSidebarSelect(tt.msg); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("target = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestHandleSidebarSelectReportsWakaTimeActivity(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	runner := &tuiWakaRunner{}
	tracker := wakatime.New(wakatime.Options{
		CLIPath:       "/usr/bin/wakatime-cli",
		ProjectFolder: "/repo",
		Plugin:        "monocle/test",
		Now:           func() time.Time { return now },
		Run:           runner.run,
	})
	m := NewApp(&stubEngine{cfg: &types.Config{}}, AppOptions{
		RepoRoot:        "/repo",
		WakaTimeTracker: tracker,
	})

	_ = m.handleSidebarSelect(sidebarSelectMsg{path: "internal/app.go"})
	tracker.Wait()

	want := []string{
		"--entity", "/repo/internal/app.go",
		"--entity-type", "file",
		"--category", "code reviewing",
		"--heartbeat-rate-limit-seconds", "0",
		"--sync-ai-disabled",
		"--plugin", "monocle/test",
		"--project-folder", "/repo",
	}
	if got := runner.call(0); !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}
