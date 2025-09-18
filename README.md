# Repo Explanation Tool

A production-ready Golang application that helps onboard new developers by analyzing and explaining codebases using advanced LLM integration. The tool automatically crawls code repositories, processes files through a sophisticated map-reduce pipeline, and generates comprehensive summaries of project architecture, purpose, and functionality.

## 🚀 Key Features

### **Intelligent Code Analysis**
- **Hierarchical Analysis**: Map-reduce pipeline (file → folder → project)
- **LLM Integration**: OpenAI GPT-4o-mini for cost-effective, accurate analysis
- **Smart File Processing**: Respects .gitignore, filters by file type, chunks large files
- **Caching System**: Hash-based caching for idempotent operations
- **Rate Limiting**: Built-in OpenAI API rate limiting and error handling

### **Dual Mode Application**
- **Interactive CLI**: REPL-style interface for repository analysis
- **Web API**: RESTful API with health check endpoint
- **Flexible Deployment**: Run as CLI tool or web service

### **Production Ready**
- **Security**: Secret redaction, configurable file filtering
- **Performance**: Concurrent processing, worker pools, intelligent chunking
- **Reliability**: Comprehensive error handling, graceful degradation
- **Configuration**: YAML + environment variable support

## 📁 Project Structure

```
├── analyzer/              # Core analysis pipeline
├── cache/                 # Caching system with file hashing
├── cli/                   # Interactive CLI/REPL
├── cmd/                   # Standalone entry points
├── config/                # Configuration management
├── controllers/           # HTTP request handlers  
├── internal/              # Internal packages
│   ├── chunker/          # File chunking system
│   ├── gitignore/        # .gitignore parser
│   ├── openai/           # OpenAI client with rate limiting
│   └── pipeline/         # Analysis pipeline orchestration
├── models/                # Data models
├── routes/                # API routes
├── config.yaml           # Main configuration file
├── .env.example          # Environment variables template
└── README.md             # This file
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
```bash
# Test CLI mode
./bin/repo-explanation -mode=cli

# Test server mode
./bin/repo-explanation -mode=server
```

## 🧠 Repository Analysis Usage

### **Interactive CLI Analysis**
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

### **Example Analysis Session**
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

### **API Server Mode**
```bash
# Start as web server
./bin/repo-explanation -mode=server

# Health check
curl http://localhost:8080/health
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

**"OpenAI API key not configured"**
```bash
# Set environment variable
export OPENAI_API_KEY=sk-your-key-here
# OR update .env file
```

**"Rate limit exceeded"**
- Adjust `rate_limiting` settings in config.yaml
- Check your OpenAI tier limits

**"Analysis taking too long"**
- Reduce `concurrent_workers`
- Enable caching: `cache.enabled: true`
- Check file size limits: `max_file_size_mb`

**"Too many files"**
- Add patterns to .gitignore
- Adjust `supported_extensions` in config
- Use `skip_secret_files` to exclude unnecessary files
