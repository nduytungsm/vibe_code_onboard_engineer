package secrets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretVariable represents a required environment variable or secret
type SecretVariable struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "api_key", "database_url", "secret", "config", "credential"
	Example     string `json:"example,omitempty"`
	Required    bool   `json:"required"`
	Source      string `json:"source"` // file where it was found
}

// ServiceSecrets represents secrets for a specific service/project
type ServiceSecrets struct {
	ServiceName string            `json:"service_name"`
	ServicePath string            `json:"service_path"`
	Variables   []SecretVariable  `json:"variables"`
	ConfigFiles []string          `json:"config_files"` // files that were analyzed
}

// ProjectSecrets contains all secrets for the entire project
type ProjectSecrets struct {
	ProjectType     string           `json:"project_type"`     // "monorepo", "single-service"
	Services        []ServiceSecrets `json:"services"`
	GlobalSecrets   []SecretVariable `json:"global_secrets"`   // project-wide secrets
	TotalVariables  int              `json:"total_variables"`
	RequiredCount   int              `json:"required_count"`
	Summary         string           `json:"summary"`
}

// SecretExtractor analyzes configuration files to find required secrets
type SecretExtractor struct {
	projectPath string
}

// NewSecretExtractor creates a new secret extractor
func NewSecretExtractor(projectPath string) *SecretExtractor {
	return &SecretExtractor{
		projectPath: projectPath,
	}
}

// ExtractSecrets analyzes the project and extracts all required secrets
func (se *SecretExtractor) ExtractSecrets() (*ProjectSecrets, error) {
	fmt.Printf("🔐 [DEBUG] Starting secret extraction for project: %s\n", se.projectPath)
	
	// Find all config files in the project
	configFiles, err := se.findConfigFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %v", err)
	}
	
	fmt.Printf("🔍 [DEBUG] Found %d config files to analyze\n", len(configFiles))
	
	// Determine if this is a monorepo or single service
	isMonorepo := se.isMonorepo(configFiles)
	
	var services []ServiceSecrets
	var globalSecrets []SecretVariable
	
	if isMonorepo {
		services = se.extractMonorepoSecrets(configFiles)
	} else {
		// Single service - treat as one service
		singleService := se.extractServiceSecrets("main", se.projectPath, configFiles)
		services = []ServiceSecrets{singleService}
	}
	
	// Extract global/project-wide secrets
	globalSecrets = se.extractGlobalSecrets(configFiles)
	
	// Calculate totals
	totalVars := len(globalSecrets)
	requiredCount := 0
	
	for _, secret := range globalSecrets {
		if secret.Required {
			requiredCount++
		}
	}
	
	for _, service := range services {
		totalVars += len(service.Variables)
		for _, variable := range service.Variables {
			if variable.Required {
				requiredCount++
			}
		}
	}
	
	projectType := "single-service"
	if isMonorepo {
		projectType = "monorepo"
	}
	
	summary := se.generateSummary(totalVars, requiredCount, len(services))
	
	return &ProjectSecrets{
		ProjectType:    projectType,
		Services:       services,
		GlobalSecrets:  globalSecrets,
		TotalVariables: totalVars,
		RequiredCount:  requiredCount,
		Summary:        summary,
	}, nil
}

// findConfigFiles searches for configuration files in the project
func (se *SecretExtractor) findConfigFiles() ([]string, error) {
	var configFiles []string
	
	fmt.Printf("🔍 [DEBUG] Searching for config files in: %s\n", se.projectPath)
	
	// Walk through project directory
	err := filepath.Walk(se.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}
		
		if info.IsDir() {
			// Skip certain directories
			dirName := filepath.Base(path)
			if dirName == "node_modules" || dirName == ".git" || dirName == "vendor" || dirName == "dist" || dirName == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		
		fileName := filepath.Base(path)
		fileExt := filepath.Ext(fileName)
		
		// Check for specific patterns
		isConfigFile := false
		
		// Check for .env files (any file starting with .env)
		if strings.HasPrefix(fileName, ".env") {
			isConfigFile = true
			fmt.Printf("📋 [DEBUG] Found .env file: %s\n", path)
		}
		
		// Check for .yaml and .yml files
		if fileExt == ".yaml" || fileExt == ".yml" {
			isConfigFile = true
			fmt.Printf("📋 [DEBUG] Found YAML file: %s\n", path)
		}
		
		// Check for other common config files
		if fileName == "config.json" || fileName == "application.properties" || fileName == "docker-compose.yml" || fileName == "docker-compose.yaml" {
			isConfigFile = true
			fmt.Printf("📋 [DEBUG] Found config file: %s\n", path)
		}
		
		if isConfigFile {
			configFiles = append(configFiles, path)
		}
		
		return nil
	})
	
	fmt.Printf("✅ [DEBUG] Found %d config files total\n", len(configFiles))
	for i, file := range configFiles {
		fmt.Printf("   %d. %s\n", i+1, file)
	}
	
	return configFiles, err
}

// isMonorepo determines if this is a monorepo structure
func (se *SecretExtractor) isMonorepo(configFiles []string) bool {
	// Check for multiple service directories with their own configs
	serviceDirs := make(map[string]bool)
	
	for _, file := range configFiles {
		relPath := strings.TrimPrefix(file, se.projectPath)
		relPath = strings.TrimPrefix(relPath, "/")
		
		// If config file is in a subdirectory, it might be a service
		if strings.Contains(relPath, "/") {
			parts := strings.Split(relPath, "/")
			if len(parts) >= 2 {
				// Check if it looks like a service directory
				serviceDir := parts[0]
				if se.isServiceDirectory(serviceDir) {
					serviceDirs[serviceDir] = true
				}
			}
		}
	}
	
	// If we have multiple service directories, it's likely a monorepo
	return len(serviceDirs) > 1
}

// isServiceDirectory checks if a directory name looks like a service
func (se *SecretExtractor) isServiceDirectory(dirName string) bool {
	servicePatterns := []string{
		"service", "svc", "api", "backend", "frontend", 
		"web", "app", "server", "client", "worker",
		"auth", "user", "payment", "notification",
		"microservice", "ms",
	}
	
	lowerDir := strings.ToLower(dirName)
	for _, pattern := range servicePatterns {
		if strings.Contains(lowerDir, pattern) {
			return true
		}
	}
	
	return false
}

// extractMonorepoSecrets extracts secrets for each service in a monorepo
func (se *SecretExtractor) extractMonorepoSecrets(configFiles []string) []ServiceSecrets {
	serviceFiles := make(map[string][]string)
	
	// Group config files by service directory
	for _, file := range configFiles {
		relPath := strings.TrimPrefix(file, se.projectPath)
		relPath = strings.TrimPrefix(relPath, "/")
		
		if strings.Contains(relPath, "/") {
			parts := strings.Split(relPath, "/")
			serviceDir := parts[0]
			
			if se.isServiceDirectory(serviceDir) {
				if serviceFiles[serviceDir] == nil {
					serviceFiles[serviceDir] = []string{}
				}
				serviceFiles[serviceDir] = append(serviceFiles[serviceDir], file)
			}
		}
	}
	
	var services []ServiceSecrets
	for serviceName, files := range serviceFiles {
		servicePath := filepath.Join(se.projectPath, serviceName)
		service := se.extractServiceSecrets(serviceName, servicePath, files)
		services = append(services, service)
	}
	
	return services
}

// extractServiceSecrets extracts secrets for a single service
func (se *SecretExtractor) extractServiceSecrets(serviceName, servicePath string, configFiles []string) ServiceSecrets {
	var variables []SecretVariable
	var analyzedFiles []string
	
	for _, file := range configFiles {
		fileName := filepath.Base(file)
		analyzedFiles = append(analyzedFiles, fileName)
		
		fileVars := se.parseConfigFile(file)
		variables = append(variables, fileVars...)
	}
	
	// Remove duplicates and merge information
	variables = se.deduplicateVariables(variables)
	
	return ServiceSecrets{
		ServiceName: serviceName,
		ServicePath: servicePath,
		Variables:   variables,
		ConfigFiles: analyzedFiles,
	}
}

// extractGlobalSecrets extracts project-wide secrets from root config files
func (se *SecretExtractor) extractGlobalSecrets(configFiles []string) []SecretVariable {
	var globalSecrets []SecretVariable
	
	// Only analyze config files in the root directory for global secrets
	for _, file := range configFiles {
		relPath := strings.TrimPrefix(file, se.projectPath)
		relPath = strings.TrimPrefix(relPath, "/")
		
		// If file is in root directory (no subdirectories)
		if !strings.Contains(relPath, "/") {
			fileVars := se.parseConfigFile(file)
			globalSecrets = append(globalSecrets, fileVars...)
		}
	}
	
	return se.deduplicateVariables(globalSecrets)
}

// parseConfigFile analyzes a single config file for secrets
func (se *SecretExtractor) parseConfigFile(filePath string) []SecretVariable {
	var variables []SecretVariable
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("⚠️ [DEBUG] Could not read config file %s: %v\n", filePath, err)
		return variables
	}
	
	fileName := filepath.Base(filePath)
	fileExt := filepath.Ext(fileName)
	
	fmt.Printf("🔍 [DEBUG] Parsing config file: %s\n", fileName)
	
	switch fileExt {
	case ".env":
		variables = se.parseEnvFile(string(content), fileName)
	case ".yaml", ".yml":
		variables = se.parseYamlFile(string(content), fileName)
	case ".json":
		variables = se.parseJsonFile(string(content), fileName)
	case ".toml":
		variables = se.parseTomlFile(string(content), fileName)
	case ".properties":
		variables = se.parsePropertiesFile(string(content), fileName)
	default:
		// Try to parse as env format by default
		variables = se.parseEnvFile(string(content), fileName)
	}
	
	return variables
}

// parseEnvFile parses .env format files
func (se *SecretExtractor) parseEnvFile(content, fileName string) []SecretVariable {
	var variables []SecretVariable
	
	fmt.Printf("🔍 [DEBUG] Parsing .env file: %s\n", fileName)
	
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse KEY=VALUE format
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) >= 1 {
				key := strings.TrimSpace(parts[0])
				value := ""
				if len(parts) == 2 {
					value = strings.TrimSpace(parts[1])
					// Remove quotes from value
					value = strings.Trim(value, `"'`)
				}
				
				fmt.Printf("   Line %d: %s=%s\n", lineNum, key, value)
				
				// Check if this is an empty/missing value that needs to be configured
				if se.isEmptyOrPlaceholder(value) {
					secret := SecretVariable{
						Name:        key,
						Description: se.generateDescription(key, value),
						Type:        se.determineSecretType(key),
						Example:     se.generateExample(key),
						Required:    true, // Empty values are always required
						Source:      fileName,
					}
					variables = append(variables, secret)
					fmt.Printf("   ✓ Found required variable: %s (empty value)\n", key)
				} else if value != "" {
					fmt.Printf("   ○ Variable %s has value, skipping\n", key)
				}
			}
		}
	}
	
	fmt.Printf("📋 [DEBUG] Extracted %d required variables from %s\n", len(variables), fileName)
	return variables
}

// parseYamlFile parses YAML configuration files
func (se *SecretExtractor) parseYamlFile(content, fileName string) []SecretVariable {
	var variables []SecretVariable
	
	fmt.Printf("🔍 [DEBUG] Parsing YAML file: %s\n", fileName)
	
	// Look for environment variable references in various formats
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\{([^}]+)\}`),           // ${VAR_NAME} or ${VAR_NAME:default}
		regexp.MustCompile(`\$([A-Z_][A-Z0-9_]*)`),   // $VAR_NAME
		regexp.MustCompile(`env:\s*([A-Z_][A-Z0-9_]*)`), // env: VAR_NAME (common in docker-compose)
	}
	
	seen := make(map[string]bool)
	
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		fmt.Printf("   Pattern %s found %d matches\n", pattern.String(), len(matches))
		
		for _, match := range matches {
			var envVar string
			if len(match) > 1 {
				envVar = match[1]
				// Remove default values from ${VAR:default} format
				if strings.Contains(envVar, ":") {
					envVar = strings.Split(envVar, ":")[0]
				}
			}
			
			if envVar != "" && !seen[envVar] {
				seen[envVar] = true
				
				secret := SecretVariable{
					Name:        envVar,
					Description: se.generateDescription(envVar, ""),
					Type:        se.determineSecretType(envVar),
					Example:     se.generateExample(envVar),
					Required:    true, // Variables referenced in YAML are typically required
					Source:      fileName,
				}
				variables = append(variables, secret)
				fmt.Printf("   ✓ Found required variable: %s (referenced in YAML)\n", envVar)
			}
		}
	}
	
	// Also check for key: value pairs where value is empty or placeholder
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				// Check if this looks like an environment variable and has empty/placeholder value
				if se.looksLikeSecret(key) && se.isEmptyOrPlaceholder(value) {
					if !seen[key] {
						seen[key] = true
						secret := SecretVariable{
							Name:        key,
							Description: se.generateDescription(key, value),
							Type:        se.determineSecretType(key),
							Example:     se.generateExample(key),
							Required:    true,
							Source:      fileName,
						}
						variables = append(variables, secret)
						fmt.Printf("   ✓ Found empty config key: %s (line %d)\n", key, i+1)
					}
				}
			}
		}
	}
	
	fmt.Printf("📋 [DEBUG] Extracted %d required variables from %s\n", len(variables), fileName)
	return variables
}

// parseJsonFile parses JSON configuration files
func (se *SecretExtractor) parseJsonFile(content, fileName string) []SecretVariable {
	// Similar to YAML, look for environment variable patterns
	return se.parseYamlFile(content, fileName)
}

// parseTomlFile parses TOML configuration files  
func (se *SecretExtractor) parseTomlFile(content, fileName string) []SecretVariable {
	// Similar to YAML, look for environment variable patterns
	return se.parseYamlFile(content, fileName)
}

// parsePropertiesFile parses Java properties files
func (se *SecretExtractor) parsePropertiesFile(content, fileName string) []SecretVariable {
	// Similar to env files but with different comment style
	return se.parseEnvFile(content, fileName)
}

// isRequiredSecret determines if a variable represents a required secret
func (se *SecretExtractor) isRequiredSecret(key, value, fileName string) bool {
	// Don't include variables that already have values (unless they're examples)
	isExampleFile := strings.Contains(strings.ToLower(fileName), "example") || 
					strings.Contains(strings.ToLower(fileName), "template") ||
					strings.Contains(strings.ToLower(fileName), "sample")
	
	// If it's not an example file and has a value, it might not need user input
	if !isExampleFile && value != "" && !se.isPlaceholderValue(value) {
		return false
	}
	
	// Check if the key looks like a secret
	return se.looksLikeSecret(key)
}

// looksLikeSecret determines if a key name suggests it's a secret
func (se *SecretExtractor) looksLikeSecret(key string) bool {
	lowerKey := strings.ToLower(key)
	
	secretPatterns := []string{
		"secret", "key", "token", "password", "pass", "pwd",
		"api_key", "apikey", "auth", "credential", "cert",
		"private", "public", "jwt", "oauth", "client_id",
		"client_secret", "database_url", "db_url", "redis_url",
		"smtp_", "email_", "mail_", "webhook", "endpoint",
		"host", "port", "user", "username", "encryption",
	}
	
	for _, pattern := range secretPatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}
	
	return false
}

// isPlaceholderValue checks if a value is a placeholder
func (se *SecretExtractor) isPlaceholderValue(value string) bool {
	placeholders := []string{
		"your_", "replace_", "changeme", "example",
		"placeholder", "todo", "fixme", "xxx", "***",
		"<", ">", "{", "}", "localhost", "127.0.0.1",
	}
	
	lowerValue := strings.ToLower(value)
	for _, placeholder := range placeholders {
		if strings.Contains(lowerValue, placeholder) {
			return true
		}
	}
	
	return false
}

// isEmptyOrPlaceholder checks if a value is empty or a placeholder
func (se *SecretExtractor) isEmptyOrPlaceholder(value string) bool {
	// Empty values
	if value == "" || value == `""` || value == `''` {
		return true
	}
	
	// Common placeholder patterns
	placeholders := []string{
		"your_", "replace_", "changeme", "example",
		"placeholder", "todo", "fixme", "xxx", "***",
		"<", ">", "{", "}", "localhost", "127.0.0.1",
		"null", "undefined", "tbd", "insert", "add_",
	}
	
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	for _, placeholder := range placeholders {
		if strings.Contains(lowerValue, placeholder) {
			return true
		}
	}
	
	return false
}

// isRequired determines if a variable is required
func (se *SecretExtractor) isRequired(key, value, fileName string) bool {
	// If it's empty or a placeholder, it's likely required
	return value == "" || se.isPlaceholderValue(value)
}

// determineSecretType categorizes the type of secret
func (se *SecretExtractor) determineSecretType(key string) string {
	lowerKey := strings.ToLower(key)
	
	if strings.Contains(lowerKey, "api") && strings.Contains(lowerKey, "key") {
		return "api_key"
	}
	if strings.Contains(lowerKey, "database") || strings.Contains(lowerKey, "db") {
		return "database_url"
	}
	if strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "token") {
		return "secret"
	}
	if strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "pwd") {
		return "credential"
	}
	
	return "config"
}

// generateDescription creates a helpful description for the variable
func (se *SecretExtractor) generateDescription(key, value string) string {
	lowerKey := strings.ToLower(key)
	
	descriptions := map[string]string{
		"openai_api_key":     "OpenAI API key for LLM integration. Get from https://platform.openai.com/api-keys",
		"database_url":       "Database connection string (e.g., postgresql://user:password@host:port/database)",
		"redis_url":          "Redis connection string (e.g., redis://user:password@host:port)",
		"jwt_secret":         "Secret key for JWT token signing. Should be a long, random string",
		"api_key":            "API key for external service integration",
		"client_id":          "OAuth client ID for authentication",
		"client_secret":      "OAuth client secret for authentication",
		"smtp_host":          "SMTP server hostname for sending emails",
		"smtp_port":          "SMTP server port (usually 587 for TLS or 465 for SSL)",
		"smtp_user":          "SMTP username for email authentication", 
		"smtp_password":      "SMTP password for email authentication",
	}
	
	// Check for exact matches first
	for pattern, desc := range descriptions {
		if strings.Contains(lowerKey, pattern) {
			return desc
		}
	}
	
	// Generate generic description based on key pattern
	if strings.Contains(lowerKey, "url") {
		return fmt.Sprintf("URL/endpoint for %s service", strings.Replace(key, "_URL", "", -1))
	}
	if strings.Contains(lowerKey, "host") {
		return fmt.Sprintf("Hostname for %s service", strings.Replace(key, "_HOST", "", -1))
	}
	if strings.Contains(lowerKey, "port") {
		return fmt.Sprintf("Port number for %s service", strings.Replace(key, "_PORT", "", -1))
	}
	if strings.Contains(lowerKey, "key") {
		return fmt.Sprintf("API key or access key for %s", strings.Replace(key, "_KEY", "", -1))
	}
	
	return fmt.Sprintf("Required configuration value for %s", key)
}

// generateExample creates an example value for the variable
func (se *SecretExtractor) generateExample(key string) string {
	lowerKey := strings.ToLower(key)
	
	if strings.Contains(lowerKey, "url") {
		if strings.Contains(lowerKey, "database") || strings.Contains(lowerKey, "db") {
			return "postgresql://username:password@localhost:5432/database_name"
		}
		if strings.Contains(lowerKey, "redis") {
			return "redis://localhost:6379"
		}
		return "https://api.example.com"
	}
	
	if strings.Contains(lowerKey, "key") {
		return "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	}
	
	if strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "token") {
		return "your-secret-token-here"
	}
	
	if strings.Contains(lowerKey, "host") {
		return "localhost"
	}
	
	if strings.Contains(lowerKey, "port") {
		return "3000"
	}
	
	return "your-value-here"
}

// deduplicateVariables removes duplicate variables and merges information
func (se *SecretExtractor) deduplicateVariables(variables []SecretVariable) []SecretVariable {
	seen := make(map[string]*SecretVariable)
	
	for _, variable := range variables {
		if existing, exists := seen[variable.Name]; exists {
			// Merge information - prefer more detailed descriptions
			if len(variable.Description) > len(existing.Description) {
				existing.Description = variable.Description
			}
			if variable.Example != "" && existing.Example == "" {
				existing.Example = variable.Example
			}
			// Mark as required if any source says it's required
			if variable.Required {
				existing.Required = true
			}
			// Combine sources
			if !strings.Contains(existing.Source, variable.Source) {
				existing.Source = existing.Source + ", " + variable.Source
			}
		} else {
			// Make a copy to avoid pointer issues
			newVar := variable
			seen[variable.Name] = &newVar
		}
	}
	
	// Convert back to slice
	result := make([]SecretVariable, 0, len(seen))
	for _, variable := range seen {
		result = append(result, *variable)
	}
	
	return result
}

// generateSummary creates a summary of the secrets analysis
func (se *SecretExtractor) generateSummary(total, required, services int) string {
	if total == 0 {
		return "No configuration secrets detected. This project may not require additional environment variables."
	}
	
	summary := fmt.Sprintf("Found %d environment variables", total)
	if required > 0 {
		summary += fmt.Sprintf(", %d of which are required to be configured", required)
	}
	
	if services > 1 {
		summary += fmt.Sprintf(" across %d services", services)
	}
	
	summary += ". Review each variable and provide appropriate values before running the project."
	
	return summary
}
