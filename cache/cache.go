package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"repo-explanation/config"
	"repo-explanation/internal/openai"
)

// Cache handles caching of analysis results
type Cache struct {
	config *config.Config
}

// CacheEntry represents a cached analysis result
type CacheEntry struct {
	ContentHash string      `json:"content_hash"`
	Timestamp   time.Time   `json:"timestamp"`
	Result      interface{} `json:"result"`
}

// NewCache creates a new cache instance
func NewCache(cfg *config.Config) *Cache {
	return &Cache{config: cfg}
}

// GetFileSummary retrieves cached file summary if available and valid
func (c *Cache) GetFileSummary(filepath, content string) (*openai.FileSummary, bool) {
	if !c.config.Cache.Enabled {
		return nil, false
	}

	hash := c.hashContent(content)
	cacheFile := c.getFileCachePath(filepath, "file")
	
	entry, err := c.loadCacheEntry(cacheFile)
	if err != nil {
		return nil, false
	}

	// Check if hash matches and entry is not expired
	if entry.ContentHash != hash || c.isExpired(entry.Timestamp) {
		return nil, false
	}

	// Convert result to FileSummary
	resultBytes, err := json.Marshal(entry.Result)
	if err != nil {
		return nil, false
	}

	var summary openai.FileSummary
	if err := json.Unmarshal(resultBytes, &summary); err != nil {
		return nil, false
	}

	return &summary, true
}

// SetFileSummary caches a file summary
func (c *Cache) SetFileSummary(filepath, content string, summary *openai.FileSummary) error {
	if !c.config.Cache.Enabled {
		return nil
	}

	hash := c.hashContent(content)
	cacheFile := c.getFileCachePath(filepath, "file")
	
	entry := CacheEntry{
		ContentHash: hash,
		Timestamp:   time.Now(),
		Result:      summary,
	}

	return c.saveCacheEntry(cacheFile, entry)
}

// GetFolderSummary retrieves cached folder summary
func (c *Cache) GetFolderSummary(folderPath string, fileSummaries map[string]openai.FileSummary) (*openai.FolderSummary, bool) {
	if !c.config.Cache.Enabled {
		return nil, false
	}

	hash := c.hashFileSummaries(fileSummaries)
	cacheFile := c.getFileCachePath(folderPath, "folder")
	
	entry, err := c.loadCacheEntry(cacheFile)
	if err != nil {
		return nil, false
	}

	if entry.ContentHash != hash || c.isExpired(entry.Timestamp) {
		return nil, false
	}

	resultBytes, err := json.Marshal(entry.Result)
	if err != nil {
		return nil, false
	}

	var summary openai.FolderSummary
	if err := json.Unmarshal(resultBytes, &summary); err != nil {
		return nil, false
	}

	return &summary, true
}

// SetFolderSummary caches a folder summary
func (c *Cache) SetFolderSummary(folderPath string, fileSummaries map[string]openai.FileSummary, summary *openai.FolderSummary) error {
	if !c.config.Cache.Enabled {
		return nil
	}

	hash := c.hashFileSummaries(fileSummaries)
	cacheFile := c.getFileCachePath(folderPath, "folder")
	
	entry := CacheEntry{
		ContentHash: hash,
		Timestamp:   time.Now(),
		Result:      summary,
	}

	return c.saveCacheEntry(cacheFile, entry)
}

// GetProjectSummary retrieves cached project summary
func (c *Cache) GetProjectSummary(projectPath string, folderSummaries map[string]openai.FolderSummary) (*openai.ProjectSummary, bool) {
	if !c.config.Cache.Enabled {
		return nil, false
	}

	// Use URL-based cache path if projectPath looks like a URL, otherwise use traditional path-based
	var cacheFile string
	var hash string
	
	if strings.HasPrefix(projectPath, "http") {
		// URL-based caching - use only URL as cache key for stability
		hash = c.hashContent(projectPath + "_stable") // Add stable suffix for cache versioning
		safeFilename := c.getSafeFilenameFromURL(projectPath)
		urlHash := c.hashContent(projectPath)
		filename := fmt.Sprintf("%s_project_%s.json", safeFilename, urlHash[:8])
		cacheFile = filepath.Join(c.config.Cache.Directory, filename)
	} else {
		// Traditional path-based caching - use content hash
		hash = c.hashFolderSummaries(folderSummaries)
		cacheFile = c.getFileCachePath(projectPath, "project")
	}
	
	entry, err := c.loadCacheEntry(cacheFile)
	if err != nil {
		return nil, false
	}

	// For URL-based caching, only check expiration (ignore content hash variations)
	// For path-based caching, check both hash and expiration
	if strings.HasPrefix(projectPath, "http") {
		if c.isExpired(entry.Timestamp) {
			return nil, false
		}
	} else {
		if entry.ContentHash != hash || c.isExpired(entry.Timestamp) {
			return nil, false
		}
	}

	resultBytes, err := json.Marshal(entry.Result)
	if err != nil {
		return nil, false
	}

	var summary openai.ProjectSummary
	if err := json.Unmarshal(resultBytes, &summary); err != nil {
		return nil, false
	}

	return &summary, true
}

// SetProjectSummary caches a project summary
func (c *Cache) SetProjectSummary(projectPath string, folderSummaries map[string]openai.FolderSummary, summary *openai.ProjectSummary) error {
	if !c.config.Cache.Enabled {
		return nil
	}

	// Use URL-based cache path if projectPath looks like a URL, otherwise use traditional path-based
	var cacheFile string
	var hash string
	
	if strings.HasPrefix(projectPath, "http") {
		// URL-based caching - use only URL as cache key for stability
		hash = c.hashContent(projectPath + "_stable") // Add stable suffix for cache versioning
		safeFilename := c.getSafeFilenameFromURL(projectPath)
		urlHash := c.hashContent(projectPath)
		filename := fmt.Sprintf("%s_project_%s.json", safeFilename, urlHash[:8])
		cacheFile = filepath.Join(c.config.Cache.Directory, filename)
	} else {
		// Traditional path-based caching - use content hash
		hash = c.hashFolderSummaries(folderSummaries)
		cacheFile = c.getFileCachePath(projectPath, "project")
	}
	
	entry := CacheEntry{
		ContentHash: hash,
		Timestamp:   time.Now(),
		Result:      summary,
	}

	return c.saveCacheEntry(cacheFile, entry)
}

// GetRepositoryDetails retrieves cached detailed repository analysis
func (c *Cache) GetRepositoryDetails(repositoryURL string, folderSummaries map[string]openai.FolderSummary, fileSummaries map[string]openai.FileSummary, importantFiles map[string]string) (*openai.RepositoryAnalysis, bool) {
	if !c.config.Cache.Enabled {
		return nil, false
	}

	// Create composite hash from all inputs
	hash := c.hashRepositoryDetailsInputs(folderSummaries, fileSummaries, importantFiles)
	cacheFile := c.getRepositoryDetailsCachePath(repositoryURL)
	
	entry, err := c.loadCacheEntry(cacheFile)
	if err != nil {
		return nil, false
	}

	// Check if hash matches and entry is not expired
	if entry.ContentHash != hash || c.isExpired(entry.Timestamp) {
		return nil, false
	}

	// Convert result to RepositoryAnalysis
	resultBytes, err := json.Marshal(entry.Result)
	if err != nil {
		return nil, false
	}

	var analysis openai.RepositoryAnalysis
	if err := json.Unmarshal(resultBytes, &analysis); err != nil {
		return nil, false
	}

	return &analysis, true
}

// SetRepositoryDetails caches detailed repository analysis
func (c *Cache) SetRepositoryDetails(repositoryURL string, folderSummaries map[string]openai.FolderSummary, fileSummaries map[string]openai.FileSummary, importantFiles map[string]string, analysis *openai.RepositoryAnalysis) error {
	if !c.config.Cache.Enabled {
		return nil
	}

	hash := c.hashRepositoryDetailsInputs(folderSummaries, fileSummaries, importantFiles)
	cacheFile := c.getRepositoryDetailsCachePath(repositoryURL)
	
	entry := CacheEntry{
		ContentHash: hash,
		Timestamp:   time.Now(),
		Result:      analysis,
	}

	return c.saveCacheEntry(cacheFile, entry)
}

// ClearCache removes all cached entries
func (c *Cache) ClearCache() error {
	return os.RemoveAll(c.config.Cache.Directory)
}

// hashContent creates a hash of content for cache key
func (c *Cache) hashContent(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// hashFileSummaries creates a hash of file summaries for cache key
func (c *Cache) hashFileSummaries(summaries map[string]openai.FileSummary) string {
	data, _ := json.Marshal(summaries)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// hashFolderSummaries creates a hash of folder summaries for cache key
func (c *Cache) hashFolderSummaries(summaries map[string]openai.FolderSummary) string {
	data, _ := json.Marshal(summaries)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// hashRepositoryDetailsInputs creates a hash for all repository details inputs
func (c *Cache) hashRepositoryDetailsInputs(folderSummaries map[string]openai.FolderSummary, fileSummaries map[string]openai.FileSummary, importantFiles map[string]string) string {
	type compositeInput struct {
		FolderSummaries map[string]openai.FolderSummary `json:"folder_summaries"`
		FileSummaries   map[string]openai.FileSummary   `json:"file_summaries"`
		ImportantFiles  map[string]string               `json:"important_files"`
	}
	
	input := compositeInput{
		FolderSummaries: folderSummaries,
		FileSummaries:   fileSummaries,
		ImportantFiles:  importantFiles,
	}
	
	data, _ := json.Marshal(input)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// getFileCachePath generates cache file path
func (c *Cache) getFileCachePath(originalPath, cacheType string) string {
	// Create safe filename from path
	safeFilename := filepath.Base(originalPath)
	if safeFilename == "." || safeFilename == "/" {
		safeFilename = "root"
	}
	
	// Add hash of full path to avoid collisions
	pathHash := c.hashContent(originalPath)
	filename := fmt.Sprintf("%s_%s_%s.json", safeFilename, cacheType, pathHash[:8])
	
	return filepath.Join(c.config.Cache.Directory, filename)
}

// getRepositoryDetailsCachePath generates cache file path for repository details
func (c *Cache) getRepositoryDetailsCachePath(repositoryURL string) string {
	// Create safe filename from repository URL
	safeFilename := c.getSafeFilenameFromURL(repositoryURL)
	
	// Add hash of full URL to avoid collisions
	urlHash := c.hashContent(repositoryURL)
	filename := fmt.Sprintf("%s_details_%s.json", safeFilename, urlHash[:8])
	
	return filepath.Join(c.config.Cache.Directory, filename)
}

// getSafeFilenameFromURL creates a safe filename from repository URL
func (c *Cache) getSafeFilenameFromURL(url string) string {
	// Extract owner/repo from GitHub URL
	// e.g., https://github.com/owner/repo -> owner-repo
	url = strings.TrimSuffix(url, ".git")
	if strings.HasPrefix(url, "https://github.com/") {
		parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("%s-%s", parts[0], parts[1])
		}
	}
	
	// Fallback: use domain and sanitize
	if strings.HasPrefix(url, "http") {
		parts := strings.Split(url, "/")
		if len(parts) >= 3 {
			domain := strings.Replace(parts[2], ".", "-", -1)
			if len(parts) >= 5 {
				return fmt.Sprintf("%s-%s-%s", domain, parts[3], parts[4])
			}
			return domain
		}
	}
	
	// Ultimate fallback: sanitize the whole URL
	safe := strings.NewReplacer(
		"/", "-",
		":", "-",
		".", "-",
		"?", "-",
		"&", "-",
		"=", "-",
		" ", "_",
	).Replace(url)
	
	// Limit length
	if len(safe) > 50 {
		safe = safe[:50]
	}
	
	return safe
}

// loadCacheEntry loads cache entry from file
func (c *Cache) loadCacheEntry(filePath string) (*CacheEntry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// saveCacheEntry saves cache entry to file
func (c *Cache) saveCacheEntry(filePath string, entry CacheEntry) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// isExpired checks if cache entry is expired
func (c *Cache) isExpired(timestamp time.Time) bool {
	return time.Since(timestamp) > c.config.GetCacheTTL()
}
