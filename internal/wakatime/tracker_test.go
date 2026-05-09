package wakatime

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (r *fakeRunner) run(_ context.Context, _ string, args []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, append([]string(nil), args...))
	return nil
}

func (r *fakeRunner) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *fakeRunner) call(i int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls[i]...)
}

func TestEnabledFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"on", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv(EnvEnabled, tt.value)
			if got := EnabledFromEnv(); got != tt.want {
				t.Fatalf("EnabledFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActivityBuildsCLIArgs(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	runner := &fakeRunner{}
	tracker := New(Options{
		CLIPath:       "/usr/bin/wakatime-cli",
		ProjectFolder: "/repo",
		Plugin:        "monocle/test",
		Now:           func() time.Time { return now },
		Run:           runner.run,
	})

	tracker.Activity(FileTarget("/repo/file.go"))
	tracker.Wait()

	want := []string{
		"--entity", "/repo/file.go",
		"--entity-type", "file",
		"--category", "code reviewing",
		"--plugin", "monocle/test",
		"--project-folder", "/repo",
	}
	if got := runner.call(0); !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestRateLimitAndTick(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	runner := &fakeRunner{}
	tracker := New(Options{
		CLIPath:     "/usr/bin/wakatime-cli",
		Interval:    time.Minute,
		IdleTimeout: 10 * time.Minute,
		Now:         func() time.Time { return now },
		Run:         runner.run,
	})

	tracker.Activity(FileTarget("/repo/file.go"))
	tracker.Wait()
	if got := runner.count(); got != 1 {
		t.Fatalf("calls after first activity = %d, want 1", got)
	}

	now = now.Add(30 * time.Second)
	tracker.Activity(FileTarget("/repo/file.go"))
	tracker.Wait()
	if got := runner.count(); got != 1 {
		t.Fatalf("calls inside rate limit = %d, want 1", got)
	}

	now = now.Add(31 * time.Second)
	tracker.Tick(Target{})
	tracker.Wait()
	if got := runner.count(); got != 2 {
		t.Fatalf("calls after tick = %d, want 2", got)
	}
}

func TestIdleSuppressesTickAndFinalHeartbeat(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	runner := &fakeRunner{}
	tracker := New(Options{
		CLIPath:     "/usr/bin/wakatime-cli",
		Interval:    time.Minute,
		IdleTimeout: 10 * time.Minute,
		Now:         func() time.Time { return now },
		Run:         runner.run,
	})

	tracker.Activity(AppTarget("monocle artifact: plan"))
	tracker.Wait()

	now = now.Add(11 * time.Minute)
	tracker.Tick(Target{})
	tracker.Stop()

	if got := runner.count(); got != 1 {
		t.Fatalf("calls after idle tick/stop = %d, want 1", got)
	}
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv(EnvEnabled, "")
	tracker, err := NewFromEnv("/repo", "dev")
	if err != nil {
		t.Fatalf("NewFromEnv disabled returned error: %v", err)
	}
	if tracker != nil {
		t.Fatal("NewFromEnv disabled returned tracker")
	}

	t.Setenv(EnvEnabled, "1")
	t.Setenv("PATH", t.TempDir())
	tracker, err = NewFromEnv("/repo", "dev")
	if tracker != nil {
		t.Fatal("NewFromEnv missing CLI returned tracker")
	}
	if err == nil || !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("NewFromEnv missing CLI error = %v, want exec not found", err)
	}
}
