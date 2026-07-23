package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"gitboard/internal/db"
	"gitboard/internal/platform"
	"gitboard/internal/scanner"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:web/dist
var assets embed.FS

func init() {
	// Ensure PATH includes common binary directories (git may not be in PATH when launched from Finder)
	ensurePath()
	// Redirect logs to file so crashes can be diagnosed when launched from Finder
	setupLogging()
}

func main() {
	log.Println("GitBoard starting...")

	// Open database
	database, err := db.InitDB(platform.GetDbPath())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Detect git user
	gitUser := platform.GetGitUserName()
	if gitUser != "" {
		log.Printf("Git user detected: %s", gitUser)
	} else {
		log.Println("No git user detected; personal stats will be empty")
	}

	// Default scan root and auto-scan
	existingRoots, _ := db.GetScanRoots(database)
	if len(existingRoots) == 0 {
		homeDir, err := os.UserHomeDir()
		defaultRoots := []string{homeDir}
		if err != nil {
			defaultRoots = []string{"."}
		}
		log.Println("First launch — auto-scanning repositories...")
		for _, root := range defaultRoots {
			log.Printf("  Scanning: %s", root)
		}
		depthStr, _ := db.GetConfig(database, "scan_depth")
		maxDepth := 5
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 && d <= 10 {
			maxDepth = d
		}
		// Set default scan roots
		for _, root := range defaultRoots {
			if err := db.AddScanRoot(database, root); err != nil {
				log.Printf("  add scan root error: %v", err)
			}
		}
		repos, err := scanner.ScanRepositories(defaultRoots, maxDepth)
		if err != nil {
			log.Printf("Auto-scan error: %v", err)
		} else {
			log.Printf("Found %d repositories", len(repos))
		}
	}

	// Create app with dependencies
	app := NewApp(database, gitUser)

	// Launch Wails
	err = wails.Run(&options.App{
		Title:     "GitBoard",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func ensurePath() {
	path := os.Getenv("PATH")
	if runtime.GOOS == "windows" {
		// Common Windows binary directories
		extras := []string{
			`C:\Program Files\Git\cmd`,
			`C:\Program Files\Git\bin`,
			`C:\Program Files (x86)\Git\cmd`,
			`C:\Program Files (x86)\Git\bin`,
			`C:\Windows\System32`,
			`C:\Windows`,
		}
		for _, d := range extras {
			if path == "" {
				path = d
			} else {
				path = d + string(os.PathListSeparator) + path
			}
		}
	} else {
		// Unix-like: prepend common binary directories
		extras := "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
		if path == "" {
			path = extras
		} else {
			path = extras + string(os.PathListSeparator) + path
		}
	}
	os.Setenv("PATH", path)
}

func setupLogging() {
	var logDir string
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		logDir = filepath.Join(home, "Library", "Logs", "GitBoard")
	case "windows":
		logDir = filepath.Join(os.TempDir(), "GitBoard")
	default: // linux and others
		// Follow XDG Base Directory specification
		if cacheDir, err := os.UserCacheDir(); err == nil {
			logDir = filepath.Join(cacheDir, "gitboard")
		} else {
			logDir = filepath.Join(os.TempDir(), "gitboard")
		}
	}
	_ = os.MkdirAll(logDir, 0750)
	logFile := filepath.Join(logDir, "gitboard.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err == nil {
		log.SetOutput(f)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("=== GitBoard log started ===")
	log.Printf("PATH=%s", os.Getenv("PATH"))
}
