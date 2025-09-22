package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"repo-explanation/cache"
	"repo-explanation/config"
	"repo-explanation/internal/chunker"
	"repo-explanation/internal/database"
	"repo-explanation/internal/detector"
	"repo-explanation/internal/microservices"
	"repo-explanation/internal/openai"
	"repo-explanation/internal/relationships"
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
	ProjectType     *detector.DetectionResult        `json:"project_type"`
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
	
	projectType := projectDetector.DetectProjectType(detectorFiles)
	
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
	
	// Convert pointer maps to value maps for the detailed analysis
	fileSummariesForAnalysis := make(map[string]openai.FileSummary)
	for k, v := range fileSummaries {
		fileSummariesForAnalysis[k] = *v
	}
	
	folderSummariesForAnalysis := make(map[string]openai.FolderSummary)
	for k, v := range folderSummaries {
		folderSummariesForAnalysis[k] = *v
	}
	
	detailedAnalysis, err := a.openaiClient.AnalyzeRepositoryDetails(ctx, a.crawler.basePath, folderSummariesForAnalysis, fileSummariesForAnalysis, importantFiles)
	if err != nil {
		fmt.Printf("⚠️  Detailed analysis failed: %v\n", err)
		// Continue without detailed analysis
	} else {
		projectSummary.DetailedAnalysis = detailedAnalysis
		fmt.Println("✅ Detailed analysis complete!")
	}
	
	// Phase 6: Enhance with intelligent microservice discovery if it's a monorepo
	var discoveredServices []microservices.DiscoveredService
	if projectSummary.DetailedAnalysis != nil && projectSummary.DetailedAnalysis.RepoLayout == "monorepo" {
		fmt.Println("🔍 Discovering microservices...")
		discoveredServices = a.enhanceWithMicroserviceDiscovery(ctx, files, projectType, projectSummary)
		fmt.Println("✅ Microservice discovery complete!")
		
		// Phase 7: Discover service relationships using the discovered services
		if len(discoveredServices) > 1 {
			fmt.Println("🔗 Discovering service relationships...")
			a.discoverServiceRelationships(files, discoveredServices, projectSummary)
			fmt.Println("✅ Service relationship discovery complete!")
		}
	}

	// Phase 8: Extract database schema from migrations (always run for backend projects)
	if projectType != nil && (strings.ToLower(string(projectType.PrimaryType)) == "backend" || 
							  strings.ToLower(string(projectType.PrimaryType)) == "fullstack") {
		fmt.Println("🗃️  Discovering database schema...")
		a.extractDatabaseSchema(files)
		fmt.Println("✅ Database schema extraction complete!")
	}
	
	fmt.Println("✅ Project analysis complete!")
	
	return &AnalysisResult{
		ProjectSummary:  projectSummary,
		FolderSummaries: folderSummaries,
		FileSummaries:   fileSummaries,
		ProjectType:     projectType,
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
func (a *Analyzer) enhanceWithMicroserviceDiscovery(ctx context.Context, files []FileInfo, projectType *detector.DetectionResult, projectSummary *openai.ProjectSummary) []microservices.DiscoveredService {
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
	var enhancedServices []openai.MonorepoService
	for _, service := range discoveredServices {
		enhancedService := openai.MonorepoService{
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
func (a *Analyzer) discoverServiceRelationships(files []FileInfo, discoveredServices []microservices.DiscoveredService, projectSummary *openai.ProjectSummary) {
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
			return
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
}

// extractDatabaseSchema extracts database schema from SQL migration files
func (a *Analyzer) extractDatabaseSchema(files []FileInfo) {
	projectPath := a.crawler.basePath
	
	// Convert files to map for schema extraction
	fileMap := make(map[string]string)
	for _, file := range files {
		content, err := a.crawler.ReadFile(file)
		if err == nil {
			fileMap[file.RelativePath] = content
		}
	}

	// Create schema extractor
	schemaExtractor := database.NewSchemaExtractor()

	// Extract schema from migrations
	schema, err := schemaExtractor.ExtractSchemaFromMigrations(projectPath, fileMap)
	if err != nil {
		fmt.Printf("⚠️  Database schema extraction failed: %v\n", err)
		return
	}

	if len(schema.Tables) == 0 {
		fmt.Println("🗃️  No database tables found in migrations")
		return
	}

	// Generate PlantUML ERD
	pumlContent := schemaExtractor.GeneratePlantUML()

	// Save PlantUML file
	if err := schemaExtractor.SavePlantUMLFile(projectPath, pumlContent); err != nil {
		fmt.Printf("⚠️  Failed to save PlantUML file: %v\n", err)
		return
	}

	// Display summary
	fmt.Printf("🗃️  Found %d database tables in %s\n", len(schema.Tables), schema.MigrationPath)
	for tableName, table := range schema.Tables {
		fmt.Printf("   📊 %s (%d columns)\n", tableName, len(table.Columns))
	}
}
