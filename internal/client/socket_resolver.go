package client

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/josephschmitt/monocle/internal/adapters"
	"github.com/josephschmitt/monocle/internal/protocol"
)

const socketProbeTimeout = 500 * time.Millisecond

// SocketCandidate is a live monocle serve socket and the repo it owns.
type SocketCandidate struct {
	SocketPath string
	RepoRoot   string
}

// AmbiguousSocketError reports that several running nested monocle sessions
// match the requested workspace, so the caller must choose a repo explicitly.
type AmbiguousSocketError struct {
	WorkDir    string
	Candidates []SocketCandidate
}

func (e *AmbiguousSocketError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "multiple running monocle sessions found under %s:\n\n", e.WorkDir)
	for i, candidate := range e.Candidates {
		fmt.Fprintf(&b, "%d. %s", i+1, candidate.RepoRoot)
		if candidate.SocketPath != "" {
			fmt.Fprintf(&b, " (socket %s)", candidate.SocketPath)
		}
		b.WriteByte('\n')
	}
	b.WriteString("\nAsk the user which repo to use, then rerun with -C <repo>.")
	return b.String()
}

// ResolveSocketPath returns the socket path for a review command. Explicit
// socket overrides win. Otherwise the deterministic socket for workdir/CWD is
// tried first; if it is not running and the workdir is a non-git parent, a
// unique running nested monocle session is discovered automatically.
func ResolveSocketPath(socketOverride, workdir string) (string, error) {
	if socketPath := explicitSocket(socketOverride); socketPath != "" {
		return socketPath, nil
	}
	base, repoRoot, nonGitMode, err := resolveBase(workdir)
	if err != nil {
		return "", err
	}
	return resolveSocketFromBase(base, repoRoot, nonGitMode)
}

// ResolveSocketPathFromRoots resolves a socket from an MCP client's file roots.
// Exact live root matches win before nested discovery. If no root is usable it
// falls back to the current working directory behavior.
func ResolveSocketPathFromRoots(socketOverride string, roots []string) (string, error) {
	if socketPath := explicitSocket(socketOverride); socketPath != "" {
		return socketPath, nil
	}

	var firstDefault string
	var candidates []SocketCandidate
	var candidateBase string
	for _, root := range roots {
		base, repoRoot, nonGitMode, err := resolveBase(root)
		if err != nil {
			continue
		}

		defaultSocket := adapters.DefaultSocketPath(repoRoot)
		if firstDefault == "" {
			firstDefault = defaultSocket
		}
		if IsSocketReachable(defaultSocket) {
			return defaultSocket, nil
		}
		if !nonGitMode {
			continue
		}

		rootCandidates := discoverSocketCandidates(base)
		if len(rootCandidates) == 0 {
			continue
		}
		if candidateBase == "" {
			candidateBase = base
		}
		candidates = append(candidates, rootCandidates...)
	}

	candidates = uniqueCandidates(candidates)
	if len(candidates) == 1 {
		return candidates[0].SocketPath, nil
	}
	if len(candidates) > 1 {
		return "", &AmbiguousSocketError{WorkDir: candidateBase, Candidates: candidates}
	}
	if firstDefault != "" {
		return firstDefault, nil
	}
	return ResolveSocketPath("", "")
}

// IsSocketReachable reports whether a Unix socket exists and accepts a local
// connection. It does not validate that the peer speaks the monocle protocol.
func IsSocketReachable(socketPath string) bool {
	if socketPath == "" {
		return false
	}
	if _, err := os.Stat(socketPath); errors.Is(err, os.ErrNotExist) {
		return false
	}
	conn, err := net.DialTimeout("unix", socketPath, 250*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func explicitSocket(socketOverride string) string {
	if socketOverride != "" {
		return socketOverride
	}
	return os.Getenv("MONOCLE_SOCKET")
}

func resolveBase(workdir string) (base string, repoRoot string, nonGitMode bool, err error) {
	if workdir == "" {
		workdir, err = os.Getwd()
		if err != nil {
			return "", "", false, fmt.Errorf("get cwd: %w", err)
		}
	}
	abs, err := filepath.Abs(workdir)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve workdir: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", "", false, fmt.Errorf("--workdir: %w", err)
	}
	if !info.IsDir() {
		return "", "", false, fmt.Errorf("--workdir: %s is not a directory", abs)
	}

	repoRoot = adapters.FindRepoRoot(abs)
	_, statErr := os.Stat(filepath.Join(repoRoot, ".git"))
	return cleanPath(abs), cleanPath(repoRoot), statErr != nil, nil
}

func resolveSocketFromBase(base, repoRoot string, nonGitMode bool) (string, error) {
	defaultSocket := adapters.DefaultSocketPath(repoRoot)
	if IsSocketReachable(defaultSocket) {
		return defaultSocket, nil
	}
	if !nonGitMode {
		return defaultSocket, nil
	}

	candidates := discoverSocketCandidates(base)
	if len(candidates) == 0 {
		return defaultSocket, nil
	}
	if len(candidates) == 1 {
		return candidates[0].SocketPath, nil
	}
	return "", &AmbiguousSocketError{WorkDir: base, Candidates: candidates}
}

func discoverSocketCandidates(base string) []SocketCandidate {
	paths, err := filepath.Glob("/tmp/monocle-*.sock")
	if err != nil {
		return nil
	}
	var candidates []SocketCandidate
	for _, path := range paths {
		repoRoot, ok := querySocketRepoRoot(path)
		if !ok {
			continue
		}
		if !sameOrDescendant(repoRoot, base) {
			continue
		}
		candidates = append(candidates, SocketCandidate{SocketPath: path, RepoRoot: repoRoot})
	}
	return uniqueCandidates(candidates)
}

func querySocketRepoRoot(socketPath string) (string, bool) {
	c, err := Connect(socketPath)
	if err != nil {
		return "", false
	}
	defer c.Close()

	resp, err := c.Request(&protocol.GetRepoInfoMsg{Type: protocol.TypeGetRepoInfo}, socketProbeTimeout)
	if err != nil {
		return "", false
	}
	info, ok := resp.(*protocol.GetRepoInfoResponse)
	if !ok || info.Info.Root == "" {
		return "", false
	}
	return cleanPath(info.Info.Root), true
}

func sameOrDescendant(path, base string) bool {
	rel, err := filepath.Rel(cleanPath(base), cleanPath(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func uniqueCandidates(candidates []SocketCandidate) []SocketCandidate {
	seen := make(map[string]bool, len(candidates))
	unique := make([]SocketCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.SocketPath + "\x00" + candidate.RepoRoot
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, candidate)
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].RepoRoot == unique[j].RepoRoot {
			return unique[i].SocketPath < unique[j].SocketPath
		}
		return unique[i].RepoRoot < unique[j].RepoRoot
	})
	return unique
}

func cleanPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}
