package grouper

import (
	"path/filepath"
	"sort"
	"strings"

	"git-dashboard/internal/scanner"
)

// ProjectGroup represents a grouped project containing one or more repositories.
type ProjectGroup struct {
	Name         string             // display name
	RootPath     string             // project root directory
	Repos        []scanner.RepoInfo // repositories in this group
	IsAutoGrouped bool              // whether this was auto-grouped
}

// GroupRepositories groups discovered repositories into projects based on directory structure.
//
// Rules:
// 1. If a parent directory contains a single git repo, the parent becomes the project.
// 2. If a parent directory contains multiple git repos, they are grouped under the parent.
// 3. If a parent directory is itself a git repo and contains sub-repos, the parent repo
//    is kept as a separate project from its children.
func GroupRepositories(repos []scanner.RepoInfo) []ProjectGroup {
	if len(repos) == 0 {
		return nil
	}

	// Sort by depth (shallow first) to process parent repos first
	sorted := make([]scanner.RepoInfo, len(repos))
	copy(sorted, repos)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Depth != sorted[j].Depth {
			return sorted[i].Depth < sorted[j].Depth
		}
		return sorted[i].Path < sorted[j].Path
	})

	// Build a set of all repo paths for quick lookup
	repoSet := make(map[string]bool)
	for _, r := range repos {
		repoSet[r.Path] = true
	}

	// Track which repos have been assigned to a group
	assigned := make(map[string]bool)
	var groups []ProjectGroup

	for _, repo := range sorted {
		if assigned[repo.Path] {
			continue
		}

		parentDir := filepath.Dir(repo.Path)

		// Check if parent is also a git repo (Rule 3)
		if repoSet[parentDir] && !assigned[parentDir] {
			// Parent is a repo, make it a separate project
			parentGroup := ProjectGroup{
				Name:          filepath.Base(parentDir),
				RootPath:      parentDir,
				Repos:         []scanner.RepoInfo{{Path: parentDir, Depth: 0}},
				IsAutoGrouped: true,
			}
			groups = append(groups, parentGroup)
			assigned[parentDir] = true

			// Now handle the child repo
			childGroup := groupSiblings(parentDir, repoSet, assigned, sorted)
			if len(childGroup.Repos) > 0 {
				groups = append(groups, childGroup)
			}
			continue
		}

		// Collect all repos sharing the same parent (Rules 1 & 2)
		group := groupSiblings(parentDir, repoSet, assigned, sorted)
		if len(group.Repos) > 0 {
			groups = append(groups, group)
		}
	}

	return groups
}

// groupSiblings groups all unassigned repos that share the same parent directory.
func groupSiblings(parentDir string, repoSet map[string]bool, assigned map[string]bool, repos []scanner.RepoInfo) ProjectGroup {
	group := ProjectGroup{
		Name:          filepath.Base(parentDir),
		RootPath:      parentDir,
		IsAutoGrouped: true,
	}

	// Handle edge case: if parentDir is "/" or root
	if parentDir == "/" || parentDir == "." || strings.HasSuffix(parentDir, ":\\") || strings.HasSuffix(parentDir, ":") {
		// For root-level repos, use the repo path itself
		for _, r := range repos {
			if !assigned[r.Path] && isDirectChild(parentDir, r.Path) {
				group.Repos = append(group.Repos, r)
				assigned[r.Path] = true
			}
		}
		if len(group.Repos) > 0 {
			group.Name = filepath.Base(group.Repos[0].Path)
			group.RootPath = group.Repos[0].Path
		}
		return group
	}

	for _, r := range repos {
		if assigned[r.Path] {
			continue
		}
		if filepath.Dir(r.Path) == parentDir {
			group.Repos = append(group.Repos, r)
			assigned[r.Path] = true
		}
	}

	return group
}

// isDirectChild checks if child is an immediate child of parent.
func isDirectChild(parent, child string) bool {
	dir := filepath.Dir(child)
	return dir == parent
}

// AdjustProjectLevelUp expands a project's scope to include all repos under the parent directory.
func AdjustProjectLevelUp(group *ProjectGroup, allRepos []scanner.RepoInfo) *ProjectGroup {
	parentDir := filepath.Dir(group.RootPath)
	newGroup := &ProjectGroup{
		Name:          filepath.Base(parentDir),
		RootPath:      parentDir,
		IsAutoGrouped: false,
	}

	for _, r := range allRepos {
		if strings.HasPrefix(r.Path, parentDir) {
			newGroup.Repos = append(newGroup.Repos, r)
		}
	}
	return newGroup
}

// AdjustProjectLevelDown splits a project into individual sub-projects.
func AdjustProjectLevelDown(group *ProjectGroup) []ProjectGroup {
	var groups []ProjectGroup
	for _, repo := range group.Repos {
		g := ProjectGroup{
			Name:          filepath.Base(repo.Path),
			RootPath:      repo.Path,
			Repos:         []scanner.RepoInfo{repo},
			IsAutoGrouped: false,
		}
		groups = append(groups, g)
	}
	return groups
}
