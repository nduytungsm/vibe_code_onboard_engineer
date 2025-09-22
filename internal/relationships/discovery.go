package relationships

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"repo-explanation/internal/microservices"
)

// EvidenceType represents the type of evidence for a service relationship
type EvidenceType string

const (
	ConfigEvidence  EvidenceType = "config"
	ImportEvidence  EvidenceType = "import"
	NetworkEvidence EvidenceType = "network"
)

// ServiceRelationship represents a dependency between two services
type ServiceRelationship struct {
	From         string        `json:"from"`           // Source service name
	To           string        `json:"to"`             // Target service name
	EvidenceType EvidenceType  `json:"evidence_type"`  // Type of evidence
	Evidence     string        `json:"evidence"`       // Specific evidence found
	FilePath     string        `json:"file_path"`      // File where evidence was found
	Confidence   float64       `json:"confidence"`     // Confidence level (0.0-1.0)
}

// ServiceGraph represents the complete service dependency graph
type ServiceGraph struct {
	Services      []microservices.DiscoveredService `json:"services"`
	Relationships []ServiceRelationship             `json:"relationships"`
	ProjectPath   string                            `json:"project_path"`
	GeneratedAt   time.Time                         `json:"generated_at"`
	MermaidGraph  string                            `json:"mermaid_graph"`
}

// MermaidOutput represents the JSON output format for Mermaid graphs
type MermaidOutput struct {
	Mermaid string `json:"mermaid"`
}

// RelationshipDiscovery discovers relationships between microservices
type RelationshipDiscovery struct {
	services    []microservices.DiscoveredService
	serviceMap  map[string]microservices.DiscoveredService // name -> service mapping
	fileContent map[string]string                          // file path -> content
}

// NewRelationshipDiscovery creates a new relationship discovery instance
func NewRelationshipDiscovery(services []microservices.DiscoveredService, fileContent map[string]string) *RelationshipDiscovery {
	serviceMap := make(map[string]microservices.DiscoveredService)
	for _, service := range services {
		serviceMap[service.Name] = service
		// Also add common variations
		serviceMap[service.Name+"-service"] = service
		serviceMap[strings.ReplaceAll(service.Name, "-", "_")] = service
	}

	return &RelationshipDiscovery{
		services:    services,
		serviceMap:  serviceMap,
		fileContent: fileContent,
	}
}

// DiscoverRelationships discovers all service relationships using deterministic evidence
func (rd *RelationshipDiscovery) DiscoverRelationships(projectPath string) (*ServiceGraph, error) {
	var relationships []ServiceRelationship

	// 1. Parse explicit references in config files
	configRels := rd.discoverConfigRelationships()
	relationships = append(relationships, configRels...)

	// 2. Analyze code imports for cross-service clients
	importRels := rd.discoverImportRelationships()
	relationships = append(relationships, importRels...)

	// 3. Parse network calls in code
	networkRels := rd.discoverNetworkRelationships()
	relationships = append(relationships, networkRels...)

	// Deduplicate relationships
	relationships = rd.deduplicateRelationships(relationships)

	// Generate Mermaid graph
	mermaidGraph := rd.generateMermaidGraph(relationships)

	return &ServiceGraph{
		Services:      rd.services,
		Relationships: relationships,
		ProjectPath:   projectPath,
		GeneratedAt:   time.Now(),
		MermaidGraph:  mermaidGraph,
	}, nil
}

// discoverConfigRelationships finds relationships in config files
func (rd *RelationshipDiscovery) discoverConfigRelationships() []ServiceRelationship {
	var relationships []ServiceRelationship

	for filePath, content := range rd.fileContent {
		fileName := strings.ToLower(filepath.Base(filePath))

		// Parse Docker Compose files
		if fileName == "docker-compose.yml" || fileName == "docker-compose.yaml" {
			rels := rd.parseDockerCompose(filePath, content)
			relationships = append(relationships, rels...)
		}

		// Parse Kubernetes manifests
		if strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") {
			if rd.looksLikeK8sManifest(content) {
				rels := rd.parseK8sManifest(filePath, content)
				relationships = append(relationships, rels...)
			}
		}

		// Parse config files (.env, config.yaml, values.yaml, etc.)
		if rd.looksLikeConfigFile(fileName) {
			rels := rd.parseConfigFile(filePath, content)
			relationships = append(relationships, rels...)
		}
	}

	return relationships
}

// discoverImportRelationships finds relationships through code imports
func (rd *RelationshipDiscovery) discoverImportRelationships() []ServiceRelationship {
	var relationships []ServiceRelationship

	for filePath, content := range rd.fileContent {
		// Only analyze Go files for now
		if !strings.HasSuffix(filePath, ".go") {
			continue
		}

		// Determine which service this file belongs to
		serviceOwner := rd.getServiceOwnerFromPath(filePath)
		if serviceOwner == "" {
			continue
		}

		// Parse imports
		rels := rd.parseGoImports(filePath, content, serviceOwner)
		relationships = append(relationships, rels...)
	}

	return relationships
}

// discoverNetworkRelationships finds relationships through network calls
func (rd *RelationshipDiscovery) discoverNetworkRelationships() []ServiceRelationship {
	var relationships []ServiceRelationship

	for filePath, content := range rd.fileContent {
		// Only analyze code files
		if !rd.isCodeFile(filePath) {
			continue
		}

		serviceOwner := rd.getServiceOwnerFromPath(filePath)
		if serviceOwner == "" {
			continue
		}

		// Parse network calls
		rels := rd.parseNetworkCalls(filePath, content, serviceOwner)
		relationships = append(relationships, rels...)
	}

	return relationships
}

// parseDockerCompose parses Docker Compose files for service dependencies
func (rd *RelationshipDiscovery) parseDockerCompose(filePath, content string) []ServiceRelationship {
	var relationships []ServiceRelationship

	// Parse YAML structure
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &compose); err != nil {
		return relationships
	}

	services, ok := compose["services"].(map[interface{}]interface{})
	if !ok {
		return relationships
	}

	for serviceName, serviceConfig := range services {
		serviceNameStr := fmt.Sprintf("%v", serviceName)

		// Check if this service is in our discovered services
		if _, exists := rd.serviceMap[serviceNameStr]; !exists {
			continue
		}

		if config, ok := serviceConfig.(map[interface{}]interface{}); ok {
			// 1. Check depends_on
			if dependsOn, exists := config["depends_on"]; exists {
				deps := rd.extractDependencies(dependsOn)
				for _, dep := range deps {
					if _, exists := rd.serviceMap[dep]; exists {
						relationships = append(relationships, ServiceRelationship{
							From:         serviceNameStr,
							To:           dep,
							EvidenceType: ConfigEvidence,
							Evidence:     fmt.Sprintf("depends_on: %s", dep),
							FilePath:     filePath,
							Confidence:   1.0,
						})
					}
				}
			}

			// 2. Check environment variables for service URLs
			if env, exists := config["environment"]; exists {
				envRels := rd.parseEnvironmentVars(env, serviceNameStr, filePath)
				relationships = append(relationships, envRels...)
			}
		}
	}

	return relationships
}

// parseK8sManifest parses Kubernetes manifests for service references
func (rd *RelationshipDiscovery) parseK8sManifest(filePath, content string) []ServiceRelationship {
	var relationships []ServiceRelationship

	// Look for service references in environment variables
	envRegex := regexp.MustCompile(`(?i)(?:name|key):\s*(\w*SERVICE\w*URL\w*)\s*(?:value|default):\s*(?:http://|https://)?([a-zA-Z0-9-_]+)`)
	matches := envRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			targetService := match[2]
			if _, exists := rd.serviceMap[targetService]; exists {
				// Try to determine which service this manifest belongs to
				ownerService := rd.getServiceOwnerFromPath(filePath)
				if ownerService != "" && ownerService != targetService {
					relationships = append(relationships, ServiceRelationship{
						From:         ownerService,
						To:           targetService,
						EvidenceType: ConfigEvidence,
						Evidence:     fmt.Sprintf("K8s env var: %s -> %s", match[1], targetService),
						FilePath:     filePath,
						Confidence:   0.9,
					})
				}
			}
		}
	}

	return relationships
}

// parseConfigFile parses config files for service URLs and references
func (rd *RelationshipDiscovery) parseConfigFile(filePath, content string) []ServiceRelationship {
	var relationships []ServiceRelationship

	// Look for service URL patterns
	// USER_SERVICE_URL=http://users:8080
	// PAYMENT_GRPC_ADDR=payments:50051
	serviceURLRegex := regexp.MustCompile(`(?i)([A-Z_]*SERVICE[A-Z_]*|[A-Z_]*GRPC[A-Z_]*|[A-Z_]*API[A-Z_]*)=(?:http://|https://|grpc://)?([a-zA-Z0-9-_]+)`)
	matches := serviceURLRegex.FindAllStringSubmatch(content, -1)

	ownerService := rd.getServiceOwnerFromPath(filePath)

	for _, match := range matches {
		if len(match) >= 3 {
			targetService := match[2]
			// Remove common suffixes/prefixes to match our service names
			targetService = strings.TrimSuffix(targetService, "-service")
			targetService = strings.TrimPrefix(targetService, "service-")

			if _, exists := rd.serviceMap[targetService]; exists && ownerService != "" && ownerService != targetService {
				relationships = append(relationships, ServiceRelationship{
					From:         ownerService,
					To:           targetService,
					EvidenceType: ConfigEvidence,
					Evidence:     fmt.Sprintf("Config: %s -> %s", match[1], targetService),
					FilePath:     filePath,
					Confidence:   0.8,
				})
			}
		}
	}

	return relationships
}

// parseGoImports analyzes Go imports for cross-service dependencies
func (rd *RelationshipDiscovery) parseGoImports(filePath, content, serviceOwner string) []ServiceRelationship {
	var relationships []ServiceRelationship

	// Look for internal imports that reference other service clients
	// e.g., "github.com/yourorg/monorepo/services/user/pkg/client"
	// e.g., "internal/clients/userservice"
	importRegex := regexp.MustCompile(`import\s+(?:[a-zA-Z_]\w*\s+)?"([^"]+)"`)
	matches := importRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			importPath := match[1]

			// Check if this import references another service
			targetService := rd.extractServiceFromImport(importPath)
			if targetService != "" && targetService != serviceOwner {
				if _, exists := rd.serviceMap[targetService]; exists {
					relationships = append(relationships, ServiceRelationship{
						From:         serviceOwner,
						To:           targetService,
						EvidenceType: ImportEvidence,
						Evidence:     fmt.Sprintf("Import: %s", importPath),
						FilePath:     filePath,
						Confidence:   0.9,
					})
				}
			}
		}
	}

	// Also look for gRPC client imports
	grpcRegex := regexp.MustCompile(`([a-zA-Z0-9_]+)pb\.New([A-Z][a-zA-Z0-9_]*)Client`)
	grpcMatches := grpcRegex.FindAllStringSubmatch(content, -1)

	for _, match := range grpcMatches {
		if len(match) >= 3 {
			serviceName := strings.ToLower(match[1])
			if serviceName != serviceOwner {
				if _, exists := rd.serviceMap[serviceName]; exists {
					relationships = append(relationships, ServiceRelationship{
						From:         serviceOwner,
						To:           serviceName,
						EvidenceType: ImportEvidence,
						Evidence:     fmt.Sprintf("gRPC client: %s", match[0]),
						FilePath:     filePath,
						Confidence:   0.95,
					})
				}
			}
		}
	}

	return relationships
}

// parseNetworkCalls analyzes network calls in code
func (rd *RelationshipDiscovery) parseNetworkCalls(filePath, content, serviceOwner string) []ServiceRelationship {
	var relationships []ServiceRelationship

	// Look for HTTP calls
	httpRegex := regexp.MustCompile(`(?i)(?:http\.(?:Get|Post|Put|Delete)|http\.NewRequest|client\.(?:Get|Post|Put|Delete))\s*\(\s*[^,)]*["` + "`" + `]([^"` + "`" + `]+)["` + "`" + `]`)
	matches := httpRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			url := match[1]
			targetService := rd.extractServiceFromURL(url)
			if targetService != "" && targetService != serviceOwner {
				if _, exists := rd.serviceMap[targetService]; exists {
					relationships = append(relationships, ServiceRelationship{
						From:         serviceOwner,
						To:           targetService,
						EvidenceType: NetworkEvidence,
						Evidence:     fmt.Sprintf("HTTP call: %s", url),
						FilePath:     filePath,
						Confidence:   0.8,
					})
				}
			}
		}
	}

	// Look for gRPC dial calls
	grpcRegex := regexp.MustCompile(`grpc\.Dial\s*\(\s*[^,)]*["` + "`" + `]([^"` + "`" + `]+)["` + "`" + `]`)
	grpcMatches := grpcRegex.FindAllStringSubmatch(content, -1)

	for _, match := range grpcMatches {
		if len(match) >= 2 {
			address := match[1]
			targetService := rd.extractServiceFromAddress(address)
			if targetService != "" && targetService != serviceOwner {
				if _, exists := rd.serviceMap[targetService]; exists {
					relationships = append(relationships, ServiceRelationship{
						From:         serviceOwner,
						To:           targetService,
						EvidenceType: NetworkEvidence,
						Evidence:     fmt.Sprintf("gRPC dial: %s", address),
						FilePath:     filePath,
						Confidence:   0.85,
					})
				}
			}
		}
	}

	return relationships
}

// Helper functions

func (rd *RelationshipDiscovery) extractDependencies(dependsOn interface{}) []string {
	var deps []string

	switch v := dependsOn.(type) {
	case []interface{}:
		for _, dep := range v {
			deps = append(deps, fmt.Sprintf("%v", dep))
		}
	case string:
		deps = append(deps, v)
	}

	return deps
}

func (rd *RelationshipDiscovery) parseEnvironmentVars(env interface{}, serviceOwner, filePath string) []ServiceRelationship {
	var relationships []ServiceRelationship

	switch envVars := env.(type) {
	case []interface{}:
		for _, envVar := range envVars {
			if envStr, ok := envVar.(string); ok {
				rel := rd.parseEnvString(envStr, serviceOwner, filePath)
				if rel != nil {
					relationships = append(relationships, *rel)
				}
			}
		}
	case map[interface{}]interface{}:
		for key, value := range envVars {
			envStr := fmt.Sprintf("%v=%v", key, value)
			rel := rd.parseEnvString(envStr, serviceOwner, filePath)
			if rel != nil {
				relationships = append(relationships, *rel)
			}
		}
	}

	return relationships
}

func (rd *RelationshipDiscovery) parseEnvString(envStr, serviceOwner, filePath string) *ServiceRelationship {
	// Look for service URL patterns in environment variables
	serviceURLRegex := regexp.MustCompile(`([A-Z_]*SERVICE[A-Z_]*|[A-Z_]*API[A-Z_]*)=(?:http://|https://)?([a-zA-Z0-9-_]+)`)
	matches := serviceURLRegex.FindStringSubmatch(envStr)

	if len(matches) >= 3 {
		targetService := matches[2]
		targetService = strings.TrimSuffix(targetService, "-service")

		if _, exists := rd.serviceMap[targetService]; exists && targetService != serviceOwner {
			return &ServiceRelationship{
				From:         serviceOwner,
				To:           targetService,
				EvidenceType: ConfigEvidence,
				Evidence:     fmt.Sprintf("Docker env: %s", envStr),
				FilePath:     filePath,
				Confidence:   0.9,
			}
		}
	}

	return nil
}

func (rd *RelationshipDiscovery) getServiceOwnerFromPath(filePath string) string {
	// Check if path contains cmd/{service-name}
	if strings.Contains(filePath, "/cmd/") {
		parts := strings.Split(filePath, "/")
		for i, part := range parts {
			if part == "cmd" && i+1 < len(parts) {
				serviceName := parts[i+1]
				serviceName = strings.TrimSuffix(serviceName, "-service")
				if _, exists := rd.serviceMap[serviceName]; exists {
					return serviceName
				}
				// Try with original name
				if _, exists := rd.serviceMap[parts[i+1]]; exists {
					return parts[i+1]
				}
			}
		}
	}

	// Check if path contains services/{service-name}
	if strings.Contains(filePath, "/services/") {
		parts := strings.Split(filePath, "/")
		for i, part := range parts {
			if part == "services" && i+1 < len(parts) {
				serviceName := parts[i+1]
				if _, exists := rd.serviceMap[serviceName]; exists {
					return serviceName
				}
			}
		}
	}

	return ""
}

func (rd *RelationshipDiscovery) extractServiceFromImport(importPath string) string {
	// Look for patterns like:
	// "github.com/yourorg/monorepo/services/user/pkg/client" -> user
	// "internal/clients/userservice" -> user
	// "pkg/clients/payment" -> payment

	parts := strings.Split(importPath, "/")
	
	// Pattern: .../services/{service}/...
	for i, part := range parts {
		if part == "services" && i+1 < len(parts) {
			serviceName := parts[i+1]
			return strings.TrimSuffix(serviceName, "-service")
		}
	}
	
	// Pattern: .../clients/{service}...
	for i, part := range parts {
		if part == "clients" && i+1 < len(parts) {
			serviceName := parts[i+1]
			serviceName = strings.TrimSuffix(serviceName, "service")
			serviceName = strings.TrimSuffix(serviceName, "client")
			return serviceName
		}
	}

	return ""
}

func (rd *RelationshipDiscovery) extractServiceFromURL(url string) string {
	// Look for patterns like:
	// "http://users:8080/api/v1/profile" -> users
	// "https://payment-service:443/charge" -> payment
	// "http://localhost:8001/auth" -> could be auth based on port

	// Remove protocol
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	
	// Extract hostname
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		hostPort := parts[0]
		host := strings.Split(hostPort, ":")[0]
		
		// Skip localhost and IP addresses
		if host == "localhost" || rd.isIPAddress(host) {
			return ""
		}
		
		// Clean up service name
		serviceName := strings.TrimSuffix(host, "-service")
		return serviceName
	}

	return ""
}

func (rd *RelationshipDiscovery) extractServiceFromAddress(address string) string {
	// Similar to extractServiceFromURL but for gRPC addresses
	// "users:50051" -> users
	// "payment-service:9090" -> payment
	
	parts := strings.Split(address, ":")
	if len(parts) > 0 {
		host := parts[0]
		
		// Skip localhost and IP addresses
		if host == "localhost" || rd.isIPAddress(host) {
			return ""
		}
		
		serviceName := strings.TrimSuffix(host, "-service")
		return serviceName
	}

	return ""
}

func (rd *RelationshipDiscovery) looksLikeK8sManifest(content string) bool {
	return strings.Contains(content, "apiVersion:") && 
		   (strings.Contains(content, "kind: Deployment") || 
			strings.Contains(content, "kind: ConfigMap") ||
			strings.Contains(content, "kind: Service"))
}

func (rd *RelationshipDiscovery) looksLikeConfigFile(fileName string) bool {
	configFiles := []string{".env", "config.yaml", "config.yml", "values.yaml", "values.yml", "application.yaml", "application.yml"}
	for _, cf := range configFiles {
		if fileName == cf || strings.HasSuffix(fileName, cf) {
			return true
		}
	}
	return false
}

func (rd *RelationshipDiscovery) isCodeFile(filePath string) bool {
	codeExtensions := []string{".go", ".js", ".ts", ".py", ".java", ".cs", ".cpp", ".c", ".rb", ".php"}
	for _, ext := range codeExtensions {
		if strings.HasSuffix(filePath, ext) {
			return true
		}
	}
	return false
}

func (rd *RelationshipDiscovery) isIPAddress(host string) bool {
	// Simple check for IP addresses
	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		for _, part := range parts {
			if len(part) == 0 || len(part) > 3 {
				return false
			}
			for _, char := range part {
				if char < '0' || char > '9' {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (rd *RelationshipDiscovery) deduplicateRelationships(relationships []ServiceRelationship) []ServiceRelationship {
	seen := make(map[string]bool)
	var unique []ServiceRelationship

	for _, rel := range relationships {
		key := fmt.Sprintf("%s->%s:%s", rel.From, rel.To, rel.EvidenceType)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, rel)
		}
	}

	return unique
}

// ConsoleVisualization creates an ASCII visualization of the service graph
func (sg *ServiceGraph) ConsoleVisualization() string {
	if len(sg.Services) == 0 {
		return "No services found."
	}

	var result strings.Builder
	
	result.WriteString("🔗 SERVICE DEPENDENCY GRAPH\n")
	result.WriteString(strings.Repeat("═", 50) + "\n\n")

	// Group relationships by source service
	relationshipsBySource := make(map[string][]ServiceRelationship)
	for _, rel := range sg.Relationships {
		relationshipsBySource[rel.From] = append(relationshipsBySource[rel.From], rel)
	}

	// Sort services for consistent output
	var sortedServices []string
	for _, service := range sg.Services {
		sortedServices = append(sortedServices, service.Name)
	}
	sort.Strings(sortedServices)

	// Display each service and its dependencies
	for _, serviceName := range sortedServices {
		result.WriteString(fmt.Sprintf("📦 %s\n", strings.ToUpper(serviceName)))
		
		if relationships, hasRels := relationshipsBySource[serviceName]; hasRels {
			for _, rel := range relationships {
				// Choose icon based on evidence type
				var icon string
				switch rel.EvidenceType {
				case ConfigEvidence:
					icon = "⚙️"
				case ImportEvidence:
					icon = "📦"
				case NetworkEvidence:
					icon = "🌐"
				default:
					icon = "🔗"
				}
				
				result.WriteString(fmt.Sprintf("  │\n"))
				result.WriteString(fmt.Sprintf("  ├─%s─► %s\n", icon, strings.ToUpper(rel.To)))
				result.WriteString(fmt.Sprintf("  │     %s (%.1f)\n", rel.Evidence, rel.Confidence))
			}
		} else {
			result.WriteString("  │\n")
			result.WriteString("  └── (no dependencies)\n")
		}
		result.WriteString("\n")
	}

	// Summary statistics
	result.WriteString("📊 DEPENDENCY SUMMARY\n")
	result.WriteString(strings.Repeat("─", 30) + "\n")
	result.WriteString(fmt.Sprintf("Services: %d\n", len(sg.Services)))
	result.WriteString(fmt.Sprintf("Dependencies: %d\n", len(sg.Relationships)))
	
	// Evidence type breakdown
	evidenceCount := make(map[EvidenceType]int)
	for _, rel := range sg.Relationships {
		evidenceCount[rel.EvidenceType]++
	}
	
	if len(evidenceCount) > 0 {
		result.WriteString("Evidence types:\n")
		for evidenceType, count := range evidenceCount {
			var icon string
			switch evidenceType {
			case ConfigEvidence:
				icon = "⚙️"
			case ImportEvidence:
				icon = "📦"
			case NetworkEvidence:
				icon = "🌐"
			}
			result.WriteString(fmt.Sprintf("  %s %s: %d\n", icon, evidenceType, count))
		}
	}

	return result.String()
}

// generateMermaidGraph creates a Mermaid.js graph from service relationships
func (rd *RelationshipDiscovery) generateMermaidGraph(relationships []ServiceRelationship) string {
	var mermaid strings.Builder
	
	// Start with graph definition
	mermaid.WriteString("graph TD\\n")
	
	// Add service nodes with styling
	serviceSet := make(map[string]bool)
	for _, service := range rd.services {
		serviceName := rd.sanitizeServiceName(service.Name)
		serviceSet[serviceName] = true
		
		// Add service node with API type styling
		switch service.APIType {
		case microservices.HTTPService:
			mermaid.WriteString(fmt.Sprintf("  %s[%s - HTTP]\\n", serviceName, service.Name))
		case microservices.GRPCService:
			mermaid.WriteString(fmt.Sprintf("  %s{%s - gRPC}\\n", serviceName, service.Name))
		case microservices.GraphQLService:
			mermaid.WriteString(fmt.Sprintf("  %s(%s - GraphQL)\\n", serviceName, service.Name))
		default:
			mermaid.WriteString(fmt.Sprintf("  %s[%s]\\n", serviceName, service.Name))
		}
	}
	
	// Add relationships/edges
	if len(relationships) > 0 {
		mermaid.WriteString("\\n")
		for _, rel := range relationships {
			fromService := rd.sanitizeServiceName(rel.From)
			toService := rd.sanitizeServiceName(rel.To)
			
			// Determine edge label based on evidence type
			var edgeLabel string
			switch rel.EvidenceType {
			case ConfigEvidence:
				edgeLabel = "config"
			case ImportEvidence:
				edgeLabel = "import"
			case NetworkEvidence:
				if strings.Contains(strings.ToLower(rel.Evidence), "grpc") {
					edgeLabel = "grpc"
				} else {
					edgeLabel = "http"
				}
			default:
				edgeLabel = "depends"
			}
			
			// Add edge with label
			mermaid.WriteString(fmt.Sprintf("  %s -->|%s| %s\\n", fromService, edgeLabel, toService))
		}
	}
	
	// Add styling for better visualization
	mermaid.WriteString("\\n")
	mermaid.WriteString("  classDef httpService fill:#e1f5fe,stroke:#01579b,stroke-width:2px\\n")
	mermaid.WriteString("  classDef grpcService fill:#f3e5f5,stroke:#4a148c,stroke-width:2px\\n")
	mermaid.WriteString("  classDef graphqlService fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px\\n")
	
	return mermaid.String()
}

// sanitizeServiceName creates a valid Mermaid node identifier
func (rd *RelationshipDiscovery) sanitizeServiceName(serviceName string) string {
	// Replace hyphens and special characters with underscores
	sanitized := strings.ReplaceAll(serviceName, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	return sanitized
}

// GenerateMermaidJSON creates the JSON output format for Mermaid graphs
func (sg *ServiceGraph) GenerateMermaidJSON() (string, error) {
	output := MermaidOutput{
		Mermaid: sg.MermaidGraph,
	}
	
	jsonBytes, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Mermaid JSON: %v", err)
	}
	
	return string(jsonBytes), nil
}

// SaveToFile saves the service graph to a cache file
func (sg *ServiceGraph) SaveToFile(cacheDir string) error {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}
	
	// Generate filename based on project path
	filename := generateCacheFilename(sg.ProjectPath)
	filePath := filepath.Join(cacheDir, filename)
	
	// Marshal the service graph to JSON
	jsonData, err := json.MarshalIndent(sg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal service graph: %v", err)
	}
	
	// Write to file
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}
	
	return nil
}

// LoadFromFile loads a service graph from a cache file if it exists and is recent
func LoadServiceGraphFromFile(projectPath, cacheDir string) (*ServiceGraph, error) {
	filename := generateCacheFilename(projectPath)
	filePath := filepath.Join(cacheDir, filename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // File doesn't exist, not an error
	}
	
	// Read file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %v", err)
	}
	
	// Unmarshal JSON
	var serviceGraph ServiceGraph
	if err := json.Unmarshal(jsonData, &serviceGraph); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service graph: %v", err)
	}
	
	// Check if cache is recent (less than 24 hours old)
	if time.Since(serviceGraph.GeneratedAt) > 24*time.Hour {
		return nil, nil // Cache is stale, regenerate
	}
	
	return &serviceGraph, nil
}

// generateCacheFilename creates a consistent filename from project path
func generateCacheFilename(projectPath string) string {
	// Replace path separators and special characters
	filename := strings.ReplaceAll(projectPath, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "")
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.Trim(filename, "_")
	
	if filename == "" {
		filename = "root"
	}
	
	return filename + "_service_graph.json"
}

// Helper function for SaveToFile method (fix scope issue)
func (rd *RelationshipDiscovery) generateCacheFilename(projectPath string) string {
	return generateCacheFilename(projectPath)
}
