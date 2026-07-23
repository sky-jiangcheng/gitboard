package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitboard/internal/db"
)

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

// stripFrontmatter removes a leading YAML frontmatter block (between --- markers
// on their own lines) from a markdown string. If no frontmatter is present, the
// input is returned as-is. Only considers the very first two lines for each marker
// to avoid being fooled by horizontal rules later in the text.
func stripFrontmatter(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "---") {
		return s
	}
	// Find the end of the first line containing the opening "---"
	idx := strings.Index(s, "\n")
	if idx < 0 {
		return s // no newline after "---", not valid frontmatter
	}
	// Check if the first line is exactly "---" (optional trailing whitespace)
	firstLine := strings.TrimSpace(s[:idx])
	if firstLine != "---" {
		return s
	}
	// Look for closing "---" on a line by itself
	remainder := s[idx+1:]
	lines := strings.Split(remainder, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			// Return everything after this closing marker line
			if i+1 < len(lines) {
				return strings.TrimLeft(strings.Join(lines[i+1:], "\n"), "\r\n")
			}
			return ""
		}
	}
	// No closing marker found; return the remainder as-is.
	return strings.TrimLeft(remainder, "\r\n")
}
