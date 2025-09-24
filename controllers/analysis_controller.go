package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"repo-explanation/config"
	"repo-explanation/internal/pipeline"
)

type AnalysisController struct {
	config *config.Config
}

type AnalysisRequest struct {
	URL   string `json:"url" validate:"required"`
	Type  string `json:"type" validate:"required"`
	Token string `json:"token,omitempty"` // GitHub personal access token for private repos
}

type AnalysisResponse struct {
	Status     string                 `json:"status"`
	Message    string                 `json:"message,omitempty"`
	Results    *pipeline.AnalysisResult `json:"results,omitempty"`
	Repository *RepositoryInfo        `json:"repository,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type RepositoryInfo struct {
	URL       string `json:"url"`
	Name      string `json:"name"`
	Owner     string `json:"owner"`
	LocalPath string `json:"local_path,omitempty"`
}

type StreamEvent struct {
	Type      string      `json:"type"`      // "progress", "stage", "data", "complete", "error"
	Stage     string      `json:"stage"`     // Current stage description
	Progress  int         `json:"progress"`  // Progress percentage (0-100)
	Data      interface{} `json:"data"`      // Partial or complete analysis data
	Message   string      `json:"message"`   // Status message
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Use the pipeline's ProgressCallback type to avoid type conflicts

func NewAnalysisController() *AnalysisController {
	// Try multiple config paths for different environments
	configPaths := []string{
		"config.yaml",        // Same directory (Docker container)
		"../config.yaml",     // Parent directory (local development)
		"./config.yaml",      // Current working directory
		"/app/config.yaml",   // Absolute path in container
	}
	
	var cfg *config.Config
	var err error
	
	for _, path := range configPaths {
		cfg, err = config.LoadConfig(path)
		if err == nil {
			break
		}
	}
	
	if cfg == nil {
		panic(fmt.Sprintf("Failed to load config from any path %v: %v", configPaths, err))
	}
	
	return &AnalysisController{
		config: cfg,
	}
}

func (ac *AnalysisController) AnalyzeRepository(c echo.Context) error {
	// Parse request
	var req AnalysisRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, AnalysisResponse{
			Status: "error",
			Error:  "Invalid request format",
		})
	}

	// Validate GitHub URL
	if req.Type != "github_url" {
		return c.JSON(http.StatusBadRequest, AnalysisResponse{
			Status: "error",
			Error:  "Only GitHub URLs are supported",
		})
	}

	if !isValidGitHubURL(req.URL) {
		return c.JSON(http.StatusBadRequest, AnalysisResponse{
			Status: "error",
			Error:  "Invalid GitHub URL format",
		})
	}

	// Extract repository info
	repoInfo := extractRepoInfo(req.URL)
	
	// Create temporary directory for cloning
	tempDir := filepath.Join(os.TempDir(), "repo-analysis", fmt.Sprintf("%s-%s-%d", 
		repoInfo.Owner, repoInfo.Name, time.Now().Unix()))
	
	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, AnalysisResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to create temp directory: %v", err),
		})
	}

	// Clean up temp directory after analysis
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("Warning: Failed to clean up temp directory %s: %v\n", tempDir, err)
		}
	}()

	repoInfo.LocalPath = tempDir
	
	// Clone the repository (try public first, then with token if needed)
	c.Logger().Infof("Cloning repository %s to %s", req.URL, tempDir)
	
	// First try public access
	err := cloneRepository(req.URL, tempDir, "")
	if err != nil {
		c.Logger().Warnf("Public clone failed for %s: %v", req.URL, err)
		
		// Check if this looks like a private repo error and we have a token
		if isPrivateRepoError(err) {
			if req.Token == "" {
				return c.JSON(http.StatusUnauthorized, AnalysisResponse{
					Status: "auth_required",
					Error:  "Repository appears to be private. Please provide a GitHub personal access token.",
					Repository: &repoInfo,
				})
			}
			
			// Try again with token
			c.Logger().Infof("Retrying clone with authentication token for %s", req.URL)
			err = cloneRepository(req.URL, tempDir, req.Token)
			if err != nil {
				c.Logger().Errorf("Authenticated clone failed for %s: %v", req.URL, err)
				return c.JSON(http.StatusUnauthorized, AnalysisResponse{
					Status: "error",
					Error:  fmt.Sprintf("Failed to clone repository with provided token: %v", err),
					Repository: &repoInfo,
				})
			}
		} else {
			// Not a private repo error, return the original error
			c.Logger().Errorf("Clone failed for %s: %v", req.URL, err)
			return c.JSON(http.StatusInternalServerError, AnalysisResponse{
				Status: "error",
				Error:  fmt.Sprintf("Failed to clone repository: %v", err),
				Repository: &repoInfo,
			})
		}
	}
	
	c.Logger().Infof("Successfully cloned repository %s", req.URL)

	// Perform analysis using existing pipeline
	c.Logger().Infof("Starting analysis of cloned repository")
	analyzer, err := pipeline.NewAnalyzer(ac.config, tempDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AnalysisResponse{
			Status:     "error", 
			Error:      fmt.Sprintf("Failed to create analyzer: %v", err),
			Repository: &repoInfo,
		})
	}

	// Run analysis with extended timeout (30 minutes for large repositories)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Minute)
	defer cancel()

	// Create a channel to handle analysis result or timeout
	resultChan := make(chan *pipeline.AnalysisResult, 1)
	errorChan := make(chan error, 1)

	// Run analysis in a goroutine
	go func() {
		c.Logger().Infof("Analysis pipeline started for %s", req.URL)
		results, err := analyzer.AnalyzeProject(ctx)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- results
	}()

	// Wait for either completion or timeout
	select {
	case results := <-resultChan:
		c.Logger().Infof("Analysis completed successfully for %s", req.URL)
		return c.JSON(http.StatusOK, AnalysisResponse{
			Status:     "success",
			Message:    "Repository analysis completed successfully",
			Results:    results,
			Repository: &repoInfo,
		})
		
	case err := <-errorChan:
		c.Logger().Errorf("Analysis failed for %s: %v", req.URL, err)
		return c.JSON(http.StatusInternalServerError, AnalysisResponse{
			Status:     "error",
			Error:      fmt.Sprintf("Analysis failed: %v", err), 
			Repository: &repoInfo,
		})
		
	case <-ctx.Done():
		c.Logger().Warnf("Analysis timed out for %s after 30 minutes", req.URL)
		return c.JSON(http.StatusRequestTimeout, AnalysisResponse{
			Status:     "timeout",
			Error:      "Analysis timed out after 30 minutes. The repository may be too large or complex for analysis.",
			Repository: &repoInfo,
		})
	}
}

// isValidGitHubURL validates if the URL is a valid GitHub repository URL
func isValidGitHubURL(url string) bool {
	return strings.HasPrefix(url, "https://github.com/") && strings.Count(url, "/") >= 4
}

// extractRepoInfo extracts owner and repository name from GitHub URL
func extractRepoInfo(url string) RepositoryInfo {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")
	
	// Split URL to get owner and repo
	parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
	
	owner := ""
	name := ""
	
	if len(parts) >= 2 {
		owner = parts[0]
		name = parts[1]
	}

	return RepositoryInfo{
		URL:   url,
		Owner: owner,
		Name:  name,
	}
}

// cloneRepository clones a GitHub repository to the specified directory
func cloneRepository(url, destDir, token string) error {
	// Ensure we're using HTTPS URL format
	cloneURL := normalizeGitHubURL(url)
	
	// If we have a token, inject it into the URL for authentication
	if token != "" {
		cloneURL = injectTokenIntoURL(cloneURL, token)
	}
	
	fmt.Printf("DEBUG: Original URL: %s, Clone URL: %s (token: %t)\n", url, maskTokenInURL(cloneURL), token != "")
	
	// Set timeout for clone operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	// Use git clone command with HTTPS and explicit config to prevent SSH rewriting
	cmd := exec.CommandContext(ctx, "git", 
		"-c", "url.https://github.com/.insteadof=ssh://git@github.com/",
		"-c", "url.https://github.com/.insteadof=git@github.com:",
		"clone", "--depth", "1", cloneURL, destDir)
	
	// Set environment to avoid SSH key prompts and force HTTPS
	cmd.Env = append(os.Environ(), 
		"GIT_TERMINAL_PROMPT=0", // Disable interactive prompts
		"GIT_ASKPASS=echo",      // Provide empty password for HTTPS
		"GIT_CONFIG_GLOBAL=/dev/null", // Ignore global git config
		"GIT_CONFIG_SYSTEM=/dev/null", // Ignore system git config
	)
	
	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %v, output: %s", err, string(output))
	}

	return nil
}

// normalizeGitHubURL ensures the URL is in HTTPS format for public cloning
func normalizeGitHubURL(url string) string {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")
	
	// Convert SSH format to HTTPS if needed
	if strings.HasPrefix(url, "git@github.com:") {
		// Convert git@github.com:owner/repo to https://github.com/owner/repo
		url = "https://github.com/" + strings.TrimPrefix(url, "git@github.com:")
	}
	
	// Ensure HTTPS format
	if !strings.HasPrefix(url, "https://github.com/") {
		return url // Return as-is if not a recognized GitHub URL
	}
	
	// Add .git suffix for reliable cloning
	return url + ".git"
}

// injectTokenIntoURL adds a GitHub personal access token to the URL for authentication
func injectTokenIntoURL(url, token string) string {
	// Convert https://github.com/owner/repo.git to https://token@github.com/owner/repo.git
	if strings.HasPrefix(url, "https://github.com/") {
		return strings.Replace(url, "https://github.com/", fmt.Sprintf("https://%s@github.com/", token), 1)
	}
	return url
}

// maskTokenInURL masks the token in URL for safe logging
func maskTokenInURL(url string) string {
	// Replace any token in the URL with asterisks for logging
	if strings.Contains(url, "@github.com/") {
		parts := strings.Split(url, "@github.com/")
		if len(parts) == 2 {
			return "https://***@github.com/" + parts[1]
		}
	}
	return url
}

// isPrivateRepoError checks if the error indicates a private repository access issue
func isPrivateRepoError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "authentication failed") ||
		   strings.Contains(errStr, "invalid username or token") ||
		   strings.Contains(errStr, "repository not found") ||
		   strings.Contains(errStr, "password authentication is not supported") ||
		   strings.Contains(errStr, "permission denied")
}

// StreamAnalyzeRepository provides real-time analysis progress via Server-Sent Events
func (ac *AnalysisController) StreamAnalyzeRepository(c echo.Context) error {
	fmt.Println("🔍 [STREAM] Starting StreamAnalyzeRepository")
	
	// Parse request
	var req AnalysisRequest
	if err := c.Bind(&req); err != nil {
		fmt.Printf("❌ [STREAM] Failed to bind request: %v\n", err)
		return c.JSON(http.StatusBadRequest, AnalysisResponse{
			Status: "error",
			Error:  "Invalid request format",
		})
	}
	
	fmt.Printf("✅ [STREAM] Request parsed: URL=%s, Type=%s, HasToken=%v\n", req.URL, req.Type, req.Token != "")

	// Validate GitHub URL
	if req.Type != "github_url" {
		fmt.Printf("❌ [STREAM] Invalid request type: %s\n", req.Type)
		return c.JSON(http.StatusBadRequest, AnalysisResponse{
			Status: "error",
			Error:  "Only GitHub URLs are supported",
		})
	}

	if !isValidGitHubURL(req.URL) {
		fmt.Printf("❌ [STREAM] Invalid GitHub URL: %s\n", req.URL)
		return c.JSON(http.StatusBadRequest, AnalysisResponse{
			Status: "error",
			Error:  "Invalid GitHub URL format",
		})
	}
	
	fmt.Println("✅ [STREAM] URL validation passed")

	// Set up SSE headers
	fmt.Println("🔧 [STREAM] Setting up SSE headers")
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Response().Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	fmt.Println("✅ [STREAM] SSE headers configured")

	// Create progress callback for streaming updates
	fmt.Println("🔧 [STREAM] Creating progress callback")
	progressCallback := pipeline.ProgressCallback(func(eventType, stage, message string, progress int, data interface{}) {
		fmt.Printf("📡 [STREAM] Progress callback: type=%s, stage=%s, progress=%d, message=%s\n", eventType, stage, progress, message)
		
		event := StreamEvent{
			Type:      eventType,
			Stage:     stage,
			Progress:  progress,
			Data:      data,
			Message:   message,
			Timestamp: time.Now(),
		}
		
		eventJSON, err := json.Marshal(event)
		if err != nil {
			fmt.Printf("❌ [STREAM] Failed to marshal event: %v\n", err)
			return
		}
		
		fmt.Printf("📤 [STREAM] Sending event: %s\n", string(eventJSON))
		fmt.Fprintf(c.Response(), "data: %s\n\n", string(eventJSON))
		c.Response().Flush()
	})

	// Send initial progress event
	fmt.Println("🚀 [STREAM] Sending initial progress event")
	progressCallback("progress", "🚀 Initializing analysis...", "Starting repository analysis", 0, nil)

	// Extract repository info
	fmt.Println("🔍 [STREAM] Extracting repository info")
	repoInfo := extractRepoInfo(req.URL)
	fmt.Printf("✅ [STREAM] Repository info: Owner=%s, Name=%s\n", repoInfo.Owner, repoInfo.Name)
	
	// Create temporary directory for cloning
	fmt.Println("📁 [STREAM] Creating temporary directory")
	tempDir := filepath.Join(os.TempDir(), "repo-analysis", fmt.Sprintf("%s-%s-%d", 
		repoInfo.Owner, repoInfo.Name, time.Now().Unix()))
	fmt.Printf("📁 [STREAM] Temp directory: %s\n", tempDir)
	
	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		fmt.Printf("❌ [STREAM] Failed to create temp directory: %v\n", err)
		progressCallback("error", "", "Failed to create temporary directory", 0, nil)
		return nil
	}
	fmt.Println("✅ [STREAM] Temporary directory created")

	// Clean up temp directory after analysis
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("Warning: Failed to clean up temp directory %s: %v\n", tempDir, err)
		}
	}()

	repoInfo.LocalPath = tempDir
	
	// Clone the repository with progress updates
	fmt.Println("📂 [STREAM] Starting repository clone")
	progressCallback("progress", "📂 Cloning repository from GitHub...", "Downloading repository files", 5, nil)
	
	// First try public access
	fmt.Println("🔓 [STREAM] Attempting public repository clone")
	err := cloneRepository(req.URL, tempDir, "")
	if err != nil {
		fmt.Printf("⚠️ [STREAM] Public clone failed: %v\n", err)
		
		// Check if this looks like a private repo error and we have a token
		if isPrivateRepoError(err) {
			fmt.Println("🔒 [STREAM] Detected private repository error")
			if req.Token == "" {
				fmt.Println("❌ [STREAM] No token provided for private repository")
				progressCallback("error", "", "Repository appears to be private. Please provide a GitHub personal access token.", 0, map[string]interface{}{
					"auth_required": true,
					"repository":    repoInfo,
				})
				return nil
			}
			
			// Try again with token
			fmt.Println("🔐 [STREAM] Retrying with provided token")
			progressCallback("progress", "🔐 Authenticating with GitHub...", "Using provided access token", 8, nil)
			err = cloneRepository(req.URL, tempDir, req.Token)
			if err != nil {
				fmt.Printf("❌ [STREAM] Clone with token failed: %v\n", err)
				progressCallback("error", "", fmt.Sprintf("Failed to clone repository with provided token: %v", err), 0, nil)
				return nil
			}
		} else {
			fmt.Printf("❌ [STREAM] Clone failed with non-private error: %v\n", err)
			progressCallback("error", "", fmt.Sprintf("Failed to clone repository: %v", err), 0, nil)
			return nil
		}
	}
	
	fmt.Println("✅ [STREAM] Repository cloned successfully")
	
	progressCallback("progress", "✅ Repository cloned successfully", "Repository files downloaded", 15, nil)

	// Perform analysis with progress updates
	fmt.Println("🔬 [STREAM] Creating analyzer")
	analyzer, err := pipeline.NewAnalyzer(ac.config, tempDir)
	if err != nil {
		fmt.Printf("❌ [STREAM] Failed to create analyzer: %v\n", err)
		progressCallback("error", "", fmt.Sprintf("Failed to create analyzer: %v", err), 0, nil)
		return nil
	}
	fmt.Println("✅ [STREAM] Analyzer created successfully")

	// Run analysis with extended timeout and progress callbacks
	fmt.Println("⏱️ [STREAM] Setting up 30-minute context timeout")
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Minute)
	defer cancel()

	// Run streaming analysis
	fmt.Println("🚀 [STREAM] Starting streaming analysis")
	results, err := ac.runStreamingAnalysis(ctx, analyzer, progressCallback)
	if err != nil {
		fmt.Printf("❌ [STREAM] Analysis failed: %v\n", err)
		progressCallback("error", "", fmt.Sprintf("Analysis failed: %v", err), 0, nil)
		return nil
	}
	
	fmt.Println("✅ [STREAM] Analysis completed successfully")

	// Send completion event with full results
	fmt.Println("🎉 [STREAM] Sending completion event")
	progressCallback("complete", "🎉 Analysis complete!", "Repository analysis finished successfully", 100, results)
	
	fmt.Println("✅ [STREAM] StreamAnalyzeRepository completed")
	return nil
}

// runStreamingAnalysis runs the analysis pipeline with progress callbacks
func (ac *AnalysisController) runStreamingAnalysis(ctx context.Context, analyzer *pipeline.Analyzer, callback pipeline.ProgressCallback) (*pipeline.AnalysisResult, error) {
	fmt.Println("🔄 [STREAM] Entering runStreamingAnalysis")
	
	// Create custom analyzer that emits progress
	fmt.Println("📊 [STREAM] Calling analyzer.AnalyzeProjectWithProgress")
	result, err := analyzer.AnalyzeProjectWithProgress(ctx, callback)
	
	if err != nil {
		fmt.Printf("❌ [STREAM] AnalyzeProjectWithProgress failed: %v\n", err)
		return nil, err
	}
	
	fmt.Println("✅ [STREAM] AnalyzeProjectWithProgress completed successfully")
	return result, nil
}
