package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OS type constants
const (
	OSWindows = "windows"
	OSDarwin  = "darwin"
	OSLinux   = "linux"
)

// DetectOS returns the current operating system name.
func DetectOS() string {
	return runtime.GOOS
}

// DefaultScanRoots returns platform-specific default scan root directories.
// Windows: all drive letters except C: (the system drive).
// macOS: the user's home directory.
// Linux: the user's home directory.
func DefaultScanRoots() []string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		roots := []string{}
		for _, drive := range getWindowsDrives() {
			upper := strings.ToUpper(drive)
			if upper != "C:" && upper != "C:\\" {
				roots = append(roots, drive)
			}
		}
		if len(roots) == 0 {
			roots = append(roots, home)
		}
		return roots
	case "darwin":
		return []string{home}
	default: // linux and others
		return []string{home}
	}
}

// getWindowsDrives enumerates available drive letters on Windows.
// On non-Windows platforms, returns an empty slice.
func getWindowsDrives() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	drives := []string{}
	for c := 'A'; c <= 'Z'; c++ {
		path := string(c) + ":\\"
		if _, err := os.Stat(path); err == nil {
			drives = append(drives, path)
		}
	}
	return drives
}

// OpenBrowser opens the specified URL in the system's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux and others
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// CheckGitInstalled checks whether git is available in the system PATH.
func CheckGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetGitUserName returns the git user.name from global or local config.
func GetGitUserName() string {
	cmd := exec.Command("git", "config", "user.name")
	out, err := cmd.Output()
	if err != nil {
		// fallback to OS username
		if u, e := os.UserHomeDir(); e == nil {
			return filepath.Base(u)
		}
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// GetDbPath returns the path to the SQLite database file.
// The database is stored in the user's config directory.
func GetDbPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.TempDir(), "git-dashboard")
	}
	dir := filepath.Join(configDir, "git-dashboard")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return filepath.Join(os.TempDir(), "git-dashboard.db")
	}
	return filepath.Join(dir, "dashboard.db")
}

// GetPort returns the server port from environment or a random available port.
func GetPort() string {
	if port := os.Getenv("GIT_DASHBOARD_PORT"); port != "" {
		return port
	}
	return "18731"
}

// ServerURL returns the full local server URL.
func ServerURL(port string) string {
	return fmt.Sprintf("http://localhost:%s", port)
}
