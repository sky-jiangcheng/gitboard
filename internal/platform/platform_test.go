package platform

import (
	"runtime"
	"testing"
)

func TestDetectOS(t *testing.T) {
	os := DetectOS()
	if os == "" {
		t.Fatal("DetectOS returned empty string")
	}
}

func TestDefaultScanRoots(t *testing.T) {
	roots := DefaultScanRoots()
	if len(roots) == 0 {
		t.Fatal("DefaultScanRoots returned empty slice")
	}
	for _, root := range roots {
		if root == "" {
			t.Error("DefaultScanRoots contains empty path")
		}
	}
}

func TestGetWindowsDrives(t *testing.T) {
	drives := getWindowsDrives()
	if runtime.GOOS != "windows" && len(drives) != 0 {
		t.Error("getWindowsDrives should return nil on non-Windows")
	}
}

func TestCheckGitInstalled(t *testing.T) {
	result := CheckGitInstalled()
	// Git should be available in CI/dev environments
	if !result {
		t.Log("Git not found in PATH (may be expected in some environments)")
	}
}

func TestGetGitUserName(t *testing.T) {
	name := GetGitUserName()
	if name == "" {
		t.Error("GetGitUserName returned empty string")
	}
	if name == "unknown" {
		t.Log("Git user name not configured, using fallback")
	}
}

func TestGetDbPath(t *testing.T) {
	path := GetDbPath()
	if path == "" {
		t.Fatal("GetDbPath returned empty string")
	}
}

func TestGetPort(t *testing.T) {
	port := GetPort()
	if port == "" {
		t.Fatal("GetPort returned empty string")
	}
}

func TestServerURL(t *testing.T) {
	url := ServerURL("28731")
	expected := "http://localhost:28731"
	if url != expected {
		t.Errorf("ServerURL = %s, want %s", url, expected)
	}
}

func TestOpenBrowser_ValidURL(t *testing.T) {
	tests := []string{
		"http://localhost:28731",
		"https://example.com",
	}
	for _, u := range tests {
		err := OpenBrowser(u)
		// In headless/CI environments, xdg-open/open may not be available.
		// We only care that the URL validation passes and no error is returned
		// from the validation step. ENOENT from the exec is expected.
		if err != nil && err.Error() == "exec: \"xdg-open\": executable file not found in $PATH" {
			continue
		}
		if err != nil {
			t.Errorf("OpenBrowser(%s) returned error: %v", u, err)
		}
	}
}

func TestOpenBrowser_InvalidURL(t *testing.T) {
	tests := []string{
		"javascript:alert(1)",
		"file:///etc/passwd",
		"ftp://evil.com",
		"   ",
		"not-a-url",
	}
	for _, u := range tests {
		err := OpenBrowser(u)
		if err == nil {
			t.Errorf("OpenBrowser(%s) should have returned error", u)
		}
	}
}
