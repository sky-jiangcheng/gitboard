package main

import (
	"fmt"
	"log"
	"strconv"

	"gitboard/internal/db"
	"gitboard/internal/knowledge"
	"gitboard/internal/stats"
)

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
					pathBase(repo.Path), repo.Path, project.LevelOverride-1,
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
		parentDir := pathDir(project.RootPath)
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
				if pathDir(sroot) == parentDir {
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
			if _, err := tx.Exec("UPDATE projects SET is_auto_grouped = 0, root_path = ?, name = ? WHERE id = ?", parentDir, pathBase(parentDir), id); err != nil {
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

// ToggleStar flips the starred status of a project.
func (a *App) ToggleStar(projectID int64) (bool, error) {
	return db.ToggleProjectStar(a.db, projectID)
}

// ProjectOverview is the mined-knowledge payload for a project detail page.
type ProjectOverview struct {
	ReadmeExcerpt string                  `json:"readme_excerpt"`
	TechStack     []knowledge.Tech        `json:"tech_stack"`
	Languages     []knowledge.LanguageStat `json:"languages"`
	RecentCommits []stats.RecentCommit    `json:"recent_commits"`
	Cached        bool                    `json:"cached"`
	Mining        bool                    `json:"mining,omitempty"`
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
			jsonUnmarshalSafe(meta.TechStack, &resp.TechStack)
			jsonUnmarshalSafe(meta.Languages, &resp.Languages)
			resp.ReadmeExcerpt = meta.ReadmeExcerpt
			resp.Cached = true
		}
	}

	// Mine fresh when no cache was found — trigger async so the API returns quickly.
	if !resp.Cached {
		resp.Mining = true
		go a.mineAndCacheAsync(projectID, cacheRepoID, project.RootPath, repos)
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

// mineAndCacheAsync runs knowledge mining in the background and caches the result.
func (a *App) mineAndCacheAsync(projectID, cacheRepoID int64, rootPath string, repos []db.Repository) {
	minePath := rootPath
	if minePath == "" && len(repos) > 0 {
		minePath = repos[0].Path
	}
	if minePath == "" {
		return
	}
	k, err := knowledge.Mine(minePath)
	if err != nil || k == nil {
		return
	}
	if cacheRepoID > 0 {
		ts, _ := marshalJSON(k.TechStack)
		ls, _ := marshalJSON(k.Languages)
		_ = db.UpsertRepoMeta(a.db, cacheRepoID, string(ts), k.ReadmeExcerpt, string(ls))
	}
}
