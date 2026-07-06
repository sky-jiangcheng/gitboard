package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"git-dashboard/internal/db"
	"git-dashboard/internal/grouper"
	"git-dashboard/internal/platform"
	"git-dashboard/internal/scanner"
	"git-dashboard/internal/server"
)

//go:embed web/dist/*
var staticFiles embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Check git availability
	if !platform.CheckGitInstalled() {
		return fmt.Errorf("git is not installed or not in PATH. Please install git first")
	}

	// Get git user name
	gitUser := platform.GetGitUserName()
	log.Printf("Git user: %s", gitUser)

	// Initialize database
	dbPath := platform.GetDbPath()
	log.Printf("Database path: %s", dbPath)
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Check if first run (no scan roots configured)
	scanRoots, err := db.GetScanRoots(database)
	if err != nil {
		return fmt.Errorf("failed to get scan roots: %w", err)
	}

	isFirstRun := len(scanRoots) == 0
	if isFirstRun {
		log.Println("First run detected, applying default scan roots")
		defaultRoots := platform.DefaultScanRoots()
		for _, root := range defaultRoots {
			db.AddScanRoot(database, root)
			log.Printf("Added scan root: %s", root)
		}
		scanRoots = defaultRoots
	}

	// Run initial scan
	scanDepthStr, _ := db.GetConfig(database, "scan_depth")
	scanDepth := 5
	if scanDepthStr != "" {
		fmt.Sscanf(scanDepthStr, "%d", &scanDepth)
	}

	repos, err := scanner.ScanRepositories(scanRoots, scanDepth)
	if err != nil {
		log.Printf("Warning: scan error: %v", err)
	}
	log.Printf("Found %d git repositories", len(repos))

	// Group repositories into projects
	groups := grouper.GroupRepositories(repos)
	log.Printf("Grouped into %d projects", len(groups))

	// Persist to database (only on first run or if repos changed)
	if isFirstRun || dbNeedsRefresh(database, repos) {
		db.DeleteAllRepositories(database)
		db.DeleteAllProjects(database)

		for _, group := range groups {
			projectID, err := db.UpsertProject(database, group.Name, group.RootPath, 0, group.IsAutoGrouped)
			if err != nil {
				log.Printf("Warning: failed to upsert project %s: %v", group.Name, err)
				continue
			}
			for _, repo := range group.Repos {
				db.UpsertRepository(database, repo.Path, projectID)
			}
		}
	}

	// Create HTTP server
	srv := server.NewServer(database, gitUser)

	// Try to serve embedded frontend, fall back to message if not built
	staticFS, err := fs.Sub(staticFiles, "web/dist")
	if err != nil || isStaticEmpty(staticFS) {
		log.Println("Frontend not built. API only mode.")
	} else {
		srv.RegisterStatic(staticFS)
		log.Println("Frontend static files registered")
	}

	// Start HTTP server
	port := platform.GetPort()
	addr := server.FormatAddr(port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		httpServer.Close()
	}()

	serverURL := platform.ServerURL(port)
	log.Printf("Git Dashboard starting at %s", serverURL)

	// Open browser
	if err := platform.OpenBrowser(serverURL); err != nil {
		log.Printf("Failed to open browser: %v (server running at %s)", err, serverURL)
	}

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// dbNeedsRefresh checks if the database needs to be refreshed based on scan results.
func dbNeedsRefresh(database *sql.DB, repos []scanner.RepoInfo) bool {
	existing, err := db.GetAllRepositories(database)
	if err != nil {
		return true
	}
	if len(existing) != len(repos) {
		return true
	}
	existingPaths := make(map[string]bool)
	for _, r := range existing {
		existingPaths[r.Path] = true
	}
	for _, r := range repos {
		if !existingPaths[r.Path] {
			return true
		}
	}
	return false
}

// isStaticEmpty checks if the embedded filesystem is empty.
func isStaticEmpty(fsys fs.FS) bool {
	f, err := fsys.Open(".")
	if err != nil {
		return true
	}
	defer f.Close()
	entries, err := f.(fs.ReadDirFile).ReadDir(1)
	if err != nil {
		return true
	}
	return len(entries) == 0
}
