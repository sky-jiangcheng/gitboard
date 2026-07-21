package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitboard/internal/db"
	"gitboard/internal/grouper"
	"gitboard/internal/knowledge"
	"gitboard/internal/scanner"
	"gitboard/internal/stats"
)

// App is the main application struct whose public methods are exposed to the
// frontend via Wails Bind. The ctx is set during OnStartup.
type App struct {
	ctx         context.Context
	db          *sql.DB
	gitUser     string
	scanMu      sync.Mutex
	scanning    bool
	backfilling bool
	scanCancel  context.CancelFunc
}

// NewApp creates a new App instance with dependencies injected.
func NewApp(database *sql.DB, gitUser string) *App {
	return &App{
		db:      database,
		gitUser: gitUser,
	}
}

// startup is called at application startup.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.ensureHistoryBackfilled()
}

// shutdown is called when the application exits.
func (a *App) shutdown(ctx context.Context) {
	if a.scanCancel != nil {
		a.scanCancel()
	}
	if a.db != nil {
		a.db.Close()
	}
}

// Health returns a health-check payload for the frontend.
func (a *App) Health() map[string]interface{} {
	if err := a.db.Ping(); err != nil {
		return map[string]interface{}{"status": "error", "message": "database unavailable"}
	}
	return map[string]interface{}{"status": "ok", "version": "1.5.0"}
}

// ProjectResponse is the enriched project payload sent to the frontend.
type ProjectResponse struct {
	db.Project
	RepoCount     int  `json:"repo_count"`
	TotalAdded    int  `json:"total_added"`
	TotalDeleted  int  `json:"total_deleted"`
	MyAdded       int  `json:"my_added"`
	MyDeleted     int  `json:"my_deleted"`
	MyFiles       int  `json:"my_files"`
	IsWorkday     bool `json:"is_workday"`
	BelowStandard bool `json:"below_standard"`
}

// GetProjects returns enriched project summaries, optionally filtered by date.
func (a *App) GetProjects(date string, starredOnly bool) []ProjectResponse {
	if date == "" {
		date = stats.GetYesterdayDate()
	}
	if err := stats.ValidateDate(date); err != nil {
		log.Printf("invalid date: %v", err)
		return nil
	}

	var projects []db.Project
	var err error
	if starredOnly {
		projects, err = db.GetStarredProjects(a.db)
	} else {
		projects, err = db.GetAllProjects(a.db)
	}
	if err != nil {
		log.Printf("get projects error: %v", err)
		return nil
	}

	codeStdStr, _ := db.GetConfig(a.db, "daily_code_standard")
	codeStd, _ := strconv.Atoi(codeStdStr)
	if codeStd == 0 {
		codeStd = 500
	}
	isWorkday := stats.IsWorkday(date)

	var result []ProjectResponse
	for _, p := range projects {
		statsList, _ := db.GetStatsByProject(a.db, p.ID, date)
		if len(statsList) == 0 {
			repos, _ := db.GetRepositoriesByProjectID(a.db, p.ID)
			if len(repos) > 0 {
				a.refreshProjectStats(p.ID, date)
				statsList, _ = db.GetStatsByProject(a.db, p.ID, date)
			}
		}
		repos, _ := db.GetRepositoriesByProjectID(a.db, p.ID)

		pr := ProjectResponse{
			Project:   p,
			RepoCount: len(repos),
			IsWorkday: isWorkday,
		}
		for _, st := range statsList {
			pr.TotalAdded += st.LinesAdded
			pr.TotalDeleted += st.LinesDeleted
			if st.Author == a.gitUser {
				pr.MyAdded += st.LinesAdded
				pr.MyDeleted += st.LinesDeleted
				pr.MyFiles += st.FilesChanged
			}
		}
		pr.BelowStandard = isWorkday && pr.MyAdded < codeStd
		result = append(result, pr)
	}
	return result
}

// RepoInfo is a repository record with embedded daily stats.
type RepoInfo struct {
	db.Repository
	Stats []db.DailyStat `json:"stats"`
}

// ProjectDetailResponse is the full project detail payload.
type ProjectDetailResponse struct {
	*db.Project
	Repos []RepoInfo `json:"repos"`
}

// GetProjectDetail returns a project with all its repositories and stats.
func (a *App) GetProjectDetail(id int64) (*ProjectDetailResponse, error) {
	project, err := db.GetProjectByID(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	repos, _ := db.GetRepositoriesByProjectID(a.db, id)

	var repoList []RepoInfo
	for _, repo := range repos {
		statsList, _ := db.GetStatsByRepositoryAndDate(a.db, repo.ID, "")
		if statsList == nil {
			statsList = []db.DailyStat{}
		}
		repoList = append(repoList, RepoInfo{Repository: repo, Stats: statsList})
	}
	return &ProjectDetailResponse{Project: project, Repos: repoList}, nil
}

// GetProjectStats returns daily stats for a project, optionally filtered by date.
func (a *App) GetProjectStats(id int64, date string) []db.DailyStat {
	if date == "" {
		date = stats.GetYesterdayDate()
	}
	if err := stats.ValidateDate(date); err != nil {
		return nil
	}
	statsList, err := db.GetStatsByProject(a.db, id, date)
	if err != nil {
		log.Printf("get stats error: %v", err)
		return nil
	}
	if len(statsList) == 0 {
		a.refreshProjectStats(id, date)
		statsList, _ = db.GetStatsByProject(a.db, id, date)
	}
	return statsList
}

// LevelUpdateResult holds the result of a level change operation.
type LevelUpdateResult struct {
	Success  bool `json:"success"`
	NewLevel int  `json:"new_level"`
}

// UpdateProjectLevel adjusts a project's grouping: "down" splits a multi-repo
// project into per-repo projects (keeping the original for the first repo so its
// notes/todos survive); "up" merges sibling projects that share the same parent
// directory into this one (moving their repos, notes, and todos along). Both
// operate in a single transaction so the project graph never ends up half-changed.
func (a *App) UpdateProjectLevel(id int64, direction string) (*LevelUpdateResult, error) {
	if direction != "up" && direction != "down" {
		return nil, fmt.Errorf("direction must be 'up' or 'down'")
	}
	project, err := db.GetProjectByID(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	repos, err := db.GetRepositoriesByProjectID(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load repos")
	}

	tx, err := a.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction")
	}
	defer tx.Rollback() //nolint:errcheck

	newLevel := project.LevelOverride

	if direction == "down" {
		// Split: every repo except the first becomes its own project.
		if len(repos) <= 1 {
			if _, err := tx.Exec("UPDATE projects SET is_auto_grouped = 0 WHERE id = ?", id); err != nil {
				return nil, fmt.Errorf("failed to update project")
			}
		} else {
			for _, repo := range repos[1:] {
				res, err := tx.Exec(
					"INSERT INTO projects (name, root_path, level_override, is_auto_grouped) VALUES (?, ?, ?, 0)",
					filepath.Base(repo.Path), repo.Path, project.LevelOverride-1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create sub-project: %w", err)
				}
				newID, err := res.LastInsertId()
				if err != nil {
					return nil, fmt.Errorf("failed to get new project id: %w", err)
				}
				if _, err := tx.Exec("UPDATE repositories SET project_id = ? WHERE id = ?", newID, repo.ID); err != nil {
					return nil, fmt.Errorf("failed to reassign repo: %w", err)
				}
			}
			// Keep the original project bound to its first repo.
			if _, err := tx.Exec("UPDATE projects SET is_auto_grouped = 0, root_path = ? WHERE id = ?", repos[0].Path, id); err != nil {
				return nil, fmt.Errorf("failed to update project: %w", err)
			}
		}
		newLevel = project.LevelOverride - 1
	} else {
		// Up / merge: absorb sibling projects that share the same parent directory.
		parentDir := filepath.Dir(project.RootPath)
		if parentDir != "" && parentDir != "/" && parentDir != "." {
			rows, err := tx.Query("SELECT id, root_path FROM projects WHERE id != ? AND root_path LIKE ?", id, parentDir+"/%")
			if err != nil {
				return nil, fmt.Errorf("failed to query siblings: %w", err)
			}
			var siblingIDs []int64
			for rows.Next() {
				var sid int64
				var sroot string
				if err := rows.Scan(&sid, &sroot); err != nil {
					rows.Close()
					return nil, fmt.Errorf("failed to scan sibling: %w", err)
				}
				if filepath.Dir(sroot) == parentDir {
					siblingIDs = append(siblingIDs, sid)
				}
			}
			rows.Close()

			for _, sid := range siblingIDs {
				if _, err := tx.Exec("UPDATE repositories SET project_id = ? WHERE project_id = ?", id, sid); err != nil {
					return nil, fmt.Errorf("failed to move repos: %w", err)
				}
				if _, err := tx.Exec("UPDATE project_notes SET project_id = ? WHERE project_id = ?", id, sid); err != nil {
					return nil, fmt.Errorf("failed to move notes: %w", err)
				}
				if _, err := tx.Exec("UPDATE project_todos SET project_id = ? WHERE project_id = ?", id, sid); err != nil {
					return nil, fmt.Errorf("failed to move todos: %w", err)
				}
				if _, err := tx.Exec("DELETE FROM projects WHERE id = ?", sid); err != nil {
					return nil, fmt.Errorf("failed to remove merged project: %w", err)
				}
			}
			if _, err := tx.Exec("UPDATE projects SET is_auto_grouped = 0, root_path = ?, name = ? WHERE id = ?", parentDir, filepath.Base(parentDir), id); err != nil {
				return nil, fmt.Errorf("failed to update project: %w", err)
			}
		} else if _, err := tx.Exec("UPDATE projects SET is_auto_grouped = 0 WHERE id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to update project: %w", err)
		}
		newLevel = project.LevelOverride + 1
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit level change: %w", err)
	}
	return &LevelUpdateResult{Success: true, NewLevel: newLevel}, nil
}

// ProjectOverview is the mined-knowledge payload for a project detail page.
type ProjectOverview struct {
	ReadmeExcerpt  string                    `json:"readme_excerpt"`
	TechStack      []knowledge.Tech          `json:"tech_stack"`
	Languages      []knowledge.LanguageStat `json:"languages"`
	RecentCommits  []stats.RecentCommit      `json:"recent_commits"`
	Cached         bool                      `json:"cached"`
}

// GetProjectOverview returns mined knowledge for a project: README excerpt,
// detected tech stack, language breakdown, and recent commits. Mined results
// are cached in repo_meta so repeated loads do not re-walk the working tree.
func (a *App) GetProjectOverview(projectID int64) (*ProjectOverview, error) {
	project, err := db.GetProjectByID(a.db, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	repos, _ := db.GetRepositoriesByProjectID(a.db, projectID)

	resp := &ProjectOverview{}

	// Cache is keyed by the first repo id when available.
	var cacheRepoID int64
	if len(repos) > 0 {
		cacheRepoID = repos[0].ID
	}
	if cacheRepoID > 0 {
		if meta, err := db.GetRepoMeta(a.db, cacheRepoID); err == nil && meta != nil && meta.TechStack != "" {
			_ = json.Unmarshal([]byte(meta.TechStack), &resp.TechStack)
			_ = json.Unmarshal([]byte(meta.Languages), &resp.Languages)
			resp.ReadmeExcerpt = meta.ReadmeExcerpt
			resp.Cached = true
		}
	}

	// Mine fresh when no cache was found.
	if !resp.Cached {
		minePath := project.RootPath
		if minePath == "" && len(repos) > 0 {
			minePath = repos[0].Path
		}
		if minePath != "" {
			if k, err := knowledge.Mine(minePath); err == nil && k != nil {
				resp.ReadmeExcerpt = k.ReadmeExcerpt
				resp.TechStack = k.TechStack
				resp.Languages = k.Languages
				if cacheRepoID > 0 {
					ts, _ := json.Marshal(k.TechStack)
					ls, _ := json.Marshal(k.Languages)
					_ = db.UpsertRepoMeta(a.db, cacheRepoID, string(ts), k.ReadmeExcerpt, string(ls))
				}
			}
		}
	}

	// Recent commits are always fresh.
	repoPaths := make([]string, 0, len(repos))
	for _, r := range repos {
		repoPaths = append(repoPaths, r.Path)
	}
	if commits, err := stats.GetRecentCommits(repoPaths, a.gitUser, 8); err == nil {
		resp.RecentCommits = commits
	}
	return resp, nil
}

// ScanResult holds the result of a scan operation.
type ScanResult struct {
	Success    bool `json:"success"`
	ReposFound int  `json:"repos_found"`
	Projects   int  `json:"projects"`
}

// ScanStatus holds the current scanning progress.
type ScanStatus struct {
	Running    bool   `json:"running"`
	Backfilling bool  `json:"backfilling"`
	Message    string `json:"message"`
	Progress   int    `json:"progress"`
	Total      int    `json:"total"`
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
	a.scanMu.Unlock()
	msg := ""
	if running {
		msg = "正在扫描仓库…"
	} else if backfilling {
		msg = "正在回填历史数据…"
	}
	return &ScanStatus{
		Running:    running,
		Backfilling: backfilling,
		Message:    msg,
	}
}

// runFullScan performs the actual scan + history backfill.
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

	if err := db.DeleteAllRepositories(a.db); err != nil {
		log.Printf("delete repos error: %v", err)
	}
	if err := db.DeleteAllProjects(a.db); err != nil {
		log.Printf("delete projects error: %v", err)
	}

	for _, group := range groups {
		select {
		case <-ctx.Done():
			return
		default:
		}
		projectID, err := db.UpsertProject(a.db, group.Name, group.RootPath, 0, group.IsAutoGrouped)
		if err != nil {
			log.Printf("upsert project error: %v", err)
			continue
		}
		for _, repo := range group.Repos {
			if err := db.UpsertRepository(a.db, repo.Path, projectID); err != nil {
				log.Printf("upsert repo error: %v", err)
			}
		}
	}

	a.refreshAllStatsWithCancel(ctx)
	_ = db.SetConfig(a.db, "last_stats_backfill", stats.GetTodayDate())
	log.Printf("scan complete: %d repos, %d projects", len(repos), len(groups))
}

// ensureHistoryBackfilled checks if we need to update stats, and backfills missing days.
// Runs in background at startup. Uses config to track last backfill date to avoid repeating.
func (a *App) ensureHistoryBackfilled() {
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
	ctx, cancel := context.WithCancel(context.Background())
	a.scanCancel = cancel
	a.scanMu.Unlock()

	hasData := false
	for _, repo := range repos {
		select {
		case <-ctx.Done():
			log.Printf("stats refresh cancelled")
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
	a.scanMu.Lock()
	a.backfilling = false
	a.scanCancel = nil
	a.scanMu.Unlock()
	log.Printf("stats backfill %s, has data: %v", today, hasData)
}

// ConfigData holds the application configuration sent to the frontend.
type ConfigData struct {
	Config    map[string]string `json:"config"`
	ScanRoots []string          `json:"scan_roots"`
}

// GetConfig returns all configuration settings and scan roots.
func (a *App) GetConfig() (*ConfigData, error) {
	configs, err := db.GetAllConfigs(a.db)
	if err != nil {
		return nil, fmt.Errorf("failed to load config")
	}
	roots, _ := db.GetScanRoots(a.db)
	return &ConfigData{Config: configs, ScanRoots: roots}, nil
}

// UpdateConfig sets a single configuration key-value pair.
func (a *App) UpdateConfig(key, value string) error {
	allowed := map[string]bool{"daily_code_standard": true, "scan_depth": true, "git_author": true}
	if !allowed[key] {
		return fmt.Errorf("unknown config key: %s", key)
	}
	// Validate numeric configs
	if key != "git_author" {
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("config value must be a number")
		}
	}
	return db.SetConfig(a.db, key, value)
}

// UpdateScanRoots replaces the entire scan root list atomically.
func (a *App) UpdateScanRoots(scanRoots []string) error {
	if err := db.ReplaceScanRoots(a.db, scanRoots); err != nil {
		return fmt.Errorf("failed to update scan roots: %w", err)
	}
	return nil
}

// SummaryData holds the daily summary payload.
type SummaryData struct {
	Date         string `json:"date"`
	RepoCount    int    `json:"repo_count"`
	TotalFiles   int    `json:"total_files"`
	TotalAdded   int    `json:"total_added"`
	TotalDeleted int    `json:"total_deleted"`
	MyAdded      int    `json:"my_added"`
	MyDeleted    int    `json:"my_deleted"`
	MyFiles      int    `json:"my_files"`
	IsWorkday    bool   `json:"is_workday"`
}

// GetSummary returns aggregated stats for all repositories on a given date.
func (a *App) GetSummary(date string) (*SummaryData, error) {
	if date == "" {
		date = stats.GetYesterdayDate()
	}
	if err := stats.ValidateDate(date); err != nil {
		return nil, fmt.Errorf("invalid date format")
	}

	allStats, err := db.GetStatsByDate(a.db, date)
	if err != nil {
		return nil, fmt.Errorf("failed to load summary")
	}

	summary := &SummaryData{Date: date, IsWorkday: stats.IsWorkday(date)}
	repoSet := make(map[int64]bool)
	for _, st := range allStats {
		repoSet[st.RepositoryID] = true
		summary.TotalFiles += st.FilesChanged
		summary.TotalAdded += st.LinesAdded
		summary.TotalDeleted += st.LinesDeleted
		if st.Author == a.gitUser {
			summary.MyAdded += st.LinesAdded
			summary.MyDeleted += st.LinesDeleted
			summary.MyFiles += st.FilesChanged
		}
	}
	summary.RepoCount = len(repoSet)
	return summary, nil
}

// -- Todo Bind methods --

// ListTodos returns all todo items for a project.
func (a *App) ListTodos(projectID int64) []db.Todo {
	todos, err := db.ListTodos(a.db, projectID)
	if err != nil {
		log.Printf("list todos error: %v", err)
		return nil
	}
	if todos == nil {
		todos = []db.Todo{}
	}
	return todos
}

// CreateTodo creates a new todo for a project.
func (a *App) CreateTodo(projectID int64, title string) (*db.Todo, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	return db.CreateTodo(a.db, projectID, title)
}

// ToggleTodo flips the completed status of a todo.
func (a *App) ToggleTodo(todoID int64) error {
	return db.ToggleTodo(a.db, todoID)
}

// DeleteTodo removes a todo.
func (a *App) DeleteTodo(todoID int64) error {
	return db.DeleteTodo(a.db, todoID)
}

// ReorderTodos updates the sort_order for a list of todo IDs.
func (a *App) ReorderTodos(todoIDs []int64) error {
	return db.ReorderTodos(a.db, todoIDs)
}

// -- Note Bind methods --

// ListNotes returns all notes for a project.
func (a *App) ListNotes(projectID int64) []db.Note {
	notes, err := db.ListNotes(a.db, projectID)
	if err != nil {
		log.Printf("list notes error: %v", err)
		return nil
	}
	if notes == nil {
		notes = []db.Note{}
	}
	return notes
}

// CreateNote creates a new note for a project.
func (a *App) CreateNote(projectID int64, content string) (*db.Note, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is required")
	}
	return db.CreateNote(a.db, projectID, content)
}

// UpdateNote updates the content of a note.
func (a *App) UpdateNote(noteID int64, content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return db.UpdateNote(a.db, noteID, content)
}

// DeleteNote removes a note.
func (a *App) DeleteNote(noteID int64) error {
	return db.DeleteNote(a.db, noteID)
}

// -- Knowledge hub Bind methods --

// NoteWithProject is a note joined with its parent project (global knowledge hub).
type NoteWithProject = db.NoteWithProject

// ListAllNotes returns every note across all projects, joined with project info,
// ordered pinned first then most recently updated.
func (a *App) ListAllNotes() []NoteWithProject {
	notes, err := db.ListAllNotes(a.db)
	if err != nil {
		log.Printf("list all notes error: %v", err)
		return nil
	}
	if notes == nil {
		notes = []db.NoteWithProject{}
	}
	return notes
}

// ListAllTags returns the distinct set of tags used across all notes.
func (a *App) ListAllTags() []string {
	tags, err := db.ListAllTags(a.db)
	if err != nil {
		log.Printf("list all tags error: %v", err)
		return nil
	}
	if tags == nil {
		tags = []string{}
	}
	return tags
}

// CreateNoteWithMeta creates a note with explicit title, tags, kind, and source.
func (a *App) CreateNoteWithMeta(projectID int64, title, content, tags, kind, source string) (*db.Note, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is required")
	}
	return db.CreateNoteEx(a.db, projectID, title, content, tags, kind, source)
}

// UpdateNoteMeta updates a note's editable metadata (title, tags, kind, pinned).
func (a *App) UpdateNoteMeta(noteID int64, title, tags, kind string, pinned bool) error {
	return db.UpdateNoteMeta(a.db, noteID, title, tags, kind, pinned)
}

// PinNote sets or clears the pinned flag on a note.
func (a *App) PinNote(noteID int64, pinned bool) error {
	return db.PinNote(a.db, noteID, pinned)
}

// ImportResult summarizes a Claude memory import run.
type ImportResult struct {
	Synced  int `json:"synced"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

// ImportClaudeMemory imports notes from Claude's per-project memory directory
// (~/.claude/projects/*/memory/*.md) into GitBoard, matching each to a project
// by name or repository path. Imports are idempotent (re-running updates existing
// notes rather than duplicating them) and use parameterized queries throughout.
func (a *App) ImportClaudeMemory() (*ImportResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve home directory")
	}
	claudeDir := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		// No Claude memory directory yet; treat as a successful no-op.
		return &ImportResult{}, nil
	}

	projects, _ := db.GetAllProjects(a.db)
	repos, _ := db.GetAllRepositories(a.db)

	result := &ImportResult{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		memDir := filepath.Join(claudeDir, e.Name(), "memory")
		memEntries, err := os.ReadDir(memDir)
		if err != nil {
			continue
		}

		displayName := claudeDisplayName(e.Name())
		if len(displayName) < 2 {
			result.Skipped++
			continue
		}
		pid := matchClaudeProject(displayName, projects, repos)
		if pid == 0 {
			result.Skipped++
			continue
		}

		for _, m := range memEntries {
			if m.IsDir() || !strings.HasSuffix(m.Name(), ".md") {
				continue
			}
			base := strings.TrimSuffix(m.Name(), ".md")
			if base == "MEMORY" {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(memDir, m.Name()))
			if err != nil {
				continue
			}
			body := stripFrontmatter(string(raw))
			title := claudeNoteTitle(base)
			kind := "knowledge"

			if existing, _ := db.GetNoteBySourceTitle(a.db, pid, "claude", title); existing != nil {
				_ = db.UpdateNote(a.db, existing.ID, body)
				_ = db.UpdateNoteMeta(a.db, existing.ID, title, "", kind, existing.Pinned)
				result.Updated++
			} else {
				if _, err := db.CreateNoteEx(a.db, pid, title, body, "", kind, "claude"); err == nil {
					result.Synced++
				}
			}
		}
	}
	return result, nil
}

// claudeDisplayName extracts the final path segment from a Claude project dir name
// like "-Users-name-Workspace-ProjectName" -> "ProjectName".
func claudeDisplayName(dirName string) string {
	s := dirName
	if strings.HasPrefix(s, "-") {
		s = strings.TrimPrefix(s, "-")
	}
	parts := strings.Split(s, "-")
	return parts[len(parts)-1]
}

// claudeNoteTitle maps a Claude memory filename to a human-readable note title.
func claudeNoteTitle(filename string) string {
	switch filename {
	case "project":
		return "项目知识"
	case "user":
		return "用户信息"
	case "feedback":
		return "反馈记录"
	case "reference":
		return "参考信息"
	default:
		return filename
	}
}

// matchClaudeProject finds the GitBoard project id for a Claude memory dir,
// preferring exact name, then repo path suffix, then name containment.
func matchClaudeProject(displayName string, projects []db.Project, repos []db.Repository) int64 {
	lower := strings.ToLower(displayName)
	// 1. exact name
	for _, p := range projects {
		if p.Name == displayName {
			return p.ID
		}
	}
	// 2. repository path ending with /displayName
	for _, r := range repos {
		rp := strings.ToLower(r.Path)
		if strings.HasSuffix(rp, "/"+lower) || strings.HasSuffix(rp, "/"+lower+".git") {
			if r.ProjectID != nil {
				return *r.ProjectID
			}
		}
	}
	// 3. project name containment
	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.Name), lower) {
			return p.ID
		}
	}
	return 0
}

// stripFrontmatter removes a leading YAML frontmatter block (between --- markers)
// from a markdown string. If no frontmatter is present, the input is returned as-is.
func stripFrontmatter(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "---") {
		return s
	}
	rest := strings.TrimPrefix(s, "---")
	rest = strings.TrimLeft(rest, "\r\n")
	if idx := strings.Index(rest, "\n---"); idx >= 0 {
		rest = rest[idx+len("\n---"):]
		return strings.TrimLeft(rest, "\r\n")
	}
	// No closing marker; return the remainder.
	return rest
}

// HeatmapResponse holds heatmap data for the frontend.
type HeatmapResponse struct {
	Days []db.HeatmapDay `json:"days"`
}

// GetHeatmapData returns daily commit stats for the past year.
func (a *App) GetHeatmapData() *HeatmapResponse {
	endDate := stats.GetTodayDate()
	startDate := time.Now().AddDate(0, 0, -365).Format("2006-01-02")

	days, err := db.GetHeatmapData(a.db, startDate, endDate, a.gitUser)
	if err != nil {
		log.Printf("get heatmap error: %v", err)
		return &HeatmapResponse{Days: []db.HeatmapDay{}}
	}
	if days == nil {
		days = []db.HeatmapDay{}
	}
	return &HeatmapResponse{Days: days}
}

// StatusBarData holds real-time status information.
type StatusBarData struct {
	CurrentTime      string `json:"current_time"`
	LastCommitTime   string `json:"last_commit_time"`
	LastCommitRepo   string `json:"last_commit_repo"`
	LastCommitBranch string `json:"last_commit_branch"`
	LastCommitMsg    string `json:"last_commit_msg"`
}

// GetStatusBar returns current status bar information.
func (a *App) GetStatusBar() *StatusBarData {
	repos, _ := db.GetAllRepositories(a.db)
	repoPaths := make([]string, 0, len(repos))
	for _, r := range repos {
		repoPaths = append(repoPaths, r.Path)
	}

	data := &StatusBarData{
		CurrentTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	recent, err := stats.GetRecentCommit(repoPaths, a.gitUser)
	if err == nil && recent != nil {
		data.LastCommitTime = recent.Time
		data.LastCommitRepo = filepath.Base(recent.Repo)
		data.LastCommitBranch = recent.Branch
		data.LastCommitMsg = recent.Message
	}

	return data
}

// -- Summary Bind method --

// GetTodoCounts returns incomplete and total todo counts per project.
func (a *App) GetTodoCounts() []db.TodoCount {
	counts, err := db.GetTodoCounts(a.db)
	if err != nil {
		log.Printf("get todo counts error: %v", err)
		return nil
	}
	if counts == nil {
		counts = []db.TodoCount{}
	}
	return counts
}

// GetNoteCounts returns the count of notes per project.
func (a *App) GetNoteCounts() []db.NoteCount {
	counts, err := db.GetNoteCounts(a.db)
	if err != nil {
		log.Printf("get note counts error: %v", err)
		return nil
	}
	if counts == nil {
		counts = []db.NoteCount{}
	}
	return counts
}

// SearchHit is the unified search result type exposed to the frontend.
type SearchHit = db.SearchHit

// SearchNotes searches note content/title/tags across all projects,
// returning ranked hits with context snippets.
func (a *App) SearchNotes(query string) []SearchHit {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	results, err := db.SearchNotes(a.db, query)
	if err != nil {
		log.Printf("search notes error: %v", err)
		return nil
	}
	if results == nil {
		results = []db.SearchHit{}
	}
	return results
}

// SearchAll searches notes and todos together, returning ranked unified hits.
func (a *App) SearchAll(query string) []SearchHit {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	results, err := db.SearchAll(a.db, query)
	if err != nil {
		log.Printf("search all error: %v", err)
		return nil
	}
	if results == nil {
		results = []db.SearchHit{}
	}
	return results
}

// ToggleStar flips the starred status of a project.
func (a *App) ToggleStar(projectID int64) (bool, error) {
	return db.ToggleProjectStar(a.db, projectID)
}

// --- helpers (not exposed to frontend) ---

func (a *App) refreshAllStatsWithCancel(ctx context.Context) {
	repos, err := db.GetAllRepositories(a.db)
	if err != nil {
		return
	}

	startDate := time.Now().AddDate(0, 0, -365).Format("2006-01-02")
	endDate := stats.GetTodayDate()

	for _, repo := range repos {
		select {
		case <-ctx.Done():
			log.Printf("stats refresh cancelled")
			return
		default:
		}

		allEntries, err := stats.QueryStatsRange(repo.Path, startDate, endDate, "")
		if err == nil && allEntries != nil {
			for _, e := range allEntries {
				if e.FilesChanged > 0 || e.LinesAdded > 0 || e.LinesDeleted > 0 {
					_ = db.UpsertDailyStat(a.db, repo.ID, e.Date, "all",
						e.FilesChanged, e.LinesAdded, e.LinesDeleted)
				}
			}
		}

		if a.gitUser != "" {
			myEntries, err := stats.QueryStatsRange(repo.Path, startDate, endDate, a.gitUser)
			if err == nil && myEntries != nil {
				for _, e := range myEntries {
					if e.FilesChanged > 0 || e.LinesAdded > 0 || e.LinesDeleted > 0 {
						_ = db.UpsertDailyStat(a.db, repo.ID, e.Date, a.gitUser,
							e.FilesChanged, e.LinesAdded, e.LinesDeleted)
					}
				}
			}
		}
	}
}

func (a *App) refreshProjectStats(projectID int64, date string) {
	repos, err := db.GetRepositoriesByProjectID(a.db, projectID)
	if err != nil {
		return
	}
	for _, repo := range repos {
		allResult, err := stats.QueryStats(repo.Path, date, "")
		if err != nil {
			continue
		}
		if err := db.UpsertDailyStat(a.db, repo.ID, date, "all",
			allResult.FilesChanged, allResult.LinesAdded, allResult.LinesDeleted); err != nil {
			log.Printf("upsert daily stat error: %v", err)
		}
		if a.gitUser != "" {
			myResult, err := stats.QueryStats(repo.Path, date, a.gitUser)
			if err != nil {
				continue
			}
			if err := db.UpsertDailyStat(a.db, repo.ID, date, a.gitUser,
				myResult.FilesChanged, myResult.LinesAdded, myResult.LinesDeleted); err != nil {
				log.Printf("upsert daily stat error: %v", err)
			}
		}
	}
}
