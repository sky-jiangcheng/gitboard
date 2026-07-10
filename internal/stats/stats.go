package stats

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
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

// datePattern validates YYYY-MM-DD format.
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// safeAuthorPattern allows only safe characters for git author matching.
// Allowed: letters, digits, space, dot, underscore, hyphen, at-sign.
var safeAuthorPattern = regexp.MustCompile(`^[a-zA-Z0-9 ._\-@]+$`)

// ValidateDate checks that the date string matches YYYY-MM-DD format.
func ValidateDate(date string) error {
	if !datePattern.MatchString(date) {
		return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", date)
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date: %s", date)
	}
	// Ensure the parsed date matches the input (catches things like "0000-00-00")
	if t.Format("2006-01-02") != date {
		return fmt.Errorf("invalid date: %s", date)
	}
	return nil
}

// ValidateAuthor checks that the author string contains only safe characters.
func ValidateAuthor(author string) error {
	if author == "" {
		return nil
	}
	if !safeAuthorPattern.MatchString(author) {
		return fmt.Errorf("invalid author name: contains unsafe characters")
	}
	return nil
}

// QueryStats runs git log --shortstat for the given repository, date, and optional author.
// Returns aggregated statistics. All user-supplied parameters are validated.
func QueryStats(repoPath, date, author string) (*Result, error) {
	if err := ValidateDate(date); err != nil {
		return nil, err
	}
	if err := ValidateAuthor(author); err != nil {
		return nil, err
	}

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

	//nolint:gosec // date and author are validated by ValidateDate/ValidateAuthor above
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("query timed out after %v", QueryTimeout)
		}
		return &Result{}, nil
	}

	return parseShortStat(string(out))
}

// QueryMultiBranch runs QueryStats for multiple branches and aggregates results.
func QueryMultiBranch(repoPath, date string, branches []string) (*Result, error) {
	if err := ValidateDate(date); err != nil {
		return nil, err
	}

	result := &Result{}
	seen := make(map[string]bool)

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" || !isSafeRefName(branch) {
			continue
		}

		r, err := QueryStatsForBranch(repoPath, date, branch)
		if err != nil {
			continue
		}

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

// safeRefPattern allows safe git ref names (branches, tags).
var safeRefPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/\-]*$`)

// isSafeRefName checks that a git ref name contains only safe characters.
func isSafeRefName(ref string) bool {
	if len(ref) == 0 || len(ref) > 255 {
		return false
	}
	return safeRefPattern.MatchString(ref)
}

// QueryStatsForBranch queries stats for a specific branch.
func QueryStatsForBranch(repoPath, date, branch string) (*branchResult, error) {
	if err := ValidateDate(date); err != nil {
		return nil, err
	}
	if !isSafeRefName(branch) {
		return nil, fmt.Errorf("invalid branch name")
	}

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

	//nolint:gosec // date and branch are validated by ValidateDate/isSafeRefName above
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
func parseShortStat(output string) (*Result, error) {
	result := &Result{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

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

// parseStatLine parses a single shortstat line.
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

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// RecentCommit holds information about the most recent commit.
type RecentCommit struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Repo    string `json:"repo"`
	Branch  string `json:"branch"`
}

// GetRecentCommit queries the most recent commit across all repositories.
func GetRecentCommit(repoPaths []string, filterAuthor string) (*RecentCommit, error) {
	var best *RecentCommit

	for _, repoPath := range repoPaths {
		args := []string{
			"log", "-1",
			"--pretty=format:%H%n%an%n%at%n%s%n%D",
		}
		if filterAuthor != "" {
			args = append(args, "--author="+filterAuthor)
		}

		ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repoPath
		out, err := cmd.Output()
		cancel()
		if err != nil {
			continue
		}

		lines := strings.Split(string(out), "\n")
		if len(lines) < 4 {
			continue
		}

		ts, _ := strconv.ParseInt(lines[2], 10, 64)
		if ts == 0 {
			continue
		}

		branch := ""
		if len(lines) >= 5 {
			branch = extractBranch(lines[4])
		}

		commitTime := time.Unix(ts, 0)
		if best == nil || commitTime.After(time.Unix(parseTimestamp(best.Time), 0)) {
			best = &RecentCommit{
				Time:    commitTime.Format("2006-01-02 15:04:05"),
				Author:  lines[1],
				Message: lines[3],
				Repo:    repoPath,
				Branch:  branch,
			}
		}
	}

	return best, nil
}

func extractBranch(refString string) string {
	if strings.Contains(refString, "HEAD -> ") {
		parts := strings.Split(refString, "HEAD -> ")
		if len(parts) > 1 {
			b := strings.Split(parts[1], ",")[0]
			return strings.TrimSpace(b)
		}
	}
	return ""
}

func parseTimestamp(s string) int64 {
	t, _ := time.Parse("2006-01-02 15:04:05", s)
	return t.Unix()
}

func GetTodayDate() string {
	return time.Now().Format("2006-01-02")
}

func GetYesterdayDate() string {
	return time.Now().AddDate(0, 0, -1).Format("2006-01-02")
}

func IsWorkday(date string) bool {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return true
	}
	day := t.Weekday()
	return day != time.Saturday && day != time.Sunday
}
