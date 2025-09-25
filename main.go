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
	"repo-explanation/routes"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	mode := flag.String("mode", "server", "Mode to run: 'server', 'cli', or 'debug-db'")
	flag.Parse()

	switch *mode {
	case "server":
		runServer()
	case "cli":
		runCLI()
	case "debug-db":
		runDebugDB()
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

	fmt.Println("\n✅ Database schema extraction completed successfully!")
	fmt.Println("🎯 SUCCESS: Generated single migration file representing final database state!")
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
