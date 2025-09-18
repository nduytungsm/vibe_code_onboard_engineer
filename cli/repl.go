package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"repo-explanation/config"
	"repo-explanation/internal/pipeline"
)

type REPL struct {
	scanner    *bufio.Scanner
	running    bool
	pathSet    bool
	targetPath string
}

func NewREPL() *REPL {
	return &REPL{
		scanner: bufio.NewScanner(os.Stdin),
		running: true,
		pathSet: false,
	}
}

func (r *REPL) Start() {
	fmt.Println("🚀 Repo Explanation CLI Started")

	// First, prompt for folder path
	if !r.promptForPath() {
		return
	}

	// Then start command loop
	fmt.Println("Type 'try me' to test, '/end' to exit")
	fmt.Print("> ")

	for r.running && r.scanner.Scan() {
		input := strings.TrimSpace(r.scanner.Text())
		r.processCommand(input)

		if r.running {
			fmt.Print("> ")
		}
	}

	if err := r.scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func (r *REPL) promptForPath() bool {
	fmt.Print("Please enter the relative path to a folder: ")

	if !r.scanner.Scan() {
		return false
	}

	input := strings.TrimSpace(r.scanner.Text())
	if input == "" {
		fmt.Println("Path cannot be empty")
		return false
	}

	// Expand path (handle ~ and other special cases)
	expandedPath, err := r.expandPath(input)
	if err != nil {
		fmt.Printf("Invalid path: %v\n", err)
		return false
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		fmt.Printf("Invalid path: %v\n", err)
		return false
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Printf("Path does not exist: %v\n", err)
		return false
	}

	if !info.IsDir() {
		fmt.Printf("Path is not a directory: %s\n", absPath)
		return false
	}

	r.targetPath = absPath
	r.pathSet = true

	// Count folders and report
	folderCount, err := r.countFolders(absPath)
	if err != nil {
		fmt.Printf("Error counting folders: %v\n", err)
		return false
	}

	fmt.Printf("Total number of folders in '%s': %d\n", input, folderCount)
	
	// Start repository analysis
	if err := r.analyzeRepository(); err != nil {
		fmt.Printf("Error analyzing repository: %v\n", err)
		return false
	}
	
	fmt.Println()
	return true
}

func (r *REPL) expandPath(path string) (string, error) {
	// Handle tilde expansion for home directory
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %v", err)
		}

		if path == "~" {
			return usr.HomeDir, nil
		} else if strings.HasPrefix(path, "~/") {
			return filepath.Join(usr.HomeDir, path[2:]), nil
		}
		// For cases like ~username, we don't handle those here
		return path, nil
	}

	return path, nil
}

func (r *REPL) countFolders(rootPath string) (int, error) {
	count := 0

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Count directories, but skip the root directory itself
		if info.IsDir() && path != rootPath {
			count++
		}

		return nil
	})

	return count, err
}

func (r *REPL) loadConfig() (*config.Config, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}
	
	// List of possible config file locations
	configPaths := []string{
		filepath.Join(cwd, "config.yaml"),       // Current directory
		filepath.Join(cwd, "..", "config.yaml"), // Parent directory
		"config.yaml",                           // Relative to current dir
	}
	
	var lastErr error
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("Found config file: %s\n", path)
			cfg, err := config.LoadConfig(path)
			if err != nil {
				fmt.Printf("Error loading config from %s: %v\n", path, err)
				lastErr = err
				continue
			}
			return cfg, nil
		}
	}
	
	// If no config file found, return the last error or a generic error
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("config.yaml not found in any of the expected locations: %v", configPaths)
}

func (r *REPL) analyzeRepository() error {
	// Find and load configuration
	cfg, err := r.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Validate API key
	if cfg.OpenAI.APIKey == "" {
		return fmt.Errorf("OpenAI API key not configured. Please set OPENAI_API_KEY environment variable or update config.yaml")
	}

	fmt.Println("\n🧠 Starting repository analysis with LLM...")
	startTime := time.Now()

	// Create analyzer
	analyzer, err := pipeline.NewAnalyzer(cfg, r.targetPath)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %v", err)
	}

	// Run analysis
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := analyzer.AnalyzeProject(ctx)
	if err != nil {
		return fmt.Errorf("analysis failed: %v", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("\n⏱️  Analysis completed in %.2f seconds\n", duration.Seconds())

	// Display results
	r.displayAnalysisResults(result)

	return nil
}

func (r *REPL) displayAnalysisResults(result *pipeline.AnalysisResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 REPOSITORY ANALYSIS RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	// Display project type summary at the top
	if result.ProjectType != nil {
		result.ProjectType.PrintSummary()
		fmt.Println()
	}

	if result.ProjectSummary != nil {
		fmt.Println("\n🎯 PURPOSE:")
		fmt.Printf("   %s\n", result.ProjectSummary.Purpose)

		fmt.Println("\n🏗️  ARCHITECTURE:")
		fmt.Printf("   %s\n", result.ProjectSummary.Architecture)

		if len(result.ProjectSummary.DataModels) > 0 {
			fmt.Println("\n📋 DATA MODELS:")
			for _, model := range result.ProjectSummary.DataModels {
				fmt.Printf("   • %s\n", model)
			}
		}

		if len(result.ProjectSummary.ExternalServices) > 0 {
			fmt.Println("\n🔗 EXTERNAL SERVICES:")
			for _, service := range result.ProjectSummary.ExternalServices {
				fmt.Printf("   • %s\n", service)
			}
		}

		fmt.Println("\n📝 SUMMARY:")
		fmt.Printf("   %s\n", result.ProjectSummary.Summary)
	}

	// Show statistics
	if stats, ok := result.Stats["total_files"].(int); ok && stats > 0 {
		fmt.Println("\n📈 STATISTICS:")
		fmt.Printf("   • Files analyzed: %d\n", stats)
		if totalSize, ok := result.Stats["total_size_mb"].(float64); ok {
			fmt.Printf("   • Total size: %.2f MB\n", totalSize)
		}
		if extensions, ok := result.Stats["extensions"].(map[string]int); ok {
			fmt.Println("   • File types:")
			for ext, count := range extensions {
				if ext == "" {
					ext = "(no extension)"
				}
				fmt.Printf("     - %s: %d files\n", ext, count)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

func (r *REPL) processCommand(input string) {
	switch input {
	case "try me":
		fmt.Println("i am here")
	case "/end":
		fmt.Println("Goodbye! 👋")
		r.running = false
	case "":
		// Do nothing for empty input
	default:
		fmt.Println("unsupported function")
	}
}
