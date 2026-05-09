package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeRegister_AddsPermissions(t *testing.T) {
	setupTestSkills(t)
	dir := t.TempDir()
	projDir := filepath.Join(dir, "project")
	os.MkdirAll(projDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	adapter := &OpenCodeAdapter{}
	if err := adapter.Register(false); err != nil {
		t.Fatalf("register: %v", err)
	}

	data, err := ReadJSONFile(filepath.Join(projDir, "opencode.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	perm, ok := data["permission"].(map[string]any)
	if !ok {
		t.Fatal("permission key should exist")
	}
	bash, ok := perm["bash"].(map[string]any)
	if !ok {
		t.Fatal("permission.bash should exist")
	}
	if bash["monocle *"] != "allow" {
		t.Errorf("expected monocle * = allow, got %v", bash["monocle *"])
	}
}

func TestOpenCodeRegister_SkillsModeInstallsSkillsAndCommands(t *testing.T) {
	setupTestSkills(t)
	dir := t.TempDir()
	projDir := filepath.Join(dir, "project")
	os.MkdirAll(projDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	adapter := &OpenCodeAdapter{}
	if err := adapter.Register(false); err != nil {
		t.Fatalf("register: %v", err)
	}

	skillPath := filepath.Join(projDir, ".opencode", "skills", "get-feedback", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill should be installed: %v", err)
	}

	commandPath := filepath.Join(projDir, ".opencode", "commands", "get-feedback.md")
	data, err := os.ReadFile(commandPath)
	if err != nil {
		t.Fatalf("command should be installed: %v", err)
	}
	if !strings.Contains(string(data), "monocle review get-feedback") {
		t.Fatalf("command should use CLI feedback command, got:\n%s", string(data))
	}
}

func TestOpenCodeConfigPaths_SkillsModeIncludesSkillsAndCommands(t *testing.T) {
	adapter := &OpenCodeAdapter{}
	paths := adapter.ConfigPaths(false)

	wantSkill := filepath.Join(".opencode", "skills", "get-feedback", "SKILL.md")
	wantCommand := filepath.Join(".opencode", "commands", "get-feedback.md")
	if !containsPath(paths, wantSkill) {
		t.Fatalf("ConfigPaths missing skill path %q in %#v", wantSkill, paths)
	}
	if !containsPath(paths, wantCommand) {
		t.Fatalf("ConfigPaths missing command path %q in %#v", wantCommand, paths)
	}
}

func TestOpenCodeRegister_MCPModeKeepsMCPCommandBodies(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "project")
	os.MkdirAll(projDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	adapter := &OpenCodeAdapter{Mode: ModeMCPTools}
	if err := adapter.Register(false); err != nil {
		t.Fatalf("register: %v", err)
	}

	commandPath := filepath.Join(projDir, ".opencode", "commands", "get-feedback.md")
	data, err := os.ReadFile(commandPath)
	if err != nil {
		t.Fatalf("command should be installed: %v", err)
	}
	if !strings.Contains(string(data), "get_feedback") {
		t.Fatalf("MCP-mode command should reference MCP tool, got:\n%s", string(data))
	}
}

func containsPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}

func TestOpenCodeRegister_PreservesExistingConfig(t *testing.T) {
	setupTestSkills(t)
	dir := t.TempDir()
	projDir := filepath.Join(dir, "project")
	os.MkdirAll(projDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	// Write existing config
	configPath := filepath.Join(projDir, "opencode.json")
	existing := map[string]any{
		"theme": "dark",
		"permission": map[string]any{
			"bash": map[string]any{
				"git *": "allow",
			},
		},
	}
	if err := WriteJSONFile(configPath, existing); err != nil {
		t.Fatalf("write config: %v", err)
	}

	adapter := &OpenCodeAdapter{}
	if err := adapter.Register(false); err != nil {
		t.Fatalf("register: %v", err)
	}

	data, err := ReadJSONFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	// Verify existing config preserved
	if data["theme"] != "dark" {
		t.Error("theme should be preserved")
	}
	bash := data["permission"].(map[string]any)["bash"].(map[string]any)
	if bash["git *"] != "allow" {
		t.Error("existing git permission should be preserved")
	}
	if bash["monocle *"] != "allow" {
		t.Error("monocle permission should be added")
	}
}

func TestOpenCodeRegister_Idempotent(t *testing.T) {
	setupTestSkills(t)
	dir := t.TempDir()
	projDir := filepath.Join(dir, "project")
	os.MkdirAll(projDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	adapter := &OpenCodeAdapter{}
	if err := adapter.Register(false); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := adapter.Register(false); err != nil {
		t.Fatalf("second register: %v", err)
	}

	data, err := ReadJSONFile(filepath.Join(projDir, "opencode.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	bash := data["permission"].(map[string]any)["bash"].(map[string]any)
	if len(bash) != 1 {
		t.Errorf("expected 1 bash permission entry, got %d", len(bash))
	}
}

func TestOpenCodeUnregister_RemovesPermissions(t *testing.T) {
	setupTestSkills(t)
	dir := t.TempDir()
	projDir := filepath.Join(dir, "project")
	os.MkdirAll(projDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	adapter := &OpenCodeAdapter{}
	if err := adapter.Register(false); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := adapter.Unregister(false); err != nil {
		t.Fatalf("unregister: %v", err)
	}

	// opencode.json should be removed (was only monocle permissions)
	configPath := filepath.Join(projDir, "opencode.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatal("opencode.json should be removed when only monocle permissions existed")
	}
}

func TestOpenCodeUnregister_PreservesOtherPermissions(t *testing.T) {
	setupTestSkills(t)
	dir := t.TempDir()
	projDir := filepath.Join(dir, "project")
	os.MkdirAll(projDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	// Write config with monocle + other permissions
	configPath := filepath.Join(projDir, "opencode.json")
	existing := map[string]any{
		"permission": map[string]any{
			"bash": map[string]any{
				"git *":     "allow",
				"monocle *": "allow",
			},
		},
	}
	if err := WriteJSONFile(configPath, existing); err != nil {
		t.Fatalf("write config: %v", err)
	}

	adapter := &OpenCodeAdapter{}
	if err := adapter.Unregister(false); err != nil {
		t.Fatalf("unregister: %v", err)
	}

	data, err := ReadJSONFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	bash := data["permission"].(map[string]any)["bash"].(map[string]any)
	if bash["git *"] != "allow" {
		t.Error("git permission should be preserved")
	}
	if _, ok := bash["monocle *"]; ok {
		t.Error("monocle permission should be removed")
	}
}
