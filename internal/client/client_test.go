package client_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/josephschmitt/monocle/internal/adapters"
	"github.com/josephschmitt/monocle/internal/client"
	"github.com/josephschmitt/monocle/internal/core"
	"github.com/josephschmitt/monocle/internal/db"
	"github.com/josephschmitt/monocle/internal/protocol"
)

func setupTestEngine(t *testing.T) (*core.Engine, string) {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	tmpDir := t.TempDir()
	cfg := core.DefaultConfig()
	engine, err := core.NewEngine(cfg, database, tmpDir, true /* nonGitMode */)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	_, err = engine.StartSession(core.SessionOptions{
		Agent:    "test",
		RepoRoot: tmpDir,
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	hash := sha256.Sum256([]byte(t.Name()))
	socketPath := fmt.Sprintf("/tmp/monocle-test-%s.sock", hex.EncodeToString(hash[:])[:8])
	if err := engine.StartServer(socketPath); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() { engine.Shutdown() })

	return engine, socketPath
}

func setupTestEngineAt(t *testing.T, repoRoot, socketPath string) *core.Engine {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := core.DefaultConfig()
	engine, err := core.NewEngine(cfg, database, repoRoot, true /* nonGitMode */)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	_, err = engine.StartSession(core.SessionOptions{
		Agent:    "test",
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	_ = os.Remove(socketPath)
	if err := engine.StartServer(socketPath); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() { engine.Shutdown() })

	return engine
}

func TestClient_LargeDiffResponse(t *testing.T) {
	engine, socketPath := setupTestEngine(t)
	repoRoot := engine.GetSession().RepoRoot
	content := bytes.Repeat([]byte("x"), 2*1024*1024)
	if err := os.WriteFile(filepath.Join(repoRoot, "large.txt"), content, 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}

	c, err := client.Connect(socketPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	resp, err := c.Request(&protocol.GetFileDiffMsg{Type: protocol.TypeGetFileDiff, Path: "large.txt"}, client.DefaultTimeout)
	if err != nil {
		t.Fatalf("large diff request: %v", err)
	}
	diff := resp.(*protocol.GetFileDiffResponse)
	if diff.Error != "" {
		t.Fatalf("large diff error: %s", diff.Error)
	}
	if len(diff.Diff.Hunks) != 1 || len(diff.Diff.Hunks[0].Lines) != 1 {
		t.Fatalf("unexpected large diff shape: %#v", diff.Diff.Hunks)
	}
	if got := len(diff.Diff.Hunks[0].Lines[0].Content); got != len(content) {
		t.Fatalf("large diff content length = %d, want %d", got, len(content))
	}
}

func TestClient_ReviewStatus(t *testing.T) {
	_, socketPath := setupTestEngine(t)

	c, err := client.Connect(socketPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	msg := &protocol.GetReviewStatusMsg{Type: protocol.TypeGetReviewStatus}
	resp, err := c.Request(msg, client.DefaultTimeout)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	status, ok := resp.(*protocol.GetReviewStatusResponse)
	if !ok {
		t.Fatalf("expected *GetReviewStatusResponse, got %T", resp)
	}
	if status.Status != "no_feedback" {
		t.Errorf("status = %q, want %q", status.Status, "no_feedback")
	}
}

func TestClient_PollFeedback_NoWait(t *testing.T) {
	_, socketPath := setupTestEngine(t)

	c, err := client.Connect(socketPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	msg := &protocol.PollFeedbackMsg{Type: protocol.TypePollFeedback, Wait: false}
	resp, err := c.Request(msg, client.DefaultTimeout)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	feedback, ok := resp.(*protocol.PollFeedbackResponse)
	if !ok {
		t.Fatalf("expected *PollFeedbackResponse, got %T", resp)
	}
	if feedback.HasFeedback {
		t.Error("expected no feedback")
	}
}

func TestClient_SubmitContent(t *testing.T) {
	_, socketPath := setupTestEngine(t)

	c, err := client.Connect(socketPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	msg := &protocol.SubmitContentMsg{
		Type:        protocol.TypeSubmitContent,
		ID:          "test-plan",
		Title:       "Test Plan",
		Content:     "# My Plan\n\nDo the thing.",
		ContentType: "md",
		IsPlan:      true,
	}
	resp, err := c.Request(msg, client.DefaultTimeout)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	submit, ok := resp.(*protocol.SubmitContentResponse)
	if !ok {
		t.Fatalf("expected *SubmitContentResponse, got %T", resp)
	}
	if !submit.Success {
		t.Errorf("expected success, got message: %s", submit.Message)
	}
}

func TestClient_AddFiles(t *testing.T) {
	_, socketPath := setupTestEngine(t)

	c, err := client.Connect(socketPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	msg := &protocol.AddAdditionalFilesMsg{
		Type:  protocol.TypeAddAdditionalFiles,
		Paths: []string{t.TempDir()},
	}
	resp, err := c.Request(msg, client.DefaultTimeout)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	add, ok := resp.(*protocol.AddAdditionalFilesResponse)
	if !ok {
		t.Fatalf("expected *AddAdditionalFilesResponse, got %T", resp)
	}
	if !add.Success {
		t.Errorf("expected success, got message: %s", add.Message)
	}
}

func TestClient_ErrNotRunning(t *testing.T) {
	_, err := client.Connect("/tmp/monocle-does-not-exist.sock")
	if err != client.ErrNotRunning {
		t.Errorf("expected ErrNotRunning, got %v", err)
	}
}

func TestResolveSocketPath_ExplicitOverrideWins(t *testing.T) {
	parent := t.TempDir()
	t.Setenv("MONOCLE_SOCKET", "/tmp/from-env.sock")

	got, err := client.ResolveSocketPath("/tmp/from-flag.sock", parent)
	if err != nil {
		t.Fatalf("resolve socket: %v", err)
	}
	if got != "/tmp/from-flag.sock" {
		t.Fatalf("socket = %q, want explicit flag", got)
	}
}

func TestResolveSocketPath_DiscoversUniqueNestedSession(t *testing.T) {
	t.Setenv("MONOCLE_SOCKET", "")
	parent := t.TempDir()
	repo := filepath.Join(parent, "nested-app")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	socketPath := adapters.DefaultSocketPath(repo)
	setupTestEngineAt(t, repo, socketPath)

	got, err := client.ResolveSocketPath("", parent)
	if err != nil {
		t.Fatalf("resolve socket: %v", err)
	}
	if got != socketPath {
		t.Fatalf("socket = %q, want nested repo socket %q", got, socketPath)
	}
}

func TestResolveSocketPath_AmbiguousNestedSessions(t *testing.T) {
	t.Setenv("MONOCLE_SOCKET", "")
	parent := t.TempDir()
	repoA := filepath.Join(parent, "app")
	repoB := filepath.Join(parent, "infra")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("mkdir repo A: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("mkdir repo B: %v", err)
	}
	setupTestEngineAt(t, repoA, adapters.DefaultSocketPath(repoA))
	setupTestEngineAt(t, repoB, adapters.DefaultSocketPath(repoB))

	_, err := client.ResolveSocketPath("", parent)
	if err == nil {
		t.Fatal("expected ambiguity error, got nil")
	}
	var ambiguous *client.AmbiguousSocketError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected AmbiguousSocketError, got %T: %v", err, err)
	}
	if len(ambiguous.Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(ambiguous.Candidates))
	}
	if ambiguous.Candidates[0].RepoRoot != repoA || ambiguous.Candidates[1].RepoRoot != repoB {
		t.Fatalf("candidate roots = %#v, want %q and %q", ambiguous.Candidates, repoA, repoB)
	}
	if got := ambiguous.Error(); !bytes.Contains([]byte(got), []byte("rerun with -C <repo>")) {
		t.Fatalf("ambiguity error should tell agent how to retry, got: %s", got)
	}
}

func TestResolveSocketPath_IgnoresStaleNestedSocket(t *testing.T) {
	t.Setenv("MONOCLE_SOCKET", "")
	parent := t.TempDir()
	repo := filepath.Join(parent, "app")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	staleSocket := adapters.DefaultSocketPath(repo)
	_ = os.Remove(staleSocket)
	if err := os.WriteFile(staleSocket, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale socket: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(staleSocket) })

	got, err := client.ResolveSocketPath("", parent)
	if err != nil {
		t.Fatalf("resolve socket: %v", err)
	}
	want := adapters.DefaultSocketPath(parent)
	if got != want {
		t.Fatalf("socket = %q, want parent default %q", got, want)
	}
}

func TestResolveSocketPath_ExactLiveParentWins(t *testing.T) {
	t.Setenv("MONOCLE_SOCKET", "")
	parent := t.TempDir()
	repo := filepath.Join(parent, "app")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	parentSocket := adapters.DefaultSocketPath(parent)
	setupTestEngineAt(t, parent, parentSocket)
	setupTestEngineAt(t, repo, adapters.DefaultSocketPath(repo))

	got, err := client.ResolveSocketPath("", parent)
	if err != nil {
		t.Fatalf("resolve socket: %v", err)
	}
	if got != parentSocket {
		t.Fatalf("socket = %q, want exact parent socket %q", got, parentSocket)
	}
}
