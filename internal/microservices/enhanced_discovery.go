package microservices

import (
	"path/filepath"
	"regexp"
	"strings"
	"fmt"
	"sort"
)

// EnhancedServiceDiscovery provides deterministic microservice detection
// using multiple patterns: main.go files, Makefile commands, docker-compose services
type EnhancedServiceDiscovery struct {
	projectPath string
	projectType string
	debug       bool
}

// ServiceCandidate represents a potential microservice discovered through various patterns
type ServiceCandidate struct {
	Name           string                 `json:"name"`
	Path           string                 `json:"path"`
	EntryPoint     string                 `json:"entry_point"`
	DetectionType  string                 `json:"detection_type"` // "main_go", "makefile", "docker_compose", "directory"
	Confidence     float64               `json:"confidence"`     // 0.0 to 1.0
	Evidence       []string              `json:"evidence"`
	Language       string                `json:"language"`
	APIType        ServiceType           `json:"api_type"`
	Port           string                `json:"port,omitempty"`
	Description    string                `json:"description,omitempty"`
}

// NewEnhancedServiceDiscovery creates a new enhanced service discovery instance
func NewEnhancedServiceDiscovery(projectPath, projectType string) *EnhancedServiceDiscovery {
	return &EnhancedServiceDiscovery{
		projectPath: projectPath,
		projectType: projectType,
		debug:       true,
	}
}

// DiscoverMicroservices discovers microservices using multiple deterministic patterns
func (esd *EnhancedServiceDiscovery) DiscoverMicroservices(files map[string]string) ([]DiscoveredService, error) {
	if esd.debug {
		fmt.Printf("🔍 Enhanced microservice discovery starting...\n")
	}
	
	var allCandidates []ServiceCandidate
	
	// Pattern 1: Multiple main.go files (highest confidence)
	mainGoCandidates := esd.discoverFromMainGoFiles(files)
	allCandidates = append(allCandidates, mainGoCandidates...)
	if esd.debug && len(mainGoCandidates) > 0 {
		fmt.Printf("📄 Found %d main.go based services\n", len(mainGoCandidates))
	}
	
	// Pattern 2: Makefile service commands
	makefileCandidates := esd.discoverFromMakefile(files)
	allCandidates = append(allCandidates, makefileCandidates...)
	if esd.debug && len(makefileCandidates) > 0 {
		fmt.Printf("🔨 Found %d Makefile based services\n", len(makefileCandidates))
	}
	
	// Pattern 3: Docker Compose services  
	dockerComposeCandidates := esd.discoverFromDockerCompose(files)
	allCandidates = append(allCandidates, dockerComposeCandidates...)
	if esd.debug && len(dockerComposeCandidates) > 0 {
		fmt.Printf("🐳 Found %d Docker Compose based services\n", len(dockerComposeCandidates))
	}
	
	// Pattern 4: Package.json based services (for Node.js/npm workspaces)
	packageJsonCandidates := esd.discoverFromPackageJson(files)
	allCandidates = append(allCandidates, packageJsonCandidates...)
	if esd.debug && len(packageJsonCandidates) > 0 {
		fmt.Printf("📦 Found %d package.json based services\n", len(packageJsonCandidates))
	}
	
	// Pattern 5: Directory structure patterns (services/, apps/, cmd/)
	directoryCandidates := esd.discoverFromDirectoryStructure(files)
	allCandidates = append(allCandidates, directoryCandidates...)
	if esd.debug && len(directoryCandidates) > 0 {
		fmt.Printf("📁 Found %d directory structure based services\n", len(directoryCandidates))
	}
	
	// Merge and deduplicate candidates
	mergedCandidates := esd.mergeCandidates(allCandidates)
	
	// Filter and enhance with API detection
	finalServices := esd.filterAndEnhanceServices(mergedCandidates, files)
	
	if esd.debug {
		fmt.Printf("✅ Enhanced discovery complete: %d final microservices detected\n", len(finalServices))
		for i, service := range finalServices {
			fmt.Printf("   %d. %s (%s) - %s [%.1f confidence]\n", 
				i+1, service.Name, service.Path, service.EntryPoint, 
				esd.findCandidateByName(mergedCandidates, service.Name).Confidence)
		}
	}
	
	return finalServices, nil
}

// discoverFromMainGoFiles discovers services by finding multiple main.go files
func (esd *EnhancedServiceDiscovery) discoverFromMainGoFiles(files map[string]string) []ServiceCandidate {
	var candidates []ServiceCandidate
	var mainGoFiles []string
	
	// Find all main.go files
	for filePath := range files {
		if strings.HasSuffix(filePath, "main.go") {
			mainGoFiles = append(mainGoFiles, filePath)
		}
	}
	
	// If multiple main.go files exist, treat each as a potential service
	if len(mainGoFiles) > 1 {
		for _, mainGoFile := range mainGoFiles {
			serviceName := esd.extractServiceNameFromPath(mainGoFile)
			servicePath := filepath.Dir(mainGoFile)
			
			if serviceName != "" {
				candidates = append(candidates, ServiceCandidate{
					Name:          serviceName,
					Path:          servicePath,
					EntryPoint:    mainGoFile,
					DetectionType: "main_go",
					Confidence:    0.9, // High confidence for multiple main.go
					Evidence:      []string{fmt.Sprintf("main.go file at %s", mainGoFile)},
					Language:      "Go",
					APIType:       HTTPService, // Will be refined later
				})
			}
		}
	} else if len(mainGoFiles) == 1 {
		// Single main.go - check if it's in a service-like directory structure
		mainGoFile := mainGoFiles[0]
		if esd.isServiceLikeStructure(mainGoFile) {
			serviceName := esd.extractServiceNameFromPath(mainGoFile)
			if serviceName != "" {
				candidates = append(candidates, ServiceCandidate{
					Name:          serviceName,
					Path:          filepath.Dir(mainGoFile),
					EntryPoint:    mainGoFile,
					DetectionType: "main_go",
					Confidence:    0.7, // Medium confidence for single main.go in service structure
					Evidence:      []string{fmt.Sprintf("main.go file in service structure: %s", mainGoFile)},
					Language:      "Go",
					APIType:       HTTPService,
				})
			}
		}
	}
	
	return candidates
}

// discoverFromMakefile discovers services from Makefile run-* commands
func (esd *EnhancedServiceDiscovery) discoverFromMakefile(files map[string]string) []ServiceCandidate {
	var candidates []ServiceCandidate
	
	// Find Makefile
	var makefileContent string
	var makefilePath string
	for filePath, content := range files {
		filename := strings.ToLower(filepath.Base(filePath))
		if filename == "makefile" || filename == "makefile.mk" || filename == "gnumakefile" {
			makefileContent = content
			makefilePath = filePath
			break
		}
	}
	
	if makefileContent == "" {
		return candidates
	}
	
	// Pattern 1: run-servicename targets
	runServiceRegex := regexp.MustCompile(`^run-(\w+):|^start-(\w+):`)
	
	// Pattern 2: service start commands with go run ./cmd/servicename
	goRunCmdRegex := regexp.MustCompile(`go\s+run\s+\./cmd/(\w+)`)
	
	// Pattern 3: docker-compose up servicename
	dockerComposeUpRegex := regexp.MustCompile(`docker-compose\s+up\s+(\w+)`)
	
	// Pattern 4: Service build targets
	serviceBuildRegex := regexp.MustCompile(`^build-(\w+):|^(\w+)-build:`)
	
	lines := strings.Split(makefileContent, "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check run-* targets
		if matches := runServiceRegex.FindStringSubmatch(line); len(matches) > 1 {
			for _, match := range matches[1:] {
				if match != "" {
					candidates = append(candidates, esd.createMakefileCandidate(match, "run target", makefilePath, lineNum+1))
				}
			}
		}
		
		// Check go run ./cmd/* commands
		if matches := goRunCmdRegex.FindStringSubmatch(line); len(matches) > 1 {
			serviceName := matches[1]
			candidates = append(candidates, esd.createMakefileCandidate(serviceName, "go run cmd", makefilePath, lineNum+1))
		}
		
		// Check docker-compose up commands
		if matches := dockerComposeUpRegex.FindStringSubmatch(line); len(matches) > 1 {
			serviceName := matches[1]
			if serviceName != "all" && serviceName != "-d" { // Exclude common flags
				candidates = append(candidates, esd.createMakefileCandidate(serviceName, "docker-compose up", makefilePath, lineNum+1))
			}
		}
		
		// Check build targets
		if matches := serviceBuildRegex.FindStringSubmatch(line); len(matches) > 1 {
			for _, match := range matches[1:] {
				if match != "" && !esd.isCommonMakeTarget(match) {
					candidates = append(candidates, esd.createMakefileCandidate(match, "build target", makefilePath, lineNum+1))
				}
			}
		}
	}
	
	return candidates
}

// discoverFromDockerCompose discovers services from docker-compose.yml
func (esd *EnhancedServiceDiscovery) discoverFromDockerCompose(files map[string]string) []ServiceCandidate {
	var candidates []ServiceCandidate
	
	// Find docker-compose files
	var dockerComposeContent string
	var dockerComposePath string
	for filePath, content := range files {
		filename := strings.ToLower(filepath.Base(filePath))
		if filename == "docker-compose.yml" || filename == "docker-compose.yaml" || 
		   filename == "docker-compose.override.yml" || filename == "docker-compose.prod.yml" {
			dockerComposeContent = content
			dockerComposePath = filePath
			break
		}
	}
	
	if dockerComposeContent == "" {
		return candidates
	}
	
	// Simple YAML parsing for services section
	// Look for service definitions under "services:"
	lines := strings.Split(dockerComposeContent, "\n")
	inServicesSection := false
	
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Detect services section
		if trimmedLine == "services:" {
			inServicesSection = true
			continue
		}
		
		// Exit services section when we hit another top-level section
		if inServicesSection && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(trimmedLine, "#") {
			if trimmedLine != "services:" {
				inServicesSection = false
			}
		}
		
		// Parse service names (indented under services)
		if inServicesSection && strings.Contains(line, ":") {
			// Check if this line defines a service (2-space or 4-space indentation)
			if (strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ")) ||
			   (strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "\t\t")) {
				
				serviceName := strings.TrimSpace(strings.Split(line, ":")[0])
				if serviceName != "" && !esd.isCommonDockerComposeKey(serviceName) {
					candidates = append(candidates, ServiceCandidate{
						Name:          serviceName,
						Path:          fmt.Sprintf("service-%s", serviceName), // Will be refined later
						EntryPoint:    dockerComposePath,
						DetectionType: "docker_compose",
						Confidence:    0.8, // High confidence for docker-compose services
						Evidence:      []string{fmt.Sprintf("Docker Compose service definition at line %d", i+1)},
						Language:      "Unknown", // Will be inferred later
						APIType:       HTTPService,
					})
				}
			}
		}
	}
	
	return candidates
}

// discoverFromPackageJson discovers services from multiple package.json files (Node.js workspaces)
func (esd *EnhancedServiceDiscovery) discoverFromPackageJson(files map[string]string) []ServiceCandidate {
	var candidates []ServiceCandidate
	var packageJsonFiles []string
	
	// Find all package.json files
	for filePath := range files {
		if strings.HasSuffix(filePath, "package.json") {
			packageJsonFiles = append(packageJsonFiles, filePath)
		}
	}
	
	// If multiple package.json files, each could be a service
	if len(packageJsonFiles) > 1 {
		for _, packagePath := range packageJsonFiles {
			// Skip root package.json in favor of nested ones
			if packagePath == "package.json" {
				continue
			}
			
			serviceName := esd.extractServiceNameFromPath(packagePath)
			servicePath := filepath.Dir(packagePath)
			
			if serviceName != "" {
				candidates = append(candidates, ServiceCandidate{
					Name:          serviceName,
					Path:          servicePath,
					EntryPoint:    packagePath,
					DetectionType: "package_json",
					Confidence:    0.7, // Medium-high confidence for multiple package.json
					Evidence:      []string{fmt.Sprintf("package.json file at %s", packagePath)},
					Language:      "Node.js",
					APIType:       HTTPService,
				})
			}
		}
	}
	
	return candidates
}

// discoverFromDirectoryStructure discovers services from conventional directory patterns
func (esd *EnhancedServiceDiscovery) discoverFromDirectoryStructure(files map[string]string) []ServiceCandidate {
	var candidates []ServiceCandidate
	serviceDirectories := make(map[string]bool)
	
	// Look for service-like directory patterns
	for filePath := range files {
		dirs := strings.Split(filePath, "/")
		
		for i, dir := range dirs {
			// Check for service directory patterns
			if esd.isServiceDirectory(dir) && i < len(dirs)-1 {
				// Get the service name (next directory after services/apps/cmd)
				if i+1 < len(dirs) {
					serviceName := dirs[i+1]
					serviceKey := dir + "/" + serviceName
					
					if !serviceDirectories[serviceKey] && !esd.isCommonSubdirectory(serviceName) {
						serviceDirectories[serviceKey] = true
						
						servicePath := strings.Join(dirs[:i+2], "/")
						candidates = append(candidates, ServiceCandidate{
							Name:          serviceName,
							Path:          servicePath,
							EntryPoint:    esd.findEntryPointInDirectory(servicePath, files),
							DetectionType: "directory",
							Confidence:    0.6, // Medium confidence for directory patterns
							Evidence:      []string{fmt.Sprintf("Service directory structure: %s", servicePath)},
							Language:      esd.inferLanguageFromPath(servicePath, files),
							APIType:       HTTPService,
						})
					}
				}
			}
		}
	}
	
	return candidates
}

// Helper functions

func (esd *EnhancedServiceDiscovery) extractServiceNameFromPath(filePath string) string {
	// Extract service name from various path patterns
	dirs := strings.Split(filePath, "/")
	
	for i, dir := range dirs {
		if esd.isServiceDirectory(dir) && i+1 < len(dirs) {
			return dirs[i+1] // Service name after services/cmd/apps directory
		}
	}
	
	// If no service directory, use the containing directory name
	if len(dirs) > 1 {
		return dirs[len(dirs)-2] // Directory containing the file
	}
	
	// Default to project name
	return filepath.Base(esd.projectPath)
}

func (esd *EnhancedServiceDiscovery) isServiceLikeStructure(mainGoPath string) bool {
	// Check if main.go is in a service-like directory structure
	dirs := strings.Split(mainGoPath, "/")
	
	for _, dir := range dirs {
		if esd.isServiceDirectory(dir) {
			return true
		}
	}
	
	// Check for common service patterns in path
	servicePatterns := []string{"api", "server", "service", "gateway", "worker", "daemon"}
	for _, pattern := range servicePatterns {
		if strings.Contains(mainGoPath, pattern) {
			return true
		}
	}
	
	return false
}

func (esd *EnhancedServiceDiscovery) isServiceDirectory(dir string) bool {
	serviceDirectories := []string{
		"services", "service", "apps", "app", "cmd", "commands", 
		"microservices", "micro", "api", "apis", "servers",
	}
	
	for _, serviceDir := range serviceDirectories {
		if strings.ToLower(dir) == serviceDir {
			return true
		}
	}
	
	return false
}

func (esd *EnhancedServiceDiscovery) createMakefileCandidate(serviceName, evidenceType, makefilePath string, lineNum int) ServiceCandidate {
	return ServiceCandidate{
		Name:          serviceName,
		Path:          fmt.Sprintf("cmd/%s", serviceName), // Assume cmd structure
		EntryPoint:    fmt.Sprintf("cmd/%s/main.go", serviceName),
		DetectionType: "makefile",
		Confidence:    0.8, // High confidence for Makefile entries
		Evidence:      []string{fmt.Sprintf("Makefile %s at line %d", evidenceType, lineNum)},
		Language:      "Go", // Assume Go for Makefile patterns
		APIType:       HTTPService,
	}
}

func (esd *EnhancedServiceDiscovery) isCommonMakeTarget(target string) bool {
	commonTargets := []string{
		"build", "test", "clean", "install", "lint", "fmt", "vet", 
		"dev", "prod", "docker", "deploy", "up", "down", "all",
	}
	
	for _, common := range commonTargets {
		if strings.ToLower(target) == common {
			return true
		}
	}
	
	return false
}

func (esd *EnhancedServiceDiscovery) isCommonDockerComposeKey(key string) bool {
	commonKeys := []string{
		"version", "volumes", "networks", "secrets", "configs",
		"build", "image", "ports", "environment", "depends_on",
		"restart", "command", "entrypoint", "working_dir",
	}
	
	for _, common := range commonKeys {
		if strings.ToLower(key) == common {
			return true
		}
	}
	
	return false
}

func (esd *EnhancedServiceDiscovery) isCommonSubdirectory(dir string) bool {
	commonDirs := []string{
		"test", "tests", "testing", "spec", "specs",
		"vendor", "node_modules", ".git", ".github",
		"docs", "doc", "documentation", "examples", "example",
		"build", "dist", "target", "bin", "lib", "pkg",
	}
	
	for _, common := range commonDirs {
		if strings.ToLower(dir) == common {
			return true
		}
	}
	
	return false
}

func (esd *EnhancedServiceDiscovery) findEntryPointInDirectory(servicePath string, files map[string]string) string {
	// Try to find the main entry point for the service
	candidates := []string{
		servicePath + "/main.go",
		servicePath + "/cmd/main.go", 
		servicePath + "/index.js",
		servicePath + "/server.js",
		servicePath + "/app.js",
		servicePath + "/src/index.js",
		servicePath + "/src/main.js",
		servicePath + "/package.json",
	}
	
	for _, candidate := range candidates {
		if _, exists := files[candidate]; exists {
			return candidate
		}
	}
	
	return servicePath // Default to directory path
}

func (esd *EnhancedServiceDiscovery) inferLanguageFromPath(servicePath string, files map[string]string) string {
	// Check for language-specific files in the service directory
	for filePath := range files {
		if strings.HasPrefix(filePath, servicePath) {
			if strings.HasSuffix(filePath, ".go") {
				return "Go"
			} else if strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".ts") {
				return "Node.js"
			} else if strings.HasSuffix(filePath, ".py") {
				return "Python"
			} else if strings.HasSuffix(filePath, ".java") {
				return "Java"
			}
		}
	}
	
	return "Unknown"
}

// mergeCandidates combines candidates from different detection methods and removes duplicates
func (esd *EnhancedServiceDiscovery) mergeCandidates(candidates []ServiceCandidate) []ServiceCandidate {
	candidateMap := make(map[string]ServiceCandidate)
	
	for _, candidate := range candidates {
		key := strings.ToLower(candidate.Name)
		
		if existing, exists := candidateMap[key]; exists {
			// Merge evidence and use highest confidence
			merged := existing
			if candidate.Confidence > existing.Confidence {
				merged = candidate
			}
			merged.Evidence = append(merged.Evidence, candidate.Evidence...)
			merged.DetectionType += "," + candidate.DetectionType
			candidateMap[key] = merged
		} else {
			candidateMap[key] = candidate
		}
	}
	
	// Convert back to slice and sort by confidence
	var mergedCandidates []ServiceCandidate
	for _, candidate := range candidateMap {
		mergedCandidates = append(mergedCandidates, candidate)
	}
	
	sort.Slice(mergedCandidates, func(i, j int) bool {
		return mergedCandidates[i].Confidence > mergedCandidates[j].Confidence
	})
	
	return mergedCandidates
}

// filterAndEnhanceServices converts candidates to final services with API detection
func (esd *EnhancedServiceDiscovery) filterAndEnhanceServices(candidates []ServiceCandidate, files map[string]string) []DiscoveredService {
	var services []DiscoveredService
	
	for _, candidate := range candidates {
		// Only include candidates with reasonable confidence
		if candidate.Confidence < 0.5 {
			continue
		}
		
		// Enhance with API detection
		apiType := esd.detectAPIType(candidate, files)
		
		service := DiscoveredService{
			Name:        candidate.Name,
			Path:        candidate.Path,
			EntryPoint:  candidate.EntryPoint,
			APIType:     apiType,
			Port:        candidate.Port,
			Description: esd.generateDescription(candidate),
		}
		
		services = append(services, service)
	}
	
	return services
}

func (esd *EnhancedServiceDiscovery) detectAPIType(candidate ServiceCandidate, files map[string]string) ServiceType {
	// Check the entry point file for API patterns
	if content, exists := files[candidate.EntryPoint]; exists {
		return esd.analyzeAPIType(content, candidate.Language)
	}
	
	// Default to HTTP service
	return HTTPService
}

func (esd *EnhancedServiceDiscovery) analyzeAPIType(content, language string) ServiceType {
	// Go API patterns
	if language == "Go" {
		if strings.Contains(content, "grpc") || strings.Contains(content, "google.golang.org/grpc") {
			return GRPCService
		}
		if strings.Contains(content, "graphql") {
			return GraphQLService
		}
		// Default Go service is HTTP
		return HTTPService
	}
	
	// Node.js API patterns  
	if language == "Node.js" {
		if strings.Contains(content, "apollo") || strings.Contains(content, "graphql") {
			return GraphQLService
		}
		// Default Node.js service is HTTP
		return HTTPService
	}
	
	return HTTPService
}

func (esd *EnhancedServiceDiscovery) generateDescription(candidate ServiceCandidate) string {
	detectionTypes := strings.Split(candidate.DetectionType, ",")
	return fmt.Sprintf("%s service detected via %s", candidate.Language, strings.Join(detectionTypes, " and "))
}

func (esd *EnhancedServiceDiscovery) findCandidateByName(candidates []ServiceCandidate, name string) ServiceCandidate {
	for _, candidate := range candidates {
		if candidate.Name == name {
			return candidate
		}
	}
	return ServiceCandidate{}
}
