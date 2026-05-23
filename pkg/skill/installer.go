package skill

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func InstallSkill() error {
	fmt.Println("Installing copilot-infra skill...")

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	// Target global skill directories
	dirs := []string{
		filepath.Join(home, ".gemini", "skills", "copilot-infra"),
		filepath.Join(home, ".qoder", "skills", "copilot-infra"),
		filepath.Join(home, ".config", "github-copilot", "skills", "copilot-infra"),
	}

	installed := false
	for _, dir := range dirs {
		// Clean up directory to ensure no residual source code or unwanted files remain
		if err := cleanSkillDir(dir); err != nil {
			fmt.Printf("Warning: failed to clean skill directory %s: %v\n", dir, err)
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}

		dest := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(dest, SkillContent, 0644); err == nil {
			fmt.Printf("Successfully installed skill at %s\n", dest)
			installed = true
		}
	}

	if !installed {
		return fmt.Errorf("failed to install skill in any agent directory")
	}

	return nil
}

func cleanSkillDir(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return os.Remove(dir)
	}

	// Read and remove everything inside the directory to start completely fresh
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}

