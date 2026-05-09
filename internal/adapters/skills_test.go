package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestSkills populates a temp directory with stub SKILL.md files and sets
// SkillsSourceOverride so InstallSkills reads from there instead of the repo.
func setupTestSkills(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range SkillNames {
		skillDir := filepath.Join(dir, name)
		os.MkdirAll(skillDir, 0755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# "+name+"\nTest skill content.\n"), 0644)
	}
	SkillsSourceOverride = dir
	t.Cleanup(func() { SkillsSourceOverride = "" })
}

func TestInstallSkills(t *testing.T) {
	setupTestSkills(t)
	dest := t.TempDir()

	if err := InstallSkills(dest); err != nil {
		t.Fatalf("InstallSkills: %v", err)
	}

	for _, name := range SkillNames {
		path := filepath.Join(dest, name, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("skill %s not installed: %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("skill %s is empty", name)
		}
	}
}

func TestRemoveSkills(t *testing.T) {
	setupTestSkills(t)
	dest := t.TempDir()

	if err := InstallSkills(dest); err != nil {
		t.Fatalf("InstallSkills: %v", err)
	}

	RemoveSkills(dest)

	for _, name := range SkillNames {
		path := filepath.Join(dest, name, "SKILL.md")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("skill %s should have been removed", name)
		}
	}
}

func TestResolveSkillsSource_LocalDirectory(t *testing.T) {
	origOverride := SkillsSourceOverride
	SkillsSourceOverride = ""
	defer func() { SkillsSourceOverride = origOverride }()

	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	for _, name := range SkillNames {
		os.MkdirAll(filepath.Join(skillsDir, name), 0755)
		os.WriteFile(filepath.Join(skillsDir, name, "SKILL.md"), []byte("# test"), 0644)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	src, err := resolveSkillsSource()
	if err != nil {
		t.Fatalf("resolveSkillsSource: %v", err)
	}
	if src != "skills" {
		t.Fatalf("expected 'skills', got %q", src)
	}
}

func TestResolveSkillsSource_Override(t *testing.T) {
	dir := t.TempDir()
	SkillsSourceOverride = dir
	defer func() { SkillsSourceOverride = "" }()

	src, err := resolveSkillsSource()
	if err != nil {
		t.Fatalf("resolveSkillsSource: %v", err)
	}
	if src != dir {
		t.Fatalf("expected %q, got %q", dir, src)
	}
}
