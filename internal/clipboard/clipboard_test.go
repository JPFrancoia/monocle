package clipboard

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOSC52Sequence(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("STY", "")

	encoded := base64.StdEncoding.EncodeToString([]byte("copy me"))
	want := fmt.Sprintf("\x1b]52;c;%s\x07", encoded)
	if got := osc52Sequence("copy me"); got != want {
		t.Fatalf("osc52Sequence() = %q, want %q", got, want)
	}
}

func TestOSC52SequenceTmuxPassthrough(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,123,0")
	t.Setenv("STY", "")

	encoded := base64.StdEncoding.EncodeToString([]byte("copy me"))
	inner := fmt.Sprintf("\x1b]52;c;%s\x07", encoded)
	want := fmt.Sprintf("\x1bPtmux;\x1b%s\x1b\\", inner)
	if got := osc52Sequence("copy me"); got != want {
		t.Fatalf("osc52Sequence() = %q, want %q", got, want)
	}
}

func TestCopySystemPrefersWlCopyOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux clipboard command selection only")
	}

	dir := t.TempDir()
	record := filepath.Join(dir, "record")
	for _, name := range []string{"wl-copy", "xclip", "xsel"} {
		path := filepath.Join(dir, name)
		script := fmt.Sprintf("#!/bin/sh\nprintf %%s %q > %q\nwhile IFS= read -r _; do :; done\nexit 0\n", name, record)
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatalf("write fake %s: %v", name, err)
		}
	}
	t.Setenv("PATH", dir)

	if err := copySystem("copied text"); err != nil {
		t.Fatalf("copySystem() error = %v", err)
	}
	gotBytes, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if got := strings.TrimSpace(string(gotBytes)); got != "wl-copy" {
		t.Fatalf("clipboard command = %q, want wl-copy", got)
	}
}
