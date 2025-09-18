package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	hash := c.hashFolderSummaries(folderSummaries)
	cacheFile := c.getFileCachePath(projectPath, "project")
	
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

	hash := c.hashFolderSummaries(folderSummaries)
	cacheFile := c.getFileCachePath(projectPath, "project")
	
	entry := CacheEntry{
		ContentHash: hash,
		Timestamp:   time.Now(),
		Result:      summary,
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
