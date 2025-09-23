package controllers

import (
	"context"
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

func NewAnalysisController() *AnalysisController {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		// Try to load from parent directory if not found
		cfg, err = config.LoadConfig("../config.yaml")
		if err != nil {
			panic(fmt.Sprintf("Failed to load config: %v", err))
		}
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
