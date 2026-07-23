package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gitboard/internal/db"
	"gitboard/internal/stats"
)

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
		data.LastCommitRepo = pathBase(recent.Repo)
		data.LastCommitBranch = recent.Branch
		data.LastCommitMsg = recent.Message
	}

	return data
}

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

// ExportProjectStats returns all stats in CSV format suitable for spreadsheet import.
func (a *App) ExportProjectStats(projectID int64) string {
	statsList, err := db.GetStatsByProject(a.db, projectID, "")
	if err != nil {
		return ""
	}
	if len(statsList) == 0 {
		a.refreshProjectStats(projectID, "")
		statsList, _ = db.GetStatsByProject(a.db, projectID, "")
	}

	var sb strings.Builder
	sb.WriteString("date,author,files_changed,lines_added,lines_deleted\n")
	for _, st := range statsList {
		sb.WriteString(fmt.Sprintf("%s,%s,%d,%d,%d\n",
			st.StatDate, st.Author, st.FilesChanged, st.LinesAdded, st.LinesDeleted))
	}
	return sb.String()
}

// ExportHeatmapCSV returns heatmap data as CSV for spreadsheet use.
func (a *App) ExportHeatmapCSV() string {
	days := a.GetHeatmapData().Days
	var sb strings.Builder
	sb.WriteString("date,lines_added,lines_deleted,commits\n")
	for _, d := range days {
		sb.WriteString(fmt.Sprintf("%s,%d,%d,%d\n", d.Date, d.LinesAdded, d.LinesDeleted, d.Commits))
	}
	return sb.String()
}
