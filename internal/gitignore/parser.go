package gitignore

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GitIgnore represents a parsed .gitignore file
type GitIgnore struct {
	patterns []pattern
}

type pattern struct {
	regex     *regexp.Regexp
	negate    bool
	dirOnly   bool
	absolute  bool
	original  string
}

// NewGitIgnore creates a new GitIgnore parser
func NewGitIgnore() *GitIgnore {
	return &GitIgnore{
		patterns: make([]pattern, 0),
	}
}

// LoadFromFile loads patterns from a .gitignore file
func (g *GitIgnore) LoadFromFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // .gitignore doesn't exist, that's fine
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if err := g.AddPattern(line); err != nil {
			// Log but don't fail on invalid patterns
			continue
		}
	}

	return scanner.Err()
}

// AddPattern adds a single gitignore pattern
func (g *GitIgnore) AddPattern(line string) error {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	p := pattern{
		original: line,
	}

	// Check for negation
	if strings.HasPrefix(line, "!") {
		p.negate = true
		line = line[1:]
	}

	// Check for directory-only pattern
	if strings.HasSuffix(line, "/") {
		p.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	// Check for absolute path
	if strings.HasPrefix(line, "/") {
		p.absolute = true
		line = line[1:]
	}

	// Convert gitignore pattern to regex
	regexPattern := g.convertToRegex(line)
	
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return err
	}
	
	p.regex = regex
	g.patterns = append(g.patterns, p)
	
	return nil
}

// IsIgnored checks if a file path should be ignored
func (g *GitIgnore) IsIgnored(filePath string, isDir bool) bool {
	// Normalize path separators
	filePath = filepath.ToSlash(filePath)
	
	ignored := false
	
	for _, p := range g.patterns {
		if g.matchesPattern(p, filePath, isDir) {
			ignored = !p.negate
		}
	}
	
	return ignored
}

// matchesPattern checks if a path matches a specific pattern
func (g *GitIgnore) matchesPattern(p pattern, filePath string, isDir bool) bool {
	// Directory-only patterns only match directories
	if p.dirOnly && !isDir {
		return false
	}
	
	if p.absolute {
		// Match from root
		return p.regex.MatchString(filePath)
	} else {
		// Match basename or full path
		basename := filepath.Base(filePath)
		return p.regex.MatchString(basename) || p.regex.MatchString(filePath)
	}
}

// convertToRegex converts gitignore pattern to regex
func (g *GitIgnore) convertToRegex(pattern string) string {
	// Escape regex special characters except * and ?
	pattern = regexp.QuoteMeta(pattern)
	
	// Convert gitignore wildcards to regex
	pattern = strings.ReplaceAll(pattern, `\*\*`, ".*")  // ** matches any path
	pattern = strings.ReplaceAll(pattern, `\*`, "[^/]*") // * matches within path segment
	pattern = strings.ReplaceAll(pattern, `\?`, ".")     // ? matches single character
	
	// Anchor the pattern
	pattern = "^" + pattern + "$"
	
	return pattern
}

// LoadDefault loads common ignore patterns
func (g *GitIgnore) LoadDefault() {
	defaultPatterns := []string{
		// Version control
		".git/",
		".svn/",
		".hg/",
		".bzr/",
		
		// IDE and editors
		".vscode/",
		".idea/",
		"*.swp",
		"*.swo",
		"*~",
		".DS_Store",
		"Thumbs.db",
		
		// Dependencies and build artifacts
		"node_modules/",
		"vendor/",
		"build/",
		"dist/",
		"target/",
		"*.o",
		"*.so",
		"*.dylib",
		"*.dll",
		"*.exe",
		
		// Logs and temporary files
		"*.log",
		"*.tmp",
		"*.temp",
		".cache/",
		
		// Binary files
		"*.jpg",
		"*.jpeg",
		"*.png",
		"*.gif",
		"*.svg",
		"*.pdf",
		"*.zip",
		"*.tar",
		"*.gz",
		"*.rar",
		"*.7z",
		
		// Large data files
		"*.db",
		"*.sqlite",
		"*.sqlite3",
	}
	
	for _, pattern := range defaultPatterns {
		g.AddPattern(pattern)
	}
}
