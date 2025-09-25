# 🚀 Repository Analyzer

A **production-ready Golang application** that revolutionizes developer onboarding by providing comprehensive codebase analysis through advanced LLM integration. Transform complex repositories into clear, actionable insights with real-time progress tracking and beautiful visualizations.

## 🌟 New Features

### **🎨 Modern Web Interface**
- **GitHub URL Analysis**: Simply paste a GitHub URL and get instant insights
- **Real-time Progress**: Live streaming updates with detailed progress indicators
- **Multi-tab Visualization**: Organized results across Overview, Analysis, Services, Database, Relationships, and Files tabs
- **Authentication Support**: Private repository access with GitHub tokens

### **🗄️ Advanced Database Analysis**
- **Streaming Schema Extraction**: Professional-grade migration analysis with real-time progress
- **Mermaid ERD Generation**: Beautiful database relationship diagrams
- **Comprehensive DDL Support**: CREATE/ALTER/DROP tables, constraints, indexes, enums, views
- **Multi-dialect Support**: PostgreSQL, MySQL, SQLite compatibility

### **🔗 Service Discovery & Relationships**
- **Microservice Detection**: Automatic service identification and mapping
- **Dependency Visualization**: Clear service relationship diagrams
- **Architecture Analysis**: Monolith vs microservices detection
- **Tech Stack Identification**: Comprehensive technology stack analysis

## 🚀 Key Features

### **Intelligent Code Analysis**
- **Hierarchical Analysis**: Map-reduce pipeline (file → folder → project)
- **LLM Integration**: OpenAI GPT-4o-mini for cost-effective, accurate analysis
- **Smart File Processing**: Respects .gitignore, filters by file type, chunks large files
- **Caching System**: Hash-based caching for idempotent operations
- **Rate Limiting**: Built-in OpenAI API rate limiting and error handling

### **Dual Mode Application**
- **Modern Web Application**: React-based frontend with intuitive GitHub URL analysis
- **Interactive CLI**: REPL-style interface for local repository analysis
- **Streaming API**: Real-time Server-Sent Events with progress updates
- **Flexible Deployment**: Docker Compose, single container, or cloud platform deployment

### **Production Ready**
- **Security**: Secret redaction, configurable file filtering
- **Performance**: Concurrent processing, worker pools, intelligent chunking
- **Reliability**: Comprehensive error handling, graceful degradation
- **Configuration**: YAML + environment variable support

## 📁 Project Structure

```
├── frontend/                    # React Web Application
│   ├── src/
│   │   ├── App.jsx             # Main application with tab system
│   │   ├── utils/api.js        # Streaming API client
│   │   └── components/ui/      # shadcn/ui component library
│   ├── package.json            # Frontend dependencies
│   └── vite.config.js          # Vite build configuration
├── controllers/                 # HTTP request handlers
│   └── analysis_controller.go  # Streaming analysis endpoints
├── internal/                    # Internal packages
│   ├── pipeline/               # Analysis orchestration
│   │   ├── analyzer.go         # Main analysis pipeline
│   │   └── crawler.go          # Repository file discovery
│   ├── database/               # Database schema analysis
│   │   ├── schema.go           # Legacy schema extractor
│   │   └── streaming_extractor.go  # New streaming extractor
│   ├── detector/               # Project type detection
│   │   ├── project_type.go     # 8-category classification
│   │   └── display.go          # Results visualization
│   ├── microservices/          # Service discovery
│   ├── relationships/          # Service relationship mapping
│   ├── openai/                 # LLM integration
│   ├── chunker/                # File processing
│   └── gitignore/              # Repository filtering
├── cache/                       # Analysis result caching
├── cli/                        # Interactive CLI interface
├── docker-compose.yml          # Production deployment
├── config.yaml                # Main configuration
├── .env.example               # Environment template
└── README.md                  # This documentation
```

## 🛠️ Setup & Installation

### Prerequisites
- Go 1.23.0+
- OpenAI API key

### 1. Clone and Build
```bash
git clone <repository>
cd repo-explanation
go mod tidy
go build -o bin/repo-explanation .
```

### 2. Configure OpenAI API
```bash
# Copy the example environment file
cp .env.example .env

# Edit .env and add your OpenAI API key
OPENAI_API_KEY=sk-your-openai-api-key-here
```

### 3. Test Installation

#### **Quick Start - Web Application**
```bash
# Start with Docker Compose (Recommended)
docker-compose up -d

# Access the application
open http://localhost
```

#### **CLI Mode (Local Analysis)**
```bash
# Interactive CLI for local repositories
./bin/repo-explanation -mode=cli

# When prompted, enter a local path:
# Please enter the relative path to a folder: ./my-project
```

#### **Server Mode (API Only)**
```bash
# Start API server only
./bin/repo-explanation -mode=server

# Access API at http://localhost:8080
curl http://localhost:8080/health
```

## 🧠 Repository Analysis Usage

### **🌐 Web Application Analysis (Recommended)**

#### **Step 1: Access the Application**
```
1. Start: docker-compose up -d
2. Open: http://localhost
3. Interface: Modern React application with real-time progress
```

#### **Step 2: Analyze GitHub Repository**
```
1. Paste GitHub URL: https://github.com/owner/repository
2. Optional: Add GitHub token for private repositories
3. Click "Analyze Repository"
4. Watch real-time progress with detailed stages
```

#### **Step 3: Explore Results Across Tabs**

**📊 Overview Tab**
- Project type and confidence score
- Quick statistics (files, languages, architecture)
- Two-sentence summary for new developers

**🔬 Analysis Tab** 
- Detailed architectural analysis
- Purpose and technology stack
- Folder-by-folder breakdown

**🏢 Services Tab**
- Discovered microservices and their purposes
- Service technology stacks and endpoints
- API and configuration details

**🗄️ Database Tab**
- Interactive Mermaid ERD diagrams
- Table structures with columns and constraints
- Primary keys, foreign keys, and relationships
- Migration history analysis

**🔗 Relationships Tab**
- Service dependency mapping
- Communication patterns
- Architecture visualization

**📁 Files Tab**
- Individual file analysis results
- Functions, imports, and complexity scores
- Security insights and recommendations

### **🖥️ CLI Analysis (Local Repositories)**
```bash
# Start the CLI
./bin/repo-explanation -mode=cli

# When prompted, enter a path to analyze:
Please enter the relative path to a folder: ~/my-project

# The tool will:
# 1. Count folders in the path
# 2. Discover and filter code files
# 3. Analyze files with LLM in parallel
# 4. Generate folder summaries
# 5. Create final project overview
# 6. Display comprehensive results
```

### **Analysis Output**
The tool provides:
- **Purpose**: Why this repository exists
- **Architecture**: High-level architectural patterns (MVC, microservices, etc.)
- **Data Models**: Key data structures and relationships
- **External Services**: APIs, databases, and integrations
- **Two-sentence Summary**: Concise explanation for new developers

### **🎯 Real-World Analysis Examples**

#### **Web Application Example: Go Backend Project**
```
🚀 Repository: https://github.com/gin-gonic/gin
📊 Progress: [████████████████████] 100% Complete

📋 OVERVIEW RESULTS:
   🎯 Type: Backend (Confidence: 9.2/10)
   📊 Files: 116 analyzed (0.79 MB)
   🛠️ Languages: Go (95 files), YAML (10 files)
   🏗️ Architecture: Monolith

🔬 DETAILED ANALYSIS:
   Purpose: High-performance HTTP web framework for Go
   Tech Stack: Go, Testing frameworks, Documentation tools
   Architecture: Clean, well-structured framework library

🗄️ DATABASE SCHEMA:
   Status: No database migrations found
   Type: Framework/Library project

🏢 SERVICES DISCOVERED:
   • gin-framework (Go) - Core HTTP framework implementation
   • examples (Go) - Usage examples and demonstrations
   • testing (Go) - Comprehensive test suite

🔗 RELATIONSHIPS:
   • Framework-to-examples dependency
   • Test-to-framework validation relationship
```

#### **Frontend Project Example**
```
🚀 Repository: https://github.com/facebook/react
📊 Progress: [████████████████████] 100% Complete

📋 OVERVIEW RESULTS:
   🎯 Type: Frontend (Confidence: 8.7/10)
   📊 Files: 2,847 analyzed (45.2 MB)
   🛠️ Languages: JavaScript (1,205), TypeScript (892), Flow (301)
   🏗️ Architecture: Monorepo

🔬 DETAILED ANALYSIS:
   Purpose: JavaScript library for building user interfaces
   Tech Stack: JavaScript, TypeScript, Flow, Jest, Rollup
   Architecture: Component-based, declarative UI framework

🏢 MONOREPO SERVICES:
   • react (JavaScript) - Core React library
   • react-dom (JavaScript) - DOM renderer for React
   • scheduler (JavaScript) - Cooperative scheduling for React
   • react-reconciler (JavaScript) - React reconciliation algorithm
```

### **Example CLI Analysis Session**
```
🚀 Repo Explanation CLI Started
Please enter the relative path to a folder: ./my-go-project
Total number of folders in './my-go-project': 8

🧠 Starting repository analysis with LLM...
🔍 Discovering files...
📁 Found 23 files (0.45 MB)
🧠 Analyzing files...
📊 Processed 10/23 files
📊 Processed 20/23 files
✅ Analyzed 23 files
📂 Analyzing folders...
✅ Analyzed 5 folders
🏗️ Analyzing project...
✅ Project analysis complete!

⏱️ Analysis completed in 45.67 seconds

================================================================================
📊 REPOSITORY ANALYSIS RESULTS
================================================================================

🎯 PURPOSE:
   A RESTful API service for user management with authentication and authorization features

🏗️ ARCHITECTURE:
   Clean architecture following MVC pattern with dependency injection and layered structure

📋 DATA MODELS:
   • User (authentication and profile)
   • Role (authorization system)
   • Session (user sessions)

🔗 EXTERNAL SERVICES:
   • PostgreSQL database
   • Redis cache
   • JWT authentication

📝 SUMMARY:
   This is a user management microservice built with Go that provides authentication and authorization capabilities through REST APIs. The service follows clean architecture principles with proper separation of concerns and includes caching, database persistence, and JWT-based security.

📈 STATISTICS:
   • Files analyzed: 23
   • Total size: 0.45 MB
   • File types:
     - .go: 18 files
     - .sql: 3 files
     - .yaml: 2 files
================================================================================

Type 'try me' to test, '/end' to exit
>
```

## 🚀 Deployment

### **Production Deployment (Recommended)**

#### **Docker Compose - Full Stack**
```bash
# Quick deployment script
./deploy.sh

# Manual deployment
docker-compose up -d

# View application logs
docker-compose logs -f

# Health check
curl http://localhost/health  # Frontend
curl http://localhost:8080/health  # Backend API
```

**Architecture:**
- **Frontend**: React app served by Nginx on port 80
- **Backend**: Go API server on port 8080  
- **Services**: Automatic restarts and health checks
- **Networking**: Internal Docker network with external access

#### **Single Container Deployment**
```bash
# Build and run combined container
docker build -f Dockerfile.combined -t repo-analyzer .
docker run -d -p 8080:8080 --name repo-analyzer repo-analyzer

# Access application at http://localhost:8080
```

#### **Cloud Platform Deployment**

**Railway**
```bash
railway up  # Automatic deployment from repository
```

**Render/Heroku**
- Connect GitHub repository
- Use `Dockerfile.combined` for build
- Set `OPENAI_API_KEY` environment variable

### **Development Setup**
```bash
# Backend development
go run main.go -mode=server

# Frontend development  
cd frontend && npm run dev

# Access frontend at http://localhost:5173
# API available at http://localhost:8080
```

## ⚙️ Configuration

### **Main Configuration** (`config.yaml`)
```yaml
# OpenAI API Configuration
openai:
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o-mini"          # Cost-effective model
  max_tokens_per_request: 4000
  temperature: 0.1              # Low for consistent results

# Rate Limiting (adjust based on your OpenAI tier)
rate_limiting:
  requests_per_minute: 500
  requests_per_day: 10000
  concurrent_workers: 5

# File Processing
file_processing:
  max_file_size_mb: 10
  chunk_size_tokens: 3000
  supported_extensions:        # Add/remove as needed
    - ".go"
    - ".js" 
    - ".py"
    # ... (see config.yaml for full list)

# Caching
cache:
  enabled: true
  directory: "./cache"
  ttl_hours: 24

# Security
security:
  redact_secrets: true         # Redact API keys, passwords
  skip_secret_files:           # Skip these files entirely
    - ".env"
    - "*.key"
    - "*.pem"
```

### **Environment Variables** (`.env`)
```bash
OPENAI_API_KEY=sk-your-actual-key-here
```

## 🔧 Advanced Usage

### **🌐 Streaming API**

#### **Analyze GitHub Repository**
```bash
curl -X POST http://localhost:8080/api/analyze/stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "url": "https://github.com/owner/repository",
    "type": "github_url",
    "token": "ghp_optional_private_repo_token"
  }'
```

**Response: Server-Sent Events (SSE)**
```
data: {"type":"progress","stage":"🚀 Initializing analysis...","progress":0,"message":"Starting repository analysis"}

data: {"type":"progress","stage":"📂 Cloning repository...","progress":5,"message":"Downloading repository files"}

data: {"type":"data","stage":"Project type detected","progress":32,"data":{"project_type":{"primary_type":"Backend","confidence":8.5}}}

data: {"type":"data","stage":"Database schema extracted","progress":92,"data":{"database_schema":{"tables":{...},"mermaid":"erDiagram..."}}}

data: {"type":"complete","stage":"🎉 Analysis complete!","progress":100,"data":{"project_summary":{...},"database_schema":{...}}}
```

#### **Traditional API (Non-streaming)**
```bash
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://github.com/owner/repository",
    "type": "github_url"
  }'
```

#### **Health Check**
```bash
curl http://localhost:8080/health
# Response: {"message":"Server is running","service":"repo-explanation","status":"healthy"}
```

### **Custom Configuration**
```bash
# Use custom config file
REPO_CONFIG=./custom-config.yaml ./bin/repo-explanation -mode=cli
```

### **Cache Management**
```bash
# Clear analysis cache
rm -rf ./cache/

# Disable caching in config.yaml
cache:
  enabled: false
```

## 📊 Cost & Performance

### **OpenAI API Costs** (GPT-4o-mini pricing)
- Input: $0.15 per 1M tokens
- Output: $0.60 per 1M tokens
- **Typical project (50 files, 2MB)**: ~$0.05-$0.15

### **Performance Optimizations**
- **Caching**: Reuses previous analysis results
- **Chunking**: Processes large files efficiently  
- **Concurrency**: Parallel file processing
- **Rate Limiting**: Respects API limits
- **Incremental**: Only reprocesses changed files

## 🏗️ Architecture Details

### **Map-Reduce Pipeline**
1. **Map Phase**: Analyze each file individually
   - Chunk large files (3k tokens max)
   - Extract language, purpose, functions, imports
   - Identify security risks and side effects
   
2. **Reduce Phase 1**: Aggregate files into folder summaries
   - Combine file analyses by directory
   - Identify module purposes and dependencies
   
3. **Reduce Phase 2**: Create project overview
   - Synthesize folder summaries
   - Generate architecture description
   - Create final two-sentence summary

### **Key Components**
- **Crawler**: Discovers files, respects .gitignore
- **Chunker**: Splits large files intelligently
- **OpenAI Client**: Handles API calls with rate limiting
- **Cache**: Hash-based result caching
- **Pipeline**: Orchestrates the analysis workflow

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit pull request

## 📄 License

[License information]

## 🆘 Troubleshooting

### **Common Issues**

**"Frontend not loading"**
```bash
# Check if services are running
docker-compose ps

# View frontend logs
docker-compose logs frontend

# Restart frontend service
docker-compose restart frontend
```

**"GitHub repository cloning failed"**
```bash
# Check if repository is public or add GitHub token
# For private repositories, add token in the web interface

# Check backend logs for cloning errors
docker-compose logs backend
```

**"Analysis progress stuck or timeout"**
```bash
# Large repositories may take 5-10 minutes
# Check backend logs for detailed progress
docker-compose logs -f backend

# Increase timeout in browser if needed
# The system has 35-minute timeout built-in
```

**"OpenAI API key not configured"**
```bash
# Set environment variable before starting
export OPENAI_API_KEY=sk-your-key-here
docker-compose up -d

# OR update .env file
echo "OPENAI_API_KEY=sk-your-key-here" > .env
```

**"Rate limit exceeded"**
- Adjust `rate_limiting` settings in config.yaml
- Check your OpenAI tier limits
- Reduce `concurrent_workers` from 5 to 2-3

**"Database schema not displaying"**
```bash
# Ensure repository has SQL migration files
# Supported patterns: migrations/, sql_migrations/, db/migrate/

# Check backend logs for schema extraction details
docker-compose logs backend | grep -i database
```

**"Service discovery not working"**
- Ensure repository has clear service structure
- Works best with microservice architectures
- Check for docker-compose.yml, package.json, go.mod files

**"Analysis incomplete or partial results"**
```bash
# Check OpenAI API quota and billing
# Verify internet connectivity for API calls
# Check backend logs for specific failures
docker-compose logs backend | grep -i error
```
