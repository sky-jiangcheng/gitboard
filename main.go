package main

import (
	"embed"
	"log"
	"os"

	"gitboard/internal/db"
	"gitboard/internal/platform"
	"gitboard/internal/scanner"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:web/dist
var assets embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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
			defaultRoots = []string{"/"}
		}
		log.Println("First launch — auto-scanning repositories...")
		for _, root := range defaultRoots {
			log.Printf("  Scanning: %s", root)
		}
		depthStr, _ := db.GetConfig(database, "scan_depth")
		maxDepth := 5
		if d, err := parseInt(depthStr); err == nil && d > 0 && d <= 10 {
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
		Title:  "GitBoard",
		Width:  1280,
		Height: 800,
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

func parseInt(s string) (int, error) {
	v := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		v = v*10 + int(c-'0')
	}
	return v, nil
}
