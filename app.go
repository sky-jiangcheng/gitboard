package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gitboard/internal/db"
	"gitboard/internal/grouper"
	"gitboard/internal/scanner"
	"gitboard/internal/stats"
)

// App is the main application struct whose public methods are exposed to the
// frontend via Wails Bind. The ctx is set during OnStartup.
type App struct {
	ctx     context.Context
	db      *sql.DB
	gitUser string
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
}

// shutdown is called when the application exits.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// Health returns a health-check payload for the frontend.
func (a *App) Health() map[string]interface{} {
	if err := a.db.Ping(); err != nil {
		return map[string]interface{}{"status": "error", "message": "database unavailable"}
	}
	return map[string]interface{}{"status": "ok", "version": "1.0.0"}
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
func (a *App) GetProjects(date string) []ProjectResponse {
	if date == "" {
		date = stats.GetYesterdayDate()
	}
	if err := stats.ValidateDate(date); err != nil {
		log.Printf("invalid date: %v", err)
		return nil
	}

	projects, err := db.GetAllProjects(a.db)
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

// UpdateProjectLevel adjusts a project's grouping level up or down.
func (a *App) UpdateProjectLevel(id int64, direction string) (*LevelUpdateResult, error) {
	if direction != "up" && direction != "down" {
		return nil, fmt.Errorf("direction must be 'up' or 'down'")
	}
	project, err := db.GetProjectByID(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	newLevel := project.LevelOverride
	if direction == "up" {
		newLevel++
	} else {
		newLevel--
	}
	if err := db.UpdateProjectLevel(a.db, id, newLevel); err != nil {
		return nil, fmt.Errorf("failed to update project level")
	}
	return &LevelUpdateResult{Success: true, NewLevel: newLevel}, nil
}

// ScanResult holds the result of a scan operation.
type ScanResult struct {
	Success    bool `json:"success"`
	ReposFound int  `json:"repos_found"`
	Projects   int  `json:"projects"`
}

// TriggerScan runs a full repository scan and re-groups projects.
func (a *App) TriggerScan() (*ScanResult, error) {
	depthStr, _ := db.GetConfig(a.db, "scan_depth")
	maxDepth, _ := strconv.Atoi(depthStr)
	if maxDepth <= 0 || maxDepth > 10 {
		maxDepth = 5
	}

	roots, _ := db.GetScanRoots(a.db)
	repos, err := scanner.ScanRepositories(roots, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("scan failed")
	}

	groups := grouper.GroupRepositories(repos)

	if err := db.DeleteAllRepositories(a.db); err != nil {
		log.Printf("delete repos error: %v", err)
	}
	if err := db.DeleteAllProjects(a.db); err != nil {
		log.Printf("delete projects error: %v", err)
	}

	for _, group := range groups {
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

	a.refreshAllStats(stats.GetTodayDate())
	return &ScanResult{Success: true, ReposFound: len(repos), Projects: len(groups)}, nil
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
	allowed := map[string]bool{"daily_code_standard": true, "scan_depth": true}
	if !allowed[key] {
		return fmt.Errorf("unknown config key: %s", key)
	}
	// Validate the value is a valid integer string
	if _, err := strconv.Atoi(value); err != nil {
		return fmt.Errorf("config value must be a number")
	}
	return db.SetConfig(a.db, key, value)
}

// UpdateScanRoots replaces the entire scan root list.
func (a *App) UpdateScanRoots(scanRoots []string) error {
	existing, _ := db.GetScanRoots(a.db)
	for _, root := range existing {
		if err := db.RemoveScanRoot(a.db, root); err != nil {
			log.Printf("remove scan root error: %v", err)
		}
	}
	for _, root := range scanRoots {
		if root != "" && !strings.Contains(root, "\x00") {
			if err := db.AddScanRoot(a.db, root); err != nil {
				log.Printf("add scan root error: %v", err)
			}
		}
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

// --- helpers (not exposed to frontend) ---

func (a *App) refreshAllStats(date string) {
	repos, err := db.GetAllRepositories(a.db)
	if err != nil {
		return
	}
	for _, repo := range repos {
		allResult, err := stats.QueryStats(repo.Path, date, "")
		if err != nil {
			continue
		}
		if allResult.FilesChanged > 0 || allResult.LinesAdded > 0 || allResult.LinesDeleted > 0 {
			if err := db.UpsertDailyStat(a.db, repo.ID, date, "all",
				allResult.FilesChanged, allResult.LinesAdded, allResult.LinesDeleted); err != nil {
				log.Printf("upsert daily stat error: %v", err)
			}
		}
		if a.gitUser != "" {
			myResult, err := stats.QueryStats(repo.Path, date, a.gitUser)
			if err != nil {
				continue
			}
			if myResult.FilesChanged > 0 || myResult.LinesAdded > 0 || myResult.LinesDeleted > 0 {
				if err := db.UpsertDailyStat(a.db, repo.ID, date, a.gitUser,
					myResult.FilesChanged, myResult.LinesAdded, myResult.LinesDeleted); err != nil {
					log.Printf("upsert daily stat error: %v", err)
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
