package detector

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ProjectType represents the detected type of project
type ProjectType string

const (
	Frontend   ProjectType = "Frontend"
	Backend    ProjectType = "Backend"
	Fullstack  ProjectType = "Fullstack"
	Mobile     ProjectType = "Mobile"
	Desktop    ProjectType = "Desktop"
	Library    ProjectType = "Library"
	DevOps     ProjectType = "DevOps/Infrastructure"
	DataScience ProjectType = "Data Science"
	Unknown    ProjectType = "Unknown"
)

// DetectionResult contains the detection results with confidence scores
type DetectionResult struct {
	PrimaryType   ProjectType            `json:"primary_type"`
	SecondaryType ProjectType            `json:"secondary_type,omitempty"`
	Confidence    float64                `json:"confidence"`
	Evidence      map[string][]string    `json:"evidence"`
	Scores        map[ProjectType]float64 `json:"scores"`
}

// ProjectDetector analyzes file structures and determines project type
type ProjectDetector struct {
	frontendRules   []DetectionRule
	backendRules    []DetectionRule
	mobileRules     []DetectionRule
	desktopRules    []DetectionRule
	libraryRules    []DetectionRule
	devopsRules     []DetectionRule
	dataScienceRules []DetectionRule
}

// DetectionRule defines criteria for detecting project types
type DetectionRule struct {
	Name        string
	Score       float64
	FilePattern string
	Extensions  []string
	Directories []string
	Keywords    []string
	Required    bool
}

// NewProjectDetector creates a new project type detector
func NewProjectDetector() *ProjectDetector {
	return &ProjectDetector{
		frontendRules:    getFrontendRules(),
		backendRules:     getBackendRules(),
		mobileRules:      getMobileRules(),
		desktopRules:     getDesktopRules(),
		libraryRules:     getLibraryRules(),
		devopsRules:      getDevopsRules(),
		dataScienceRules: getDataScienceRules(),
	}
}

// FileInfo represents a discovered file (avoiding import cycle)
type FileInfo struct {
	Path         string
	RelativePath string
	Size         int64
	Extension    string
	IsDir        bool
}

// DetectProjectType analyzes files and determines project type
func (pd *ProjectDetector) DetectProjectType(files []FileInfo, fileContents map[string]string) *DetectionResult {
	scores := make(map[ProjectType]float64)
	evidence := make(map[string][]string)
	
	// Initialize scores
	scores[Frontend] = 0.0
	scores[Backend] = 0.0
	scores[Mobile] = 0.0
	scores[Desktop] = 0.0
	scores[Library] = 0.0
	scores[DevOps] = 0.0
	scores[DataScience] = 0.0
	
	// Collect file information
	extensions := make(map[string]int)
	directories := make(map[string]bool)
	filenames := make([]string, 0)
	
	for _, file := range files {
		ext := strings.ToLower(file.Extension)
		if ext != "" {
			extensions[ext]++
		}
		
		dir := filepath.Dir(file.RelativePath)
		directories[strings.ToLower(dir)] = true
		filenames = append(filenames, strings.ToLower(filepath.Base(file.Path)))
	}
	
	// Apply detection rules
	pd.applyRules(pd.frontendRules, Frontend, extensions, directories, filenames, scores, evidence)
	pd.applyRules(pd.backendRules, Backend, extensions, directories, filenames, scores, evidence)
	pd.applyRules(pd.mobileRules, Mobile, extensions, directories, filenames, scores, evidence)
	pd.applyRules(pd.desktopRules, Desktop, extensions, directories, filenames, scores, evidence)
	pd.applyRules(pd.libraryRules, Library, extensions, directories, filenames, scores, evidence)
	pd.applyRules(pd.devopsRules, DevOps, extensions, directories, filenames, scores, evidence)
	pd.applyRules(pd.dataScienceRules, DataScience, extensions, directories, filenames, scores, evidence)
	
	// Determine primary and secondary types
	primary, secondary, confidence := pd.determineTypes(scores)
	
	// Check for fullstack using command-based detection
	hasFrontendCommands := pd.hasFrontendStartupCommands(files, fileContents)
	hasBackendCommands := pd.hasBackendStartupCommands(files, fileContents)
	
	if hasFrontendCommands && hasBackendCommands {
		primary = Fullstack
		confidence = (scores[Frontend] + scores[Backend]) / 2.0
		if confidence > 10.0 {
			confidence = 10.0
		}
		// Add evidence for command-based detection
		if evidence[string(Fullstack)] == nil {
			evidence[string(Fullstack)] = []string{}
		}
		evidence[string(Fullstack)] = append(evidence[string(Fullstack)], 
			"Command-based detection: Found both frontend and backend startup commands")
	}
	
	return &DetectionResult{
		PrimaryType:   primary,
		SecondaryType: secondary,
		Confidence:    confidence,
		Evidence:      evidence,
		Scores:        scores,
	}
}

// applyRules applies detection rules for a specific project type
func (pd *ProjectDetector) applyRules(rules []DetectionRule, projectType ProjectType, 
	extensions map[string]int, directories map[string]bool, filenames []string,
	scores map[ProjectType]float64, evidence map[string][]string) {
	
	for _, rule := range rules {
		matched := false
		matchedItems := []string{}
		
		// Check file extensions
		for _, ext := range rule.Extensions {
			if count, exists := extensions[ext]; exists && count > 0 {
				matched = true
				matchedItems = append(matchedItems, fmt.Sprintf("%s files (%d)", ext, count))
				// Bonus for multiple files of the same type
				if count > 1 {
					scores[projectType] += rule.Score * float64(count) * 0.1
				}
			}
		}
		
		// Check directories
		for _, dir := range rule.Directories {
			if directories[strings.ToLower(dir)] {
				matched = true
				matchedItems = append(matchedItems, dir+" directory")
			}
		}
		
		// Check keywords in filenames
		for _, keyword := range rule.Keywords {
			for _, filename := range filenames {
				if strings.Contains(filename, strings.ToLower(keyword)) {
					matched = true
					matchedItems = append(matchedItems, "file: "+filename)
					break
				}
			}
		}
		
		// Apply score if rule matched
		if matched {
			scores[projectType] += rule.Score
			if evidence[string(projectType)] == nil {
				evidence[string(projectType)] = []string{}
			}
			evidence[string(projectType)] = append(evidence[string(projectType)], 
				rule.Name+": "+strings.Join(matchedItems, ", "))
		}
	}
}

// determineTypes finds primary and secondary project types
func (pd *ProjectDetector) determineTypes(scores map[ProjectType]float64) (ProjectType, ProjectType, float64) {
	var primary, secondary ProjectType
	var primaryScore, secondaryScore float64
	
	for pType, score := range scores {
		if score > primaryScore {
			secondary = primary
			secondaryScore = primaryScore
			primary = pType
			primaryScore = score
		} else if score > secondaryScore {
			secondary = pType
			secondaryScore = score
		}
	}
	
	// If primary score is too low, mark as unknown
	if primaryScore < 1.0 {
		return Unknown, "", 0.0
	}
	
	// Confidence calculation (0-10 scale)
	confidence := primaryScore
	if confidence > 10.0 {
		confidence = 10.0
	}
	
	// Clear secondary if it's too close to primary or too low
	if secondaryScore < 2.0 || (primaryScore-secondaryScore) < 1.0 {
		secondary = ""
	}
	
	return primary, secondary, confidence
}

// Frontend detection rules
func getFrontendRules() []DetectionRule {
	return []DetectionRule{
		{
			Name: "React Framework",
			Score: 4.0,
			Extensions: []string{".jsx", ".tsx"},
			Keywords: []string{"react", "jsx"},
		},
		{
			Name: "Vue.js Framework",
			Score: 4.0,
			Extensions: []string{".vue"},
			Keywords: []string{"vue", "nuxt"},
		},
		{
			Name: "Angular Framework",
			Score: 4.0,
			Extensions: []string{".ts"},
			Directories: []string{"src/app"},
			Keywords: []string{"angular", "ng-"},
		},
		{
			Name: "JavaScript/TypeScript",
			Score: 2.0,
			Extensions: []string{".js", ".ts", ".mjs"},
		},
		{
			Name: "Web Styling",
			Score: 2.5,
			Extensions: []string{".css", ".scss", ".sass", ".less"},
		},
		{
			Name: "HTML Templates",
			Score: 3.0,
			Extensions: []string{".html", ".htm"},
		},
		{
			Name: "Package Management",
			Score: 2.0,
			Keywords: []string{"package.json", "yarn.lock", "package-lock.json"},
		},
		{
			Name: "Build Tools",
			Score: 1.5,
			Keywords: []string{"webpack", "vite", "rollup", "parcel"},
		},
		{
			Name: "Frontend Directories",
			Score: 2.0,
			Directories: []string{"public", "src", "assets", "components"},
		},
	}
}

// Backend detection rules
func getBackendRules() []DetectionRule {
	return []DetectionRule{
		{
			Name: "Database Migrations",
			Score: 4.5, // High score - strong backend indicator
			Extensions: []string{".sql", ".js", ".ts", ".py", ".rb", ".php", ".go"},
			Keywords: []string{
				"migration", "migrate", "schema", "alter", "create_table", "drop_table",
				"add_column", "remove_column", "create table", "drop table", "alter table",
				"flyway", "liquibase", "knex", "sequelize", "prisma", "alembic", "django.db.migrations",
				"rails migration", "laravel migration", "up()", "down()", "rollback",
			},
			Directories: []string{
				"migrations", "migrate", "db/migrate", "database/migrations", "prisma/migrations",
				"sql/migrations", "resources/db/migration", "src/main/resources/db/migration",
				"alembic/versions", "db/versions", "migration",
			},
		},
		{
			Name: "Node.js Backend",
			Score: 3.0,
			Extensions: []string{".js", ".ts"},
			Keywords: []string{"express", "fastify", "koa", "server.js"},
		},
		{
			Name: "Python Backend",
			Score: 3.0,
			Extensions: []string{".py"},
			Keywords: []string{"django", "flask", "fastapi", "app.py", "main.py"},
		},
		{
			Name: "Java Backend",
			Score: 3.0,
			Extensions: []string{".java"},
			Keywords: []string{"spring", "servlet", "application.java"},
		},
		{
			Name: "Go Backend",
			Score: 3.0,
			Extensions: []string{".go"},
			Keywords: []string{"gin", "echo", "fiber", "main.go", "server.go"},
		},
		{
			Name: "C# Backend",
			Score: 3.0,
			Extensions: []string{".cs"},
			Keywords: []string{"controller", "startup.cs", "program.cs"},
		},
		{
			Name: "PHP Backend",
			Score: 3.0,
			Extensions: []string{".php"},
			Keywords: []string{"laravel", "symfony", "index.php"},
		},
		{
			Name: "Ruby Backend",
			Score: 3.0,
			Extensions: []string{".rb"},
			Keywords: []string{"rails", "sinatra", "gemfile"},
		},
		{
			Name: "Rust Backend",
			Score: 3.0,
			Extensions: []string{".rs"},
			Keywords: []string{"actix", "warp", "rocket", "axum"},
		},
		{
			Name: "API Definitions",
			Score: 2.0,
			Extensions: []string{".yaml", ".yml", ".json"},
			Keywords: []string{"api", "swagger", "openapi"},
		},
		{
			Name: "Migration File Patterns",
			Score: 4.0, // High confidence migration indicator
			Keywords: []string{
				// Timestamp patterns common in migrations
				"001_", "002_", "2023", "2024", "2025", 
				// Common migration file naming patterns
				"_create_", "_add_", "_drop_", "_alter_", "_migration",
				"V1__", "V2__", "V001_", "V002_", // Flyway patterns
				"001_initial", "002_add_users", "003_create_posts",
				// Framework-specific patterns
				"0001_initial.py", "001_create_users_table.rb", "create_users_table.php",
			},
		},
		{
			Name: "Database Files",
			Score: 2.5,
			Extensions: []string{".sql", ".db", ".sqlite"},
		},
		{
			Name: "Backend Directories",
			Score: 2.0,
			Directories: []string{"api", "server", "backend", "controllers", "models", "routes"},
		},
	}
}

// Mobile detection rules
func getMobileRules() []DetectionRule {
	return []DetectionRule{
		{
			Name: "React Native",
			Score: 4.0,
			Extensions: []string{".jsx", ".tsx"},
			Keywords: []string{"react-native", "metro.config"},
		},
		{
			Name: "Flutter",
			Score: 4.0,
			Extensions: []string{".dart"},
			Keywords: []string{"flutter", "pubspec.yaml"},
		},
		{
			Name: "iOS Development",
			Score: 4.0,
			Extensions: []string{".swift", ".m", ".h"},
			Keywords: []string{"xcode", "podfile", "info.plist"},
		},
		{
			Name: "Android Development",
			Score: 4.0,
			Extensions: []string{".java", ".kt"},
			Keywords: []string{"android", "gradle", "manifest.xml"},
		},
		{
			Name: "Xamarin",
			Score: 4.0,
			Extensions: []string{".cs", ".xaml"},
			Keywords: []string{"xamarin"},
		},
	}
}

// Desktop detection rules
func getDesktopRules() []DetectionRule {
	return []DetectionRule{
		{
			Name: "Electron",
			Score: 4.0,
			Keywords: []string{"electron", "main.js", "renderer.js"},
		},
		{
			Name: "Tauri",
			Score: 4.0,
			Extensions: []string{".rs"},
			Keywords: []string{"tauri", "tauri.conf.json"},
		},
		{
			Name: "C++ Desktop",
			Score: 3.0,
			Extensions: []string{".cpp", ".cc", ".cxx", ".h", ".hpp"},
		},
		{
			Name: "C# Desktop",
			Score: 3.0,
			Extensions: []string{".cs", ".xaml"},
			Keywords: []string{"wpf", "winforms", ".csproj"},
		},
		{
			Name: "Python Desktop",
			Score: 3.0,
			Extensions: []string{".py"},
			Keywords: []string{"tkinter", "pyqt", "kivy"},
		},
	}
}

// Library detection rules
func getLibraryRules() []DetectionRule {
	return []DetectionRule{
		{
			Name: "Package Definition",
			Score: 3.0,
			Keywords: []string{"package.json", "setup.py", "cargo.toml", "composer.json", "go.mod"},
		},
		{
			Name: "Library Structure",
			Score: 2.0,
			Directories: []string{"lib", "src", "dist", "build"},
		},
		{
			Name: "Documentation",
			Score: 1.5,
			Extensions: []string{".md"},
			Keywords: []string{"readme", "changelog", "license"},
		},
		{
			Name: "Test Directory",
			Score: 1.0,
			Directories: []string{"test", "tests", "__tests__", "spec"},
		},
	}
}

// DevOps detection rules
func getDevopsRules() []DetectionRule {
	return []DetectionRule{
		{
			Name: "Docker",
			Score: 3.0,
			Keywords: []string{"dockerfile", "docker-compose.yml", ".dockerignore"},
		},
		{
			Name: "Kubernetes",
			Score: 3.0,
			Extensions: []string{".yaml", ".yml"},
			Keywords: []string{"kubernetes", "k8s", "deployment", "service"},
		},
		{
			Name: "Terraform",
			Score: 3.0,
			Extensions: []string{".tf", ".hcl"},
		},
		{
			Name: "Ansible",
			Score: 3.0,
			Extensions: []string{".yml", ".yaml"},
			Keywords: []string{"ansible", "playbook"},
		},
		{
			Name: "CI/CD",
			Score: 2.0,
			Keywords: []string{".github", ".gitlab-ci.yml", "jenkinsfile", ".travis.yml"},
		},
	}
}

// Data Science detection rules
func getDataScienceRules() []DetectionRule {
	return []DetectionRule{
		{
			Name: "Jupyter Notebooks",
			Score: 4.0,
			Extensions: []string{".ipynb"},
		},
		{
			Name: "Python Data Science",
			Score: 3.0,
			Extensions: []string{".py"},
			Keywords: []string{"pandas", "numpy", "matplotlib", "jupyter"},
		},
		{
			Name: "R Language",
			Score: 4.0,
			Extensions: []string{".r", ".rmd"},
		},
		{
			Name: "Data Files",
			Score: 2.0,
			Extensions: []string{".csv", ".json", ".parquet", ".h5"},
		},
		{
			Name: "Requirements",
			Score: 1.0,
			Keywords: []string{"requirements.txt", "environment.yml"},
		},
	}
}

// hasFrontendStartupCommands checks if the repository has commands to start a frontend UI
func (pd *ProjectDetector) hasFrontendStartupCommands(files []FileInfo, fileContents map[string]string) bool {
	// Check package.json for frontend startup commands
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(file.RelativePath), "package.json") {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasPackageJsonFrontendCommands(content) {
				return true
			}
		}
	}
	
	// Check README files for frontend startup instructions
	for _, file := range files {
		filename := strings.ToLower(filepath.Base(file.RelativePath))
		if strings.Contains(filename, "readme") {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasReadmeFrontendCommands(content) {
				return true
			}
		}
	}
	
	// Check Makefile for frontend commands
	for _, file := range files {
		filename := strings.ToLower(filepath.Base(file.RelativePath))
		if filename == "makefile" || filename == "makefile.mk" {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasMakefileFrontendCommands(content) {
				return true
			}
		}
	}
	
	return false
}

// hasBackendStartupCommands checks if the repository has commands to start a backend service
func (pd *ProjectDetector) hasBackendStartupCommands(files []FileInfo, fileContents map[string]string) bool {
	// Check package.json for backend startup commands
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(file.RelativePath), "package.json") {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasPackageJsonBackendCommands(content) {
				return true
			}
		}
	}
	
	// Check README files for backend startup instructions
	for _, file := range files {
		filename := strings.ToLower(filepath.Base(file.RelativePath))
		if strings.Contains(filename, "readme") {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasReadmeBackendCommands(content) {
				return true
			}
		}
	}
	
	// Check Makefile for backend commands
	for _, file := range files {
		filename := strings.ToLower(filepath.Base(file.RelativePath))
		if filename == "makefile" || filename == "makefile.mk" {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasMakefileBackendCommands(content) {
				return true
			}
		}
	}
	
	// Check for Go main.go or server files
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(file.RelativePath), "main.go") || 
		   strings.Contains(strings.ToLower(file.RelativePath), "server.go") ||
		   strings.Contains(strings.ToLower(file.RelativePath), "app.go") {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasGoBackendCode(content) {
				return true
			}
		}
	}
	
	// Check for Python server files
	for _, file := range files {
		filename := strings.ToLower(filepath.Base(file.RelativePath))
		if filename == "app.py" || filename == "main.py" || filename == "server.py" ||
		   filename == "manage.py" || filename == "wsgi.py" || filename == "asgi.py" {
			content, exists := fileContents[file.RelativePath]
			if exists && pd.hasPythonBackendCode(content) {
				return true
			}
		}
	}
	
	return false
}

// hasPackageJsonFrontendCommands checks package.json for frontend development commands
func (pd *ProjectDetector) hasPackageJsonFrontendCommands(content string) bool {
	frontendCommands := []string{
		"\"dev\":", "\"start\":", "\"serve\":", "\"preview\":",
		"vite", "webpack-dev-server", "next dev", "gatsby develop",
		"react-scripts start", "vue-cli-service serve", "ng serve",
		"parcel", "rollup", "nuxt dev", "svelte-kit dev",
		"\"build\":", "\"build:client\":", "\"build:web\":",
	}
	
	contentLower := strings.ToLower(content)
	
	// Must contain scripts section
	if !strings.Contains(contentLower, "\"scripts\"") {
		return false
	}
	
	// Check for frontend-specific commands in scripts
	for _, cmd := range frontendCommands {
		if strings.Contains(contentLower, strings.ToLower(cmd)) {
			return true
		}
	}
	
	// Check for frontend dependencies
	frontendDeps := []string{
		"\"react\":", "\"vue\":", "\"angular\":", "\"svelte\":",
		"\"next\":", "\"nuxt\":", "\"gatsby\":", "\"vite\":",
		"\"webpack\":", "@vue/cli-service", "@angular/cli",
	}
	
	for _, dep := range frontendDeps {
		if strings.Contains(contentLower, strings.ToLower(dep)) {
			// Also check for dev command which is common for frontend
			if strings.Contains(contentLower, "\"dev\":") || 
			   strings.Contains(contentLower, "\"start\":") ||
			   strings.Contains(contentLower, "\"serve\":") {
				return true
			}
		}
	}
	
	return false
}

// hasPackageJsonBackendCommands checks package.json for backend startup commands
func (pd *ProjectDetector) hasPackageJsonBackendCommands(content string) bool {
	backendCommands := []string{
		"express", "fastify", "koa", "hapi", "nestjs",
		"node server", "node app", "node index", "nodemon",
		"ts-node", "pm2", "forever",
		"\"server\":", "\"api\":", "\"backend\":",
		"\"start:server\":", "\"start:api\":", "\"start:backend\":",
		"\"dev:server\":", "\"dev:api\":", "\"dev:backend\":",
	}
	
	contentLower := strings.ToLower(content)
	
	// Must contain scripts section
	if !strings.Contains(contentLower, "\"scripts\"") {
		return false
	}
	
	// Check for backend-specific commands
	for _, cmd := range backendCommands {
		if strings.Contains(contentLower, strings.ToLower(cmd)) {
			return true
		}
	}
	
	// Check for backend dependencies
	backendDeps := []string{
		"\"express\":", "\"fastify\":", "\"koa\":", "\"hapi\":",
		"\"@nestjs/core\":", "\"apollo-server\":", "\"graphql\":",
		"\"mongoose\":", "\"sequelize\":", "\"prisma\":", "\"typeorm\":",
	}
	
	for _, dep := range backendDeps {
		if strings.Contains(contentLower, strings.ToLower(dep)) {
			// Also check for start/server command
			if strings.Contains(contentLower, "\"start\":") || 
			   strings.Contains(contentLower, "\"server\":") ||
			   strings.Contains(contentLower, "\"dev\":") {
				return true
			}
		}
	}
	
	return false
}

// hasReadmeFrontendCommands checks README for frontend startup instructions
func (pd *ProjectDetector) hasReadmeFrontendCommands(content string) bool {
	frontendReadmePatterns := []string{
		"npm run dev", "npm start", "yarn dev", "yarn start",
		"pnpm dev", "pnpm start", "npm run serve", "yarn serve",
		"ng serve", "next dev", "gatsby develop", "nuxt dev",
		"vue-cli-service serve", "vite dev", "parcel",
		"open.*localhost:3000", "open.*localhost:8080", "open.*localhost:5173",
		"development server", "dev server", "local server",
		"browser.*localhost", "visit.*localhost",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, pattern := range frontendReadmePatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	
	return false
}

// hasReadmeBackendCommands checks README for backend startup instructions
func (pd *ProjectDetector) hasReadmeBackendCommands(content string) bool {
	backendReadmePatterns := []string{
		"npm run server", "npm run api", "npm run backend",
		"yarn server", "yarn api", "yarn backend",
		"node server", "node app", "node index",
		"go run main.go", "go run .", "go run server.go",
		"python app.py", "python main.py", "python server.py",
		"python manage.py runserver", "flask run", "fastapi run",
		"uvicorn", "gunicorn", "django-admin runserver",
		"./gradlew run", "mvn spring-boot:run", "java -jar",
		"rails server", "rails s", "ruby app.rb",
		"php artisan serve", "php -S localhost",
		"cargo run", "dotnet run",
		"localhost:8000", "localhost:8080", "localhost:3000/api",
		"api server", "backend server", "database server",
		"start.*server", "run.*server", "serve.*api",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, pattern := range backendReadmePatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	
	return false
}

// hasMakefileFrontendCommands checks Makefile for frontend commands
func (pd *ProjectDetector) hasMakefileFrontendCommands(content string) bool {
	frontendMakePatterns := []string{
		"dev:", "serve:", "start:", "build:", "preview:",
		"npm run dev", "npm start", "yarn dev", "yarn start",
		"ng serve", "next dev", "gatsby develop", "vite dev",
		"webpack-dev-server", "parcel", "rollup",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, pattern := range frontendMakePatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	
	return false
}

// hasMakefileBackendCommands checks Makefile for backend commands  
func (pd *ProjectDetector) hasMakefileBackendCommands(content string) bool {
	backendMakePatterns := []string{
		"server:", "api:", "backend:", "run:", "start-server:",
		"go run", "python", "java -jar", "mvn", "gradle",
		"node server", "node app", "rails server",
		"php artisan", "cargo run", "dotnet run",
		"./gradlew", "uvicorn", "gunicorn", "flask",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, pattern := range backendMakePatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	
	return false
}

// hasGoBackendCode checks Go files for backend server code
func (pd *ProjectDetector) hasGoBackendCode(content string) bool {
	goBackendPatterns := []string{
		"http.ListenAndServe", "gin.Default()", "echo.New()",
		"fiber.New()", "mux.NewRouter()", "chi.NewRouter()",
		"http.HandleFunc", "http.Handle", "net/http",
		"github.com/gin-gonic/gin", "github.com/labstack/echo",
		"github.com/gofiber/fiber", "github.com/gorilla/mux",
		"ListenAndServe", "HandleFunc", "GET(", "POST(",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, pattern := range goBackendPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return true
		}
	}
	
	return false
}

// hasPythonBackendCode checks Python files for backend server code
func (pd *ProjectDetector) hasPythonBackendCode(content string) bool {
	pythonBackendPatterns := []string{
		"from flask import", "from django", "from fastapi import",
		"flask.Flask", "Django", "FastAPI", "Tornado",
		"app = Flask", "app = FastAPI", "uvicorn.run",
		"app.run(", "wsgi", "asgi", "runserver",
		"@app.route", "@app.get", "@app.post",
		"HttpResponse", "render", "jsonify",
		"bottle.run", "pyramid", "falcon",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, pattern := range pythonBackendPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return true
		}
	}
	
	return false
}
