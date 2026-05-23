package setup

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func EnsurePulumiBackground() {
	_, err := exec.LookPath("pulumi")
	
	home, homeErr := os.UserHomeDir()
	var pulumiBinDir string
	if homeErr == nil {
		pulumiBinDir = filepath.Join(home, ".pulumi", "bin")
		// Check if it's already installed but not in PATH
		if err != nil {
			if fileExists(filepath.Join(pulumiBinDir, "pulumi")) || (runtime.GOOS == "windows" && fileExists(filepath.Join(pulumiBinDir, "pulumi.exe"))) {
				// Add to PATH and return
				path := os.Getenv("PATH")
				if !strings.Contains(path, pulumiBinDir) {
					os.Setenv("PATH", path+string(os.PathListSeparator)+pulumiBinDir)
				}
				log.Printf("Pulumi found in %s, added to PATH", pulumiBinDir)
				return
			}
		}
	}

	if err == nil {
		return // Pulumi already in PATH
	}

	go func() {
		log.Printf("Pulumi CLI not found. Installing in the background...")

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmdStr := `[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; iex ((New-Object System.Net.WebClient).DownloadString('https://get.pulumi.com/install.ps1'))`
			cmd = exec.Command("powershell.exe", "-NoProfile", "-InputFormat", "None", "-ExecutionPolicy", "Bypass", "-Command", cmdStr)
		} else {
			cmd = exec.Command("sh", "-c", "curl -fsSL https://get.pulumi.com | sh")
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to install Pulumi CLI: %v\nOutput: %s", err, string(out))
			return
		}

		if homeErr == nil {
			path := os.Getenv("PATH")
			if !strings.Contains(path, pulumiBinDir) {
				os.Setenv("PATH", path+string(os.PathListSeparator)+pulumiBinDir)
			}
		}

		log.Printf("Successfully installed Pulumi CLI in background.")
	}()
}
