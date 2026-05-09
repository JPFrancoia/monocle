package adapters

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed commands.json
var commandsJSON []byte

// CommandDef describes a slash command that wraps an MCP tool call.
type CommandDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

var (
	commandsOnce sync.Once
	commandDefs  []CommandDef
)

// loadCommands returns the embedded command definitions, parsing once.
func loadCommands() []CommandDef {
	commandsOnce.Do(func() {
		if err := json.Unmarshal(commandsJSON, &commandDefs); err != nil {
			panic(fmt.Sprintf("parse embedded commands.json: %v", err))
		}
	})
	return commandDefs
}

// CommandNames returns the names of all defined commands.
func CommandNames() []string {
	defs := loadCommands()
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	return names
}

// InstallMarkdownCommands writes command files in markdown format (Claude Code, OpenCode).
// Each command becomes dir/<name>.md with YAML frontmatter.
func InstallMarkdownCommands(dir string) error {
	defs := loadCommands()
	for _, cmd := range defs {
		content := fmt.Sprintf("---\ndescription: %s\n---\n\n%s\n", cmd.Description, cmd.Body)
		dest := filepath.Join(dir, cmd.Name+".md")
		if err := WriteFileAtomic(dest, []byte(content)); err != nil {
			return fmt.Errorf("write command %s: %w", cmd.Name, err)
		}
	}
	return nil
}

// InstallCLIMarkdownCommands writes command files that instruct agents to use
// the monocle review CLI. This is used by skills-mode integrations where MCP
// tools are intentionally not registered.
func InstallCLIMarkdownCommands(dir string) error {
	defs := loadCommands()
	for _, cmd := range defs {
		body := cliCommandBody(cmd)
		content := fmt.Sprintf("---\ndescription: %s\n---\n\n%s\n", cmd.Description, body)
		dest := filepath.Join(dir, cmd.Name+".md")
		if err := WriteFileAtomic(dest, []byte(content)); err != nil {
			return fmt.Errorf("write command %s: %w", cmd.Name, err)
		}
	}
	return nil
}

func cliCommandBody(cmd CommandDef) string {
	switch cmd.Name {
	case "get-feedback":
		return "Run `monocle review get-feedback` to retrieve pending review feedback from your reviewer.\n\n" +
			"- If feedback is available, read it carefully and act on it; the feedback contains your reviewer's comments, issues, and suggestions about your code changes\n" +
			"- If no feedback is pending, inform the user that no review feedback is available yet\n" +
			"- If Monocle reports multiple running sessions, ask the user which listed repo to use, then rerun with `-C <chosen repo>`\n\n" +
			"After receiving feedback, address the reviewer's comments in your code, then continue with your work."
	case "get-feedback-wait":
		return "Run `monocle review get-feedback --wait` to block until your reviewer submits feedback through Monocle.\n\n" +
			"## Handling the response\n\n" +
			"- Read the feedback carefully and act on it; the feedback contains your reviewer's comments, issues, and suggestions about your code changes\n" +
			"- Address the reviewer's comments in your code\n" +
			"- If Monocle reports multiple running sessions, ask the user which listed repo to use, then rerun with `-C <chosen repo>`\n" +
			"- If the reviewer requested changes, run `monocle review get-feedback --wait` again after addressing the feedback\n" +
			"- Keep iterating until the reviewer approves"
	case "review-plan":
		return "Submit a plan file to Monocle so the reviewer can see it. Does NOT wait for feedback; use `/review-plan-wait` to block until the reviewer responds.\n\n" +
			"**Important:** This is for content that isn't already a tracked file change: plans, architecture docs, summaries, etc. You do NOT need to send regular code files; Monocle automatically picks up file changes.\n\n" +
			"## Steps\n\n" +
			"1. **Find the plan file**: if the user provided a path as an argument, use that. Otherwise, find the most recently modified plan file in the project.\n\n" +
			"2. **Read the plan file** to confirm it exists and get its filename.\n\n" +
			"3. **Run `monocle review send-artifact`** with:\n" +
			"   - `--title`: The first markdown heading from the plan, or the filename if no heading found\n" +
			"   - `--file`: Absolute path to the plan file\n" +
			"   - `--id`: The plan filename (e.g. `my-plan.md`), which ensures updates replace the previous version\n" +
			"   - `--type`: `md`\n\n" +
			"4. If Monocle reports multiple running sessions, ask the user which listed repo to use, then rerun with `-C <chosen repo>`.\n\n" +
			"5. **Confirm** to the user that the plan was sent to Monocle."
	case "review-plan-wait":
		return "Submit a plan file to Monocle and block until the reviewer responds with feedback.\n\n" +
			"**Important:** This is for content that isn't already a tracked file change: plans, architecture docs, summaries, etc. You do NOT need to send regular code files; Monocle automatically picks up file changes.\n\n" +
			"## Steps\n\n" +
			"1. **Find the plan file**: if the user provided a path as an argument, use that. Otherwise, find the most recently modified plan file in the project.\n\n" +
			"2. **Read the plan file** to confirm it exists and get its filename.\n\n" +
			"3. **Run `monocle review send-artifact --wait`** with:\n" +
			"   - `--title`: The first markdown heading from the plan, or the filename if no heading found\n" +
			"   - `--file`: Absolute path to the plan file\n" +
			"   - `--id`: The plan filename (e.g. `my-plan.md`), which ensures updates replace the previous version\n" +
			"   - `--type`: `md`\n\n" +
			"4. If Monocle reports multiple running sessions, ask the user which listed repo to use, then rerun with `-C <chosen repo>`.\n\n" +
			"5. **Handle the response:**\n" +
			"   - If the reviewer approved with no comments, inform the user and continue\n" +
			"   - If the reviewer provided feedback requesting changes, share the feedback with the user and act on it; update the plan, then repeat from step 3\n" +
			"   - Keep iterating until the reviewer approves"
	default:
		return cmd.Body
	}
}

// InstallTOMLCommands writes command files in TOML format (Gemini CLI).
// Each command becomes dir/<name>.toml with description and prompt fields.
func InstallTOMLCommands(dir string) error {
	defs := loadCommands()
	for _, cmd := range defs {
		// Escape any triple quotes in body for TOML multi-line strings
		body := strings.ReplaceAll(cmd.Body, `"""`, `\"\"\"`)
		content := fmt.Sprintf("description = %q\nprompt = \"\"\"\n%s\n\"\"\"\n", cmd.Description, body)
		dest := filepath.Join(dir, cmd.Name+".toml")
		if err := WriteFileAtomic(dest, []byte(content)); err != nil {
			return fmt.Errorf("write command %s: %w", cmd.Name, err)
		}
	}
	return nil
}

// RemoveCommands removes installed command files with the given extension.
func RemoveCommands(dir string, ext string) {
	defs := loadCommands()
	for _, cmd := range defs {
		_ = RemoveFileIfExists(filepath.Join(dir, cmd.Name+ext))
	}
	// Remove dir if empty
	_ = os.Remove(dir)
}

// CommandPaths returns the paths of command files that would be installed.
func CommandPaths(dir string, ext string) []string {
	defs := loadCommands()
	paths := make([]string, len(defs))
	for i, d := range defs {
		paths[i] = filepath.Join(dir, d.Name+ext)
	}
	return paths
}
