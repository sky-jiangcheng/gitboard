package stats

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Result holds commit statistics for a single query.
type Result struct {
	FilesChanged int
	LinesAdded   int
	LinesDeleted int
}

// QueryTimeout is the maximum time allowed for a single git log query.
const QueryTimeout = 30 * time.Second

// QueryStats runs git log --shortstat for the given repository, date, and optional author.
// Returns aggregated statistics.
func QueryStats(repoPath, date, author string) (*Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	args := []string{
		"log",
		"--since=" + date + " 00:00:00",
		"--until=" + date + " 23:59:59",
		"--first-parent",
		"--pretty=tformat:",
		"--shortstat",
	}

	if author != "" {
		args = append(args, "--author="+author)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath

	out, err := cmd.Output()
	if err != nil {
		// If no commits, git returns empty output but may exit with non-zero
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("query timed out after %v", QueryTimeout)
		}
		// Empty repo or no commits is not an error
		return &Result{}, nil
	}

	return parseShortStat(string(out))
}

// QueryMultiBranch runs QueryStats for multiple branches and aggregates results.
func QueryMultiBranch(repoPath, date string, branches []string) (*Result, error) {
	result := &Result{}
	seen := make(map[string]bool) // deduplicate by commit hash

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}

		r, err := QueryStatsForBranch(repoPath, date, branch)
		if err != nil {
			continue
		}

		// Check for commit uniqueness by hash
		for _, hash := range r.commits {
			if !seen[hash] {
				seen[hash] = true
				result.FilesChanged += r.FilesChanged
				result.LinesAdded += r.LinesAdded
				result.LinesDeleted += r.LinesDeleted
			}
		}
	}

	return result, nil
}

// branchResult holds per-branch stats with commit hashes for dedup.
type branchResult struct {
	FilesChanged int
	LinesAdded   int
	LinesDeleted int
	commits      []string
}

// QueryStatsForBranch queries stats for a specific branch.
func QueryStatsForBranch(repoPath, date, branch string) (*branchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	args := []string{
		"log", branch,
		"--since=" + date + " 00:00:00",
		"--until=" + date + " 23:59:59",
		"--first-parent",
		"--pretty=format:%H",
		"--shortstat",
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("branch query timed out after %v", QueryTimeout)
		}
		return &branchResult{}, nil
	}

	return parseShortStatWithCommits(string(out))
}

// parseShortStat parses git log --shortstat output into a Result.
// Output format:
//
//	3 files changed, 15 insertions(+), 8 deletions(-)
//	1 file changed, 5 insertions(+)
func parseShortStat(output string) (*Result, error) {
	result := &Result{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip commit hash lines and blank lines
		if len(line) == 40 && isHex(line) {
			continue
		}

		files, added, deleted := parseStatLine(line)
		result.FilesChanged += files
		result.LinesAdded += added
		result.LinesDeleted += deleted
	}

	return result, nil
}

// parseShortStatWithCommits also collects commit hashes for deduplication.
func parseShortStatWithCommits(output string) (*branchResult, error) {
	result := &branchResult{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// If it's a commit hash (40 hex chars), collect it
		if len(line) == 40 && isHex(line) {
			result.commits = append(result.commits, line)
			continue
		}

		files, added, deleted := parseStatLine(line)
		result.FilesChanged += files
		result.LinesAdded += added
		result.LinesDeleted += deleted
	}

	return result, nil
}

// parseStatLine parses a single shortstat line like:
// "3 files changed, 15 insertions(+), 8 deletions(-)"
func parseStatLine(line string) (files, added, deleted int) {
	parts := strings.Split(line, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)

		if len(fields) < 2 {
			continue
		}

		val, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		keyword := fields[1]
		switch {
		case strings.HasPrefix(keyword, "file"):
			files = val
		case strings.HasPrefix(keyword, "insertion"):
			added = val
		case strings.HasPrefix(keyword, "deletion"):
			deleted = val
		}
	}
	return
}

// isHex checks if a string contains only hexadecimal characters.
func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// GetTodayDate returns today's date in YYYY-MM-DD format.
func GetTodayDate() string {
	return time.Now().Format("2006-01-02")
}

// GetYesterdayDate returns yesterday's date in YYYY-MM-DD format.
func GetYesterdayDate() string {
	return time.Now().AddDate(0, 0, -1).Format("2006-01-02")
}

// IsWorkday returns true if the given date (YYYY-MM-DD) is a weekday (Mon-Fri).
func IsWorkday(date string) bool {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return true // assume workday on error
	}
	day := t.Weekday()
	return day != time.Saturday && day != time.Sunday
}
