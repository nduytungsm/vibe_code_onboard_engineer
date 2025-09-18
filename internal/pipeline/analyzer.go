package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"repo-explanation/cache"
	"repo-explanation/config"
	"repo-explanation/internal/chunker"
	"repo-explanation/internal/openai"
)

// Analyzer orchestrates the map-reduce analysis pipeline
type Analyzer struct {
	config     *config.Config
	openaiClient *openai.Client
	cache      *cache.Cache
	crawler    *Crawler
}

// AnalysisResult contains the complete analysis result
type AnalysisResult struct {
	ProjectSummary  *openai.ProjectSummary           `json:"project_summary"`
	FolderSummaries map[string]*openai.FolderSummary `json:"folder_summaries"`
	FileSummaries   map[string]*openai.FileSummary   `json:"file_summaries"`
	Stats           map[string]interface{}           `json:"stats"`
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(cfg *config.Config, basePath string) (*Analyzer, error) {
	crawler, err := NewCrawler(cfg, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create crawler: %v", err)
	}
	
	return &Analyzer{
		config:       cfg,
		openaiClient: openai.NewClient(cfg),
		cache:        cache.NewCache(cfg),
		crawler:      crawler,
	}, nil
}

// AnalyzeProject performs the complete analysis pipeline
func (a *Analyzer) AnalyzeProject(ctx context.Context) (*AnalysisResult, error) {
	fmt.Println("🔍 Discovering files...")
	
	// Phase 1: Discover files
	files, err := a.crawler.CrawlFiles()
	if err != nil {
		return nil, fmt.Errorf("file discovery failed: %v", err)
	}
	
	stats := a.crawler.GetFileStats(files)
	fmt.Printf("📁 Found %d files (%.2f MB)\n", stats["total_files"], stats["total_size_mb"])
	
	// Phase 2: Map - Analyze individual files
	fmt.Println("🧠 Analyzing files...")
	fileSummaries, err := a.mapPhase(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("map phase failed: %v", err)
	}
	
	fmt.Printf("✅ Analyzed %d files\n", len(fileSummaries))
	
	// Phase 3: Reduce - Analyze folders
	fmt.Println("📂 Analyzing folders...")
	folderSummaries, err := a.reducePhaseFolder(ctx, fileSummaries)
	if err != nil {
		return nil, fmt.Errorf("folder reduce phase failed: %v", err)
	}
	
	fmt.Printf("✅ Analyzed %d folders\n", len(folderSummaries))
	
	// Phase 4: Final Reduce - Analyze entire project
	fmt.Println("🏗️  Analyzing project...")
	projectSummary, err := a.reducePhaseProject(ctx, folderSummaries)
	if err != nil {
		return nil, fmt.Errorf("project reduce phase failed: %v", err)
	}
	
	fmt.Println("✅ Project analysis complete!")
	
	return &AnalysisResult{
		ProjectSummary:  projectSummary,
		FolderSummaries: folderSummaries,
		FileSummaries:   fileSummaries,
		Stats:           stats,
	}, nil
}

// mapPhase analyzes individual files
func (a *Analyzer) mapPhase(ctx context.Context, files []FileInfo) (map[string]*openai.FileSummary, error) {
	fileSummaries := make(map[string]*openai.FileSummary)
	
	// Create worker pool
	workerCount := a.config.RateLimiting.ConcurrentWorkers
	jobs := make(chan FileInfo, len(files))
	results := make(chan fileResult, len(files))
	
	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.fileWorker(ctx, jobs, results)
		}()
	}
	
	// Send jobs
	go func() {
		defer close(jobs)
		for _, file := range files {
			select {
			case jobs <- file:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Collect results
	processedCount := 0
	for result := range results {
		if result.err != nil {
			fmt.Printf("⚠️  Error analyzing %s: %v\n", result.file.RelativePath, result.err)
			continue
		}
		
		fileSummaries[result.file.RelativePath] = result.summary
		processedCount++
		
		if processedCount%10 == 0 {
			fmt.Printf("📊 Processed %d/%d files\n", processedCount, len(files))
		}
	}
	
	return fileSummaries, nil
}

type fileResult struct {
	file    FileInfo
	summary *openai.FileSummary
	err     error
}

// fileWorker processes individual files
func (a *Analyzer) fileWorker(ctx context.Context, jobs <-chan FileInfo, results chan<- fileResult) {
	for file := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		summary, err := a.analyzeFile(ctx, file)
		results <- fileResult{
			file:    file,
			summary: summary,
			err:     err,
		}
	}
}

// analyzeFile analyzes a single file
func (a *Analyzer) analyzeFile(ctx context.Context, file FileInfo) (*openai.FileSummary, error) {
	// Read file content
	content, err := a.crawler.ReadFile(file)
	if err != nil {
		return nil, err
	}
	
	// Check cache first
	if summary, found := a.cache.GetFileSummary(file.Path, content); found {
		return summary, nil
	}
	
	// Chunk the file if necessary
	chunks, err := chunker.ChunkFile(content, a.config.FileProcessing.ChunkSizeTokens, file.Path)
	if err != nil {
		return nil, err
	}
	
	// For now, analyze the first chunk (or combine chunks for small files)
	var analysisContent string
	if len(chunks) == 1 {
		analysisContent = chunks[0].Content
	} else {
		// For multiple chunks, take the first chunk but add a note about file size
		analysisContent = chunks[0].Content + fmt.Sprintf("\n\n[NOTE: This file has %d chunks, analyzing first chunk only]", len(chunks))
	}
	
	// Analyze with OpenAI
	summary, err := a.openaiClient.AnalyzeFile(ctx, file.RelativePath, analysisContent)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if err := a.cache.SetFileSummary(file.Path, content, summary); err != nil {
		fmt.Printf("⚠️  Failed to cache result for %s: %v\n", file.RelativePath, err)
	}
	
	return summary, nil
}

// reducePhaseFolder analyzes folders based on their files
func (a *Analyzer) reducePhaseFolder(ctx context.Context, fileSummaries map[string]*openai.FileSummary) (map[string]*openai.FolderSummary, error) {
	folderSummaries := make(map[string]*openai.FolderSummary)
	
	// Group files by directory
	folderFiles := make(map[string]map[string]*openai.FileSummary)
	
	for filePath, summary := range fileSummaries {
		dir := filepath.Dir(filePath)
		if dir == "." {
			dir = "root"
		}
		
		if folderFiles[dir] == nil {
			folderFiles[dir] = make(map[string]*openai.FileSummary)
		}
		folderFiles[dir][filePath] = summary
	}
	
	// Analyze each folder
	for folderPath, files := range folderFiles {
		// Convert pointer map to value map for cache and API calls
		filesForAPI := make(map[string]openai.FileSummary)
		for k, v := range files {
			filesForAPI[k] = *v
		}
		
		// Check cache
		if summary, found := a.cache.GetFolderSummary(folderPath, filesForAPI); found {
			folderSummaries[folderPath] = summary
			continue
		}
		
		// Analyze with OpenAI
		summary, err := a.openaiClient.AnalyzeFolder(ctx, folderPath, filesForAPI)
		if err != nil {
			fmt.Printf("⚠️  Error analyzing folder %s: %v\n", folderPath, err)
			continue
		}
		
		folderSummaries[folderPath] = summary
		
		// Cache the result
		if err := a.cache.SetFolderSummary(folderPath, filesForAPI, summary); err != nil {
			fmt.Printf("⚠️  Failed to cache folder result for %s: %v\n", folderPath, err)
		}
	}
	
	return folderSummaries, nil
}

// reducePhaseProject creates final project summary
func (a *Analyzer) reducePhaseProject(ctx context.Context, folderSummaries map[string]*openai.FolderSummary) (*openai.ProjectSummary, error) {
	projectPath := a.crawler.basePath
	
	// Convert pointer map to value map for cache and API calls
	foldersForAPI := make(map[string]openai.FolderSummary)
	for k, v := range folderSummaries {
		foldersForAPI[k] = *v
	}
	
	// Check cache
	if summary, found := a.cache.GetProjectSummary(projectPath, foldersForAPI); found {
		return summary, nil
	}
	
	// Analyze with OpenAI
	summary, err := a.openaiClient.AnalyzeProject(ctx, projectPath, foldersForAPI)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if err := a.cache.SetProjectSummary(projectPath, foldersForAPI, summary); err != nil {
		fmt.Printf("⚠️  Failed to cache project result: %v\n", err)
	}
	
	return summary, nil
}
