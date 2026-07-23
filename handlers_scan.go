package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"gitboard/internal/db"
	"gitboard/internal/grouper"
	"gitboard/internal/scanner"
	"gitboard/internal/stats"
)

// ScanResult holds the result of a scan operation.
type ScanResult struct {
	Success    bool `json:"success"`
	ReposFound int  `json:"repos_found"`
	Projects   int  `json:"projects"`
}

// ScanStatus holds the current scanning progress.
type ScanStatus struct {
	Running     bool   `json:"running"`
	Backfilling bool   `json:"backfilling"`
	Message     string `json:"message"`
	Progress    int    `json:"progress"`
	Total       int    `json:"total"`
}

// TriggerScan starts an async full repository scan and returns immediately.
func (a *App) TriggerScan() (*ScanResult, error) {
	a.scanMu.Lock()
	if a.scanning {
		a.scanMu.Unlock()
		return nil, fmt.Errorf("scan already in progress")
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.scanCancel = cancel
	a.scanning = true
	a.scanMu.Unlock()

	go func() {
		a.runFullScan(ctx)
		a.scanMu.Lock()
		a.scanning = false
		a.scanProgress = 0
		a.scanTotal = 0
		a.scanCancel = nil
		a.scanMu.Unlock()
	}()

	return &ScanResult{Success: true}, nil
}

// GetScanStatus returns the current scan progress.
func (a *App) GetScanStatus() *ScanStatus {
	a.scanMu.Lock()
	running := a.scanning
	backfilling := a.backfilling
	progress := a.scanProgress
	total := a.scanTotal
	a.scanMu.Unlock()
	msg := ""
	if running {
		if total > 0 {
			msg = fmt.Sprintf("正在扫描仓库 %d/%d…", progress, total)
		} else {
			msg = "正在扫描仓库…"
		}
	} else if backfilling {
		msg = "正在回填历史数据…"
	}
	return &ScanStatus{
		Running:     running,
		Backfilling: backfilling,
		Message:     msg,
		Progress:    progress,
		Total:       total,
	}
}

// runFullScan performs the actual scan + history backfill.
// Uses a merge strategy: existing projects/repos are preserved (including
// their notes, todos, and starred status). Only truly orphaned data is removed.
func (a *App) runFullScan(ctx context.Context) {
	depthStr, _ := db.GetConfig(a.db, "scan_depth")
	maxDepth, _ := strconv.Atoi(depthStr)
	if maxDepth <= 0 || maxDepth > 10 {
		maxDepth = 5
	}

	roots, _ := db.GetScanRoots(a.db)
	repos, err := scanner.ScanRepositories(roots, maxDepth)
	if err != nil {
		log.Printf("scan error: %v", err)
		return
	}

	groups := grouper.GroupRepositories(repos)

	a.scanMu.Lock()
	a.scanTotal = len(groups)
	a.scanProgress = 0
	a.scanMu.Unlock()

	// Collect all scanned repo paths for stale-data cleanup
	scannedPaths := make([]string, 0, len(repos))
	for _, r := range repos {
		scannedPaths = append(scannedPaths, r.Path)
	}

	tx, err := a.db.Begin()
	if err != nil {
		log.Printf("scan transaction begin error: %v", err)
		return
	}
	defer tx.Rollback() //nolint:errcheck

	for i, group := range groups {
		select {
		case <-ctx.Done():
			return
		default:
		}
		a.scanMu.Lock()
		a.scanProgress = i + 1
		a.scanMu.Unlock()

		projectID, err := db.SyncProjectTx(tx, group.Name, group.RootPath, 0, group.IsAutoGrouped)
		if err != nil {
			log.Printf("sync project error: %v", err)
			continue
		}
		for _, repo := range group.Repos {
			if err := db.UpsertRepositoryTx(tx, repo.Path, projectID); err != nil {
				log.Printf("upsert repo error: %v", err)
			}
		}
	}

	// Remove stale repos and orphaned projects, preserving user data
	if err := db.CleanupStaleDataTx(tx, scannedPaths); err != nil {
		log.Printf("cleanup stale data error: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("scan transaction commit error: %v", err)
		return
	}

	a.refreshAllStatsWithCancel(ctx)
	_ = db.SetConfig(a.db, "last_stats_backfill", stats.GetTodayDate())
	log.Printf("scan complete: %d repos, %d projects", len(repos), len(groups))
}

// backfillTimeout is the maximum time allowed for a single backfill pass.
const backfillTimeout = 30 * time.Minute

// ensureHistoryBackfilled checks if we need to update stats, and backfills missing days.
// Runs in background at startup with a timeout. Uses config to track last backfill date
// to avoid repeating.
func (a *App) ensureHistoryBackfilled() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in backfill: %v", r)
		}
	}()

	repos, err := db.GetAllRepositories(a.db)
	if err != nil || len(repos) == 0 {
		return
	}

	lastBackfillStr, _ := db.GetConfig(a.db, "last_stats_backfill")
	lastBackfill, _ := time.Parse("2006-01-02", lastBackfillStr)

	today := stats.GetTodayDate()
	startDate := today

	if lastBackfill.IsZero() {
		startDate = time.Now().AddDate(0, 0, -365).Format("2006-01-02")
	} else {
		startDate = lastBackfill.AddDate(0, 0, 1).Format("2006-01-02")
		if startDate > today {
			return
		}
	}

	log.Printf("backfilling stats from %s to %s...", startDate, today)
	a.scanMu.Lock()
	a.backfilling = true
	ctx, cancel := context.WithTimeout(context.Background(), backfillTimeout)
	a.scanCancel = cancel
	a.scanMu.Unlock()

	// Ensure cancel is always called to release context resources
	defer func() {
		cancel()
		a.scanMu.Lock()
		a.backfilling = false
		a.scanCancel = nil
		a.scanMu.Unlock()
	}()

	hasData := false
	for _, repo := range repos {
		select {
		case <-ctx.Done():
			log.Printf("stats refresh cancelled or timed out")
			return
		default:
		}

		allEntries, err := stats.QueryStatsRange(repo.Path, startDate, today, "")
		if err == nil && allEntries != nil {
			for _, e := range allEntries {
				if e.FilesChanged > 0 || e.LinesAdded > 0 || e.LinesDeleted > 0 {
					_ = db.UpsertDailyStat(a.db, repo.ID, e.Date, "all",
						e.FilesChanged, e.LinesAdded, e.LinesDeleted)
					hasData = true
				}
			}
		}

		if a.gitUser != "" {
			myEntries, err := stats.QueryStatsRange(repo.Path, startDate, today, a.gitUser)
			if err == nil && myEntries != nil {
				for _, e := range myEntries {
					if e.FilesChanged > 0 || e.LinesAdded > 0 || e.LinesDeleted > 0 {
						_ = db.UpsertDailyStat(a.db, repo.ID, e.Date, a.gitUser,
							e.FilesChanged, e.LinesAdded, e.LinesDeleted)
						hasData = true
					}
				}
			}
		}
	}

	_ = db.SetConfig(a.db, "last_stats_backfill", today)
	log.Printf("stats backfill %s, has data: %v", today, hasData)
}
