package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"git-dashboard/internal/db"
	"git-dashboard/internal/grouper"
	"git-dashboard/internal/scanner"
	"git-dashboard/internal/stats"
)

// Server holds the HTTP server dependencies.
type Server struct {
	db       *sql.DB
	mux      *http.ServeMux
	gitUser  string
}

// NewServer creates a new HTTP server with all routes registered.
func NewServer(database *sql.DB, gitUser string) *Server {
	s := &Server{
		db:      database,
		mux:     http.NewServeMux(),
		gitUser: gitUser,
	}
	s.registerRoutes()
	return s
}

// Handler returns the http.Handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// registerRoutes registers all API and static file routes.
func (s *Server) registerRoutes() {
	// API routes
	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/projects/", s.handleProjectByID)
	s.mux.HandleFunc("/api/scan", s.handleScan)
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/summary", s.handleSummary)
}

// RegisterStatic serves embedded frontend files.
func (s *Server) RegisterStatic(staticFS fs.FS) {
	fileServer := http.FileServer(http.FS(staticFS))
	s.mux.Handle("/", fileServer)
}

// handleProjects handles GET /api/projects
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = stats.GetYesterdayDate()
	}

	projects, err := db.GetAllProjects(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get projects: "+err.Error())
		return
	}

	type projectResponse struct {
		db.Project
		RepoCount    int  `json:"repo_count"`
		TotalAdded   int  `json:"total_added"`
		TotalDeleted int  `json:"total_deleted"`
		MyAdded      int  `json:"my_added"`
		MyDeleted    int  `json:"my_deleted"`
		MyFiles      int  `json:"my_files"`
		IsWorkday    bool `json:"is_workday"`
		BelowStandard bool `json:"below_standard"`
	}

	var result []projectResponse
	codeStdStr, _ := db.GetConfig(s.db, "daily_code_standard")
	codeStd, _ := strconv.Atoi(codeStdStr)
	if codeStd == 0 {
		codeStd = 500
	}
	isWorkday := stats.IsWorkday(date)

	for _, p := range projects {
		statsList, _ := db.GetStatsByProject(s.db, p.ID, date)
		repos, _ := db.GetRepositoriesByProjectID(s.db, p.ID)

		pr := projectResponse{
			Project:    p,
			RepoCount:  len(repos),
			IsWorkday:  isWorkday,
		}

		for _, st := range statsList {
			pr.TotalAdded += st.LinesAdded
			pr.TotalDeleted += st.LinesDeleted
			if st.Author == s.gitUser {
				pr.MyAdded += st.LinesAdded
				pr.MyDeleted += st.LinesDeleted
				pr.MyFiles += st.FilesChanged
			}
		}

		pr.BelowStandard = isWorkday && pr.MyAdded < codeStd

		result = append(result, pr)
	}

	writeJSON(w, result)
}

// handleProjectByID handles routes under /api/projects/{id}/...
func (s *Server) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "project id required")
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	// Check for sub-routes
	subRoute := ""
	if len(parts) > 1 {
		subRoute = parts[1]
	}

	switch {
	case subRoute == "stats" && r.Method == http.MethodGet:
		s.handleProjectStats(w, r, id)
	case subRoute == "level" && r.Method == http.MethodPost:
		s.handleProjectLevel(w, r, id)
	case subRoute == "" && r.Method == http.MethodGet:
		s.handleProjectDetail(w, r, id)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

// handleProjectDetail handles GET /api/projects/:id
func (s *Server) handleProjectDetail(w http.ResponseWriter, r *http.Request, id int64) {
	project, err := db.GetProjectByID(s.db, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	repos, _ := db.GetRepositoriesByProjectID(s.db, id)

	type repoInfo struct {
		db.Repository
		Stats []db.DailyStat `json:"stats"`
	}

	var repoList []repoInfo
	for _, repo := range repos {
		statsList, _ := db.GetStatsByRepositoryAndDate(s.db, repo.ID, "")
		ri := repoInfo{Repository: repo, Stats: statsList}
		repoList = append(repoList, ri)
	}

	type detailResponse struct {
		*db.Project
		Repos []repoInfo `json:"repos"`
	}

	writeJSON(w, detailResponse{Project: project, Repos: repoList})
}

// handleProjectStats handles GET /api/projects/:id/stats?date=YYYY-MM-DD
func (s *Server) handleProjectStats(w http.ResponseWriter, r *http.Request, id int64) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = stats.GetYesterdayDate()
	}

	statsList, err := db.GetStatsByProject(s.db, id, date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	// If no cached stats, trigger real-time query
	if len(statsList) == 0 {
		refreshProjectStats(s.db, id, date, s.gitUser)
		statsList, _ = db.GetStatsByProject(s.db, id, date)
	}

	writeJSON(w, statsList)
}

// handleProjectLevel handles POST /api/projects/:id/level
func (s *Server) handleProjectLevel(w http.ResponseWriter, r *http.Request, id int64) {
	type levelRequest struct {
		Direction string `json:"direction"` // "up" or "down"
	}

	var req levelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	project, err := db.GetProjectByID(s.db, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// Get all repos to rebuild groups
	allRepos, err := db.GetAllRepositories(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get repositories")
		return
	}

	// Convert to scanner.RepoInfo
	var repoInfos []scanner.RepoInfo
	for _, repo := range allRepos {
		repoInfos = append(repoInfos, scanner.RepoInfo{Path: repo.Path})
	}

	// Build current group for this project
	repos, _ := db.GetRepositoriesByProjectID(s.db, id)
	var groupRepoInfos []scanner.RepoInfo
	for _, repo := range repos {
		groupRepoInfos = append(groupRepoInfos, scanner.RepoInfo{Path: repo.Path})
	}

	currentGroup := &grouper.ProjectGroup{
		Name:    project.Name,
		RootPath: project.RootPath,
		Repos:   groupRepoInfos,
	}

	var newLevel int
	switch req.Direction {
	case "up":
		newLevel = project.LevelOverride + 1
	case "down":
		newLevel = project.LevelOverride - 1
	default:
		writeError(w, http.StatusBadRequest, "direction must be 'up' or 'down'")
		return
	}

	if err := db.UpdateProjectLevel(s.db, id, newLevel); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project level")
		return
	}

	_ = currentGroup // used for level adjustment logic in future
	writeJSON(w, map[string]interface{}{"success": true, "new_level": newLevel})
}

// handleScan handles POST /api/scan
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	depthStr, _ := db.GetConfig(s.db, "scan_depth")
	maxDepth, _ := strconv.Atoi(depthStr)
	if maxDepth == 0 {
		maxDepth = 5
	}

	roots, _ := db.GetScanRoots(s.db)
	repos, err := scanner.ScanRepositories(roots, maxDepth)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
		return
	}

	// Group repositories
	groups := grouper.GroupRepositories(repos)

	// Clear existing data
	db.DeleteAllRepositories(s.db)
	db.DeleteAllProjects(s.db)

	// Persist groups and repos
	for _, group := range groups {
		projectID, err := db.UpsertProject(s.db, group.Name, group.RootPath, 0, group.IsAutoGrouped)
		if err != nil {
			continue
		}
		for _, repo := range group.Repos {
			db.UpsertRepository(s.db, repo.Path, projectID)
		}
	}

	// Auto-stats for today
	today := stats.GetTodayDate()
	refreshAllStats(s.db, today, s.gitUser)

	writeJSON(w, map[string]interface{}{
		"success":      true,
		"repos_found":  len(repos),
		"projects":     len(groups),
	})
}

// handleConfig handles GET/PUT /api/config
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		configs, err := db.GetAllConfigs(s.db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get configs")
			return
		}
		roots, _ := db.GetScanRoots(s.db)
		result := map[string]interface{}{
			"config": configs,
			"scan_roots": roots,
		}
		writeJSON(w, result)

	case http.MethodPut:
		type configUpdate struct {
			Key       string   `json:"key"`
			Value     string   `json:"value"`
			ScanRoots []string `json:"scan_roots"`
		}

		// Read body once into bytes
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read request body")
			return
		}

		var updates []configUpdate
		if err := json.Unmarshal(bodyBytes, &updates); err != nil {
			var single configUpdate
			if err2 := json.Unmarshal(bodyBytes, &single); err2 != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			updates = []configUpdate{single}
		}

		for _, u := range updates {
			if u.Key != "" {
				db.SetConfig(s.db, u.Key, u.Value)
			}
		}

		// Handle scan_roots if provided
		if len(updates) > 0 && len(updates[0].ScanRoots) > 0 {
			// Replace all scan roots
			existing, _ := db.GetScanRoots(s.db)
			for _, root := range existing {
				db.RemoveScanRoot(s.db, root)
			}
			for _, root := range updates[0].ScanRoots {
				db.AddScanRoot(s.db, root)
			}
		}

		writeJSON(w, map[string]interface{}{"success": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSummary handles GET /api/summary?date=YYYY-MM-DD
func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = stats.GetYesterdayDate()
	}

	allStats, err := db.GetStatsByDate(s.db, date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	type summaryResponse struct {
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

	summary := summaryResponse{
		Date:      date,
		IsWorkday: stats.IsWorkday(date),
	}

	// Count unique repos
	repoSet := make(map[int64]bool)
	for _, st := range allStats {
		repoSet[st.RepositoryID] = true
		summary.TotalFiles += st.FilesChanged
		summary.TotalAdded += st.LinesAdded
		summary.TotalDeleted += st.LinesDeleted
		if st.Author == s.gitUser {
			summary.MyAdded += st.LinesAdded
			summary.MyDeleted += st.LinesDeleted
			summary.MyFiles += st.FilesChanged
		}
	}
	summary.RepoCount = len(repoSet)

	writeJSON(w, summary)
}

// refreshAllStats queries stats for all repositories for a given date and caches results.
func refreshAllStats(database *sql.DB, date string, gitUser string) {
	repos, err := db.GetAllRepositories(database)
	if err != nil {
		log.Printf("refreshAllStats: failed to get repos: %v", err)
		return
	}

	for _, repo := range repos {
		// Stats for all users
		allResult, err := stats.QueryStats(repo.Path, date, "")
		if err != nil {
			log.Printf("refreshAllStats: failed to query %s: %v", repo.Path, err)
			continue
		}
		if allResult.FilesChanged > 0 || allResult.LinesAdded > 0 || allResult.LinesDeleted > 0 {
			db.UpsertDailyStat(database, repo.ID, date, "all", allResult.FilesChanged, allResult.LinesAdded, allResult.LinesDeleted)
		}

		// Stats for current user
		if gitUser != "" {
			myResult, err := stats.QueryStats(repo.Path, date, gitUser)
			if err != nil {
				continue
			}
			if myResult.FilesChanged > 0 || myResult.LinesAdded > 0 || myResult.LinesDeleted > 0 {
				db.UpsertDailyStat(database, repo.ID, date, gitUser, myResult.FilesChanged, myResult.LinesAdded, myResult.LinesDeleted)
			}
		}
	}
}

// refreshProjectStats refreshes stats for a specific project.
func refreshProjectStats(database *sql.DB, projectID int64, date string, gitUser string) {
	repos, err := db.GetRepositoriesByProjectID(database, projectID)
	if err != nil {
		return
	}
	for _, repo := range repos {
		allResult, err := stats.QueryStats(repo.Path, date, "")
		if err != nil {
			continue
		}
		db.UpsertDailyStat(database, repo.ID, date, "all", allResult.FilesChanged, allResult.LinesAdded, allResult.LinesDeleted)

		if gitUser != "" {
			myResult, err := stats.QueryStats(repo.Path, date, gitUser)
			if err != nil {
				continue
			}
			db.UpsertDailyStat(database, repo.ID, date, gitUser, myResult.FilesChanged, myResult.LinesAdded, myResult.LinesDeleted)
		}
	}
}

// -- helpers --

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Format port helper
func FormatAddr(port string) string {
	return fmt.Sprintf(":%s", port)
}
