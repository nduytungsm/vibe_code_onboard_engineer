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
	Summary       string                    `json:"summary"`
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
	
	return fmt.Sprintf(`Analyze this entire project and create a comprehensive overview. Return a JSON object:

{
  "purpose": "what this entire project/repository does",
  "architecture": "overall architecture description (MVC, microservices, etc.)",
  "data_models": ["important", "data", "structures"],
  "external_services": ["external", "apis", "databases", "services"],
  "languages": {"language": total_file_count},
  "summary": "exactly 2 sentences describing the project purpose and what it helps achieve"
}

Project path: %s
Folder summaries: %s`, projectPath, string(summariesJSON))
}
