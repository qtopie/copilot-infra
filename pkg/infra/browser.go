package infra

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BrowserProgram returns a Pulumi program that starts a browser with remote debugging enabled.
func BrowserProgram(port int, userDataDir string, logPath string) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		chromeBin, err := ResolveChromeBin()
		if err != nil {
			return err
		}

		pidFile := logPath + ".pid"
		// Ensure directories exist
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(userDataDir, 0755); err != nil {
			return err
		}

		// Start browser in background and wait for it to be ready
		createCmd := fmt.Sprintf(`nohup "%s" --remote-debugging-port=%d --user-data-dir="%s" --headless=new --no-first-run --no-default-browser-check --remote-allow-origins=* about:blank >> "%s" 2>&1 & echo $! > "%s"
echo "Waiting for browser to initialize on port %d..."
for i in {1..10}; do
  if curl -s "http://127.0.0.1:%d/json/version" >/dev/null 2>&1; then
    echo "Browser environment is ready"
    exit 0
  fi
  sleep 1
done
echo "Failed to verify browser" >&2
exit 1`,
			chromeBin, port, userDataDir, logPath, pidFile, port, port)

		deleteCmd := fmt.Sprintf("kill -9 $(cat \"%s\") || true; rm -f \"%s\"", pidFile, pidFile)

		_, err = local.NewCommand(ctx, "chrome-browser", &local.CommandArgs{
			Create: pulumi.String(createCmd),
			Delete: pulumi.String(deleteCmd),
		})

		if err != nil {
			return err
		}

		ctx.Export("chrome_bin", pulumi.String(chromeBin))
		ctx.Export("port", pulumi.Int(port))
		ctx.Export("debug_url", pulumi.String(fmt.Sprintf("http://127.0.0.1:%d", port)))

		return nil
	}
}

// ResolveChromeBin finds the Chrome/Chromium/Edge executable path.
func ResolveChromeBin() (string, error) {
	if bin := os.Getenv("CHROME_BIN"); bin != "" {
		if _, err := exec.LookPath(bin); err == nil {
			return bin, nil
		}
	}

	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		candidates := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			filepath.Join(home, "Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			filepath.Join(home, "Applications/Chromium.app/Contents/MacOS/Chromium"),
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			filepath.Join(home, "Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
		}
	case "linux":
		// Check PATH first
		commands := []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
			"chrome",
			"microsoft-edge",
			"microsoft-edge-stable",
		}
		for _, cmd := range commands {
			if path, err := exec.LookPath(cmd); err == nil {
				return path, nil
			}
		}
		// Common fixed paths
		paths := []string{
			"/snap/bin/chromium",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/microsoft-edge",
			"/usr/bin/microsoft-edge-stable",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	case "windows":
		candidates := []string{
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe",
			filepath.Join(os.Getenv("LocalAppData"), "Google\\Chrome\\Application\\chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Google\\Chrome\\Application\\chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google\\Chrome\\Application\\chrome.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
		}
	}

	return "", fmt.Errorf("could not find Google Chrome, Chromium, or Microsoft Edge. Please set CHROME_BIN environment variable")
}
