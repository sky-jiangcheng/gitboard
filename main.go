package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"gitboard/internal/db"
	"gitboard/internal/platform"
	"gitboard/internal/scanner"
	"gitboard/internal/server"
)

//go:embed web/dist/*
var embeddedFiles embed.FS

const defaultPort = "28731"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("GitBoard starting...")

	// Database (InitDB also inserts default config values)
	database, err := db.InitDB("data.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Default scan root (if no config exists)
	existingRoots, _ := db.GetScanRoots(database)
	if len(existingRoots) == 0 {
		defaultRoots := platform.DefaultScanRoots()
		for _, root := range defaultRoots {
			if err := db.AddScanRoot(database, root); err != nil {
				log.Printf("Warning: failed to add default scan root %s: %v", root, err)
			}
		}
	}

	// Git user
	gitUser := platform.GetGitUserName()
	if gitUser != "" {
		log.Printf("Git user detected: %s", gitUser)
	}

	// Parse scan depth, validate range
	scanDepthStr, _ := db.GetConfig(database, "scan_depth")
	scanDepth, err := strconv.Atoi(scanDepthStr)
	if err != nil || scanDepth < 1 || scanDepth > 10 {
		log.Printf("Invalid scan depth '%s', using default 5", scanDepthStr)
		scanDepth = 5
	}

	// Auto-scan on startup
	roots, _ := db.GetScanRoots(database)
	if len(roots) == 0 {
		log.Println("No scan roots configured. Add paths via Settings page.")
	} else {
		log.Printf("Auto-scanning with depth %d at roots: %v", scanDepth, roots)
		repos, err := scanner.ScanRepositories(roots, scanDepth)
		if err != nil {
			log.Printf("Scan error: %v", err)
		} else {
			log.Printf("Found %d repositories", len(repos))
		}
	}

	// Server
	port := defaultPort
	if envPort := os.Getenv("PORT"); envPort != "" {
		if _, err := strconv.Atoi(envPort); err == nil {
			port = envPort
		}
	}

	srv := server.NewServer(database, gitUser)

	// Serve embedded frontend
	staticFS, err := fs.Sub(embeddedFiles, "web/dist")
	if err != nil {
		log.Fatalf("Failed to load embedded frontend: %v", err)
	}
	srv.RegisterStatic(staticFS)

	addr := server.FormatAddr(port)
	listener, err := net.Listen("tcp", addr)
		if err != nil {
			//nolint:gosec // port is validated by strconv.Atoi above
			log.Printf("Failed to listen on port %s: %v", port, err)
			os.Exit(1)
		}

	actualPort := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	url := fmt.Sprintf("http://localhost:%s", actualPort)
	log.Printf("Server running at %s", url)

	// Open browser (URL is internally generated, always safe)
	go func() {
		if err := platform.OpenBrowser(url); err != nil {
			log.Printf("Failed to open browser: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	httpServer := &http.Server{
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
	if err := httpServer.Close(); err != nil {
		log.Printf("Server close error: %v", err)
	}
}
