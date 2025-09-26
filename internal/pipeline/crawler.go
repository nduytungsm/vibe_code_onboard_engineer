package pipeline

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"repo-explanation/config"
	"repo-explanation/internal/gitignore"
)

// FileInfo represents a discovered file
type FileInfo struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Size         int64  `json:"size"`
	Extension    string `json:"extension"`
	IsDir        bool   `json:"is_dir"`
}

// Crawler discovers and filters files in a directory
type Crawler struct {
	config    *config.Config
	gitIgnore *gitignore.GitIgnore
	basePath  string
}

// NewCrawler creates a new file crawler
func NewCrawler(cfg *config.Config, basePath string) (*Crawler, error) {
	// Load .gitignore files
	gitIgnore := gitignore.NewGitIgnore()
	
	// Load default patterns
	gitIgnore.LoadDefault()
	
	// Load .gitignore from base path if it exists
	gitignorePath := filepath.Join(basePath, ".gitignore")
	if err := gitIgnore.LoadFromFile(gitignorePath); err != nil {
		return nil, fmt.Errorf("failed to load .gitignore: %v", err)
	}
	
	return &Crawler{
		config:    cfg,
		gitIgnore: gitIgnore,
		basePath:  basePath,
	}, nil
}

// CrawlFiles discovers all relevant files in the directory tree
func (c *Crawler) CrawlFiles() ([]FileInfo, error) {
	var files []FileInfo
	
	err := filepath.WalkDir(c.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip files/directories we can't read
			return nil
		}
		
		// Get relative path from base
		relPath, err := filepath.Rel(c.basePath, path)
		if err != nil {
			return nil
		}
		
		// Normalize path separators for gitignore
		normalizedPath := filepath.ToSlash(relPath)
		
		// Skip root directory
		if relPath == "." {
			return nil
		}
		
		// Check if ignored by gitignore
		if c.gitIgnore.IsIgnored(normalizedPath, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir // Skip entire directory
			}
			return nil // Skip file
		}
		
		// Enhanced filtering: Skip unimportant directories entirely
		if d.IsDir() && c.isUnimportantDirectory(normalizedPath) {
			return fs.SkipDir
		}
		
		// Enhanced filtering: Skip unimportant files
		if !d.IsDir() && c.isUnimportantFile(normalizedPath) {
			return nil
		}
		
		// Get file info
		info, err := d.Info()
		if err != nil {
			return nil
		}
		
		// Skip directories for file processing
		if d.IsDir() {
			return nil
		}
		
		// Check file size limit
		maxSize := int64(c.config.FileProcessing.MaxFileSizeMB) * 1024 * 1024
		if info.Size() > maxSize {
			return nil
		}
		
		// Check if file extension is supported
		if !c.config.IsFileSupported(path) {
			return nil
		}
		
		// Check if it's a secret file
		if c.config.IsSecretFile(path) {
			return nil
		}
		
		fileInfo := FileInfo{
			Path:         path,
			RelativePath: relPath,
			Size:         info.Size(),
			Extension:    strings.ToLower(filepath.Ext(path)),
			IsDir:        false,
		}
		
		files = append(files, fileInfo)
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %v", err)
	}
	
	return files, nil
}

// ReadFile reads the content of a file
func (c *Crawler) ReadFile(fileInfo FileInfo) (string, error) {
	data, err := os.ReadFile(fileInfo.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %v", fileInfo.Path, err)
	}
	
	content := string(data)
	
	// Redact secrets if enabled
	if c.config.Security.RedactSecrets {
		content = c.redactSecrets(content)
	}
	
	return content, nil
}

// redactSecrets removes potential secrets from content
func (c *Crawler) redactSecrets(content string) string {
	// List of patterns that might contain secrets
	secretPatterns := []struct {
		pattern     string
		replacement string
	}{
		{`api_key\s*[:=]\s*["']([^"']+)["']`, `api_key: "[REDACTED]"`},
		{`password\s*[:=]\s*["']([^"']+)["']`, `password: "[REDACTED]"`},
		{`secret\s*[:=]\s*["']([^"']+)["']`, `secret: "[REDACTED]"`},
		{`token\s*[:=]\s*["']([^"']+)["']`, `token: "[REDACTED]"`},
		{`key\s*[:=]\s*["']([^"']+)["']`, `key: "[REDACTED]"`},
		// Add more patterns as needed
	}
	
	for _, sp := range secretPatterns {
		// This is a simplified redaction - in production, you'd want more sophisticated regex
		if strings.Contains(strings.ToLower(content), strings.Split(sp.pattern, `\s`)[0]) {
			// Simple replacement - in production use proper regex
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				lower := strings.ToLower(line)
				if strings.Contains(lower, "api_key") || strings.Contains(lower, "password") || 
				   strings.Contains(lower, "secret") || strings.Contains(lower, "token") {
					// Replace the value part with [REDACTED]
					if strings.Contains(line, ":") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							lines[i] = parts[0] + ": [REDACTED]"
						}
					} else if strings.Contains(line, "=") {
						parts := strings.SplitN(line, "=", 2)
						if len(parts) == 2 {
							lines[i] = parts[0] + "=[REDACTED]"
						}
					}
				}
			}
			content = strings.Join(lines, "\n")
		}
	}
	
	return content
}

// GetFileStats returns statistics about discovered files
func (c *Crawler) GetFileStats(files []FileInfo) map[string]interface{} {
	stats := make(map[string]interface{})
	
	totalFiles := len(files)
	totalSize := int64(0)
	extensionCounts := make(map[string]int)
	
	for _, file := range files {
		totalSize += file.Size
		extensionCounts[file.Extension]++
	}
	
	stats["total_files"] = totalFiles
	stats["total_size_mb"] = float64(totalSize) / (1024 * 1024)
	stats["extensions"] = extensionCounts
	
	return stats
}

// isUnimportantDirectory checks if a directory should be skipped for performance
func (c *Crawler) isUnimportantDirectory(path string) bool {
	// Convert to lowercase for case-insensitive matching
	lowerPath := strings.ToLower(path)
	
	// Skip common unimportant directories that don't provide architectural value
	unimportantDirs := []string{
		// Build outputs and dependencies
		"node_modules", "vendor", "target", "build", "dist", "out", "bin",
		".next", ".nuxt", "__pycache__", ".pytest_cache", "coverage",
		
		// IDE and editor files  
		".vscode", ".idea", ".eclipse", ".settings",
		
		// Version control and CI
		".git", ".svn", ".hg", ".github/workflows", ".gitlab-ci",
		
		// Logs and temporary files
		"logs", "tmp", "temp", ".tmp", ".cache",
		
		// Documentation that doesn't affect architecture (keep important docs)
		"docs/api", "docs/generated", "documentation/auto",
		
		// Test artifacts and reports
		"test-results", "coverage-reports", "jest-coverage", ".nyc_output",
		
		// Package manager artifacts
		".pnpm-store", ".yarn/cache", ".npm",
		
		// Language-specific build artifacts
		"cmake-build-debug", "cmake-build-release", "obj", "debug", "release",
	}
	
	for _, skipDir := range unimportantDirs {
		if strings.Contains(lowerPath, skipDir) {
			return true
		}
	}
	
	// Skip deep nested paths (likely auto-generated)
	if strings.Count(path, "/") > 8 {
		return true
	}
	
	return false
}

// isUnimportantFile checks if a file should be skipped for performance
func (c *Crawler) isUnimportantFile(path string) bool {
	lowerPath := strings.ToLower(path)
	filename := strings.ToLower(filepath.Base(path))
	
	// Skip files that don't provide architectural insight
	unimportantFiles := []string{
		// Lock files and dependencies
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "composer.lock",
		"pipfile.lock", "poetry.lock", "cargo.lock", "go.sum",
		
		// Build and compiled files
		".map", ".min.js", ".min.css", "bundle.js", "bundle.css",
		
		// IDE and editor files
		".ds_store", "thumbs.db", "desktop.ini", ".swp", ".swo",
		
		// Test files (keep main test files, skip detailed test data)
		".test.json", ".spec.json", "__snapshots__", ".coverage",
		
		// Generated files
		"generated.go", "auto_generated", ".pb.go", ".gen.go",
		
		// Documentation that doesn't affect code architecture
		"changelog", "license", "authors", "contributors", "code_of_conduct",
		
		// Configuration files that are often repetitive
		".env.example", ".env.template", ".env.sample",
	}
	
	for _, skipFile := range unimportantFiles {
		if strings.Contains(filename, skipFile) {
			return true
		}
	}
	
	// Skip very large files that are likely data/assets
	if strings.HasSuffix(lowerPath, ".sql") && strings.Contains(lowerPath, "seed") {
		return true
	}
	if strings.HasSuffix(lowerPath, ".json") && strings.Contains(lowerPath, "fixture") {
		return true
	}
	
	// Skip binary-like files even if they have text extensions
	binaryPatterns := []string{".woff", ".ttf", ".eot", ".ico", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".pdf"}
	for _, pattern := range binaryPatterns {
		if strings.HasSuffix(lowerPath, pattern) {
			return true
		}
	}
	
	return false
}
