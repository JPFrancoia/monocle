package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/josephschmitt/monocle/internal/types"
)

// TestCLIParsesWithoutIdleTimeoutFlag guards against a regression where
// ServeCmd.IdleTimeout had a default:"" tag that Kong rejected as an
// invalid duration during CLI setup — breaking every subcommand, not
// just `monocle serve`, because Kong validates all defaults upfront.
func TestCLIParsesWithoutIdleTimeoutFlag(t *testing.T) {
	// Building the parser exercises default-tag validation on every
	// field. If ServeCmd's --idle-timeout regressed to an invalid default
	// this call would fail.
	var cli CLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatalf("kong setup failed (likely a bad default on some flag): %v", err)
	}
	if _, err := parser.Parse([]string{"hooks", "on-stop", "--agent", "claude"}); err != nil {
		t.Fatalf("parse hooks on-stop: %v", err)
	}
}

func TestCLIParsesNewSessionFlag(t *testing.T) {
	var cli CLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatalf("kong setup failed: %v", err)
	}
	if _, err := parser.Parse([]string{"--new"}); err != nil {
		t.Fatalf("parse --new: %v", err)
	}
	if !cli.Run.New {
		t.Fatal("expected --new to set Run.New")
	}

	cli = CLI{}
	parser, err = kong.New(&cli)
	if err != nil {
		t.Fatalf("kong setup failed: %v", err)
	}
	if _, err := parser.Parse([]string{"-n"}); err != nil {
		t.Fatalf("parse -n: %v", err)
	}
	if !cli.Run.New {
		t.Fatal("expected -n to set Run.New")
	}
}

func TestCLIRejectsConflictingSessionFlags(t *testing.T) {
	var cli CLI
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatalf("kong setup failed: %v", err)
	}
	if _, err := parser.Parse([]string{"--new", "--continue"}); err == nil {
		t.Fatal("expected --new and --continue to conflict")
	}
}

func TestPidFilePath(t *testing.T) {
	cases := []struct {
		socket string
		want   string
	}{
		{"/tmp/monocle-abc123.sock", "/tmp/monocle-abc123.pid"},
		{"/tmp/custom", "/tmp/custom.pid"},
	}
	for _, tc := range cases {
		got := pidFilePath(tc.socket)
		if got != tc.want {
			t.Errorf("pidFilePath(%q) = %q, want %q", tc.socket, got, tc.want)
		}
	}
}

func TestWriteReadPIDFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pid")

	if err := writePIDFile(path); err != nil {
		t.Fatalf("write: %v", err)
	}

	pid, err := readPIDFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if pid != os.Getpid() {
		t.Errorf("pid = %d, want %d", pid, os.Getpid())
	}

	removePIDFile(path)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("pid file still exists after remove: %v", err)
	}
}

func TestStopServeBySocketNoPIDFile(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "missing.sock")

	stopped, err := stopServeBySocket(socketPath, 0)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if stopped {
		t.Fatal("expected no server to stop")
	}
}

func TestResolveServeIdleTimeout(t *testing.T) {
	cfgIdle := 30 * time.Minute
	flagIdle := 5 * time.Minute

	cases := []struct {
		name string
		cfg  *types.Config
		flag time.Duration
		want time.Duration
	}{
		{
			name: "default disabled",
			cfg:  &types.Config{},
			want: 0,
		},
		{
			name: "config enables idle shutdown",
			cfg:  &types.Config{IdleTimeout: types.Duration(cfgIdle)},
			want: cfgIdle,
		},
		{
			name: "flag overrides config",
			cfg:  &types.Config{IdleTimeout: types.Duration(cfgIdle)},
			flag: flagIdle,
			want: flagIdle,
		},
		{
			name: "negative flag disables",
			cfg:  &types.Config{IdleTimeout: types.Duration(cfgIdle)},
			flag: -1,
			want: -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveServeIdleTimeout(tc.cfg, tc.flag)
			if got != tc.want {
				t.Errorf("resolveServeIdleTimeout() = %s, want %s", got, tc.want)
			}
		})
	}
}
