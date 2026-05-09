package wakatime

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	EnvEnabled               = "MONOCLE_WAKATIME_ENABLED"
	CategoryCodeReviewing    = "code reviewing"
	DefaultHeartbeatInterval = 2 * time.Minute
	DefaultIdleTimeout       = 10 * time.Minute

	cliName        = "wakatime-cli"
	commandTimeout = 30 * time.Second
	entityTypeApp  = "app"
	entityTypeFile = "file"
)

// Target describes the current review item to report to wakatime-cli.
type Target struct {
	Entity     string
	EntityType string
}

// FileTarget reports activity against a real file path.
func FileTarget(path string) Target {
	return Target{Entity: path, EntityType: entityTypeFile}
}

// AppTarget reports activity against a synthetic application entity.
func AppTarget(entity string) Target {
	return Target{Entity: entity, EntityType: entityTypeApp}
}

func (t Target) empty() bool {
	return strings.TrimSpace(t.Entity) == ""
}

func (t Target) normalized() Target {
	if t.EntityType == "" {
		t.EntityType = entityTypeFile
	}
	return t
}

type runFunc func(context.Context, string, []string) error

// Options configures a Tracker. Zero values use production defaults.
type Options struct {
	CLIPath       string
	ProjectFolder string
	Plugin        string
	Interval      time.Duration
	IdleTimeout   time.Duration
	Now           func() time.Time
	Run           runFunc
}

// Tracker rate-limits wakatime-cli heartbeats for active monocle TUI usage.
type Tracker struct {
	cliPath       string
	projectFolder string
	plugin        string
	interval      time.Duration
	idleTimeout   time.Duration
	now           func() time.Time
	run           runFunc

	mu           sync.Mutex
	target       Target
	lastActivity time.Time
	lastSent     time.Time
	wg           sync.WaitGroup
}

// EnabledFromEnv reports whether WakaTime tracking is enabled for this process.
func EnabledFromEnv() bool {
	return truthy(os.Getenv(EnvEnabled))
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// PluginName returns the wakatime-cli plugin identifier for this monocle build.
func PluginName(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	return "monocle/" + version
}

// NewFromEnv constructs a tracker when MONOCLE_WAKATIME_ENABLED is truthy.
func NewFromEnv(projectFolder, version string) (*Tracker, error) {
	if !EnabledFromEnv() {
		return nil, nil
	}
	cliPath, err := exec.LookPath(cliName)
	if err != nil {
		return nil, err
	}
	return New(Options{
		CLIPath:       cliPath,
		ProjectFolder: projectFolder,
		Plugin:        PluginName(version),
	}), nil
}

// New constructs a Tracker. An empty CLIPath returns a disabled tracker.
func New(opts Options) *Tracker {
	interval := opts.Interval
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	idleTimeout := opts.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = DefaultIdleTimeout
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	run := opts.Run
	if run == nil {
		run = runCommand
	}
	plugin := opts.Plugin
	if strings.TrimSpace(plugin) == "" {
		plugin = PluginName("dev")
	}

	return &Tracker{
		cliPath:       opts.CLIPath,
		projectFolder: opts.ProjectFolder,
		plugin:        plugin,
		interval:      interval,
		idleTimeout:   idleTimeout,
		now:           now,
		run:           run,
	}
}

// Activity records user activity and sends a heartbeat when the interval allows.
func (t *Tracker) Activity(target Target) {
	t.record(target, true, false)
}

// Tick sends a periodic heartbeat while recent user activity is still within the idle cutoff.
func (t *Tracker) Tick(target Target) {
	t.record(target, false, false)
}

// Stop sends one final heartbeat for a recently active review target and waits for in-flight sends.
func (t *Tracker) Stop() {
	t.record(Target{}, false, true)
	t.Wait()
}

// Wait blocks until in-flight wakatime-cli invocations finish.
func (t *Tracker) Wait() {
	if t == nil {
		return
	}
	t.wg.Wait()
}

func (t *Tracker) record(target Target, active bool, force bool) {
	if t == nil || t.cliPath == "" {
		return
	}
	now := t.now()

	t.mu.Lock()
	if !target.empty() {
		t.target = target.normalized()
	}
	if active {
		t.lastActivity = now
	}

	current := t.target
	if current.empty() || t.lastActivity.IsZero() || now.Sub(t.lastActivity) > t.idleTimeout {
		t.mu.Unlock()
		return
	}
	if !force && !t.lastSent.IsZero() && now.Sub(t.lastSent) < t.interval {
		t.mu.Unlock()
		return
	}

	t.lastSent = now
	args := t.args(current)
	t.mu.Unlock()

	t.spawn(args)
}

func (t *Tracker) args(target Target) []string {
	args := []string{
		"--entity", target.Entity,
		"--entity-type", target.EntityType,
		"--category", CategoryCodeReviewing,
		"--plugin", t.plugin,
	}
	if t.projectFolder != "" {
		args = append(args, "--project-folder", t.projectFolder)
	}
	return args
}

func (t *Tracker) spawn(args []string) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		_ = t.run(ctx, t.cliPath, args)
	}()
}

func runCommand(ctx context.Context, cliPath string, args []string) error {
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}
