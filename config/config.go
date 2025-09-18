package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	OpenAI          OpenAIConfig          `yaml:"openai"`
	RateLimiting    RateLimitingConfig    `yaml:"rate_limiting"`
	FileProcessing  FileProcessingConfig  `yaml:"file_processing"`
	Cache           CacheConfig           `yaml:"cache"`
	Security        SecurityConfig        `yaml:"security"`
	Output          OutputConfig          `yaml:"output"`
}

type OpenAIConfig struct {
	APIKey              string  `yaml:"api_key"`
	Model               string  `yaml:"model"`
	MaxTokensPerRequest int     `yaml:"max_tokens_per_request"`
	Temperature         float32 `yaml:"temperature"`
	BaseURL             string  `yaml:"base_url"`
}

type RateLimitingConfig struct {
	RequestsPerMinute  int `yaml:"requests_per_minute"`
	RequestsPerDay     int `yaml:"requests_per_day"`
	ConcurrentWorkers  int `yaml:"concurrent_workers"`
}

type FileProcessingConfig struct {
	MaxFileSizeMB         int      `yaml:"max_file_size_mb"`
	ChunkSizeTokens       int      `yaml:"chunk_size_tokens"`
	SupportedExtensions   []string `yaml:"supported_extensions"`
}

type CacheConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
	TTLHours  int    `yaml:"ttl_hours"`
}

type SecurityConfig struct {
	RedactSecrets    bool     `yaml:"redact_secrets"`
	SkipSecretFiles  []string `yaml:"skip_secret_files"`
}

type OutputConfig struct {
	SummaryMaxLength         int    `yaml:"summary_max_length"`
	SaveIntermediateResults  bool   `yaml:"save_intermediate_results"`
	OutputDirectory          string `yaml:"output_directory"`
}

// LoadConfig loads configuration from YAML file with environment variable substitution
func LoadConfig(configPath string) (*Config, error) {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		// Only log if the error is NOT "file not found"
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: Error loading .env file: %v\n", err)
		}
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Substitute environment variables
	content := string(data)
	content = expandEnvVars(content)

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %v", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	// Ensure directories exist
	if err := config.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %v", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}

	if c.OpenAI.Model == "" {
		return fmt.Errorf("OpenAI model is required")
	}

	if c.FileProcessing.ChunkSizeTokens <= 0 {
		return fmt.Errorf("chunk size tokens must be positive")
	}

	if c.RateLimiting.RequestsPerMinute <= 0 {
		return fmt.Errorf("requests per minute must be positive")
	}

	return nil
}

// ensureDirectories creates necessary directories
func (c *Config) ensureDirectories() error {
	dirs := []string{
		c.Cache.Directory,
		c.Output.OutputDirectory,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return nil
}

// expandEnvVars expands environment variables in the format ${VAR_NAME}
func expandEnvVars(content string) string {
	return os.Expand(content, func(key string) string {
		return os.Getenv(key)
	})
}

// GetCacheTTL returns the cache TTL as a time.Duration
func (c *Config) GetCacheTTL() time.Duration {
	return time.Duration(c.Cache.TTLHours) * time.Hour
}

// IsFileSupported checks if a file extension is supported
func (c *Config) IsFileSupported(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, supportedExt := range c.FileProcessing.SupportedExtensions {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

// IsSecretFile checks if a file should be skipped due to security concerns
func (c *Config) IsSecretFile(filename string) bool {
	basename := filepath.Base(filename)
	for _, pattern := range c.Security.SkipSecretFiles {
		if matched, _ := filepath.Match(pattern, basename); matched {
			return true
		}
	}
	return false
}
