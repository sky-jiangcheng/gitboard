package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// RepoInfo holds information about a discovered git repository.
type RepoInfo struct {
	Path  string // absolute path to the repository
	Depth int    // depth from scan root
}

// MaxEntries is the maximum number of directories to scan before stopping.
const MaxEntries = 10000

// ScanRepositories recursively searches for .git directories within the given roots,
// up to the specified max depth. Returns a list of discovered repositories.
func ScanRepositories(roots []string, maxDepth int) ([]RepoInfo, error) {
	var repos []RepoInfo
	for _, root := range roots {
		found, err := scanRoot(root, maxDepth)
		if err != nil {
			// skip inaccessible roots
			continue
		}
		repos = append(repos, found...)
	}
	return repos, nil
}

// scanRoot scans a single root directory for git repositories.
func scanRoot(root string, maxDepth int) ([]RepoInfo, error) {
	var repos []RepoInfo
	entriesCount := 0

	// Check if root itself is a git repo
	rootGitDir := filepath.Join(root, ".git")
	if info, err := os.Stat(rootGitDir); err == nil && info.IsDir() {
		absPath, _ := filepath.Abs(root)
		repos = append(repos, RepoInfo{
			Path:  absPath,
			Depth: 0,
		})
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// permission denied, skip this entry
			if os.IsPermission(err) {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		// Calculate depth relative to root
		relPath, _ := filepath.Rel(root, path)
		if relPath == "." {
			return nil
		}

		depth := len(strings.Split(filepath.ToSlash(relPath), "/"))

		// Stop if exceeding max depth
		if depth > maxDepth {
			return filepath.SkipDir
		}

		entriesCount++
		if entriesCount > MaxEntries {
			return filepath.SkipAll
		}

		// Check for .git directory
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(path)
			repos = append(repos, RepoInfo{
				Path:  absPath,
				Depth: depth,
			})
			// Don't descend into git repositories
			return filepath.SkipDir
		}

		return nil
	})

	return repos, err
}
