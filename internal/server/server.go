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

	"gitboard/internal/db"
	"gitboard/internal/grouper"
	"gitboard/internal/scanner"
	"gitboard/internal/stats"
)

// MaxRequestBodySize limits incoming JSON request bodies to 1MB.
const MaxRequestBodySize = 1 << 20 // 1 MB

// Server holds the HTTP server dependencies.
type Server struct {
	db      *sql.DB
	mux     *http.ServeMux
	gitUser string
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

// Handler returns the http.Handler for the server, wrapped with middleware.
func (s *Server) Handler() http.Handler {
	return withMiddleware(s.mux)
}

// registerRoutes registers all API and static file routes.
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/projects/", s.handleProjectByID)
	s.mux.HandleFunc("/api/scan", s.handleScan)
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/summary", s.handleSummary)
}

// RegisterStatic serves embedded frontend files with SPA fallback.
// Non-file requests (e.g., /project/1) fall through to index.html for React Router.
func (s *Server) RegisterStatic(staticFS fs.FS) {
	s.mux.Handle("/", spaHandler(http.FS(staticFS)))
}

// spaHandler wraps a file server with SPA fallback: if a requested file doesn't
// exist, serve index.html instead so that React Router can handle the route.
func spaHandler(fs http.FileSystem) http.Handler {
	fileServer := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested file
		f, err := fs.Open(r.URL.Path)
		if err != nil {
			// File doesn't exist — serve index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close() //nolint:errcheck // close file handle after checking existence

		// Serve the actual file
		fileServer.ServeHTTP(w, r)
	})
}

// withMiddleware wraps a handler with CORS, recovery, and logging middleware.
func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Recovery from panics
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC recovered: %v", rec)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// limitedReader limits the request body to MaxRequestBodySize.
func limitedReader(r *http.Request) io.Reader {
	return io.LimitReader(r.Body, MaxRequestBodySize+1)
}

// readJSONBody reads and parses JSON from the request body with size limit.
func readJSONBody(r *http.Request, v interface{}) error {
	if r.Header.Get("Content-Type") != "" &&
		!strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("content-type must be application/json")
	}

	bodyBytes, err := io.ReadAll(limitedReader(r))
	if err != nil {
		return fmt.Errorf("failed to read request body")
	}
	if len(bodyBytes) > MaxRequestBodySize {
		return fmt.Errorf("request body too large (max 1MB)")
	}
	return json.Unmarshal(bodyBytes, v)
}

// handleHealth handles GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
	})
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
	if err := stats.ValidateDate(date); err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format")
		return
	}

	projects, err := db.GetAllProjects(s.db)
	if err != nil {
		log.Printf("get projects error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load projects")
		return
	}

	type projectResponse struct {
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

	var result []projectResponse
	codeStdStr, _ := db.GetConfig(s.db, "daily_code_standard")
	codeStd, _ := strconv.Atoi(codeStdStr)
	if codeStd == 0 {
		codeStd = 500
	}
	isWorkday := stats.IsWorkday(date)

	for _, p := range projects {
		statsList, _ := db.GetStatsByProject(s.db, p.ID, date)
		// Refresh stats on demand if no cached data exists for this date
		if len(statsList) == 0 {
			repos, _ := db.GetRepositoriesByProjectID(s.db, p.ID)
			if len(repos) > 0 {
				refreshProjectStats(s.db, p.ID, date, s.gitUser)
				statsList, _ = db.GetStatsByProject(s.db, p.ID, date)
			}
		}
		repos, _ := db.GetRepositoriesByProjectID(s.db, p.ID)

		pr := projectResponse{
			Project:   p,
			RepoCount: len(repos),
			IsWorkday: isWorkday,
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
		if statsList == nil {
			statsList = []db.DailyStat{}
		}
		ri := repoInfo{Repository: repo, Stats: statsList}
		repoList = append(repoList, ri)
	}

	type detailResponse struct {
		*db.Project
		Repos []repoInfo `json:"repos"`
	}

	writeJSON(w, detailResponse{Project: project, Repos: repoList})
}

func (s *Server) handleProjectStats(w http.ResponseWriter, r *http.Request, id int64) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = stats.GetYesterdayDate()
	}
	if err := stats.ValidateDate(date); err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format")
		return
	}

	statsList, err := db.GetStatsByProject(s.db, id, date)
	if err != nil {
		log.Printf("get stats error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}

	if len(statsList) == 0 {
		refreshProjectStats(s.db, id, date, s.gitUser)
		statsList, _ = db.GetStatsByProject(s.db, id, date)
	}

	writeJSON(w, statsList)
}

func (s *Server) handleProjectLevel(w http.ResponseWriter, r *http.Request, id int64) {
	type levelRequest struct {
		Direction string `json:"direction"`
	}

	var req levelRequest
	if err := readJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Direction != "up" && req.Direction != "down" {
		writeError(w, http.StatusBadRequest, "direction must be 'up' or 'down'")
		return
	}

	project, err := db.GetProjectByID(s.db, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var newLevel int
	if req.Direction == "up" {
		newLevel = project.LevelOverride + 1
	} else {
		newLevel = project.LevelOverride - 1
	}

	if err := db.UpdateProjectLevel(s.db, id, newLevel); err != nil {
		log.Printf("update project level error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update project level")
		return
	}

	writeJSON(w, map[string]interface{}{"success": true, "new_level": newLevel})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	depthStr, _ := db.GetConfig(s.db, "scan_depth")
	maxDepth, _ := strconv.Atoi(depthStr)
	if maxDepth <= 0 || maxDepth > 10 {
		maxDepth = 5
	}

	roots, _ := db.GetScanRoots(s.db)
	repos, err := scanner.ScanRepositories(roots, maxDepth)
	if err != nil {
		log.Printf("scan error: %v", err)
		writeError(w, http.StatusInternalServerError, "scan failed")
		return
	}

	groups := grouper.GroupRepositories(repos)

	// Clear and persist in a single logical operation
	if err := db.DeleteAllRepositories(s.db); err != nil {
		log.Printf("delete repos error: %v", err)
	}
	if err := db.DeleteAllProjects(s.db); err != nil {
		log.Printf("delete projects error: %v", err)
	}

	for _, group := range groups {
		projectID, err := db.UpsertProject(s.db, group.Name, group.RootPath, 0, group.IsAutoGrouped)
		if err != nil {
			log.Printf("upsert project error: %v", err)
			continue
		}
		for _, repo := range group.Repos {
			if err := db.UpsertRepository(s.db, repo.Path, projectID); err != nil {
				log.Printf("upsert repo error: %v", err)
			}
		}
	}

	today := stats.GetTodayDate()
	refreshAllStats(s.db, today, s.gitUser)

	writeJSON(w, map[string]interface{}{
		"success":     true,
		"repos_found": len(repos),
		"projects":    len(groups),
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		configs, err := db.GetAllConfigs(s.db)
		if err != nil {
			log.Printf("get configs error: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to load config")
			return
		}
		roots, _ := db.GetScanRoots(s.db)
		writeJSON(w, map[string]interface{}{
			"config":     configs,
			"scan_roots": roots,
		})

	case http.MethodPut:
		type configUpdate struct {
			Key       string   `json:"key"`
			Value     string   `json:"value"`
			ScanRoots []string `json:"scan_roots"`
		}

		var updates []configUpdate
		bodyBytes, err := io.ReadAll(limitedReader(r))
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		if len(bodyBytes) > MaxRequestBodySize {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}

		if err := json.Unmarshal(bodyBytes, &updates); err != nil {
			var single configUpdate
			if err2 := json.Unmarshal(bodyBytes, &single); err2 != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON")
				return
			}
			updates = []configUpdate{single}
		}

		for _, u := range updates {
			if u.Key != "" {
				// Whitelist allowed config keys
				if u.Key == "daily_code_standard" || u.Key == "scan_depth" {
					if _, err := strconv.Atoi(u.Value); err != nil {
						writeError(w, http.StatusBadRequest, "config value must be a number")
						return
					}
					if err := db.SetConfig(s.db, u.Key, u.Value); err != nil {
						log.Printf("set config error: %v", err)
					}
				}
			}
		}

		// Collect all scan roots from updates
		var allScanRoots []string
		for _, u := range updates {
			allScanRoots = append(allScanRoots, u.ScanRoots...)
		}

		if len(allScanRoots) > 0 {
			existing, _ := db.GetScanRoots(s.db)
			for _, root := range existing {
				if err := db.RemoveScanRoot(s.db, root); err != nil {
					log.Printf("remove scan root error: %v", err)
				}
			}
			for _, root := range allScanRoots {
				// Basic path validation
				if root != "" && !strings.Contains(root, "\x00") {
					if err := db.AddScanRoot(s.db, root); err != nil {
						log.Printf("add scan root error: %v", err)
					}
				}
			}
		}

		writeJSON(w, map[string]interface{}{"success": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = stats.GetYesterdayDate()
	}
	if err := stats.ValidateDate(date); err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format")
		return
	}

	allStats, err := db.GetStatsByDate(s.db, date)
	if err != nil {
		log.Printf("summary error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load summary")
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

func refreshAllStats(database *sql.DB, date string, gitUser string) {
	repos, err := db.GetAllRepositories(database)
	if err != nil {
		return
	}

	for _, repo := range repos {
		allResult, err := stats.QueryStats(repo.Path, date, "")
		if err != nil {
			continue
		}
		if allResult.FilesChanged > 0 || allResult.LinesAdded > 0 || allResult.LinesDeleted > 0 {
			if err := db.UpsertDailyStat(database, repo.ID, date, "all", allResult.FilesChanged, allResult.LinesAdded, allResult.LinesDeleted); err != nil {
				log.Printf("upsert daily stat error: %v", err)
			}
		}

		if gitUser != "" {
			myResult, err := stats.QueryStats(repo.Path, date, gitUser)
			if err != nil {
				continue
			}
			if myResult.FilesChanged > 0 || myResult.LinesAdded > 0 || myResult.LinesDeleted > 0 {
				if err := db.UpsertDailyStat(database, repo.ID, date, gitUser, myResult.FilesChanged, myResult.LinesAdded, myResult.LinesDeleted); err != nil {
					log.Printf("upsert daily stat error: %v", err)
				}
			}
		}
	}
}

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
		if err := db.UpsertDailyStat(database, repo.ID, date, "all", allResult.FilesChanged, allResult.LinesAdded, allResult.LinesDeleted); err != nil {
			log.Printf("upsert daily stat error: %v", err)
		}

		if gitUser != "" {
			myResult, err := stats.QueryStats(repo.Path, date, gitUser)
			if err != nil {
				continue
			}
			if err := db.UpsertDailyStat(database, repo.ID, date, gitUser, myResult.FilesChanged, myResult.LinesAdded, myResult.LinesDeleted); err != nil {
				log.Printf("upsert daily stat error: %v", err)
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("json encode error in writeError: %v", err)
	}
}

func FormatAddr(port string) string {
	return fmt.Sprintf(":%s", port)
}
