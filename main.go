package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"repo-explanation/cli"
	"repo-explanation/controllers"
	"repo-explanation/internal/database"
	"repo-explanation/internal/detector"
	"repo-explanation/internal/secrets"
	"repo-explanation/routes"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	mode := flag.String("mode", "server", "Mode to run: 'server', 'cli', 'secrets', or 'debug-db'")
	path := flag.String("path", "", "Path to analyze (for secrets mode)")
	flag.Parse()

	switch *mode {
	case "server":
		runServer()
	case "cli":
		runCLI()
	case "secrets":
		runSecretsExtraction(*path)
	case "debug-db":
		runDebugDB()
	case "test-detection":
		runDetectionTest(*path)
	default:
		fmt.Printf("Unknown mode: %s\n", *mode)
		fmt.Println("Available modes: server, cli, debug-db")
		os.Exit(1)
	}
}

func runServer() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize controllers
	healthController := controllers.NewHealthController()
	analysisController := controllers.NewAnalysisController()

	// Setup routes
	routes.SetupRoutes(e, healthController, analysisController)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}

func runCLI() {
	repl := cli.NewREPL()
	repl.Start()
}

func runSecretsExtraction(projectPath string) {
	if projectPath == "" {
		args := flag.Args()
		if len(args) == 0 {
			fmt.Println("Usage: ./analyzer-api -mode=secrets -path=<folder-path>")
			fmt.Println("   OR: ./analyzer-api -mode=secrets <folder-path>")
			fmt.Println("Example: ./analyzer-api -mode=secrets -path=./my-project")
			fmt.Println("Example: ./analyzer-api -mode=secrets ./my-project")
			os.Exit(1)
		}
		projectPath = args[0]
	}

	fmt.Printf("🔍 Extracting secrets from: %s\n", projectPath)
	
	// Create secret extractor
	extractor := secrets.NewSecretExtractor(projectPath)
	
	// Extract secrets from configuration files
	projectSecrets, err := extractor.ExtractSecrets()
	if err != nil {
		fmt.Printf("❌ Secret extraction failed: %v\n", err)
		os.Exit(1)
	}
	
	if projectSecrets == nil || projectSecrets.TotalVariables == 0 {
		fmt.Println("✅ No configuration secrets found that need to be set.")
		return
	}
	
	// Format output
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🔐 SECRET EXTRACTION RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	
	fmt.Printf("📂 Project Path: %s\n", projectPath)
	fmt.Printf("📊 Project Type: %s\n", projectSecrets.ProjectType)
	fmt.Printf("🔢 Total Variables: %d\n", projectSecrets.TotalVariables)
	fmt.Printf("⚠️  Required Variables: %d\n", projectSecrets.RequiredCount)
	fmt.Printf("📝 Summary: %s\n", projectSecrets.Summary)
	fmt.Println()
	
	// Display Global Secrets
	if len(projectSecrets.GlobalSecrets) > 0 {
		fmt.Println("🌍 GLOBAL SECRETS")
		fmt.Println(strings.Repeat("-", 40))
		for i, secret := range projectSecrets.GlobalSecrets {
			fmt.Printf("%d. %s\n", i+1, secret.Name)
			fmt.Printf("   Type: %s\n", strings.ToUpper(secret.Type))
			fmt.Printf("   Source: %s\n", secret.Source)
			fmt.Printf("   Description: %s\n", secret.Description)
			if secret.Example != "" {
				fmt.Printf("   Example: %s=%s\n", secret.Name, secret.Example)
			}
			fmt.Println()
		}
	}
	
	// Display Service-Specific Secrets
	if len(projectSecrets.Services) > 0 {
		fmt.Println("⚙️  SERVICE SECRETS")
		fmt.Println(strings.Repeat("-", 40))
		for _, service := range projectSecrets.Services {
			fmt.Printf("📦 Service: %s\n", service.ServiceName)
			fmt.Printf("📁 Path: %s\n", service.ServicePath)
			fmt.Printf("📋 Config Files: %s\n", strings.Join(service.ConfigFiles, ", "))
			fmt.Println()
			
			if len(service.Variables) > 0 {
				for i, secret := range service.Variables {
					fmt.Printf("  %d. %s\n", i+1, secret.Name)
					fmt.Printf("     Type: %s\n", strings.ToUpper(secret.Type))
					fmt.Printf("     Source: %s\n", secret.Source)
					fmt.Printf("     Description: %s\n", secret.Description)
					if secret.Example != "" {
						fmt.Printf("     Example: %s=%s\n", secret.Name, secret.Example)
					}
					fmt.Println()
				}
			} else {
				fmt.Println("  ✅ No configuration variables needed for this service")
				fmt.Println()
			}
		}
	}
	
	// Setup Instructions
	if projectSecrets.RequiredCount > 0 {
		fmt.Println("🛠️  SETUP INSTRUCTIONS")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println("To configure this project:")
		fmt.Println("1. Copy .env.example to .env (if available)")
		fmt.Printf("2. Set values for the %d required environment variables shown above\n", projectSecrets.RequiredCount)
		fmt.Println("3. Update any configuration files (config.yaml, application.properties, etc.) with your values")
		fmt.Println("4. For API keys and secrets, refer to the respective service documentation")
		fmt.Println("5. Ensure all services have access to their required environment variables")
		fmt.Println()
		fmt.Println("💡 Tip: Check each service's README or documentation for specific setup instructions.")
	}
	
	fmt.Println(strings.Repeat("=", 60))
}

func runDebugDB() {
	// Check if folder path is provided as argument
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: ./analyzer-api -mode=debug-db <folder-path>")
		fmt.Println("Example: ./analyzer-api -mode=debug-db ./my-project")
		os.Exit(1)
	}

	folderPath := args[0]
	
	fmt.Printf("🔍 DEBUG: Database Schema Extraction for: %s\n", folderPath)
	fmt.Println(strings.Repeat("=", 60))

	// Step 1: Scan for all files
	fmt.Println("📂 Step 1: Scanning for files...")
	files, err := scanFiles(folderPath)
	if err != nil {
		fmt.Printf("❌ Error scanning files: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Found %d total files\n", len(files))

	// Step 2: Filter for SQL files
	fmt.Println("\n📄 Step 2: Filtering for SQL migration files...")
	sqlFiles := filterSQLFiles(files)
	fmt.Printf("✅ Found %d SQL files\n", len(sqlFiles))
	
	for path := range sqlFiles {
		fmt.Printf("   • %s\n", path)
	}

	// Step 3: Find migration directories
	fmt.Println("\n📁 Step 3: Identifying migration directories...")
	migrationDirs := findMigrationDirectories(sqlFiles)
	fmt.Printf("✅ Found %d migration directories\n", len(migrationDirs))
	
	for _, dir := range migrationDirs {
		fmt.Printf("   • %s\n", dir)
	}

	if len(sqlFiles) == 0 {
		fmt.Println("\n❌ No SQL files found. Database extraction cannot proceed.")
		fmt.Println("💡 Make sure your project has SQL migration files in directories containing 'migration'")
		return
	}

	// Step 4: Extract schema using streaming extractor with final migration generation
	fmt.Println("\n🗄️ Step 4: Extracting database schema and generating final migration...")
	result, err := database.ExtractSchemaWithFinalMigration(folderPath, sqlFiles, func(response database.StreamingResponse) {
		fmt.Printf("   📋 %s: %s (Progress: %d/%d)\n", 
			response.Phase, response.Message, response.Progress.Current, response.Progress.Total)
	})

	if err != nil {
		fmt.Printf("❌ Schema extraction failed: %v\n", err)
		return
	}

	if result == nil || result.Schema == nil {
		fmt.Println("❌ No schema extracted (result is nil)")
		return
	}

	canonicalSchema := result.Schema
	mermaidERD := result.MermaidERD
	finalMigrationSQL := result.FinalMigrationSQL
	llmRelationships := result.LLMRelationships
	
	fmt.Printf("🔍 [DEBUG] Result fields from ExtractSchemaWithFinalMigration:\n")
	fmt.Printf("   📊 Schema: %v\n", canonicalSchema != nil)
	fmt.Printf("   📊 MermaidERD: %d chars\n", len(mermaidERD))
	fmt.Printf("   📊 FinalMigrationSQL: %d chars\n", len(finalMigrationSQL))
	fmt.Printf("   📊 LLMRelationships: %d chars\n", len(llmRelationships))

	// Step 5: Display results
	fmt.Println("\n🎉 Step 5: Extraction Results")
	fmt.Println(strings.Repeat("=", 60))
	
	fmt.Printf("📊 Tables found: %d\n", len(canonicalSchema.Tables))
	fmt.Printf("📊 Enums found: %d\n", len(canonicalSchema.Enums))
	fmt.Printf("📊 Views found: %d\n", len(canonicalSchema.Views))

	// Display table details
	if len(canonicalSchema.Tables) > 0 {
		fmt.Println("\n📋 Table Details:")
		for tableName, table := range canonicalSchema.Tables {
			fmt.Printf("\n  🏷️  Table: %s\n", tableName)
			fmt.Printf("     Columns: %d\n", len(table.Columns))
			fmt.Printf("     Primary Keys: %v\n", table.PrimaryKey)
			fmt.Printf("     Foreign Keys: %d\n", len(table.ForeignKeys))
			fmt.Printf("     Indexes: %d\n", len(table.Indexes))
			
			// Show column details
			for colName, column := range table.Columns {
				nullable := "NOT NULL"
				if column.Nullable {
					nullable = "NULL"
				}
				defaultVal := "no default"
				if column.Default != nil {
					defaultVal = *column.Default
				}
				fmt.Printf("       🔹 %s: %s (%s, default: %s)\n", colName, column.Type, nullable, defaultVal)
			}
		}
	}

	// Display Mermaid ERD
	if mermaidERD != "" {
		fmt.Println("\n🎨 Mermaid ERD Generated:")
		fmt.Println(strings.Repeat("─", 40))
		fmt.Println(mermaidERD)
		fmt.Println(strings.Repeat("─", 40))
	}

	// Step 6: Convert to legacy format
	fmt.Println("\n🔄 Step 6: Converting to legacy format...")
	legacySchema := database.ConvertToLegacySchema(canonicalSchema, "")
	
	if legacySchema == nil {
		fmt.Println("❌ Legacy conversion failed (result is nil)")
		return
	}

	fmt.Printf("✅ Legacy schema created with %d tables\n", len(legacySchema.Tables))
	
	// Display legacy format details
	fmt.Println("\n📋 Legacy Schema Details:")
	for tableName, table := range legacySchema.Tables {
		fmt.Printf("  🏷️  %s: %d columns, %d indexes\n", tableName, len(table.Columns), len(table.Indexes))
	}

	fmt.Printf("\n🔗 Foreign Key References: %d\n", len(legacySchema.ForeignKeys))
	for _, fk := range legacySchema.ForeignKeys {
		fmt.Printf("   • %s.%s\n", fk.Table, fk.Column)
	}

	// Step 7: Display final migration SQL
	if finalMigrationSQL != "" {
		fmt.Println("\n🎯 Step 7: Final Migration SQL Generated")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("📄 Generated final migration (%d characters)\n", len(finalMigrationSQL))
		fmt.Println("🚀 Users can run this single file instead of multiple migrations!\n")
		
		fmt.Println("📋 Final Migration Content:")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println(finalMigrationSQL)
		fmt.Println(strings.Repeat("─", 60))
	}

	// Step 8: Display LLM relationship analysis
	if llmRelationships != "" {
		fmt.Println("\n🤖 Step 8: LLM Relationship Analysis Results")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("📊 LLM-generated Mermaid relationships (%d characters)\n", len(llmRelationships))
		fmt.Println("🔍 Includes both explicit foreign keys AND implicit relationships!\n")
		
		fmt.Println("📋 LLM Relationship Diagram:")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println(llmRelationships)
		fmt.Println(strings.Repeat("─", 60))
	} else {
		fmt.Println("\n🤖 Step 8: LLM Relationship Analysis")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("⚠️  LLM relationship analysis was not performed or failed")
		fmt.Println("💡 This could be due to missing OpenAI configuration or API errors")
	}

	fmt.Println("\n✅ Database schema extraction completed successfully!")
	fmt.Println("🎯 SUCCESS: Generated single migration file representing final database state!")
	if llmRelationships != "" {
		fmt.Println("🤖 BONUS: LLM enhanced with implicit relationship detection!")
	}
}

// scanFiles recursively scans a directory for all files
func scanFiles(rootPath string) (map[string]string, error) {
	files := make(map[string]string)
	
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		// Get relative path
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		
		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			// Skip files we can't read
			return nil
		}
		
		files[relPath] = string(content)
		return nil
	})
	
	return files, err
}

// filterSQLFiles filters for SQL files that might contain migrations
func filterSQLFiles(files map[string]string) map[string]string {
	sqlFiles := make(map[string]string)
	
	for path, content := range files {
		if strings.HasSuffix(strings.ToLower(path), ".sql") {
			sqlFiles[path] = content
		}
	}
	
	return sqlFiles
}

// findMigrationDirectories finds directories that contain "migration" in their path
func findMigrationDirectories(sqlFiles map[string]string) []string {
	dirSet := make(map[string]bool)
	
	for path := range sqlFiles {
		dir := filepath.Dir(path)
		
		// Check if any part of the path contains "migration"
		pathParts := strings.Split(dir, string(os.PathSeparator))
		for _, part := range pathParts {
			if strings.Contains(strings.ToLower(part), "migration") {
				dirSet[dir] = true
				break
			}
		}
	}
	
	var dirs []string
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	
	return dirs
}

func runDetectionTest(projectPath string) {
	if projectPath == "" {
		fmt.Println("Please provide -path for detection testing")
		return
	}
	
	fmt.Printf("🧪 Testing project type detection for: %s\n", projectPath)
	
	// Use the project detector directly without full pipeline
	// We'll mimic what the crawler does - discover files and read key ones
	files, fileContents, err := discoverFilesForDetection(projectPath)
	if err != nil {
		fmt.Printf("❌ Error discovering files: %v\n", err)
		return
	}
	
	fmt.Printf("📁 Found %d files\n", len(files))
	
	// Use the detector directly
	detector := detector.NewProjectDetector()
	result := detector.DetectProjectType(files, fileContents)
	
	// Print detection results
	fmt.Printf("\n📊 PROJECT TYPE DETECTION RESULTS:\n")
	fmt.Printf("├── Primary Type: %s\n", result.PrimaryType)
	fmt.Printf("├── Secondary Type: %s\n", result.SecondaryType)
	fmt.Printf("├── Confidence: %.1f/10.0\n", result.Confidence)
	
	fmt.Printf("\n🎯 CONFIDENCE SCORES:\n")
	for projectType, score := range result.Scores {
		fmt.Printf("├── %s: %.2f\n", projectType, score)
	}
	
	fmt.Printf("\n🔍 DETECTION EVIDENCE:\n")
	for category, evidence := range result.Evidence {
		fmt.Printf("├── %s:\n", category)
		for _, item := range evidence {
			fmt.Printf("│   • %s\n", item)
		}
	}
	
	// Show critical diagnosis
	if result.PrimaryType == "Backend" && result.Confidence > 5.0 {
		fmt.Printf("\n⚠️  POTENTIAL ISSUE: High confidence Backend detection - verify this is correct!\n")
	}
	if result.PrimaryType == "Frontend" {
		fmt.Printf("\n✅ LOOKS GOOD: Correctly detected as Frontend project\n")
	}
}

// discoverFilesForDetection discovers files and reads important ones for detection testing
func discoverFilesForDetection(projectPath string) ([]detector.FileInfo, map[string]string, error) {
	var files []detector.FileInfo
	fileContents := make(map[string]string)
	
	err := filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip common ignore patterns
		if shouldIgnoreForDetection(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !d.IsDir() {
			relPath, _ := filepath.Rel(projectPath, path)
			info, _ := d.Info()
			
			fileInfo := detector.FileInfo{
				Path:         path,
				RelativePath: relPath,
				Size:         info.Size(),
				Extension:    strings.ToLower(filepath.Ext(path)),
				IsDir:        false,
			}
			files = append(files, fileInfo)
			
			// Read important files for detection
			if isImportantForDetection(relPath) {
				content, err := os.ReadFile(path)
				if err == nil && len(content) < 100*1024 { // Only read files < 100KB
					fileContents[path] = string(content)
				}
			}
		}
		
		return nil
	})
	
	return files, fileContents, err
}

func shouldIgnoreForDetection(path string) bool {
	ignorePatterns := []string{
		"node_modules", "vendor", "target", "build", "dist", ".git", 
		".next", ".nuxt", "coverage", "__pycache__", ".pytest_cache",
	}
	
	for _, pattern := range ignorePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func isImportantForDetection(relPath string) bool {
	importantFiles := []string{
		"package.json", "package-lock.json", "yarn.lock",
		"go.mod", "go.sum", "Cargo.toml", "requirements.txt",
		"pom.xml", "build.gradle", "composer.json",
		"angular.json", "next.config.js", "nuxt.config.js",
		"vite.config.js", "webpack.config.js",
	}
	
	fileName := strings.ToLower(filepath.Base(relPath))
	for _, important := range importantFiles {
		if fileName == important {
			return true
		}
	}
	return false
}
