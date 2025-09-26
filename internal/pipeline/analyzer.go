package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
	"repo-explanation/cache"
	"repo-explanation/config"
	"repo-explanation/internal/chunker"
	"repo-explanation/internal/database"
	"repo-explanation/internal/detector"
	"repo-explanation/internal/microservices"
	internalOpenai "repo-explanation/internal/openai"
	"repo-explanation/internal/relationships"
)

// Analyzer orchestrates the map-reduce analysis pipeline
type Analyzer struct {
	config     *config.Config
	openaiClient *internalOpenai.Client
	cache      *cache.Cache
	crawler    *Crawler
}

// HelpfulQuestion represents a project-specific question and answer pair
type HelpfulQuestion struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// AnalysisResult contains the complete analysis result
type AnalysisResult struct {
	ProjectSummary      *internalOpenai.ProjectSummary               `json:"project_summary"`
	FolderSummaries     map[string]*internalOpenai.FolderSummary     `json:"folder_summaries"`
	FileSummaries       map[string]*internalOpenai.FileSummary       `json:"file_summaries"`
	ProjectType         *detector.DetectionResult            `json:"project_type"`
	Stats               map[string]interface{}               `json:"stats"`
	Services            []microservices.DiscoveredService    `json:"services,omitempty"`
	ServiceRelationships []relationships.ServiceRelationship `json:"relationships,omitempty"`
	DatabaseSchema      *database.DatabaseSchema             `json:"database_schema,omitempty"`
	HelpfulQuestions    []HelpfulQuestion                    `json:"helpful_questions,omitempty"`
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(cfg *config.Config, basePath string) (*Analyzer, error) {
	crawler, err := NewCrawler(cfg, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create crawler: %v", err)
	}
	
	return &Analyzer{
		config:       cfg,
		openaiClient: internalOpenai.NewClient(cfg),
		cache:        cache.NewCache(cfg),
		crawler:      crawler,
	}, nil
}

// ProgressCallback defines the signature for progress callbacks
type ProgressCallback func(eventType, stage, message string, progress int, data interface{})

// AnalyzeProjectWithProgress performs the complete analysis pipeline with progress callbacks
func (a *Analyzer) AnalyzeProjectWithProgress(ctx context.Context, callback ProgressCallback) (*AnalysisResult, error) {
	// Phase 1: Discover files
	callback("progress", "🔍 Scanning project structure...", "Discovering files and directories", 20, nil)
	
	files, err := a.crawler.CrawlFiles()
	if err != nil {
		return nil, fmt.Errorf("file discovery failed: %v", err)
	}
	
	stats := a.crawler.GetFileStats(files)
	callback("progress", "📁 Files discovered", fmt.Sprintf("Found %d files (%.2f MB)", stats["total_files"].(int), stats["total_size_mb"]), 25, map[string]interface{}{
		"file_count": stats["total_files"],
		"total_size": stats["total_size_mb"],
	})
	
	// Phase 1.5: Detect project type
	callback("progress", "🎯 Detecting project type and framework...", "Analyzing project structure and dependencies", 30, nil)
	
	projectDetector := detector.NewProjectDetector()
	
	// Convert pipeline.FileInfo to detector.FileInfo to avoid import cycle
	detectorFiles := make([]detector.FileInfo, len(files))
	for i, file := range files {
		detectorFiles[i] = detector.FileInfo{
			Path:         file.Path,
			RelativePath: file.RelativePath,
			Size:         file.Size,
			Extension:    file.Extension,
			IsDir:        file.IsDir,
		}
	}
	
	// Create file contents map for command-based detection
	fileContents := make(map[string]string)
	for _, file := range files {
		content, err := a.crawler.ReadFile(file)
		if err == nil {
			fileContents[file.RelativePath] = content
		}
	}
	
	projectType := projectDetector.DetectProjectType(detectorFiles, fileContents)
	
	callback("data", "Project type detected", "Project classification complete", 32, map[string]interface{}{
		"project_type": projectType,
	})
	
	// Phase 2: Map - Analyze individual files
	callback("progress", "🧠 Analyzing individual files...", "Processing file contents with AI analysis", 35, nil)
	
	fileSummaries, err := a.mapPhaseWithProgress(ctx, files, callback)
	if err != nil {
		return nil, fmt.Errorf("map phase failed: %v", err)
	}
	
	callback("data", "File analysis complete", fmt.Sprintf("Analyzed %d files", len(fileSummaries)), 50, map[string]interface{}{
		"file_summaries": fileSummaries,
	})
	
	// Phase 3: Reduce - Analyze folders
	callback("progress", "📂 Analyzing folder structure...", "Organizing file analysis into folder summaries", 55, nil)
	
	folderSummaries, err := a.reducePhaseFolder(ctx, fileSummaries)
	if err != nil {
		return nil, fmt.Errorf("folder reduce phase failed: %v", err)
	}
	
	callback("data", "Folder analysis complete", fmt.Sprintf("Analyzed %d folders", len(folderSummaries)), 60, map[string]interface{}{
		"folder_summaries": folderSummaries,
	})
	
	// Phase 4: Final Reduce - Analyze entire project
	callback("progress", "🏗️ Generating project overview...", "Creating comprehensive project summary", 65, nil)
	
	projectSummary, err := a.reducePhaseProject(ctx, folderSummaries)
	if err != nil {
		return nil, fmt.Errorf("project reduce phase failed: %v", err)
	}
	
	callback("data", "Project overview complete", "Project summary generated", 70, map[string]interface{}{
		"project_summary": projectSummary,
	})
	
	// Phase 5: Detailed architectural analysis
	callback("progress", "🔍 Performing detailed architectural analysis...", "Deep-diving into project architecture and patterns", 72, nil)
	
	importantFiles := a.extractImportantFiles(files)
	
	// Convert pointer maps to value maps for the detailed analysis (with nil checks)
	fileSummariesForAnalysis := make(map[string]internalOpenai.FileSummary)
	for k, v := range fileSummaries {
		if v != nil {
			fileSummariesForAnalysis[k] = *v
		}
	}
	
	folderSummariesForAnalysis := make(map[string]internalOpenai.FolderSummary)
	for k, v := range folderSummaries {
		if v != nil {
			folderSummariesForAnalysis[k] = *v
		}
	}
	
	// Perform detailed analysis with error recovery
	var detailedAnalysis *internalOpenai.RepositoryAnalysis
	var detailedErr error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("⚠️  Detailed analysis panicked: %v\n", r)
				detailedAnalysis = nil
				detailedErr = fmt.Errorf("detailed analysis panicked: %v", r)
			}
		}()
		detailedAnalysis, detailedErr = a.openaiClient.AnalyzeRepositoryDetails(ctx, a.crawler.basePath, folderSummariesForAnalysis, fileSummariesForAnalysis, importantFiles)
	}()
	
	if detailedErr != nil {
		fmt.Printf("⚠️  Detailed analysis failed: %v\n", detailedErr)
		callback("data", "Detailed analysis skipped", "Analysis failed but continuing with basic analysis", 75, map[string]interface{}{
			"detailed_analysis": nil,
		})
	} else {
		projectSummary.DetailedAnalysis = detailedAnalysis
		callback("data", "Detailed analysis complete", "Architectural patterns identified", 75, map[string]interface{}{
			"detailed_analysis": detailedAnalysis,
		})
	}
	
	// Phase 6: Microservice discovery
	var discoveredServices []microservices.DiscoveredService
	var serviceRelationships []relationships.ServiceRelationship
	if projectSummary.DetailedAnalysis != nil && projectSummary.DetailedAnalysis.RepoLayout == "monorepo" {
		callback("progress", "⚙️ Analyzing microservices architecture...", "Discovering services and components", 78, nil)
		
		discoveredServices = a.enhanceWithMicroserviceDiscovery(ctx, files, projectType, projectSummary)
		
		callback("data", "Microservice discovery complete", fmt.Sprintf("Found %d services", len(discoveredServices)), 80, map[string]interface{}{
			"services": discoveredServices,
		})
		
		// Phase 7: Service relationships
		if len(discoveredServices) > 1 {
			callback("progress", "🔗 Mapping service dependencies...", "Analyzing inter-service relationships", 82, nil)
			
			serviceRelationships = a.discoverServiceRelationships(files, discoveredServices, projectSummary)
			
			callback("data", "Service relationships mapped", fmt.Sprintf("Found %d relationships", len(serviceRelationships)), 85, map[string]interface{}{
				"relationships": serviceRelationships,
			})
		}
	}

	// Phase 8: Database schema extraction (with graceful error handling)
	var databaseSchema *database.DatabaseSchema
	if projectType != nil && (strings.ToLower(string(projectType.PrimaryType)) == "backend" || 
							  strings.ToLower(string(projectType.PrimaryType)) == "fullstack") {
		callback("progress", "🗄️ Extracting database schema...", "Analyzing database migrations and schema files", 88, nil)
		
		// Graceful database schema extraction with error recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("⚠️  Database schema extraction failed with panic: %v\n", r)
					databaseSchema = nil
				}
			}()
			databaseSchema = a.extractDatabaseSchema(files)
		}()
		
		if databaseSchema != nil {
			callback("data", "Database schema extracted", "Database structure analyzed", 92, map[string]interface{}{
				"database_schema": databaseSchema,
			})
		} else {
			callback("data", "Database schema extraction skipped", "No database schema found or extraction failed", 92, map[string]interface{}{
				"database_schema": nil,
			})
		}
	}
	
	// Phase 9: Generate helpful questions
	callback("progress", "🤔 Generating helpful questions...", "Creating project-specific Q&A to accelerate development", 94, nil)
	
	helpfulQuestions := a.generateHelpfulQuestions(ctx, projectSummary, projectType, discoveredServices, databaseSchema, fileSummaries)
	
	if len(helpfulQuestions) > 0 {
		callback("data", "Helpful questions generated", fmt.Sprintf("Generated %d project-specific questions", len(helpfulQuestions)), 96, map[string]interface{}{
			"helpful_questions": helpfulQuestions,
		})
		fmt.Printf("✅ [DEBUG] Successfully generated %d helpful questions\n", len(helpfulQuestions))
	} else {
		fmt.Printf("⚠️ [DEBUG] No helpful questions generated - this may indicate an API timeout or parsing issue\n")
		// Generate fallback questions based on project type
		fallbackQuestions := a.generateFallbackQuestions(projectType, projectSummary)
		if len(fallbackQuestions) > 0 {
			callback("data", "Fallback questions generated", fmt.Sprintf("Generated %d fallback questions", len(fallbackQuestions)), 96, map[string]interface{}{
				"helpful_questions": fallbackQuestions,
			})
			helpfulQuestions = fallbackQuestions
			fmt.Printf("✅ [DEBUG] Generated %d fallback questions as backup\n", len(fallbackQuestions))
		}
	}
	
	// Final result compilation
	callback("progress", "📊 Generating comprehensive analysis...", "Compiling final analysis results", 98, nil)
	
	result := &AnalysisResult{
		ProjectSummary:       projectSummary,
		FolderSummaries:      folderSummaries,
		FileSummaries:        fileSummaries,
		ProjectType:          projectType,
		Stats:                stats,
		Services:             discoveredServices,
		ServiceRelationships: serviceRelationships,
		DatabaseSchema:       databaseSchema,
		HelpfulQuestions:     helpfulQuestions,
	}
	
	return result, nil
}

// AnalyzeProject performs the complete analysis pipeline (legacy method for backward compatibility)
func (a *Analyzer) AnalyzeProject(ctx context.Context) (*AnalysisResult, error) {
	fmt.Println("🔍 Discovering files...")
	
	// Phase 1: Discover files
	files, err := a.crawler.CrawlFiles()
	if err != nil {
		return nil, fmt.Errorf("file discovery failed: %v", err)
	}
	
	stats := a.crawler.GetFileStats(files)
	fmt.Printf("📁 Found %d files (%.2f MB)\n", stats["total_files"], stats["total_size_mb"])
	
	// Phase 1.5: Detect project type based on file structure
	fmt.Println("🔍 Detecting project type...")
	projectDetector := detector.NewProjectDetector()
	
	// Convert pipeline.FileInfo to detector.FileInfo to avoid import cycle
	detectorFiles := make([]detector.FileInfo, len(files))
	for i, file := range files {
		detectorFiles[i] = detector.FileInfo{
			Path:         file.Path,
			RelativePath: file.RelativePath,
			Size:         file.Size,
			Extension:    file.Extension,
			IsDir:        file.IsDir,
		}
	}
	
	// Create file contents map for command-based detection
	fileContents := make(map[string]string)
	for _, file := range files {
		content, err := a.crawler.ReadFile(file)
		if err == nil {
			fileContents[file.RelativePath] = content
		}
	}
	
	projectType := projectDetector.DetectProjectType(detectorFiles, fileContents)
	
	// Display project type detection results
	projectType.DisplayResult()
	
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
	
	// Phase 5: Detailed architectural analysis
	fmt.Println("🔍 Performing detailed architectural analysis...")
	importantFiles := a.extractImportantFiles(files)
	
	// Convert pointer maps to value maps for the detailed analysis (with nil checks)
	fileSummariesForAnalysis := make(map[string]internalOpenai.FileSummary)
	for k, v := range fileSummaries {
		if v != nil {
			fileSummariesForAnalysis[k] = *v
		}
	}
	
	folderSummariesForAnalysis := make(map[string]internalOpenai.FolderSummary)
	for k, v := range folderSummaries {
		if v != nil {
			folderSummariesForAnalysis[k] = *v
		}
	}
	
	// Perform detailed analysis with error recovery
	var detailedAnalysis *internalOpenai.RepositoryAnalysis
	var detailedErr error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("⚠️  Detailed analysis panicked: %v\n", r)
				detailedAnalysis = nil
				detailedErr = fmt.Errorf("detailed analysis panicked: %v", r)
			}
		}()
		detailedAnalysis, detailedErr = a.openaiClient.AnalyzeRepositoryDetails(ctx, a.crawler.basePath, folderSummariesForAnalysis, fileSummariesForAnalysis, importantFiles)
	}()
	
	if detailedErr != nil {
		fmt.Printf("⚠️  Detailed analysis failed: %v\n", detailedErr)
		// Continue without detailed analysis
	} else {
		projectSummary.DetailedAnalysis = detailedAnalysis
		fmt.Println("✅ Detailed analysis complete!")
	}
	
	// Phase 6: Enhance with intelligent microservice discovery if it's a monorepo
	var discoveredServices []microservices.DiscoveredService
	var serviceRelationships []relationships.ServiceRelationship
	if projectSummary.DetailedAnalysis != nil && projectSummary.DetailedAnalysis.RepoLayout == "monorepo" {
		fmt.Println("🔍 Discovering microservices...")
		discoveredServices = a.enhanceWithMicroserviceDiscovery(ctx, files, projectType, projectSummary)
		fmt.Println("✅ Microservice discovery complete!")
		
		// Phase 7: Discover service relationships using the discovered services
		if len(discoveredServices) > 1 {
			fmt.Println("🔗 Discovering service relationships...")
			serviceRelationships = a.discoverServiceRelationships(files, discoveredServices, projectSummary)
			fmt.Println("✅ Service relationship discovery complete!")
		}
	}

	// Phase 8: Extract database schema from migrations (with graceful error handling)
	var databaseSchema *database.DatabaseSchema
	if projectType != nil && (strings.ToLower(string(projectType.PrimaryType)) == "backend" || 
							  strings.ToLower(string(projectType.PrimaryType)) == "fullstack") {
		fmt.Println("🗃️  Discovering database schema...")
		
		// Graceful database schema extraction with error recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("⚠️  Database schema extraction failed with panic: %v\n", r)
					databaseSchema = nil
				}
			}()
			databaseSchema = a.extractDatabaseSchema(files)
		}()
		
		if databaseSchema != nil {
			fmt.Println("✅ Database schema extraction complete!")
		} else {
			fmt.Println("⚠️  Database schema extraction skipped - no schema found")
		}
	}
	
	fmt.Println("✅ Project analysis complete!")
	
	return &AnalysisResult{
		ProjectSummary:       projectSummary,
		FolderSummaries:      folderSummaries,
		FileSummaries:        fileSummaries,
		ProjectType:          projectType,
		Stats:                stats,
		Services:             discoveredServices,
		ServiceRelationships: serviceRelationships,
		DatabaseSchema:       databaseSchema,
	}, nil
}

// mapPhaseWithProgress analyzes individual files with progress callbacks
func (a *Analyzer) mapPhaseWithProgress(ctx context.Context, files []FileInfo, callback ProgressCallback) (map[string]*internalOpenai.FileSummary, error) {
	fileSummaries := make(map[string]*internalOpenai.FileSummary)
	totalFiles := len(files)
	processedCount := 0
	
	// Create buffered channels for work distribution
	jobs := make(chan FileInfo, totalFiles)
	results := make(chan fileResult, totalFiles)
	
	// Start worker goroutines
	numWorkers := a.config.RateLimiting.ConcurrentWorkers
	for i := 0; i < numWorkers; i++ {
		go a.fileWorker(ctx, jobs, results)
	}
	
	// Send all files to be processed
	for _, file := range files {
		jobs <- file
	}
	close(jobs)
	
	// Collect results and send progress updates
	for i := 0; i < totalFiles; i++ {
		select {
		case result := <-results:
			processedCount++
			
			if result.err != nil {
				fmt.Printf("⚠️ Failed to analyze file %s: %v\n", result.file.RelativePath, result.err)
				continue
			}
			
			fileSummaries[result.file.RelativePath] = result.summary
			
			// Send progress update every 5 files or at milestones
			progressPercentage := 35 + int(float64(processedCount)/float64(totalFiles)*15) // 35-50% range
			if processedCount%5 == 0 || processedCount == totalFiles {
				callback("progress", "🧠 Analyzing individual files...", 
					fmt.Sprintf("Analyzed %d/%d files", processedCount, totalFiles), 
					progressPercentage, nil)
			}
			
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	return fileSummaries, nil
}

// mapPhase analyzes individual files (legacy method for backward compatibility)
func (a *Analyzer) mapPhase(ctx context.Context, files []FileInfo) (map[string]*internalOpenai.FileSummary, error) {
	fileSummaries := make(map[string]*internalOpenai.FileSummary)
	
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
	summary *internalOpenai.FileSummary
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
func (a *Analyzer) analyzeFile(ctx context.Context, file FileInfo) (*internalOpenai.FileSummary, error) {
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
func (a *Analyzer) reducePhaseFolder(ctx context.Context, fileSummaries map[string]*internalOpenai.FileSummary) (map[string]*internalOpenai.FolderSummary, error) {
	folderSummaries := make(map[string]*internalOpenai.FolderSummary)
	
	// Group files by directory
	folderFiles := make(map[string]map[string]*internalOpenai.FileSummary)
	
	for filePath, summary := range fileSummaries {
		dir := filepath.Dir(filePath)
		if dir == "." {
			dir = "root"
		}
		
		if folderFiles[dir] == nil {
			folderFiles[dir] = make(map[string]*internalOpenai.FileSummary)
		}
		folderFiles[dir][filePath] = summary
	}
	
	// Analyze each folder
	for folderPath, files := range folderFiles {
		// Convert pointer map to value map for cache and API calls
		filesForAPI := make(map[string]internalOpenai.FileSummary)
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
func (a *Analyzer) reducePhaseProject(ctx context.Context, folderSummaries map[string]*internalOpenai.FolderSummary) (*internalOpenai.ProjectSummary, error) {
	projectPath := a.crawler.basePath
	
	// Convert pointer map to value map for cache and API calls (with nil checks)
	foldersForAPI := make(map[string]internalOpenai.FolderSummary)
	for k, v := range folderSummaries {
		if v != nil {
			foldersForAPI[k] = *v
		}
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

// extractImportantFiles extracts key files that are important for architectural analysis
func (a *Analyzer) extractImportantFiles(files []FileInfo) map[string]string {
	importantFiles := make(map[string]string)
	
	// Define important file patterns
	importantPatterns := []string{
		"readme", "readme.md", "readme.txt",
		"package.json", "go.mod", "pyproject.toml", "requirements.txt", "pom.xml", "build.gradle",
		"docker-compose.yml", "docker-compose.yaml", "dockerfile",
		"turbo.json", "lerna.json", "nx.json", "pnpm-workspace.yaml", "go.work",
		"makefile", "cargo.toml", "composer.json", "gemfile",
		".github/workflows", "k8s", "kubernetes", "terraform",
	}
	
	for _, file := range files {
		filename := strings.ToLower(filepath.Base(file.Path))
		filepath_lower := strings.ToLower(file.RelativePath)
		
		// Check if file matches important patterns
		for _, pattern := range importantPatterns {
			if strings.Contains(filename, pattern) || strings.Contains(filepath_lower, pattern) {
				// Read file content (limit to first 2000 characters for analysis)
				content, err := a.crawler.ReadFile(file)
				if err == nil {
					if len(content) > 2000 {
						content = content[:2000] + "..."
					}
					importantFiles[file.RelativePath] = content
				}
				break
			}
		}
	}
	
	return importantFiles
}

// enhanceWithMicroserviceDiscovery enhances the analysis with intelligent microservice discovery
func (a *Analyzer) enhanceWithMicroserviceDiscovery(ctx context.Context, files []FileInfo, projectType *detector.DetectionResult, projectSummary *internalOpenai.ProjectSummary) []microservices.DiscoveredService {
	if projectSummary.DetailedAnalysis == nil {
		return nil
	}

	// Determine project language/type for service discovery
	var projectTypeStr string
	if projectType != nil {
		switch strings.ToLower(string(projectType.PrimaryType)) {
		case "backend":
			// Check if it's Go or Node.js based on files
			if a.hasGoFiles(files) {
				projectTypeStr = "go"
			} else if a.hasNodeFiles(files) {
				projectTypeStr = "node.js"
			}
		case "frontend", "fullstack":
			if a.hasReactFiles(files) {
				projectTypeStr = "react.js"
			} else if a.hasNodeFiles(files) {
				projectTypeStr = "node.js"
			}
		}
	}

	if projectTypeStr == "" {
		return nil // Unsupported project type for microservice discovery
	}

	// Create service discovery instance
	discovery := microservices.NewServiceDiscovery(a.crawler.basePath, projectTypeStr)

	// Convert files to map for discovery
	fileMap := make(map[string]string)
	folderStructure := make([]string, 0)
	
	for _, file := range files {
		content, err := a.crawler.ReadFile(file)
		if err == nil {
			fileMap[file.RelativePath] = content
		}
		
		// Build folder structure
		dir := filepath.Dir(file.RelativePath)
		if dir != "." {
			folderStructure = append(folderStructure, dir)
		}
	}

	// Discover services
	discoveredServices, err := discovery.DiscoverServices(fileMap, folderStructure)
	if err != nil {
		fmt.Printf("⚠️  Service discovery failed: %v\n", err)
		return nil
	}

	// Convert discovered services to MonorepoService format
	var enhancedServices []internalOpenai.MonorepoService
	for _, service := range discoveredServices {
		enhancedService := internalOpenai.MonorepoService{
			Name:         service.Name,
			Path:         service.Path,
			Language:     a.getLanguageFromProjectType(projectTypeStr),
			ShortPurpose: a.generateServicePurpose(service.Name, service.APIType),
			APIType:      string(service.APIType),
			Port:         service.Port,
			EntryPoint:   service.EntryPoint,
		}
		enhancedServices = append(enhancedServices, enhancedService)
	}

	// Replace or enhance existing MonorepoServices with discovered ones
	if len(enhancedServices) > 0 {
		projectSummary.DetailedAnalysis.MonorepoServices = enhancedServices
		fmt.Printf("📦 Discovered %d microservices with external APIs\n", len(enhancedServices))
	}

	return discoveredServices
}

// hasGoFiles checks if the project contains Go files
func (a *Analyzer) hasGoFiles(files []FileInfo) bool {
	for _, file := range files {
		if strings.HasSuffix(file.Path, ".go") {
			return true
		}
	}
	return false
}

// hasNodeFiles checks if the project contains Node.js files
func (a *Analyzer) hasNodeFiles(files []FileInfo) bool {
	for _, file := range files {
		if strings.HasSuffix(file.RelativePath, "package.json") ||
		   strings.HasSuffix(file.Path, ".js") ||
		   strings.HasSuffix(file.Path, ".ts") {
			return true
		}
	}
	return false
}

// hasReactFiles checks if the project contains React files
func (a *Analyzer) hasReactFiles(files []FileInfo) bool {
	for _, file := range files {
		if strings.HasSuffix(file.RelativePath, "package.json") {
			content, err := a.crawler.ReadFile(file)
			if err == nil && strings.Contains(content, "\"react\"") {
				return true
			}
		}
		if strings.HasSuffix(file.Path, ".jsx") || strings.HasSuffix(file.Path, ".tsx") {
			return true
		}
	}
	return false
}

// getLanguageFromProjectType converts project type to language string
func (a *Analyzer) getLanguageFromProjectType(projectType string) string {
	switch strings.ToLower(projectType) {
	case "go", "golang":
		return "Go"
	case "node.js", "nodejs":
		return "Node.js"
	case "react.js", "reactjs":
		return "JavaScript/TypeScript"
	default:
		return "Unknown"
	}
}

// generateServicePurpose generates a purpose description based on service name and type
func (a *Analyzer) generateServicePurpose(serviceName string, apiType microservices.ServiceType) string {
	nameWords := strings.Split(strings.ToLower(serviceName), "-")
	
	// Generate purpose based on common service name patterns
	for _, word := range nameWords {
		switch word {
		case "auth", "authentication":
			return "Handles user authentication and authorization"
		case "user", "users":
			return "Manages user accounts and profiles"
		case "payment", "payments":
			return "Processes payments and billing operations"
		case "order", "orders":
			return "Manages order processing and fulfillment"
		case "product", "products", "catalog":
			return "Manages product catalog and inventory"
		case "notification", "notifications":
			return "Handles notifications and messaging"
		case "api", "gateway":
			return "API gateway routing requests to microservices"
		case "admin":
			return "Administrative interface and operations"
		case "search":
			return "Provides search and indexing capabilities"
		case "analytics":
			return "Analytics and reporting functionality"
		}
	}
	
	// Default purpose based on API type
	switch apiType {
	case microservices.HTTPService:
		return fmt.Sprintf("HTTP API service: %s", serviceName)
	case microservices.GRPCService:
		return fmt.Sprintf("gRPC service: %s", serviceName)
	case microservices.GraphQLService:
		return fmt.Sprintf("GraphQL API: %s", serviceName)
	default:
		return fmt.Sprintf("Service: %s", serviceName)
	}
}

// discoverServiceRelationships discovers relationships between microservices
func (a *Analyzer) discoverServiceRelationships(files []FileInfo, discoveredServices []microservices.DiscoveredService, projectSummary *internalOpenai.ProjectSummary) []relationships.ServiceRelationship {
	projectPath := a.crawler.basePath
	cacheDir := "./relationships_cache"
	
	// Try to load from cache first
	cachedGraph, err := relationships.LoadServiceGraphFromFile(projectPath, cacheDir)
	if err != nil {
		fmt.Printf("⚠️  Failed to load cache: %v\n", err)
	}
	
	var serviceGraph *relationships.ServiceGraph
	
	if cachedGraph != nil {
		fmt.Println("📋 Using cached service relationship data")
		serviceGraph = cachedGraph
	} else {
		fmt.Println("🔍 Analyzing service relationships...")
		
		// Convert files to map for relationship discovery
		fileMap := make(map[string]string)
		for _, file := range files {
			content, err := a.crawler.ReadFile(file)
			if err == nil {
				fileMap[file.RelativePath] = content
			}
		}

		// Create relationship discovery instance
		relationshipDiscovery := relationships.NewRelationshipDiscovery(discoveredServices, fileMap)

		// Discover relationships
		serviceGraph, err = relationshipDiscovery.DiscoverRelationships(projectPath)
		if err != nil {
			fmt.Printf("⚠️  Service relationship discovery failed: %v\n", err)
			return []relationships.ServiceRelationship{}
		}
		
		// Save to cache
		if err := serviceGraph.SaveToFile(cacheDir); err != nil {
			fmt.Printf("⚠️  Failed to save relationship cache: %v\n", err)
		}
	}

	// Display the service dependency graph
	if len(serviceGraph.Relationships) > 0 {
		fmt.Printf("\n%s\n", serviceGraph.ConsoleVisualization())
		fmt.Printf("🔗 Found %d service dependencies\n", len(serviceGraph.Relationships))
		
		// Generate and display Mermaid JSON
		mermaidJSON, err := serviceGraph.GenerateMermaidJSON()
		if err != nil {
			fmt.Printf("⚠️  Failed to generate Mermaid JSON: %v\n", err)
		} else {
			fmt.Println("\n📊 MERMAID GRAPH (JSON):")
			fmt.Println(strings.Repeat("─", 40))
			fmt.Println(mermaidJSON)
		}
	} else {
		fmt.Println("🔗 No service dependencies detected (services appear to be independent)")
		
		// Still generate Mermaid for independent services
		mermaidJSON, err := serviceGraph.GenerateMermaidJSON()
		if err != nil {
			fmt.Printf("⚠️  Failed to generate Mermaid JSON: %v\n", err)
		} else {
			fmt.Println("\n📊 MERMAID GRAPH (JSON) - Independent Services:")
			fmt.Println(strings.Repeat("─", 40))
			fmt.Println(mermaidJSON)
		}
	}
	
	// Return the discovered relationships
	if serviceGraph != nil {
		return serviceGraph.Relationships
	}
	return []relationships.ServiceRelationship{}
}

// extractDatabaseSchema extracts database schema from SQL migration files using streaming extractor
func (a *Analyzer) extractDatabaseSchema(files []FileInfo) *database.DatabaseSchema {
	// Convert files to map for schema extraction
	fileMap := make(map[string]string)
	for _, file := range files {
		content, err := a.crawler.ReadFile(file)
		if err == nil {
			fileMap[file.RelativePath] = content
		}
	}

	// Extract schema using the streaming extractor with final migration generation
	result, err := func() (*database.ExtractSchemaFromProjectResult, error) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("⚠️  Database schema extraction recovered from panic: %v\n", r)
			}
		}()
		
		return database.ExtractSchemaWithFinalMigration("", fileMap, func(response database.StreamingResponse) {
			// Progress callback for database extraction
			fmt.Printf("📋 Database extraction: %s (%s)\n", response.Phase, response.Message)
		})
	}()
	
	// Convert canonical schema to legacy format and add final migration SQL and LLM relationships
	var schema *database.DatabaseSchema
	if err == nil && result != nil && result.Schema != nil {
		schema = database.ConvertToLegacySchema(result.Schema, "")
		// Add the final migration SQL and LLM relationships to the schema
		if schema != nil {
			schema.FinalMigrationSQL = result.FinalMigrationSQL
			schema.LLMRelationships = result.LLMRelationships
		}
	}
	
	if err != nil {
		fmt.Printf("⚠️  Database schema extraction failed: %v\n", err)
		fmt.Println("   Returning partial schema if any tables were extracted...")
		
		// Return partial schema if we have any tables
		if schema != nil && len(schema.Tables) > 0 {
			fmt.Printf("🗃️  Found %d database tables despite errors\n", len(schema.Tables))
			return schema
		}
		return nil
	}

	if schema == nil || len(schema.Tables) == 0 {
		fmt.Println("🗃️  No database tables found in migrations")
		return nil
	}

	// Display summary
	fmt.Printf("🗃️  Found %d database tables\n", len(schema.Tables))
	for tableName, table := range schema.Tables {
		fmt.Printf("   📊 %s (%d columns)\n", tableName, len(table.Columns))
	}
	
	return schema
}

// generateHelpfulQuestions creates project-specific Q&A using LLM
func (a *Analyzer) generateHelpfulQuestions(ctx context.Context, projectSummary *internalOpenai.ProjectSummary, projectType *detector.DetectionResult, services []microservices.DiscoveredService, databaseSchema *database.DatabaseSchema, fileSummaries map[string]*internalOpenai.FileSummary) []HelpfulQuestion {
	fmt.Printf("🤔 [DEBUG] Starting helpful questions generation\n")
	
	// Skip if we don't have enough data for meaningful questions
	if projectSummary == nil || projectType == nil {
		fmt.Printf("❌ [DEBUG] Insufficient data for question generation (projectSummary: %v, projectType: %v)\n", projectSummary != nil, projectType != nil)
		return []HelpfulQuestion{}
	}
	
	// Build context for LLM prompt
	prompt := a.buildQuestionsPrompt(projectSummary, projectType, services, databaseSchema, fileSummaries)
	
	fmt.Printf("✅ [DEBUG] Question prompt created (%d characters)\n", len(prompt))
	
	// Call LLM to generate questions
	questions, err := a.callLLMForQuestions(ctx, prompt)
	if err != nil {
		fmt.Printf("❌ [DEBUG] LLM question generation failed: %v\n", err)
		return []HelpfulQuestion{}
	}
	
	fmt.Printf("✅ [DEBUG] Generated %d helpful questions\n", len(questions))
	return questions
}

// buildQuestionsPrompt creates a comprehensive prompt for LLM question generation
func (a *Analyzer) buildQuestionsPrompt(projectSummary *internalOpenai.ProjectSummary, projectType *detector.DetectionResult, services []microservices.DiscoveredService, databaseSchema *database.DatabaseSchema, fileSummaries map[string]*internalOpenai.FileSummary) string {
	prompt := `You are a senior software engineer helping developers understand and work with a codebase. Based on the project analysis below, generate 5-7 helpful questions with detailed answers that would help developers:

1. Understand the project structure and architecture quickly
2. Get started with development faster  
3. Avoid common pitfalls and mistakes
4. Know the most important files and entry points
5. Understand deployment and setup processes

Make the questions SPECIFIC to this project - not generic programming questions. Focus on practical, actionable information.

Return your response as a JSON array with this exact format:
[
  {
    "question": "How do I set up the development environment for this project?",
    "answer": "Detailed step-by-step answer specific to this project..."
  }
]

PROJECT ANALYSIS:
`

	// Add project type and confidence
	if projectType != nil {
		prompt += fmt.Sprintf("\nProject Type: %s (%s confidence: %.1f/10)\n", projectType.PrimaryType, projectType.SecondaryType, projectType.Confidence)
	}
	
	// Add project summary
	if projectSummary != nil {
		if projectSummary.Purpose != "" {
			prompt += fmt.Sprintf("\nProject Purpose: %s\n", projectSummary.Purpose)
		}
		if projectSummary.Architecture != "" {
			prompt += fmt.Sprintf("Architecture: %s\n", projectSummary.Architecture)
		}
		if len(projectSummary.Languages) > 0 {
			prompt += "\nPrimary Languages:\n"
			for lang, count := range projectSummary.Languages {
				prompt += fmt.Sprintf("- %s (%d files)\n", lang, count)
			}
		}
		if len(projectSummary.DataModels) > 0 {
			prompt += fmt.Sprintf("\nData Models: %v\n", projectSummary.DataModels)
		}
		if len(projectSummary.ExternalServices) > 0 {
			prompt += fmt.Sprintf("External Services: %v\n", projectSummary.ExternalServices)
		}
	}
	
	// Add services information
	if len(services) > 0 {
		prompt += "\nServices/Components:\n"
		for _, service := range services {
			prompt += fmt.Sprintf("- %s (%s): %s\n", service.Name, service.APIType, service.EntryPoint)
		}
	}
	
	// Add database information
	if databaseSchema != nil && len(databaseSchema.Tables) > 0 {
		prompt += fmt.Sprintf("\nDatabase: %d tables detected\n", len(databaseSchema.Tables))
		tableNames := make([]string, 0, len(databaseSchema.Tables))
		for tableName := range databaseSchema.Tables {
			tableNames = append(tableNames, tableName)
		}
		if len(tableNames) <= 5 {
			prompt += fmt.Sprintf("Tables: %v\n", tableNames)
		} else {
			prompt += fmt.Sprintf("Main tables: %v... (%d total)\n", tableNames[:5], len(tableNames))
		}
	}
	
	// Add key files information
	keyFiles := a.extractKeyFiles(fileSummaries)
	if len(keyFiles) > 0 {
		prompt += "\nKey Files:\n"
		for _, file := range keyFiles {
			prompt += fmt.Sprintf("- %s\n", file)
		}
	}
	
	prompt += `

IMPORTANT GUIDELINES:
- Questions must be specific to THIS project, not generic
- Focus on practical development needs (setup, architecture, deployment)
- Include specific file names, commands, and project details in answers
- Make answers actionable and detailed
- Avoid questions about general programming concepts
- Generate exactly 5-7 questions
- Return ONLY the JSON array, no other text`

	return prompt
}

// extractKeyFiles identifies the most important files for developers
func (a *Analyzer) extractKeyFiles(fileSummaries map[string]*internalOpenai.FileSummary) []string {
	keyFiles := []string{}
	
	// Look for common important files
	importantPatterns := []string{
		"package.json", "go.mod", "requirements.txt", "Cargo.toml", "pom.xml",
		"Dockerfile", "docker-compose.yml", "Makefile",
		"README.md", "CONTRIBUTING.md", "SETUP.md",
		"main.go", "app.js", "index.js", "server.js", "main.py", "app.py",
		".env.example", "config.yaml", "config.json",
	}
	
	for filePath := range fileSummaries {
		for _, pattern := range importantPatterns {
			if strings.Contains(strings.ToLower(filePath), strings.ToLower(pattern)) {
				keyFiles = append(keyFiles, filePath)
				break
			}
		}
	}
	
	// Limit to most important ones
	if len(keyFiles) > 8 {
		keyFiles = keyFiles[:8]
	}
	
	return keyFiles
}

// callLLMForQuestions makes the LLM API call for question generation
func (a *Analyzer) callLLMForQuestions(ctx context.Context, prompt string) ([]HelpfulQuestion, error) {
	fmt.Printf("🤖 [DEBUG] Starting LLM call for question generation\n")
	fmt.Printf("📝 [DEBUG] Prompt length: %d characters\n", len(prompt))
	
	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Try loading from config file as fallback
		cfg, err := config.LoadConfig("config.yaml")
		if err == nil && cfg.OpenAI.APIKey != "" {
			apiKey = cfg.OpenAI.APIKey
		} else {
			return nil, fmt.Errorf("OpenAI API key not found for question generation")
		}
	}
	
	// Create OpenAI client
	openaiCfg := openai.DefaultConfig(apiKey)
	client := openai.NewClientWithConfig(openaiCfg)
	
	// Create context with extended timeout for question generation (5 minutes)
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	// Make the API call
	resp, err := client.CreateChatCompletion(reqCtx, openai.ChatCompletionRequest{
		Model:       "gpt-3.5-turbo",
		Temperature: 0.3, // Slightly creative but still focused
		MaxTokens:   3000, // Enough for detailed Q&A
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful senior software engineer creating project-specific onboarding questions. Always return valid JSON arrays with question/answer objects. Be specific to the project details provided.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	
	if err != nil {
		fmt.Printf("❌ [DEBUG] OpenAI API call failed: %v\n", err)
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}
	
	responseContent := strings.TrimSpace(resp.Choices[0].Message.Content)
	fmt.Printf("📝 [DEBUG] LLM response length: %d characters\n", len(responseContent))
	
	// Parse JSON response
	var questions []HelpfulQuestion
	if err := json.Unmarshal([]byte(responseContent), &questions); err != nil {
		fmt.Printf("❌ [DEBUG] Failed to parse JSON response: %v\n", err)
		fmt.Printf("📝 [DEBUG] Response content: %s\n", responseContent[:minInt(500, len(responseContent))])
		return nil, fmt.Errorf("failed to parse LLM response: %v", err)
	}
	
	// Validate and filter questions
	validQuestions := []HelpfulQuestion{}
	for _, q := range questions {
		if strings.TrimSpace(q.Question) != "" && strings.TrimSpace(q.Answer) != "" {
			validQuestions = append(validQuestions, HelpfulQuestion{
				Question: strings.TrimSpace(q.Question),
				Answer:   strings.TrimSpace(q.Answer),
			})
		}
	}
	
	// Limit to 7 questions max
	if len(validQuestions) > 7 {
		validQuestions = validQuestions[:7]
	}
	
	fmt.Printf("✅ [DEBUG] Successfully parsed %d valid questions\n", len(validQuestions))
	return validQuestions, nil
}

// generateFallbackQuestions creates basic questions when LLM generation fails
func (a *Analyzer) generateFallbackQuestions(projectType *detector.DetectionResult, projectSummary *internalOpenai.ProjectSummary) []HelpfulQuestion {
	fmt.Printf("🔧 [DEBUG] Generating fallback questions for project type: %s\n", projectType.PrimaryType)
	
	fallbackQuestions := []HelpfulQuestion{}
	
	// Generic questions applicable to all project types
	fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
		Question: "How do I set up the development environment for this project?",
		Answer:   "Check for setup files like package.json (Node.js), go.mod (Go), requirements.txt (Python), or README.md for installation instructions. Look for Docker files for containerized setup.",
	})
	
	fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
		Question: "What is the main entry point of this application?",
		Answer:   "Look for files like main.go (Go), index.js (Node.js), app.py (Python), or server.js. Check the package.json scripts section for the start command.",
	})
	
	// Project type specific questions
	switch projectType.PrimaryType {
	case "Backend", "Fullstack":
		fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
			Question: "How do I configure the database connection?",
			Answer:   "Look for configuration files like .env, config.yaml, or database connection strings in the main application files. Check for migration files in folders like migrations/ or db/.",
		})
		
		fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
			Question: "What API endpoints are available?",
			Answer:   "Check route definition files, usually in folders like routes/, handlers/, or controllers/. Look for API documentation files or OpenAPI/Swagger specifications.",
		})
		
	case "Frontend":
		fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
			Question: "How do I start the development server?",
			Answer:   "Run 'npm start' or 'yarn start' for React/Vue projects. Check package.json scripts for the exact command. For static sites, look for build tools like Vite or Webpack.",
		})
		
		fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
			Question: "What UI framework or library is being used?",
			Answer:   "Check package.json dependencies for frameworks like React, Vue, Angular, or UI libraries like Material-UI, Bootstrap, or Tailwind CSS.",
		})
		
	case "Mobile":
		fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
			Question: "How do I run this mobile app?",
			Answer:   "For React Native, use 'npx react-native run-ios' or 'npx react-native run-android'. For Flutter, use 'flutter run'. Check README.md for platform-specific setup instructions.",
		})
		
	case "DevOps/Infrastructure":
		fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
			Question: "How do I deploy this infrastructure?",
			Answer:   "Look for Terraform files (*.tf), Kubernetes manifests (*.yaml in k8s folders), Docker Compose files, or CI/CD pipeline configurations in .github/workflows or .gitlab-ci.yml.",
		})
	}
	
	// Add project-specific questions based on summary if available
	if projectSummary != nil && projectSummary.Purpose != "" {
		fallbackQuestions = append(fallbackQuestions, HelpfulQuestion{
			Question: "What is the main purpose of this project?",
			Answer:   projectSummary.Purpose,
		})
	}
	
	fmt.Printf("✅ [DEBUG] Generated %d fallback questions\n", len(fallbackQuestions))
	return fallbackQuestions
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
