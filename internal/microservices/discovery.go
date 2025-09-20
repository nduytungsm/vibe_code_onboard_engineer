package microservices

import (
	"path/filepath"
	"regexp"
	"strings"
	"fmt"
)

// ServiceType represents the type of API a service exposes
type ServiceType string

const (
	HTTPService ServiceType = "http"
	GRPCService ServiceType = "grpc"
	GraphQLService ServiceType = "graphql"
)

// DiscoveredService represents a discovered microservice
type DiscoveredService struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	EntryPoint  string      `json:"entry_point"`
	APIType     ServiceType `json:"api_type"`
	Port        string      `json:"port,omitempty"`
	Description string      `json:"description,omitempty"`
}

// ServiceDiscovery handles microservice discovery in monorepos
type ServiceDiscovery struct {
	projectPath string
	projectType string
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(projectPath, projectType string) *ServiceDiscovery {
	return &ServiceDiscovery{
		projectPath: projectPath,
		projectType: projectType,
	}
}

// DiscoverServices discovers externally exposed microservices in the monorepo
func (sd *ServiceDiscovery) DiscoverServices(files map[string]string, folderStructure []string) ([]DiscoveredService, error) {
	var services []DiscoveredService
	
	switch strings.ToLower(sd.projectType) {
	case "go", "golang":
		services = sd.discoverGoServices(files, folderStructure)
	case "node.js", "nodejs":
		services = sd.discoverNodeServices(files, folderStructure)
	case "react.js", "reactjs":
		// React projects are usually single applications, but check for microfrontends
		services = sd.discoverReactServices(files, folderStructure)
	default:
		return nil, fmt.Errorf("unsupported project type for service discovery: %s", sd.projectType)
	}
	
	// Filter services based on README commands
	readmeServices := sd.parseReadmeCommands(files)
	services = sd.reconcileWithReadme(services, readmeServices)
	
	return services, nil
}

// discoverGoServices discovers Go microservices
func (sd *ServiceDiscovery) discoverGoServices(files map[string]string, folderStructure []string) []DiscoveredService {
	var services []DiscoveredService
	
	// Step 1: Scan cmd/ folders for entrypoints
	cmdServices := sd.scanCmdFolders(files, folderStructure)
	services = append(services, cmdServices...)
	
	// Step 2: Look for top-level main.go files
	topLevelServices := sd.scanTopLevelMain(files)
	services = append(services, topLevelServices...)
	
	// Step 3: Filter only externally exposed services
	externalServices := sd.filterExternallyExposed(services, files)
	
	return externalServices
}

// discoverNodeServices discovers Node.js microservices
func (sd *ServiceDiscovery) discoverNodeServices(files map[string]string, folderStructure []string) []DiscoveredService {
	var services []DiscoveredService
	
	// Look for multiple package.json files (microservices pattern)
	packageJsonFiles := sd.findFiles(files, "package.json")
	
	for _, packagePath := range packageJsonFiles {
		if packageContent, exists := files[packagePath]; exists {
			service := sd.analyzeNodeService(packagePath, packageContent, files)
			if service != nil {
				services = append(services, *service)
			}
		}
	}
	
	// Look for services/ or apps/ directories
	for _, folder := range folderStructure {
		if strings.Contains(folder, "/services/") || strings.Contains(folder, "/apps/") {
			service := sd.analyzeNodeServiceFolder(folder, files)
			if service != nil {
				services = append(services, *service)
			}
		}
	}
	
	return sd.filterExternallyExposed(services, files)
}

// discoverReactServices discovers React-based services (microfrontends)
func (sd *ServiceDiscovery) discoverReactServices(files map[string]string, folderStructure []string) []DiscoveredService {
	var services []DiscoveredService
	
	// Look for microfrontend patterns
	for _, folder := range folderStructure {
		if strings.Contains(folder, "/apps/") || strings.Contains(folder, "/packages/") {
			// Check if it has its own package.json and is a React app
			packagePath := filepath.Join(folder, "package.json")
			if packageContent, exists := files[packagePath]; exists {
				if strings.Contains(packageContent, "react") && 
				   (strings.Contains(packageContent, "react-scripts") || strings.Contains(packageContent, "vite") || strings.Contains(packageContent, "webpack")) {
					
					serviceName := filepath.Base(folder)
					services = append(services, DiscoveredService{
						Name:       serviceName,
						Path:       folder,
						EntryPoint: packagePath,
						APIType:    HTTPService,
						Description: fmt.Sprintf("React microfrontend: %s", serviceName),
					})
				}
			}
		}
	}
	
	return services
}

// scanCmdFolders scans cmd/ directories for Go service entrypoints
func (sd *ServiceDiscovery) scanCmdFolders(files map[string]string, folderStructure []string) []DiscoveredService {
	var services []DiscoveredService
	
	for _, folder := range folderStructure {
		if strings.Contains(folder, "/cmd/") {
			// Extract service name from cmd/servicename pattern
			parts := strings.Split(folder, "/")
			var serviceName string
			
			for i, part := range parts {
				if part == "cmd" && i+1 < len(parts) {
					serviceName = parts[i+1]
					break
				}
			}
			
			if serviceName != "" {
				mainGoPath := filepath.Join(folder, "main.go")
				if _, exists := files[mainGoPath]; exists {
					services = append(services, DiscoveredService{
						Name:       serviceName,
						Path:       folder,
						EntryPoint: mainGoPath,
						APIType:    HTTPService, // Will be refined in filterExternallyExposed
					})
				}
			}
		}
	}
	
	return services
}

// scanTopLevelMain looks for top-level main.go files
func (sd *ServiceDiscovery) scanTopLevelMain(files map[string]string) []DiscoveredService {
	var services []DiscoveredService
	
	for filePath := range files {
		if strings.HasSuffix(filePath, "main.go") && !strings.Contains(filePath, "/") {
			// Top-level main.go
			serviceName := strings.TrimSuffix(filepath.Base(sd.projectPath), "/")
			services = append(services, DiscoveredService{
				Name:       serviceName,
				Path:       ".",
				EntryPoint: filePath,
				APIType:    HTTPService,
			})
		}
	}
	
	return services
}

// filterExternallyExposed filters services to only include those with external APIs
func (sd *ServiceDiscovery) filterExternallyExposed(services []DiscoveredService, files map[string]string) []DiscoveredService {
	var externalServices []DiscoveredService
	
	for _, service := range services {
		if sd.hasExternalAPI(service, files) {
			externalServices = append(externalServices, service)
		}
	}
	
	return externalServices
}

// hasExternalAPI checks if a service exposes an external API endpoint
func (sd *ServiceDiscovery) hasExternalAPI(service DiscoveredService, files map[string]string) bool {
	content, exists := files[service.EntryPoint]
	if !exists {
		return false
	}
	
	switch strings.ToLower(sd.projectType) {
	case "go", "golang":
		return sd.hasGoExternalAPI(content, &service)
	case "node.js", "nodejs":
		return sd.hasNodeExternalAPI(content, &service)
	case "react.js", "reactjs":
		return sd.hasReactExternalAPI(content, &service)
	}
	
	return false
}

// hasGoExternalAPI checks for Go HTTP/gRPC server patterns
func (sd *ServiceDiscovery) hasGoExternalAPI(content string, service *DiscoveredService) bool {
	// HTTP server patterns
	httpPatterns := []string{
		"http.ListenAndServe",
		"gin.New", "gin.Default", ".Run(",
		"echo.New", ".Start(",
		"fiber.New", ".Listen(",
		"mux.NewRouter",
		"chi.NewRouter",
		"http.Server{",
	}
	
	// gRPC server patterns
	grpcPatterns := []string{
		"grpc.NewServer",
		"google.golang.org/grpc",
		"grpc.Serve",
	}
	
	// GraphQL patterns
	graphqlPatterns := []string{
		"graphql-go/graphql",
		"99designs/gqlgen",
		"/graphql",
	}
	
	// Check for HTTP server
	for _, pattern := range httpPatterns {
		if strings.Contains(content, pattern) {
			service.APIType = HTTPService
			// Extract port if possible
			if port := sd.extractPortFromGoCode(content); port != "" {
				service.Port = port
			}
			return true
		}
	}
	
	// Check for gRPC server
	for _, pattern := range grpcPatterns {
		if strings.Contains(content, pattern) {
			service.APIType = GRPCService
			return true
		}
	}
	
	// Check for GraphQL
	for _, pattern := range graphqlPatterns {
		if strings.Contains(content, pattern) {
			service.APIType = GraphQLService
			return true
		}
	}
	
	// Exclude worker-only services
	workerPatterns := []string{
		"kafka.Consumer",
		"nats.Subscribe",
		"rabbitmq",
		"sql.Open", // Database-only workers
		"cron.New",
		"time.Ticker",
	}
	
	hasOnlyWorkerPatterns := false
	for _, pattern := range workerPatterns {
		if strings.Contains(content, pattern) {
			hasOnlyWorkerPatterns = true
			break
		}
	}
	
	// If it has worker patterns but no server patterns, it's likely a worker
	return !hasOnlyWorkerPatterns
}

// hasNodeExternalAPI checks for Node.js HTTP server patterns
func (sd *ServiceDiscovery) hasNodeExternalAPI(content string, service *DiscoveredService) bool {
	// Express patterns
	expressPatterns := []string{
		"express()",
		"app.listen(",
		"server.listen(",
		".listen(process.env.PORT",
	}
	
	// Fastify patterns
	fastifyPatterns := []string{
		"fastify(",
		"fastify.listen(",
	}
	
	// NestJS patterns
	nestPatterns := []string{
		"NestFactory.create",
		"app.listen(",
	}
	
	// Next.js patterns
	nextPatterns := []string{
		"next dev",
		"next start",
		"createServer",
	}
	
	allPatterns := append(expressPatterns, fastifyPatterns...)
	allPatterns = append(allPatterns, nestPatterns...)
	allPatterns = append(allPatterns, nextPatterns...)
	
	for _, pattern := range allPatterns {
		if strings.Contains(content, pattern) {
			service.APIType = HTTPService
			return true
		}
	}
	
	return false
}

// hasReactExternalAPI checks for React app patterns
func (sd *ServiceDiscovery) hasReactExternalAPI(content string, service *DiscoveredService) bool {
	// React apps typically serve HTTP content
	reactPatterns := []string{
		"react-scripts start",
		"vite",
		"webpack-dev-server",
		"serve -s build",
	}
	
	for _, pattern := range reactPatterns {
		if strings.Contains(content, pattern) {
			service.APIType = HTTPService
			return true
		}
	}
	
	return false
}

// extractPortFromGoCode extracts port number from Go code
func (sd *ServiceDiscovery) extractPortFromGoCode(content string) string {
	portRegex := regexp.MustCompile(`[:"](\d{4,5})[:"']`)
	matches := portRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseReadmeCommands parses README for service run commands
func (sd *ServiceDiscovery) parseReadmeCommands(files map[string]string) []string {
	var services []string
	
	readmeContent := ""
	for filePath, content := range files {
		if strings.ToLower(filepath.Base(filePath)) == "readme.md" {
			readmeContent = content
			break
		}
	}
	
	if readmeContent == "" {
		return services
	}
	
	// Look for make run-* commands
	makeRunRegex := regexp.MustCompile(`make\s+run-(\w+)`)
	matches := makeRunRegex.FindAllStringSubmatch(readmeContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			services = append(services, match[1])
		}
	}
	
	// Look for go run ./cmd/* commands
	goRunRegex := regexp.MustCompile(`go\s+run\s+\./cmd/(\w+)`)
	matches = goRunRegex.FindAllStringSubmatch(readmeContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			services = append(services, match[1])
		}
	}
	
	// Look for docker-compose service names
	dockerComposeRegex := regexp.MustCompile(`docker-compose\s+up\s+(\w+)`)
	matches = dockerComposeRegex.FindAllStringSubmatch(readmeContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			services = append(services, match[1])
		}
	}
	
	return services
}

// reconcileWithReadme reconciles discovered services with README commands
func (sd *ServiceDiscovery) reconcileWithReadme(discovered []DiscoveredService, readmeServices []string) []DiscoveredService {
	// Create a map of discovered services
	discoveredMap := make(map[string]DiscoveredService)
	for _, service := range discovered {
		discoveredMap[service.Name] = service
	}
	
	// Add services mentioned in README but not discovered
	for _, readmeService := range readmeServices {
		if _, exists := discoveredMap[readmeService]; !exists {
			// Add as a potential service
			discovered = append(discovered, DiscoveredService{
				Name:        readmeService,
				Path:        fmt.Sprintf("cmd/%s", readmeService),
				EntryPoint:  fmt.Sprintf("cmd/%s/main.go", readmeService),
				APIType:     HTTPService,
				Description: fmt.Sprintf("Service mentioned in README: %s", readmeService),
			})
		}
	}
	
	return discovered
}

// analyzeNodeService analyzes a Node.js service from package.json
func (sd *ServiceDiscovery) analyzeNodeService(packagePath, packageContent string, files map[string]string) *DiscoveredService {
	// Extract service name from path
	dir := filepath.Dir(packagePath)
	serviceName := filepath.Base(dir)
	
	// Look for server entry point
	entryPoints := []string{
		filepath.Join(dir, "index.js"),
		filepath.Join(dir, "server.js"),
		filepath.Join(dir, "app.js"),
		filepath.Join(dir, "src/index.js"),
		filepath.Join(dir, "src/server.js"),
	}
	
	for _, entryPoint := range entryPoints {
		if content, exists := files[entryPoint]; exists {
			if sd.hasNodeExternalAPI(content, &DiscoveredService{}) {
				return &DiscoveredService{
					Name:       serviceName,
					Path:       dir,
					EntryPoint: entryPoint,
					APIType:    HTTPService,
				}
			}
		}
	}
	
	return nil
}

// analyzeNodeServiceFolder analyzes a Node.js service folder
func (sd *ServiceDiscovery) analyzeNodeServiceFolder(folder string, files map[string]string) *DiscoveredService {
	// Look for package.json in this folder
	packagePath := filepath.Join(folder, "package.json")
	if _, exists := files[packagePath]; exists {
		return sd.analyzeNodeService(packagePath, files[packagePath], files)
	}
	
	return nil
}

// findFiles finds all files with a specific name
func (sd *ServiceDiscovery) findFiles(files map[string]string, filename string) []string {
	var foundFiles []string
	for filePath := range files {
		if filepath.Base(filePath) == filename {
			foundFiles = append(foundFiles, filePath)
		}
	}
	return foundFiles
}
