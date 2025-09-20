package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"repo-explanation/config"
)

// Client wraps the OpenAI client with rate limiting and error handling
type Client struct {
	client      *openai.Client
	config      *config.Config
	rateLimiter *RateLimiter
}

// FileSummary represents the structured output from LLM analysis
type FileSummary struct {
	Language    string            `json:"language"`
	Purpose     string            `json:"purpose"`
	KeyTypes    []string          `json:"key_types"`
	Functions   []string          `json:"functions"`
	Imports     []string          `json:"imports"`
	SideEffects []string          `json:"side_effects,omitempty"`
	Risks       []string          `json:"risks,omitempty"`
	Complexity  string            `json:"complexity"` // "low", "medium", "high"
}

// FolderSummary represents aggregated analysis of a folder
type FolderSummary struct {
	Path        string                   `json:"path"`
	Purpose     string                   `json:"purpose"`
	Languages   map[string]int           `json:"languages"`
	KeyModules  []string                 `json:"key_modules"`
	Dependencies []string                `json:"dependencies"`
	Architecture string                  `json:"architecture"`
	FileSummaries map[string]FileSummary `json:"file_summaries"`
}

// ProjectSummary represents the final project overview
type ProjectSummary struct {
	Purpose       string                    `json:"purpose"`
	Architecture  string                    `json:"architecture"`
	DataModels    []string                  `json:"data_models"`
	ExternalServices []string               `json:"external_services"`
	Languages     map[string]int            `json:"languages"`
	FolderSummaries map[string]FolderSummary `json:"folder_summaries"`
	// Enhanced analysis fields
	DetailedAnalysis *RepositoryAnalysis     `json:"detailed_analysis,omitempty"`
}

// RepositoryAnalysis contains detailed architectural analysis
type RepositoryAnalysis struct {
	RepoSummaryLine   string             `json:"repo_summary_line"`
	Architecture      string             `json:"architecture"` // "monolith" or "microservices"
	RepoLayout        string             `json:"repo_layout"`  // "single-repo" or "monorepo"
	MainStacks        []string           `json:"main_stacks"`
	MonorepoServices  []MonorepoService  `json:"monorepo_services"`
	EvidencePaths     []string           `json:"evidence_paths"`
	Confidence        float64            `json:"confidence"`
}

// MonorepoService represents a service in a monorepo
type MonorepoService struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Language     string `json:"language"`
	ShortPurpose string `json:"short_purpose"`
	APIType      string `json:"api_type,omitempty"`      // http, grpc, graphql
	Port         string `json:"port,omitempty"`          // service port if detected
	EntryPoint   string `json:"entry_point,omitempty"`   // main.go, index.js, etc.
}

// NewClient creates a new OpenAI client with configuration
func NewClient(cfg *config.Config) *Client {
	client := openai.NewClient(cfg.OpenAI.APIKey)
	if cfg.OpenAI.BaseURL != "" {
		config := openai.DefaultConfig(cfg.OpenAI.APIKey)
		config.BaseURL = cfg.OpenAI.BaseURL
		client = openai.NewClientWithConfig(config)
	}

	rateLimiter := NewRateLimiter(
		cfg.RateLimiting.RequestsPerMinute,
		cfg.RateLimiting.RequestsPerDay,
	)

	return &Client{
		client:      client,
		config:      cfg,
		rateLimiter: rateLimiter,
	}
}

// AnalyzeFile sends file content to OpenAI for analysis
func (c *Client) AnalyzeFile(ctx context.Context, filepath, content string) (*FileSummary, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %v", err)
	}

	prompt := c.buildFileAnalysisPrompt(filepath, content)
	
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.config.OpenAI.Model,
		Temperature: c.config.OpenAI.Temperature,
		MaxTokens:   c.config.OpenAI.MaxTokensPerRequest,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a code analysis expert. Analyze the provided code and return ONLY valid JSON in the specified format. No additional text or explanations.",
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
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	var summary FileSummary
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &summary); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %v", err)
	}

	return &summary, nil
}

// AnalyzeFolder aggregates file summaries into a folder summary
func (c *Client) AnalyzeFolder(ctx context.Context, folderPath string, fileSummaries map[string]FileSummary) (*FolderSummary, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %v", err)
	}

	prompt := c.buildFolderAnalysisPrompt(folderPath, fileSummaries)
	
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.config.OpenAI.Model,
		Temperature: c.config.OpenAI.Temperature,
		MaxTokens:   c.config.OpenAI.MaxTokensPerRequest,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a software architecture expert. Analyze the provided folder structure and file summaries. Return ONLY valid JSON in the specified format.",
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
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	var summary FolderSummary
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &summary); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %v", err)
	}

	summary.FileSummaries = fileSummaries
	return &summary, nil
}

// AnalyzeProject creates the final project summary
func (c *Client) AnalyzeProject(ctx context.Context, projectPath string, folderSummaries map[string]FolderSummary) (*ProjectSummary, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %v", err)
	}

	prompt := c.buildProjectAnalysisPrompt(projectPath, folderSummaries)
	
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.config.OpenAI.Model,
		Temperature: c.config.OpenAI.Temperature,
		MaxTokens:   c.config.OpenAI.MaxTokensPerRequest,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a senior software architect. Analyze the entire project structure and create a comprehensive overview. Return ONLY valid JSON. The summary field should be exactly 2 sentences explaining what this project does and its purpose.",
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
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	var summary ProjectSummary
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &summary); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %v", err)
	}

	summary.FolderSummaries = folderSummaries
	return &summary, nil
}

// AnalyzeRepositoryDetails performs detailed architectural analysis
func (c *Client) AnalyzeRepositoryDetails(ctx context.Context, projectPath string, folderSummaries map[string]FolderSummary, fileSummaries map[string]FileSummary, importantFiles map[string]string) (*RepositoryAnalysis, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %v", err)
	}

	prompt := c.buildDetailedAnalysisPrompt(projectPath, folderSummaries, fileSummaries, importantFiles)
	
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.config.OpenAI.Model,
		Temperature: 0.0, // Very low for consistent structured output
		MaxTokens:   c.config.OpenAI.MaxTokensPerRequest,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: `You are a precise repository analyst. Output STRICT JSON only, no prose, matching the provided schema exactly. 
Do not guess. Use only evidence present in the repository summaries/metadata provided. 
If uncertain, return "" or [] and lower confidence.`,
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
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	var analysis RepositoryAnalysis
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse detailed analysis JSON: %v", err)
	}

	return &analysis, nil
}

func (c *Client) buildFileAnalysisPrompt(filepath, content string) string {
	return fmt.Sprintf(`Analyze this code file and return a JSON object with the following structure:

{
  "language": "detected programming language",
  "purpose": "brief description of what this file does",
  "key_types": ["list", "of", "important", "types/classes/structs"],
  "functions": ["list", "of", "important", "functions/methods"],
  "imports": ["list", "of", "dependencies/imports"],
  "side_effects": ["list", "of", "side", "effects", "if", "any"],
  "risks": ["list", "of", "potential", "security", "risks", "if", "any"],
  "complexity": "low|medium|high"
}

File path: %s
Content:
%s`, filepath, content)
}

func (c *Client) buildFolderAnalysisPrompt(folderPath string, fileSummaries map[string]FileSummary) string {
	summariesJSON, _ := json.Marshal(fileSummaries)
	
	return fmt.Sprintf(`Analyze this folder structure and its file summaries. Return a JSON object with this structure:

{
  "path": "%s",
  "purpose": "what this folder/module is responsible for",
  "languages": {"language": count},
  "key_modules": ["list", "of", "important", "files"],
  "dependencies": ["external", "dependencies"],
  "architecture": "brief description of the folder's architecture pattern"
}

Folder path: %s
File summaries: %s`, folderPath, folderPath, string(summariesJSON))
}

func (c *Client) buildProjectAnalysisPrompt(projectPath string, folderSummaries map[string]FolderSummary) string {
	summariesJSON, _ := json.Marshal(folderSummaries)
	
	return fmt.Sprintf(`Analyze this entire project and create a comprehensive overview. Look at component names, folder structures, route patterns, and business logic to intelligently guess the REAL purpose and business domain.

Examples of good purpose detection:
- Instead of "A React web application" → "A gift recommendation platform for helping users find personalized presents for their partners"
- Instead of "An e-commerce platform" → "A marketplace for handmade jewelry where artisans can sell custom pieces to customers"  
- Instead of "A task management app" → "A project collaboration tool for remote teams to track deadlines and share progress updates"

Return a JSON object:

{
  "purpose": "intelligent guess of the specific business purpose/domain based on component names, routes, data models, and functionality (2-3 lines max)",
  "architecture": "overall architecture description (MVC, microservices, etc.)",
  "data_models": ["important", "data", "structures"],
  "external_services": ["external", "apis", "databases", "services"],  
  "languages": {"language": total_file_count}
}

IMPORTANT: 
- Be specific about the business domain, not just the technology
- Look for clues in component names (UserProfile, ProductCatalog, OrderHistory, etc.)
- Analyze route patterns (/products, /checkout, /dashboard, etc.)
- Consider data models (User, Order, Product, etc.) to understand the domain
- If unclear, make an educated guess based on available evidence
- Keep purpose concise but specific (2-3 lines maximum)

Project path: %s
Folder summaries: %s`, projectPath, string(summariesJSON))
}

func (c *Client) buildDetailedAnalysisPrompt(projectPath string, folderSummaries map[string]FolderSummary, fileSummaries map[string]FileSummary, importantFiles map[string]string) string {
	// Convert summaries to JSON for the prompt
	folderSummariesJSON, _ := json.Marshal(folderSummaries)
	fileSummariesJSON, _ := json.Marshal(fileSummaries)
	importantFilesJSON, _ := json.Marshal(importantFiles)
	
	return fmt.Sprintf(`You are given structured summaries of a code repository (per-file and per-folder), plus key files (README, go.mod/package.json, docker/k8s manifests). 
Determine:
1) repo_summary_line: one concise sentence describing what the repo does.
2) architecture: "monolith" or "microservices". If unclear, choose "monolith".
3) repo_layout: "single-repo" or "monorepo".
4) main_stacks: top-level stacks (language + core runtimes/frameworks) only; exclude minor libs and devtools.
5) monorepo_services: IF repo_layout == "monorepo", list each deployable service with {name, path, language, short_purpose}; otherwise [].
6) evidence_paths: file or directory paths that justify your answers (README, go.mod, docker-compose.yml, k8s manifests, turbo.json, lerna.json, pnpm-workspace.yaml, go.work, apps/*, services/*, cmd/*, etc.).
7) confidence: 0.0–1.0 reflecting certainty.

Rules:
- Architecture vs Layout: "monorepo" is not an architecture. Report architecture separately as monolith/microservices, and layout as single-repo/monorepo.
- Main stacks extraction:
  - Go: go.mod modules and major imports (e.g., echo/gin/grpc, gorm/sqlx, kafka clients).
  - JS/TS: package.json "dependencies" (frameworks like express/fastify/nestjs, prisma, knex).
  - Python: pyproject/requirements (fastapi/django/flask, sqlalchemy).
  - Java: build.gradle/pom.xml (spring-boot, vertx).
  - Infra: docker-compose (services), k8s Deployments, terraform modules.
- Treat dev/test-only deps as non-stacks (linters, formatters, test libs).
- Service detection (monorepo):
  - Evidence: top-level apps/ or services/ directories, multiple cmd/* binaries, multiple Dockerfiles, workspace files (lerna/turbo/nx/pnpm-workspace/go.work), compose with multiple services, k8s with multiple Deployments.
  - Name = directory or README heading; short_purpose from that service's README or top comment.
- Be conservative: if conflicting signals, prefer fewer stacks and lower confidence.
- Output compact JSON only; no markdown.

Inputs:
- project_summaries: %s
- folder_summaries: %s
- important_files: %s

Output schema:
{
  "repo_summary_line": "string",
  "architecture": "monolith" | "microservices",
  "repo_layout": "single-repo" | "monorepo",
  "main_stacks": ["string", ...],
  "monorepo_services": [
    {"name": "string", "path": "string", "language": "string", "short_purpose": "string"}
  ],
  "evidence_paths": ["string", ...],
  "confidence": 0.0
}`, string(fileSummariesJSON), string(folderSummariesJSON), string(importantFilesJSON))
}
