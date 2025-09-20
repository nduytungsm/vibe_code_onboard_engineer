package commands

import (
	"fmt"
	"strings"

	"repo-explanation/internal/openai"
	"repo-explanation/internal/pipeline"
)

// SupportedProjectType represents supported project types
type SupportedProjectType string

const (
	ReactJS SupportedProjectType = "React.js"
	NodeJS  SupportedProjectType = "Node.js"
	Golang  SupportedProjectType = "Go"
)

// OnboardingCommands provides hardcoded commands for user onboarding
type OnboardingCommands struct {
	analysisResult *pipeline.AnalysisResult
}

// NewOnboardingCommands creates a new onboarding commands instance
func NewOnboardingCommands(result *pipeline.AnalysisResult) *OnboardingCommands {
	return &OnboardingCommands{
		analysisResult: result,
	}
}

// ExecuteCommand executes the specified onboarding command
func (oc *OnboardingCommands) ExecuteCommand(command string) error {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "list services", "services":
		return oc.ListServices()
	case "set config", "config":
		return oc.SetConfig()
	default:
		return fmt.Errorf("unsupported command: %s", command)
	}
}

// ListServices lists all services in a microservices/monorepo project
func (oc *OnboardingCommands) ListServices() error {
	// Validate project is supported
	if err := oc.validateSupportedProject(); err != nil {
		return err
	}

	// Check if project has microservices/monorepo architecture
	if !oc.isMicroservicesOrMonorepo() {
		return oc.createFramedException("Project Architecture Not Supported", 
			"This command is only available for microservices or monorepo projects.",
			"Current project appears to be a monolith or single-service application.")
	}

	// Extract services information
	services := oc.extractServices()
	if len(services) == 0 {
		return oc.createFramedException("No Services Found",
			"Unable to detect any services in this project.",
			"This might be a monolith or the analysis couldn't identify service boundaries.")
	}

	// Display services in a framed format
	oc.displayServicesFrame(services)
	return nil
}

// SetConfig handles configuration setting (placeholder implementation)
func (oc *OnboardingCommands) SetConfig() error {
	// Validate project is supported
	if err := oc.validateSupportedProject(); err != nil {
		return err
	}

	oc.displayConfigFrame()
	return nil
}

// validateSupportedProject ensures the project is React.js, Node.js, or Go
func (oc *OnboardingCommands) validateSupportedProject() error {
	if oc.analysisResult == nil || oc.analysisResult.ProjectSummary == nil {
		return oc.createFramedException("Analysis Required",
			"Project analysis must be completed before using onboarding commands.",
			"Please run the analysis first to identify the project type.")
	}

	supportedType := oc.identifyProjectType()
	if supportedType == "" {
		return oc.createFramedException("Unsupported Project Type",
			"This repository is not supported by the onboarding system.",
			"Currently supported: React.js, Node.js, and Go projects only.")
	}

	return nil
}

// identifyProjectType determines if the project is one of our supported types based on concrete file evidence
func (oc *OnboardingCommands) identifyProjectType() SupportedProjectType {
	if oc.analysisResult == nil || oc.analysisResult.ProjectSummary == nil {
		return ""
	}

	// Priority 1: Check for concrete file evidence
	projectType := oc.detectFromFileEvidence()
	if projectType != "" {
		return projectType
	}

	// Priority 2: Check detailed analysis main stacks (if available)
	projectType = oc.detectFromMainStacks()
	if projectType != "" {
		return projectType
	}

	// Priority 3: Fallback to project type classification
	return oc.detectFromProjectClassification()
}

// detectFromFileEvidence uses concrete files to determine project type
func (oc *OnboardingCommands) detectFromFileEvidence() SupportedProjectType {
	// Check if go.mod exists - strong indicator of Go project
	if oc.hasFile("go.mod") {
		return Golang
	}

	// Check package.json for JavaScript/TypeScript projects
	if oc.hasFile("package.json") {
		// Distinguish between React.js and Node.js based on dependencies and project structure
		if oc.isReactProject() {
			return ReactJS
		}
		if oc.isNodeJSProject() {
			return NodeJS
		}
		
		// Default: If frontend type detected, assume React; if backend, assume Node.js
		if oc.analysisResult.ProjectType != nil {
			primaryType := strings.ToLower(string(oc.analysisResult.ProjectType.PrimaryType))
			if primaryType == "frontend" || primaryType == "fullstack" {
				return ReactJS
			}
			if primaryType == "backend" {
				return NodeJS
			}
		}
		
		// Final fallback for package.json - default to React.js for frontend-looking projects
		return ReactJS
	}

	return ""
}

// detectFromMainStacks checks detailed analysis main stacks
func (oc *OnboardingCommands) detectFromMainStacks() SupportedProjectType {
	summary := oc.analysisResult.ProjectSummary
	
	if summary.DetailedAnalysis != nil {
		for _, stack := range summary.DetailedAnalysis.MainStacks {
			stackLower := strings.ToLower(stack)
			if strings.Contains(stackLower, "react") {
				return ReactJS
			}
			if strings.Contains(stackLower, "node") || strings.Contains(stackLower, "express") || strings.Contains(stackLower, "fastify") {
				return NodeJS
			}
			if strings.Contains(stackLower, "go") || strings.Contains(stackLower, "golang") {
				return Golang
			}
		}
	}
	
	return ""
}

// detectFromProjectClassification uses project type classification as final fallback
func (oc *OnboardingCommands) detectFromProjectClassification() SupportedProjectType {
	summary := oc.analysisResult.ProjectSummary
	
	// Check languages from regular analysis
	hasJS := false
	hasTS := false
	hasGo := false
	
	for lang := range summary.Languages {
		langLower := strings.ToLower(lang)
		if strings.Contains(langLower, "javascript") {
			hasJS = true
		}
		if strings.Contains(langLower, "typescript") {
			hasTS = true
		}
		if strings.Contains(langLower, "go") {
			hasGo = true
		}
	}
	
	// Go detection
	if hasGo {
		return Golang
	}
	
	// JavaScript/TypeScript detection
	if hasJS || hasTS {
		if oc.analysisResult.ProjectType != nil {
			primaryType := strings.ToLower(string(oc.analysisResult.ProjectType.PrimaryType))
			if primaryType == "frontend" || primaryType == "fullstack" {
				return ReactJS
			}
			if primaryType == "backend" {
				return NodeJS
			}
		}
	}

	return ""
}

// hasFile checks if a specific file exists in the project
func (oc *OnboardingCommands) hasFile(filename string) bool {
	summary := oc.analysisResult.ProjectSummary
	
	// Check in detailed analysis evidence paths
	if summary.DetailedAnalysis != nil {
		for _, path := range summary.DetailedAnalysis.EvidencePaths {
			if strings.Contains(strings.ToLower(path), strings.ToLower(filename)) {
				return true
			}
		}
	}
	
	// Check in folder summaries for files
	for _, folder := range summary.FolderSummaries {
		for filePath := range folder.FileSummaries {
			if strings.Contains(strings.ToLower(filePath), strings.ToLower(filename)) {
				return true
			}
		}
	}
	
	return false
}

// isReactProject checks if the project is specifically a React project
func (oc *OnboardingCommands) isReactProject() bool {
	// Check detailed analysis for React indicators
	if oc.analysisResult.ProjectSummary.DetailedAnalysis != nil {
		for _, stack := range oc.analysisResult.ProjectSummary.DetailedAnalysis.MainStacks {
			if strings.Contains(strings.ToLower(stack), "react") {
				return true
			}
		}
		
		// Check if summary mentions React
		summaryLower := strings.ToLower(oc.analysisResult.ProjectSummary.DetailedAnalysis.RepoSummaryLine)
		if strings.Contains(summaryLower, "react") {
			return true
		}
	}
	
	// Check project type - if frontend/fullstack with JS/TS, likely React
	if oc.analysisResult.ProjectType != nil {
		primaryType := strings.ToLower(string(oc.analysisResult.ProjectType.PrimaryType))
		if primaryType == "frontend" || primaryType == "fullstack" {
			return true
		}
	}
	
	return false
}

// isNodeJSProject checks if the project is specifically a Node.js backend project  
func (oc *OnboardingCommands) isNodeJSProject() bool {
	// Check detailed analysis for Node.js indicators
	if oc.analysisResult.ProjectSummary.DetailedAnalysis != nil {
		for _, stack := range oc.analysisResult.ProjectSummary.DetailedAnalysis.MainStacks {
			stackLower := strings.ToLower(stack)
			if strings.Contains(stackLower, "node") || strings.Contains(stackLower, "express") || 
			   strings.Contains(stackLower, "fastify") || strings.Contains(stackLower, "nest") {
				return true
			}
		}
	}
	
	// Check project type - if backend with JS/TS, likely Node.js
	if oc.analysisResult.ProjectType != nil {
		primaryType := strings.ToLower(string(oc.analysisResult.ProjectType.PrimaryType))
		if primaryType == "backend" {
			return true
		}
	}
	
	return false
}

// isMicroservicesOrMonorepo checks if the project has microservices/monorepo architecture
func (oc *OnboardingCommands) isMicroservicesOrMonorepo() bool {
	if oc.analysisResult == nil || oc.analysisResult.ProjectSummary == nil {
		return false
	}

	summary := oc.analysisResult.ProjectSummary
	
	// Check detailed analysis
	if summary.DetailedAnalysis != nil {
		return summary.DetailedAnalysis.Architecture == "microservices" || 
			   summary.DetailedAnalysis.RepoLayout == "monorepo"
	}

	// Fallback: Check if there are multiple folders suggesting services
	return len(summary.FolderSummaries) > 3 // Heuristic: more than 3 folders might indicate services
}

// extractServices extracts service information from the analysis
func (oc *OnboardingCommands) extractServices() []ServiceInfo {
	var services []ServiceInfo

	if oc.analysisResult == nil || oc.analysisResult.ProjectSummary == nil {
		return services
	}

	summary := oc.analysisResult.ProjectSummary

	// First, try to get services from detailed analysis
	if summary.DetailedAnalysis != nil && len(summary.DetailedAnalysis.MonorepoServices) > 0 {
		for _, service := range summary.DetailedAnalysis.MonorepoServices {
			services = append(services, ServiceInfo{
				Name:        service.Name,
				Path:        service.Path,
				Language:    service.Language,
				Purpose:     service.ShortPurpose,
				Type:        oc.classifyServiceType(service.ShortPurpose),
			})
		}
		return services
	}

	// Fallback: Extract from folder summaries
	for path, folderSummary := range summary.FolderSummaries {
		if oc.looksLikeService(path, folderSummary.Purpose) {
			services = append(services, ServiceInfo{
				Name:     oc.extractServiceName(path),
				Path:     path,
				Language: oc.detectLanguageFromFolder(folderSummary),
				Purpose:  folderSummary.Purpose,
				Type:     oc.classifyServiceType(folderSummary.Purpose),
			})
		}
	}

	return services
}

// ServiceInfo represents information about a service
type ServiceInfo struct {
	Name     string
	Path     string
	Language string
	Purpose  string
	Type     string
}

// looksLikeService determines if a folder looks like a service
func (oc *OnboardingCommands) looksLikeService(path, purpose string) bool {
	pathLower := strings.ToLower(path)
	purposeLower := strings.ToLower(purpose)
	
	// Service indicators
	serviceKeywords := []string{"service", "api", "server", "app", "microservice", "gateway", "auth", "user", "payment", "order"}
	
	for _, keyword := range serviceKeywords {
		if strings.Contains(pathLower, keyword) || strings.Contains(purposeLower, keyword) {
			return true
		}
	}
	
	return false
}

// extractServiceName extracts a clean service name from path
func (oc *OnboardingCommands) extractServiceName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Clean up common service suffixes/prefixes
		name = strings.TrimSuffix(name, "-service")
		name = strings.TrimSuffix(name, "_service")
		name = strings.TrimPrefix(name, "service-")
		name = strings.TrimPrefix(name, "service_")
		return strings.Title(strings.ReplaceAll(name, "-", " "))
	}
	return path
}

// detectLanguageFromFolder detects primary language from folder summary
func (oc *OnboardingCommands) detectLanguageFromFolder(folder openai.FolderSummary) string {
	if len(folder.Languages) == 0 {
		return "Unknown"
	}
	
	// Find the language with the most files
	maxCount := 0
	primaryLang := "Unknown"
	
	for lang, count := range folder.Languages {
		if count > maxCount {
			maxCount = count
			primaryLang = lang
		}
	}
	
	return primaryLang
}

// classifyServiceType classifies the type of service based on its purpose
func (oc *OnboardingCommands) classifyServiceType(purpose string) string {
	purposeLower := strings.ToLower(purpose)
	
	if strings.Contains(purposeLower, "api") || strings.Contains(purposeLower, "gateway") {
		return "API Gateway"
	}
	if strings.Contains(purposeLower, "auth") || strings.Contains(purposeLower, "user") {
		return "Authentication"
	}
	if strings.Contains(purposeLower, "database") || strings.Contains(purposeLower, "data") {
		return "Data Service"
	}
	if strings.Contains(purposeLower, "frontend") || strings.Contains(purposeLower, "ui") {
		return "Frontend"
	}
	if strings.Contains(purposeLower, "payment") || strings.Contains(purposeLower, "billing") {
		return "Payment"
	}
	
	return "Business Logic"
}

// createFramedException creates a framed exception message
func (oc *OnboardingCommands) createFramedException(title, message, suggestion string) error {
	frame := oc.createFrame([]string{
		"❌ " + title,
		"",
		message,
		"",
		"💡 " + suggestion,
	}, 60)
	
	return fmt.Errorf("\n%s", frame)
}

// displayServicesFrame displays services in a framed format
func (oc *OnboardingCommands) displayServicesFrame(services []ServiceInfo) {
	projectType := oc.identifyProjectType()
	
	lines := []string{
		fmt.Sprintf("🏗️  %s PROJECT SERVICES", strings.ToUpper(string(projectType))),
		"",
	}
	
	for i, service := range services {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, service.Name))
		lines = append(lines, fmt.Sprintf("   📁 Path: %s", service.Path))
		lines = append(lines, fmt.Sprintf("   💻 Language: %s", service.Language))
		lines = append(lines, fmt.Sprintf("   🔧 Type: %s", service.Type))
		lines = append(lines, fmt.Sprintf("   📝 Purpose: %s", service.Purpose))
		
		if i < len(services)-1 {
			lines = append(lines, "")
		}
	}
	
	frame := oc.createFrame(lines, 80)
	fmt.Println(frame)
}

// displayConfigFrame displays the config setting message with project-specific commands
func (oc *OnboardingCommands) displayConfigFrame() {
	projectType := oc.identifyProjectType()
	
	lines := []string{
		fmt.Sprintf("⚙️  %s PROJECT CONFIGURATION", strings.ToUpper(string(projectType))),
		"",
		"🔧 Setting configs...",
		"",
	}
	
	// Add project-specific setup commands
	setupCommands := oc.getProjectSetupCommands()
	if len(setupCommands) > 0 {
		lines = append(lines, "🚀 Recommended setup commands:")
		for _, cmd := range setupCommands {
			lines = append(lines, fmt.Sprintf("   $ %s", cmd))
		}
		lines = append(lines, "")
	}
	
	lines = append(lines, 
		"📋 Configuration options will be available in future versions.",
		"💡 This will allow you to customize analysis parameters,",
		"   set project-specific preferences, and configure integrations.",
	)
	
	frame := oc.createFrame(lines, 80)
	fmt.Println(frame)
}

// getProjectSetupCommands returns recommended setup commands based on project type and files
func (oc *OnboardingCommands) getProjectSetupCommands() []string {
	var commands []string
	projectType := oc.identifyProjectType()
	
	switch projectType {
	case ReactJS:
		if oc.hasFile("package.json") {
			commands = append(commands, "npm install")
			// Check for common development scripts
			if oc.hasScriptInPackageJson("dev") {
				commands = append(commands, "npm run dev")
			} else if oc.hasScriptInPackageJson("start") {
				commands = append(commands, "npm start")
			}
			if oc.hasScriptInPackageJson("build") {
				commands = append(commands, "npm run build")
			}
		}
		
	case NodeJS:
		if oc.hasFile("package.json") {
			commands = append(commands, "npm install")
			if oc.hasScriptInPackageJson("dev") {
				commands = append(commands, "npm run dev")
			} else if oc.hasScriptInPackageJson("start") {
				commands = append(commands, "npm start")
			}
			if oc.hasScriptInPackageJson("test") {
				commands = append(commands, "npm test")
			}
		}
		
	case Golang:
		if oc.hasFile("go.mod") {
			commands = append(commands, "go mod tidy")
			commands = append(commands, "go build")
			if oc.hasFile("main.go") {
				commands = append(commands, "go run main.go")
			} else {
				commands = append(commands, "go run .")
			}
		}
	}
	
	// Add common development commands if Dockerfile exists
	if oc.hasFile("dockerfile") || oc.hasFile("docker-compose.yml") {
		commands = append(commands, "docker-compose up -d")
	}
	
	// Add Makefile commands if present
	if oc.hasFile("makefile") {
		commands = append(commands, "make")
	}
	
	return commands
}

// hasScriptInPackageJson checks if a specific script exists in package.json
func (oc *OnboardingCommands) hasScriptInPackageJson(scriptName string) bool {
	// This is a simplified check - in a full implementation, we'd parse the package.json
	// For now, we'll make reasonable assumptions based on project type
	switch scriptName {
	case "dev":
		return true // Most modern projects have a dev script
	case "start":
		return true // Most Node.js/React projects have start
	case "build":
		return true // Most projects have build
	case "test":
		return true // Most projects have test
	default:
		return false
	}
}

// createFrame creates a bordered frame around text lines
func (oc *OnboardingCommands) createFrame(lines []string, width int) string {
	if width < 20 {
		width = 20
	}
	
	// Calculate the maximum line length
	maxLen := 0
	for _, line := range lines {
		// Remove ANSI color codes for length calculation
		cleanLine := strings.ReplaceAll(line, "🏗️", "")
		cleanLine = strings.ReplaceAll(cleanLine, "❌", "")
		cleanLine = strings.ReplaceAll(cleanLine, "💡", "")
		cleanLine = strings.ReplaceAll(cleanLine, "📁", "")
		cleanLine = strings.ReplaceAll(cleanLine, "💻", "")
		cleanLine = strings.ReplaceAll(cleanLine, "🔧", "")
		cleanLine = strings.ReplaceAll(cleanLine, "📝", "")
		cleanLine = strings.ReplaceAll(cleanLine, "⚙️", "")
		cleanLine = strings.ReplaceAll(cleanLine, "📋", "")
		
		if len(cleanLine) > maxLen {
			maxLen = len(cleanLine)
		}
	}
	
	// Ensure width is at least as wide as the longest line plus padding
	if maxLen+4 > width {
		width = maxLen + 4
	}
	
	var result strings.Builder
	
	// Top border
	result.WriteString("┌" + strings.Repeat("─", width-2) + "┐\n")
	
	// Content lines
	for _, line := range lines {
		padding := width - len(line) - 4 // Account for emoji width approximation
		if padding < 0 {
			padding = 0
		}
		result.WriteString(fmt.Sprintf("│ %s%s │\n", line, strings.Repeat(" ", padding)))
	}
	
	// Bottom border
	result.WriteString("└" + strings.Repeat("─", width-2) + "┘")
	
	return result.String()
}
