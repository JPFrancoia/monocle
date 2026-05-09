package adapters

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkillNames lists the skill directories packaged in releases.
var SkillNames = []string{"get-feedback", "get-feedback-wait", "review-plan", "review-plan-wait"}

// SkillsSourceOverride, if non-empty, is used instead of downloading or
// reading the local directory. Used by tests.
var SkillsSourceOverride string

// InstallSkills writes skill files to the given parent directory.
// Each skill is written as skillsDir/<name>/SKILL.md.
func InstallSkills(skillsDir string) error {
	srcDir, err := resolveSkillsSource()
	if err != nil {
		return err
	}
	for _, name := range SkillNames {
		content, err := os.ReadFile(filepath.Join(srcDir, name, "SKILL.md"))
		if err != nil {
			return fmt.Errorf("read skill %s: %w", name, err)
		}
		dest := filepath.Join(skillsDir, name, "SKILL.md")
		if err := WriteFileAtomic(dest, content); err != nil {
			return fmt.Errorf("write skill %s: %w", name, err)
		}
	}
	return nil
}

// resolveSkillsSource returns the directory containing skill subdirectories.
func resolveSkillsSource() (string, error) {
	if SkillsSourceOverride != "" {
		return SkillsSourceOverride, nil
	}
	if _, err := os.Stat("skills"); err == nil {
		return "skills", nil
	}
	return "", fmt.Errorf("skills directory not found (run from repo root)")
}

// RemoveSkills removes installed skill directories from the given parent directory.
func RemoveSkills(skillsDir string) {
	for _, name := range SkillNames {
		dir := filepath.Join(skillsDir, name)
		_ = RemoveFileIfExists(filepath.Join(dir, "SKILL.md"))
		_ = os.Remove(dir) // remove dir if empty, ignore errors
	}
}

// SkillPaths returns the paths of installed skill files relative to skillsDir.
func SkillPaths(skillsDir string) []string {
	var paths []string
	for _, name := range SkillNames {
		paths = append(paths, filepath.Join(skillsDir, name, "SKILL.md"))
	}
	return paths
}
