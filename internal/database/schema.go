package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ColumnConstraint represents various column constraints
type ColumnConstraint string

const (
	PrimaryKey ColumnConstraint = "PK"
	ForeignKey ColumnConstraint = "FK"
	Unique     ColumnConstraint = "unique"
	NotNull    ColumnConstraint = "not null"
	Default    ColumnConstraint = "default"
)

// Column represents a database column
type Column struct {
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	Constraints  []ColumnConstraint `json:"constraints"`
	DefaultValue string             `json:"default_value,omitempty"`
	References   *ForeignKeyRef     `json:"references,omitempty"`
}

// ForeignKeyRef represents a foreign key reference
type ForeignKeyRef struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

// Index represents a database index
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// Table represents a database table
type Table struct {
	Name        string            `json:"name"`
	Columns     map[string]Column `json:"columns"`
	PrimaryKeys []string          `json:"primary_keys"`
	Indexes     map[string]Index  `json:"indexes"`
}

// DatabaseSchema represents the complete database schema state
type DatabaseSchema struct {
	Tables            map[string]Table `json:"tables"`
	ForeignKeys       []ForeignKeyRef  `json:"foreign_keys"`
	MigrationPath     string           `json:"migration_path"`
	GeneratedAt       time.Time        `json:"generated_at"`
	FinalMigrationSQL string           `json:"final_migration_sql,omitempty"`
	LLMRelationships  string           `json:"llm_relationships,omitempty"`
}

// MigrationFile represents a SQL migration file
type MigrationFile struct {
	Path      string
	Name      string
	Content   string
	Timestamp string
}

// SchemaExtractor handles database schema extraction from migrations
type SchemaExtractor struct {
	schema        *DatabaseSchema
	migrationPath string
}

// NewSchemaExtractor creates a new schema extractor
func NewSchemaExtractor() *SchemaExtractor {
	return &SchemaExtractor{
		schema: &DatabaseSchema{
			Tables:      make(map[string]Table),
			ForeignKeys: make([]ForeignKeyRef, 0),
			GeneratedAt: time.Now(),
		},
	}
}

// FindMigrationFolders finds all folders containing "migrations" in their name
func (se *SchemaExtractor) FindMigrationFolders(projectPath string, files map[string]string) []string {
	var migrationFolders []string
	folderSet := make(map[string]bool)

	// Extract unique folder paths from file paths
	for filePath := range files {
		if strings.HasSuffix(strings.ToLower(filePath), ".sql") {
			dir := filepath.Dir(filePath)

			// Check if any part of the path contains "migration"
			pathParts := strings.Split(dir, string(os.PathSeparator))
			for _, part := range pathParts {
				if strings.Contains(strings.ToLower(part), "migration") {
					folderSet[dir] = true
					break
				}
			}
		}
	}

	// Convert set to slice
	for folder := range folderSet {
		migrationFolders = append(migrationFolders, folder)
	}

	return migrationFolders
}

// ExtractSchemaFromMigrations processes migration files and builds final schema
func (se *SchemaExtractor) ExtractSchemaFromMigrations(projectPath string, files map[string]string) (*DatabaseSchema, error) {
	// Find migration folders
	migrationFolders := se.FindMigrationFolders(projectPath, files)

	if len(migrationFolders) == 0 {
		return nil, fmt.Errorf("no migration folders found")
	}

	// Use the first migration folder found
	se.migrationPath = migrationFolders[0]
	se.schema.MigrationPath = se.migrationPath

	// Collect migration files from the folder
	var migrationFiles []MigrationFile
	for filePath, content := range files {
		if strings.HasPrefix(filePath, se.migrationPath) && strings.HasSuffix(strings.ToLower(filePath), ".sql") {
			filename := filepath.Base(filePath)
			migrationFiles = append(migrationFiles, MigrationFile{
				Path:      filePath,
				Name:      filename,
				Content:   content,
				Timestamp: se.extractTimestampFromFilename(filename),
			})
		}
	}

	if len(migrationFiles) == 0 {
		return nil, fmt.Errorf("no SQL files found in migration folders")
	}

	// Sort migration files by filename/timestamp
	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Name < migrationFiles[j].Name
	})

	// Process each migration file in order
	for _, migration := range migrationFiles {
		if err := se.processMigration(migration); err != nil {
			fmt.Printf("⚠️  Error processing migration %s: %v\n", migration.Name, err)
			// Continue processing other migrations instead of stopping
		}
	}

	return se.schema, nil
}

// extractTimestampFromFilename extracts timestamp from migration filename
func (se *SchemaExtractor) extractTimestampFromFilename(filename string) string {
	// Common patterns: 20231201_create_users.sql, 001_initial.sql, etc.
	timestampRegex := regexp.MustCompile(`^(\d+)`)
	matches := timestampRegex.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return filename
}

// processMigration processes a single migration file
func (se *SchemaExtractor) processMigration(migration MigrationFile) error {
	content := strings.ToUpper(migration.Content)

	// Split into statements
	statements := se.splitSQLStatements(content)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := se.processStatement(stmt); err != nil {
			return fmt.Errorf("error in statement: %v", err)
		}
	}

	return nil
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// splitSQLStatements splits SQL content into individual statements
func (se *SchemaExtractor) splitSQLStatements(content string) []string {
	// Split by semicolon
	statements := strings.Split(content, ";")
	var cleanStatements []string

	for _, stmt := range statements {
		// Clean up the statement
		lines := strings.Split(stmt, "\n")
		var cleanLines []string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip comment lines and empty lines
			if line != "" && !strings.HasPrefix(line, "--") && !strings.HasPrefix(line, "/*") {
				cleanLines = append(cleanLines, line)
			}
		}

		if len(cleanLines) > 0 {
			cleanStmt := strings.Join(cleanLines, " ")
			cleanStmt = strings.TrimSpace(cleanStmt)
			if cleanStmt != "" {
				cleanStatements = append(cleanStatements, cleanStmt)
			}
		}
	}

	return cleanStatements
}

// processStatement processes a single SQL statement
func (se *SchemaExtractor) processStatement(stmt string) error {
	stmt = strings.TrimSpace(stmt)

	if strings.HasPrefix(stmt, "CREATE TABLE") {
		return se.processCreateTable(stmt)
	} else if strings.HasPrefix(stmt, "ALTER TABLE") {
		return se.processAlterTable(stmt)
	} else if strings.HasPrefix(stmt, "DROP TABLE") {
		return se.processDropTable(stmt)
	} else if strings.HasPrefix(stmt, "CREATE INDEX") || strings.HasPrefix(stmt, "CREATE UNIQUE INDEX") {
		return se.processCreateIndex(stmt)
	}

	// Ignore other statements (INSERT, UPDATE, etc.)
	return nil
}

// processCreateTable processes CREATE TABLE statements
func (se *SchemaExtractor) processCreateTable(stmt string) error {
	// Extract table name
	tableNameRegex := regexp.MustCompile(`CREATE TABLE\s+(?:IF NOT EXISTS\s+)?(?:"?([^"\s]+)"?|\[([^\]]+)\]|([^\s(]+))`)
	matches := tableNameRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract table name from: %s", stmt)
	}

	tableName := ""
	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			tableName = strings.ToLower(matches[i])
			break
		}
	}

	// Extract column definitions
	columnRegex := regexp.MustCompile(`\(\s*(.*)\s*\)`)
	columnMatches := columnRegex.FindStringSubmatch(stmt)
	if len(columnMatches) < 2 {
		return fmt.Errorf("could not extract column definitions")
	}

	columnDefs := columnMatches[1]

	// Create table
	table := Table{
		Name:        tableName,
		Columns:     make(map[string]Column),
		PrimaryKeys: make([]string, 0),
		Indexes:     make(map[string]Index),
	}

	// Parse column definitions
	if err := se.parseColumnDefinitions(columnDefs, &table); err != nil {
		return err
	}

	se.schema.Tables[tableName] = table
	return nil
}

// parseColumnDefinitions parses column definitions from CREATE TABLE
func (se *SchemaExtractor) parseColumnDefinitions(columnDefs string, table *Table) error {
	// Split by commas, but be careful with parentheses
	columns := se.splitColumnDefinitions(columnDefs)

	for _, colDef := range columns {
		colDef = strings.TrimSpace(colDef)

		if strings.HasPrefix(colDef, "PRIMARY KEY") {
			se.parsePrimaryKeyConstraint(colDef, table)
		} else if strings.HasPrefix(colDef, "FOREIGN KEY") || strings.HasPrefix(colDef, "CONSTRAINT") {
			se.parseForeignKeyConstraint(colDef, table)
		} else if strings.HasPrefix(colDef, "UNIQUE") {
			se.parseUniqueConstraint(colDef, table)
		} else {
			// Regular column definition
			column := se.parseColumnDefinition(colDef)
			if column.Name != "" {
				table.Columns[column.Name] = column

				// Check if this column is a primary key
				for _, constraint := range column.Constraints {
					if constraint == PrimaryKey {
						table.PrimaryKeys = append(table.PrimaryKeys, column.Name)
					}
				}
			}
		}
	}

	return nil
}

// splitColumnDefinitions splits column definitions handling nested parentheses
func (se *SchemaExtractor) splitColumnDefinitions(columnDefs string) []string {
	var result []string
	var current strings.Builder
	parenLevel := 0

	for _, char := range columnDefs {
		if char == '(' {
			parenLevel++
		} else if char == ')' {
			parenLevel--
		} else if char == ',' && parenLevel == 0 {
			result = append(result, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(char)
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parseColumnDefinition parses a single column definition
func (se *SchemaExtractor) parseColumnDefinition(colDef string) Column {
	parts := strings.Fields(colDef)
	if len(parts) < 2 {
		return Column{}
	}

	column := Column{
		Name:        strings.ToLower(strings.Trim(parts[0], `"[]`)),
		Type:        strings.ToLower(parts[1]),
		Constraints: make([]ColumnConstraint, 0),
	}

	// Parse additional constraints
	colDefUpper := strings.ToUpper(colDef)

	if strings.Contains(colDefUpper, "PRIMARY KEY") {
		column.Constraints = append(column.Constraints, PrimaryKey)
	}
	if strings.Contains(colDefUpper, "NOT NULL") {
		column.Constraints = append(column.Constraints, NotNull)
	}
	if strings.Contains(colDefUpper, "UNIQUE") {
		column.Constraints = append(column.Constraints, Unique)
	}

	// Extract default value
	defaultRegex := regexp.MustCompile(`DEFAULT\s+([^,\s]+|\([^)]+\)|'[^']*')`)
	defaultMatches := defaultRegex.FindStringSubmatch(colDefUpper)
	if len(defaultMatches) > 1 {
		column.DefaultValue = defaultMatches[1]
		column.Constraints = append(column.Constraints, Default)
	}

	// Check for foreign key references
	if strings.Contains(colDefUpper, "REFERENCES") {
		fkRef := se.parseForeignKeyReference(colDef)
		if fkRef != nil {
			column.References = fkRef
			column.Constraints = append(column.Constraints, ForeignKey)
		}
	}

	return column
}

// parseForeignKeyReference parses foreign key references
func (se *SchemaExtractor) parseForeignKeyReference(colDef string) *ForeignKeyRef {
	fkRegex := regexp.MustCompile(`REFERENCES\s+([^\s(]+)\s*\(([^)]+)\)`)
	matches := fkRegex.FindStringSubmatch(strings.ToUpper(colDef))
	if len(matches) >= 3 {
		return &ForeignKeyRef{
			Table:  strings.ToLower(strings.Trim(matches[1], `"[]`)),
			Column: strings.ToLower(strings.Trim(matches[2], `"[]`)),
		}
	}
	return nil
}

// parsePrimaryKeyConstraint parses PRIMARY KEY constraints
func (se *SchemaExtractor) parsePrimaryKeyConstraint(constraint string, table *Table) {
	pkRegex := regexp.MustCompile(`PRIMARY KEY\s*\(([^)]+)\)`)
	matches := pkRegex.FindStringSubmatch(constraint)
	if len(matches) > 1 {
		columns := strings.Split(matches[1], ",")
		for _, col := range columns {
			col = strings.ToLower(strings.TrimSpace(strings.Trim(col, `"[]`)))
			table.PrimaryKeys = append(table.PrimaryKeys, col)
		}
	}
}

// parseForeignKeyConstraint parses FOREIGN KEY constraints
func (se *SchemaExtractor) parseForeignKeyConstraint(constraint string, table *Table) {
	// FOREIGN KEY (column) REFERENCES table(column)
	fkRegex := regexp.MustCompile(`FOREIGN KEY\s*\(([^)]+)\)\s*REFERENCES\s+([^\s(]+)\s*\(([^)]+)\)`)
	matches := fkRegex.FindStringSubmatch(constraint)
	if len(matches) >= 4 {
		localCol := strings.ToLower(strings.TrimSpace(strings.Trim(matches[1], `"[]`)))
		refTable := strings.ToLower(strings.TrimSpace(strings.Trim(matches[2], `"[]`)))
		refCol := strings.ToLower(strings.TrimSpace(strings.Trim(matches[3], `"[]`)))

		// Update column constraint if column exists
		if col, exists := table.Columns[localCol]; exists {
			col.References = &ForeignKeyRef{Table: refTable, Column: refCol}
			col.Constraints = append(col.Constraints, ForeignKey)
			table.Columns[localCol] = col
		} else {
			// If column doesn't exist yet, create it (this can happen with ADD CONSTRAINT)
			table.Columns[localCol] = Column{
				Name:        localCol,
				Type:        "bigint", // Assume bigint for FK columns
				Constraints: []ColumnConstraint{ForeignKey},
				References:  &ForeignKeyRef{Table: refTable, Column: refCol},
			}
		}

		// Add to schema foreign keys
		se.schema.ForeignKeys = append(se.schema.ForeignKeys, ForeignKeyRef{
			Table:  refTable,
			Column: refCol,
		})
	}
}

// parseUniqueConstraint parses UNIQUE constraints
func (se *SchemaExtractor) parseUniqueConstraint(constraint string, table *Table) {
	uniqueRegex := regexp.MustCompile(`UNIQUE\s*\(([^)]+)\)`)
	matches := uniqueRegex.FindStringSubmatch(constraint)
	if len(matches) > 1 {
		columns := strings.Split(matches[1], ",")
		indexName := fmt.Sprintf("unique_%s_%s", table.Name, strings.Join(columns, "_"))

		var cleanColumns []string
		for _, col := range columns {
			cleanColumns = append(cleanColumns, strings.ToLower(strings.TrimSpace(strings.Trim(col, `"[]`))))
		}

		table.Indexes[indexName] = Index{
			Name:    indexName,
			Columns: cleanColumns,
			Unique:  true,
		}
	}
}

// processAlterTable processes ALTER TABLE statements
func (se *SchemaExtractor) processAlterTable(stmt string) error {
	// Extract table name
	tableNameRegex := regexp.MustCompile(`ALTER TABLE\s+([^\s]+)`)
	matches := tableNameRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract table name from ALTER TABLE")
	}

	tableName := strings.ToLower(strings.Trim(matches[1], `"[]`))

	// Get or create table
	table, exists := se.schema.Tables[tableName]
	if !exists {
		table = Table{
			Name:        tableName,
			Columns:     make(map[string]Column),
			PrimaryKeys: make([]string, 0),
			Indexes:     make(map[string]Index),
		}
	}

	if strings.Contains(stmt, "ADD COLUMN") || strings.Contains(stmt, "ADD ") {
		return se.processAddColumn(stmt, &table)
	} else if strings.Contains(stmt, "DROP COLUMN") {
		return se.processDropColumn(stmt, &table)
	} else if strings.Contains(stmt, "ALTER COLUMN") || strings.Contains(stmt, "MODIFY COLUMN") {
		return se.processAlterColumn(stmt, &table)
	} else if strings.Contains(stmt, "ADD CONSTRAINT") {
		return se.processAddConstraint(stmt, &table)
	} else if strings.Contains(stmt, "DROP CONSTRAINT") {
		return se.processDropConstraint(stmt, &table)
	}

	se.schema.Tables[tableName] = table
	return nil
}

// processAddColumn processes ADD COLUMN statements
func (se *SchemaExtractor) processAddColumn(stmt string, table *Table) error {
	addColumnRegex := regexp.MustCompile(`ADD\s+(?:COLUMN\s+)?(.+)`)
	matches := addColumnRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract column definition from ADD COLUMN")
	}

	columnDef := matches[1]
	column := se.parseColumnDefinition(columnDef)
	if column.Name != "" {
		table.Columns[column.Name] = column

		// Check if this column is a primary key
		for _, constraint := range column.Constraints {
			if constraint == PrimaryKey {
				table.PrimaryKeys = append(table.PrimaryKeys, column.Name)
			}
		}
	}

	return nil
}

// processDropColumn processes DROP COLUMN statements
func (se *SchemaExtractor) processDropColumn(stmt string, table *Table) error {
	dropColumnRegex := regexp.MustCompile(`DROP\s+COLUMN\s+([^\s,]+)`)
	matches := dropColumnRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract column name from DROP COLUMN")
	}

	columnName := strings.ToLower(strings.Trim(matches[1], `"[]`))
	delete(table.Columns, columnName)

	// Remove from primary keys if present
	var newPKs []string
	for _, pk := range table.PrimaryKeys {
		if pk != columnName {
			newPKs = append(newPKs, pk)
		}
	}
	table.PrimaryKeys = newPKs

	return nil
}

// processAlterColumn processes ALTER/MODIFY COLUMN statements
func (se *SchemaExtractor) processAlterColumn(stmt string, table *Table) error {
	alterColumnRegex := regexp.MustCompile(`(?:ALTER|MODIFY)\s+COLUMN\s+([^\s]+)\s+(.+)`)
	matches := alterColumnRegex.FindStringSubmatch(stmt)
	if len(matches) < 3 {
		return fmt.Errorf("could not extract column info from ALTER COLUMN")
	}

	columnName := strings.ToLower(strings.Trim(matches[1], `"[]`))
	newDef := matches[2]

	// Update existing column
	if _, exists := table.Columns[columnName]; exists {
		newColumn := se.parseColumnDefinition(columnName + " " + newDef)
		if newColumn.Name != "" {
			newColumn.Name = columnName // Preserve original name
			table.Columns[columnName] = newColumn
		}
	}

	return nil
}

// processAddConstraint processes ADD CONSTRAINT statements
func (se *SchemaExtractor) processAddConstraint(stmt string, table *Table) error {
	if strings.Contains(stmt, "PRIMARY KEY") {
		se.parsePrimaryKeyConstraint(stmt, table)
	} else if strings.Contains(stmt, "FOREIGN KEY") {
		se.parseForeignKeyConstraint(stmt, table)
	} else if strings.Contains(stmt, "UNIQUE") {
		se.parseUniqueConstraint(stmt, table)
	}
	return nil
}

// processDropConstraint processes DROP CONSTRAINT statements
func (se *SchemaExtractor) processDropConstraint(stmt string, table *Table) error {
	// Simple implementation - could be enhanced
	return nil
}

// processDropTable processes DROP TABLE statements
func (se *SchemaExtractor) processDropTable(stmt string) error {
	tableNameRegex := regexp.MustCompile(`DROP TABLE\s+(?:IF EXISTS\s+)?([^\s;]+)`)
	matches := tableNameRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract table name from DROP TABLE")
	}

	tableName := strings.ToLower(strings.Trim(matches[1], `"[]`))
	delete(se.schema.Tables, tableName)

	return nil
}

// processCreateIndex processes CREATE INDEX statements
func (se *SchemaExtractor) processCreateIndex(stmt string) error {
	indexRegex := regexp.MustCompile(`CREATE\s+(UNIQUE\s+)?INDEX\s+([^\s]+)\s+ON\s+([^\s(]+)\s*\(([^)]+)\)`)
	matches := indexRegex.FindStringSubmatch(stmt)
	if len(matches) < 5 {
		return fmt.Errorf("could not parse CREATE INDEX statement")
	}

	isUnique := matches[1] != ""
	indexName := strings.ToLower(strings.Trim(matches[2], `"[]`))
	tableName := strings.ToLower(strings.Trim(matches[3], `"[]`))
	columnList := matches[4]

	var columns []string
	for _, col := range strings.Split(columnList, ",") {
		columns = append(columns, strings.ToLower(strings.TrimSpace(strings.Trim(col, `"[]`))))
	}

	if table, exists := se.schema.Tables[tableName]; exists {
		table.Indexes[indexName] = Index{
			Name:    indexName,
			Columns: columns,
			Unique:  isUnique,
		}
		se.schema.Tables[tableName] = table
	}

	return nil
}

// GeneratePlantUML generates PlantUML ERD from the final schema
func (se *SchemaExtractor) GeneratePlantUML() string {
	var puml strings.Builder

	puml.WriteString("@startuml\n")
	puml.WriteString("!define MASTER_COLOR #E8F4FD\n")
	puml.WriteString("!define DETAIL_COLOR #FFF2CC\n")
	puml.WriteString("\n")

	// Sort tables for consistent output
	var tableNames []string
	for tableName := range se.schema.Tables {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)

	// Generate entity definitions
	for _, tableName := range tableNames {
		table := se.schema.Tables[tableName]
		puml.WriteString(fmt.Sprintf("entity %s {\n", table.Name))

		// Sort columns for consistent output
		var columnNames []string
		for colName := range table.Columns {
			columnNames = append(columnNames, colName)
		}
		sort.Strings(columnNames)

		// Add primary key columns first
		for _, pkCol := range table.PrimaryKeys {
			if col, exists := table.Columns[pkCol]; exists {
				puml.WriteString(fmt.Sprintf("  * %s : %s [PK]\n", col.Name, col.Type))
			}
		}

		// Add separator if there are primary keys
		if len(table.PrimaryKeys) > 0 {
			puml.WriteString("  --\n")
		}

		// Add other columns
		for _, colName := range columnNames {
			col := table.Columns[colName]

			// Skip if already added as primary key
			isPrimaryKey := false
			for _, pk := range table.PrimaryKeys {
				if pk == colName {
					isPrimaryKey = true
					break
				}
			}
			if isPrimaryKey {
				continue
			}

			// Build column definition
			var constraints []string
			for _, constraint := range col.Constraints {
				switch constraint {
				case ForeignKey:
					constraints = append(constraints, "FK")
				case Unique:
					constraints = append(constraints, "unique")
				case NotNull:
					constraints = append(constraints, "not null")
				case Default:
					if col.DefaultValue != "" {
						constraints = append(constraints, fmt.Sprintf("default %s", col.DefaultValue))
					}
				}
			}

			constraintStr := ""
			if len(constraints) > 0 {
				constraintStr = fmt.Sprintf(" [%s]", strings.Join(constraints, ", "))
			}

			puml.WriteString(fmt.Sprintf("  %s : %s%s\n", col.Name, col.Type, constraintStr))
		}

		puml.WriteString("}\n\n")
	}

	// Generate relationships
	for _, tableName := range tableNames {
		table := se.schema.Tables[tableName]

		for _, col := range table.Columns {
			if col.References != nil {
				puml.WriteString(fmt.Sprintf("%s::%s --> %s::%s\n",
					col.References.Table, col.References.Column, table.Name, col.Name))
			}
		}
	}

	puml.WriteString("\n@enduml\n")

	return puml.String()
}

// SavePlantUMLFile saves the PlantUML content to a file
func (se *SchemaExtractor) SavePlantUMLFile(projectPath, pumlContent string) error {
	// Create output directory
	outputDir := "./database_schemas"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate filename from project path
	filename := generateSchemaFilename(projectPath)
	filePath := filepath.Join(outputDir, filename)

	// Write PUML content to file
	if err := os.WriteFile(filePath, []byte(pumlContent), 0o644); err != nil {
		return fmt.Errorf("failed to write PUML file: %v", err)
	}

	fmt.Printf("📄 Database schema saved to: %s\n", filePath)
	return nil
}

// generateSchemaFilename creates a filename from project path
func generateSchemaFilename(projectPath string) string {
	// Replace path separators and special characters
	filename := strings.ReplaceAll(projectPath, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "")
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.Trim(filename, "_")

	if filename == "" {
		filename = "root"
	}

	return filename + "_database_schema.puml"
}
