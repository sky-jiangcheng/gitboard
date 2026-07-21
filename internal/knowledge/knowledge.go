// Package knowledge mines descriptive knowledge out of a git repository's
// working tree: the README, the detected tech stack (from manifest files), and
// a coarse language breakdown by file extension. Results are cached by the
// caller (db.RepoMeta) so repeated dashboard loads do not re-walk the tree.
package knowledge

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Tech is a single detected technology entry.
type Tech struct {
	Name     string `json:"name"`
	Category string `json:"category"` // "language" | "framework" | "tool"
}

// LanguageStat is a language with its file count, used for the breakdown list.
type LanguageStat struct {
	Language string `json:"language"`
	Count    int    `json:"count"`
}

// RepoKnowledge is the aggregated, mineable knowledge for one repository.
type RepoKnowledge struct {
	ReadmeExcerpt string         `json:"readme_excerpt"`
	TechStack     []Tech         `json:"tech_stack"`
	Languages     []LanguageStat `json:"languages"`
}

// maxReadmeBytes bounds how much of a README we keep (enough to preview).
const maxReadmeBytes = 8 * 1024

// maxReadmeLines bounds the number of lines kept.
const maxReadmeLines = 200

// maxScanFiles bounds the language-counting walk so huge monorepos stay fast.
const maxScanFiles = 20000

// skipDirs are directory names we never descend into when counting languages.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"build": true, "target": true, ".venv": true, "venv": true, "__pycache__": true,
	".idea": true, ".vscode": true, "Pods": true, ".next": true, ".cache": true,
}

// manifestTech maps a top-level manifest filename to the tech it implies.
var manifestTech = map[string]Tech{
	"package.json":     {"JavaScript / TypeScript", "language"},
	"go.mod":            {"Go", "language"},
	"Cargo.toml":        {"Rust", "language"},
	"pom.xml":           {"Java (Maven)", "language"},
	"build.gradle":      {"Java (Gradle)", "language"},
	"build.gradle.kts":  {"Java (Gradle)", "language"},
	"requirements.txt":  {"Python", "language"},
	"pyproject.toml":    {"Python", "language"},
	"setup.py":          {"Python", "language"},
	"composer.json":     {"PHP", "language"},
	"Gemfile":           {"Ruby", "language"},
	"Package.swift":     {"Swift", "language"},
	"pubspec.yaml":      {"Dart / Flutter", "framework"},
	"mix.exs":           {"Elixir", "language"},
	"CMakeLists.txt":    {"C / C++", "language"},
	"docker-compose.yml": {"Docker Compose", "tool"},
	"Dockerfile":        {"Docker", "tool"},
}

// extLanguage maps a file extension to a language label.
var extLanguage = map[string]string{
	".go": "Go", ".js": "JavaScript", ".jsx": "JavaScript", ".mjs": "JavaScript",
	".ts": "TypeScript", ".tsx": "TypeScript",
	".py": "Python", ".rb": "Ruby", ".rs": "Rust",
	".java": "Java", ".kt": "Kotlin", ".scala": "Scala",
	".c": "C", ".h": "C", ".cpp": "C++", ".cc": "C++", ".hpp": "C++",
	".cs": "C#", ".php": "PHP", ".swift": "Swift",
	".m": "Objective-C", ".mm": "Objective-C++",
	".vue": "Vue", ".svelte": "Svelte",
	".sh": "Shell", ".bash": "Shell", ".zsh": "Shell",
	".lua": "Lua", ".ex": "Elixir", ".exs": "Elixir",
	".clj": "Clojure", ".dart": "Dart",
	".sql": "SQL", ".html": "HTML", ".css": "CSS", ".scss": "SCSS",
	".json": "JSON", ".yaml": "YAML", ".yml": "YAML", ".toml": "TOML",
	".md": "Markdown",
}

// ErrNotARepo is returned when the path is not an accessible directory.
var ErrNotARepo = errors.New("not an accessible repository directory")

// ExtractREADME finds and reads the repository's README, returning a bounded
// excerpt. Returns an empty string (no error) when no README is present.
func ExtractREADME(repoPath string) (string, error) {
	info, err := os.Stat(repoPath)
	if err != nil || !info.IsDir() {
		return "", ErrNotARepo
	}

	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return "", nil
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isReadme(name) {
			continue
		}
		f, err := os.Open(filepath.Join(repoPath, name))
		if err != nil {
			continue
		}
		defer f.Close()

		var sb strings.Builder
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 256*1024)
		lineNo := 0
		for scanner.Scan() {
			line := scanner.Text()
			sb.WriteString(line)
			sb.WriteByte('\n')
			lineNo++
			if sb.Len() >= maxReadmeBytes || lineNo >= maxReadmeLines {
				break
			}
		}
		return strings.TrimSpace(sb.String()), nil
	}
	return "", nil
}

// isReadme reports whether a filename is a README (case-insensitive, any extension).
func isReadme(name string) bool {
	base := strings.ToLower(name)
	if !strings.HasPrefix(base, "readme") {
		return false
	}
	// README, README.md, README.rst, README.txt ...
	return base == "readme" || strings.HasPrefix(base, "readme.")
}

// DetectTechStack inspects top-level manifest files and returns detected tech.
func DetectTechStack(repoPath string) ([]Tech, error) {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, ErrNotARepo
	}

	var techs []Tech
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if t, ok := manifestTech[name]; ok {
			key := t.Name
			if !seen[key] {
				seen[key] = true
				techs = append(techs, t)
			}
		}
	}

	// C# projects: any *.csproj at top level.
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".csproj") {
			if !seen["C#"] {
				seen["C#"] = true
				techs = append(techs, Tech{Name: "C#", Category: "language"})
			}
		}
	}

	sort.Slice(techs, func(i, j int) bool { return techs[i].Name < techs[j].Name })
	return techs, nil
}

// DetectLanguages walks the repo counting files per language extension, skipping
// dependency/build directories. Returns the top languages by file count.
func DetectLanguages(repoPath string) ([]LanguageStat, error) {
	counts := make(map[string]int)
	scanned := 0

	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		scanned++
		if scanned > maxScanFiles {
			return filepath.SkipAll
		}
		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := extLanguage[ext]; ok {
			counts[lang]++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	stats := make([]LanguageStat, 0, len(counts))
	for lang, n := range counts {
		stats = append(stats, LanguageStat{Language: lang, Count: n})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count != stats[j].Count {
			return stats[i].Count > stats[j].Count
		}
		return stats[i].Language < stats[j].Language
	})

	// Keep the top 8 to avoid noise.
	if len(stats) > 8 {
		stats = stats[:8]
	}
	return stats, nil
}

// Mine aggregates README, tech stack, and language breakdown for a repository.
func Mine(repoPath string) (*RepoKnowledge, error) {
	readme, err := ExtractREADME(repoPath)
	if err != nil && err != ErrNotARepo {
		// A non-repo path yields empty knowledge, not a hard failure for callers.
		readme = ""
	}

	techs, err := DetectTechStack(repoPath)
	if err != nil {
		techs = nil
	}
	langs, err := DetectLanguages(repoPath)
	if err != nil {
		langs = nil
	}

	return &RepoKnowledge{
		ReadmeExcerpt: readme,
		TechStack:     techs,
		Languages:     langs,
	}, nil
}
