# 🔐 Secret Extraction CLI Usage Guide

## Quick Start

The secret extraction CLI mode allows you to analyze any project folder and extract all required environment variables and configuration secrets that need to be set.

### Usage Options

1. **Using the `-path` flag:**
   ```bash
   ./analyzer-api -mode=secrets -path=/path/to/your/project
   ```

2. **Using positional argument:**
   ```bash
   ./analyzer-api -mode=secrets /path/to/your/project
   ```

### Examples

1. **Analyze current directory:**
   ```bash
   ./analyzer-api -mode=secrets -path=.
   ```

2. **Analyze a specific project:**
   ```bash
   ./analyzer-api -mode=secrets -path=/Users/username/my-app
   ```

3. **Analyze with relative path:**
   ```bash
   ./analyzer-api -mode=secrets ../my-project
   ```

## What It Does

The CLI tool will:

1. **🔍 Scan Configuration Files:**
   - `.env*` files (`.env`, `.env.example`, `.env.local`, etc.)
   - `*.yaml` and `*.yml` files
   - `config.json`, `application.properties`
   - `docker-compose.yml`

2. **📋 Extract Required Variables:**
   - Keys with empty values
   - Placeholder values (e.g., "your_key_here", "changeme")
   - Environment variable references in YAML (`${VAR_NAME}`, `$VAR_NAME`)

3. **📊 Categorize by Type:**
   - **API_KEY**: API keys and access tokens
   - **DATABASE_URL**: Database connection strings  
   - **SECRET**: General secrets and tokens
   - **CREDENTIAL**: Passwords and credentials
   - **CONFIG**: General configuration values

4. **🏗️ Detect Project Structure:**
   - **Single Service**: All secrets shown as global
   - **Monorepo**: Secrets grouped by service/directory

## Output Format

```
============================================================
🔐 SECRET EXTRACTION RESULTS
============================================================
📂 Project Path: /path/to/project
📊 Project Type: single-service
🔢 Total Variables: 15
⚠️  Required Variables: 12
📝 Summary: Found 15 environment variables, 12 of which are required...

🌍 GLOBAL SECRETS
----------------------------------------
1. OPENAI_API_KEY
   Type: API_KEY
   Source: config.yaml
   Description: OpenAI API key for LLM integration
   Example: OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxx

2. DATABASE_URL
   Type: DATABASE_URL
   Source: .env.example
   Description: Database connection string
   Example: DATABASE_URL=postgresql://user:password@host:port/db

🛠️  SETUP INSTRUCTIONS
----------------------------------------
To configure this project:
1. Copy .env.example to .env (if available)
2. Set values for the 12 required environment variables shown above
3. Update any configuration files with your values
4. For API keys, refer to the respective service documentation
5. Ensure all services have access to their environment variables
============================================================
```

## Testing on Your Local Database

To test the secret extraction on your own projects:

1. **Build the analyzer:**
   ```bash
   go build -o analyzer-api .
   ```

2. **Run on any project:**
   ```bash
   ./analyzer-api -mode=secrets -path=/path/to/your/db/project
   ```

The tool will show you exactly which environment variables need to be configured, with helpful descriptions and examples for each one.

## Integration with Other Modes

- **Server mode** (default): `./analyzer-api -mode=server`
- **Interactive CLI**: `./analyzer-api -mode=cli`
- **Secret extraction**: `./analyzer-api -mode=secrets -path=/path`
- **Database debug**: `./analyzer-api -mode=debug-db /path`
