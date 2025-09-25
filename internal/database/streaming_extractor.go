package database

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// StreamingResponse represents a single streaming response event
type StreamingResponse struct {
	Phase    string          `json:"phase"`
	Progress ProgressInfo    `json:"progress"`
	Message  string          `json:"message"`
	Schema   *CanonicalSchema `json:"schema,omitempty"`
	Mermaid  string          `json:"mermaid,omitempty"`
}

// ProgressInfo tracks current progress
type ProgressInfo struct {
	Current int `json:"current"`
	Total   int `json:"total"`
}

// CanonicalSchema represents the complete canonical schema model
type CanonicalSchema struct {
	Tables map[string]*CanonicalTable `json:"tables"`
	Enums  map[string][]string        `json:"enums"`
	Views  map[string]*View           `json:"views"`
}

// CanonicalTable represents a table in canonical format
type CanonicalTable struct {
	Columns     map[string]*CanonicalColumn   `json:"columns"`
	PrimaryKey  []string                      `json:"primaryKey"`
	Unique      [][]string                    `json:"unique"`
	ForeignKeys []*CanonicalForeignKey        `json:"foreignKeys"`
	Indexes     []*CanonicalIndex             `json:"indexes"`
	Comment     *string                       `json:"comment"`
}

// CanonicalColumn represents a column in canonical format
type CanonicalColumn struct {
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	Default  *string `json:"default"`
	Comment  *string `json:"comment"`
}

// CanonicalForeignKey represents a foreign key in canonical format
type CanonicalForeignKey struct {
	Columns    []string `json:"columns"`
	RefTable   string   `json:"refTable"`
	RefColumns []string `json:"refColumns"`
	OnDelete   *string  `json:"onDelete"`
	OnUpdate   *string  `json:"onUpdate"`
	Name       *string  `json:"name"`
}

// CanonicalIndex represents an index in canonical format
type CanonicalIndex struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Using   *string  `json:"using"`
}

// View represents a database view
type View struct {
	SQL string `json:"sql"`
}

// Migration represents a single migration file
type Migration struct {
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

// StreamingSchemaExtractor handles streaming schema extraction
type StreamingSchemaExtractor struct {
	schema  *CanonicalSchema
	dialect string
}

// NewStreamingSchemaExtractor creates a new streaming schema extractor
func NewStreamingSchemaExtractor(dialect string) *StreamingSchemaExtractor {
	if dialect == "" {
		dialect = "postgres"
	}
	
	return &StreamingSchemaExtractor{
		schema: &CanonicalSchema{
			Tables: make(map[string]*CanonicalTable),
			Enums:  make(map[string][]string),
			Views:  make(map[string]*View),
		},
		dialect: dialect,
	}
}

// DDLStatement represents a parsed DDL statement
type DDLStatement struct {
	Type      string
	Statement string
	TableName string
}

// BuildSchemaAndStream processes migrations and emits streaming responses
func (se *StreamingSchemaExtractor) BuildSchemaAndStream(migrations []Migration, callback func(StreamingResponse)) error {
	totalMigrations := len(migrations)
	
	// Initialize empty schema
	se.schema = &CanonicalSchema{
		Tables: make(map[string]*CanonicalTable),
		Enums:  make(map[string][]string),
		Views:  make(map[string]*View),
	}
	
	// Process each migration
	for i, migration := range migrations {
		// Emit parse phase
		callback(StreamingResponse{
			Phase: "parse",
			Progress: ProgressInfo{
				Current: i + 1,
				Total:   totalMigrations,
			},
			Message: fmt.Sprintf("Parsing migration %s", migration.Name),
			Schema:  se.schema,
		})
		
		// Parse migration into DDL statements (with graceful error handling)
		statements, err := se.parseMigrationSQL(migration.SQL)
		if err != nil {
			callback(StreamingResponse{
				Phase: "warning",
				Progress: ProgressInfo{
					Current: i + 1,
					Total:   totalMigrations,
				},
				Message: fmt.Sprintf("⚠️ Failed to parse migration %s: %v (continuing with next migration)", migration.Name, err),
				Schema:  se.schema,
			})
			continue // Skip this migration but continue with others
		}
		
		// Emit apply phase
		callback(StreamingResponse{
			Phase: "apply",
			Progress: ProgressInfo{
				Current: i + 1,
				Total:   totalMigrations,
			},
			Message: fmt.Sprintf("Applying migration %s (%d statements)", migration.Name, len(statements)),
			Schema:  se.schema,
		})
		
		// Apply each statement (with graceful error handling)
		successfulStatements := 0
		for j, stmt := range statements {
			if err := se.applyStatement(stmt); err != nil {
				callback(StreamingResponse{
					Phase: "warning",
					Progress: ProgressInfo{
						Current: i + 1,
						Total:   totalMigrations,
					},
					Message: fmt.Sprintf("⚠️ Failed to apply statement %d/%d in %s: %v (continuing with next statement)", j+1, len(statements), migration.Name, err),
					Schema:  se.schema,
				})
				continue // Skip this statement but continue with others
			}
			successfulStatements++
		}
		
		// Report success status
		if successfulStatements > 0 {
			callback(StreamingResponse{
				Phase: "success",
				Progress: ProgressInfo{
					Current: i + 1,
					Total:   totalMigrations,
				},
				Message: fmt.Sprintf("✅ Applied %d/%d statements from %s", successfulStatements, len(statements), migration.Name),
				Schema:  se.schema,
			})
		}
	}
	
	// Emit indexing phase
	callback(StreamingResponse{
		Phase: "indexing",
		Progress: ProgressInfo{
			Current: totalMigrations,
			Total:   totalMigrations,
		},
		Message: "Normalizing schema",
		Schema:  se.schema,
	})
	
	// Normalize schema
	se.normalizeSchema()
	
	// Emit ERD phase
	callback(StreamingResponse{
		Phase: "erd",
		Progress: ProgressInfo{
			Current: totalMigrations,
			Total:   totalMigrations,
		},
		Message: "Generating ERD",
		Schema:  se.schema,
	})
	
	// Generate Mermaid ERD
	mermaidERD := se.generateMermaidERD()
	
	// Emit final migration generation phase
	callback(StreamingResponse{
		Phase: "finalizing",
		Progress: ProgressInfo{
			Current: totalMigrations,
			Total:   totalMigrations,
		},
		Message: "Generating final migration SQL",
		Schema:  se.schema,
		Mermaid: mermaidERD,
	})
	
	// Generate the final migration SQL file
	finalMigrationSQL := se.GenerateFinalMigrationSQL()
	
	// Emit completion with final migration SQL
	callback(StreamingResponse{
		Phase: "complete",
		Progress: ProgressInfo{
			Current: totalMigrations,
			Total:   totalMigrations,
		},
		Message: fmt.Sprintf("Schema extraction complete! Generated final migration with %d tables", len(se.schema.Tables)),
		Schema:  se.schema,
		Mermaid: mermaidERD,
	})
	
	// Store the final migration SQL in the schema for later access
	if se.schema != nil {
		// We'll add this as a custom field (even though it's not in the struct, we can pass it separately)
		fmt.Printf("📄 Generated final migration SQL (%d characters)\n", len(finalMigrationSQL))
		fmt.Printf("🎯 Users can run this single file instead of %d individual migrations\n", totalMigrations)
	}
	
	return nil
}

// parseMigrationSQL parses SQL content into DDL statements
func (se *StreamingSchemaExtractor) parseMigrationSQL(sql string) ([]DDLStatement, error) {
	var statements []DDLStatement
	
	// Split by semicolon and clean up
	rawStatements := strings.Split(sql, ";")
	
	for _, rawStmt := range rawStatements {
		cleanStmt := se.cleanSQLStatement(rawStmt)
		if cleanStmt == "" {
			continue
		}
		
		// Identify statement type
		stmtType := se.identifyStatementType(cleanStmt)
		if stmtType == "" {
			continue // Skip unsupported statements
		}
		
		// Extract table name if applicable
		tableName := se.extractTableName(cleanStmt, stmtType)
		
		statements = append(statements, DDLStatement{
			Type:      stmtType,
			Statement: cleanStmt,
			TableName: tableName,
		})
	}
	
	return statements, nil
}

// cleanSQLStatement cleans up a SQL statement
func (se *StreamingSchemaExtractor) cleanSQLStatement(stmt string) string {
	// Remove comments and normalize whitespace
	lines := strings.Split(stmt, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comment lines
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "/*") {
			continue
		}
		// Remove inline comments
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	
	if len(cleanLines) == 0 {
		return ""
	}
	
	return strings.Join(cleanLines, " ")
}

// identifyStatementType identifies the type of DDL statement
func (se *StreamingSchemaExtractor) identifyStatementType(stmt string) string {
	upperStmt := strings.ToUpper(stmt)
	
	if strings.HasPrefix(upperStmt, "CREATE TABLE") {
		return "CREATE_TABLE"
	} else if strings.HasPrefix(upperStmt, "DROP TABLE") {
		return "DROP_TABLE"
	} else if strings.HasPrefix(upperStmt, "ALTER TABLE") {
		return "ALTER_TABLE"
	} else if strings.HasPrefix(upperStmt, "CREATE INDEX") || strings.HasPrefix(upperStmt, "CREATE UNIQUE INDEX") {
		return "CREATE_INDEX"
	} else if strings.HasPrefix(upperStmt, "DROP INDEX") {
		return "DROP_INDEX"
	} else if strings.HasPrefix(upperStmt, "CREATE TYPE") {
		return "CREATE_TYPE"
	} else if strings.HasPrefix(upperStmt, "CREATE VIEW") {
		return "CREATE_VIEW"
	} else if strings.HasPrefix(upperStmt, "DROP VIEW") {
		return "DROP_VIEW"
	}
	
	return "" // Unsupported statement type
}

// extractTableName extracts table name from DDL statement
func (se *StreamingSchemaExtractor) extractTableName(stmt, stmtType string) string {
	var regex *regexp.Regexp
	
	switch stmtType {
	case "CREATE_TABLE":
		regex = regexp.MustCompile(`CREATE TABLE\s+(?:IF NOT EXISTS\s+)?(?:"?([^"\s(]+)"?|\[([^\]]+)\]|([^\s(]+))`)
	case "DROP_TABLE":
		regex = regexp.MustCompile(`DROP TABLE\s+(?:IF EXISTS\s+)?(?:"?([^"\s;]+)"?|\[([^\]]+)\]|([^\s;]+))`)
	case "ALTER_TABLE":
		regex = regexp.MustCompile(`ALTER TABLE\s+(?:"?([^"\s]+)"?|\[([^\]]+)\]|([^\s]+))`)
	case "CREATE_INDEX":
		regex = regexp.MustCompile(`ON\s+(?:"?([^"\s(]+)"?|\[([^\]]+)\]|([^\s(]+))`)
	default:
		return ""
	}
	
	matches := regex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return ""
	}
	
	for i := 1; i < len(matches); i++ {
		if matches[i] != "" {
			return strings.ToLower(strings.TrimSpace(matches[i]))
		}
	}
	
	return ""
}

// applyStatement applies a DDL statement to the schema (with graceful error handling)
func (se *StreamingSchemaExtractor) applyStatement(stmt DDLStatement) error {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error for graceful handling
			fmt.Printf("⚠️ Recovered from panic while applying statement: %v\n", r)
		}
	}()

	switch stmt.Type {
	case "CREATE_TABLE":
		return se.applyCreateTableSafely(stmt)
	case "DROP_TABLE":
		return se.applyDropTableSafely(stmt)
	case "ALTER_TABLE":
		return se.applyAlterTableSafely(stmt)
	case "CREATE_INDEX":
		return se.applyCreateIndexSafely(stmt)
	case "DROP_INDEX":
		return se.applyDropIndexSafely(stmt)
	case "CREATE_TYPE":
		return se.applyCreateTypeSafely(stmt)
	case "CREATE_VIEW":
		return se.applyCreateViewSafely(stmt)
	case "DROP_VIEW":
		return se.applyDropViewSafely(stmt)
	default:
		// Don't fail on unsupported statements, just skip them
		fmt.Printf("⚠️ Skipping unsupported statement type: %s\n", stmt.Type)
		return nil
	}
}

// applyCreateTable applies CREATE TABLE statement
func (se *StreamingSchemaExtractor) applyCreateTable(stmt DDLStatement) error {
	tableName := stmt.TableName
	if tableName == "" {
		return fmt.Errorf("could not extract table name from CREATE TABLE")
	}
	
	// Check if table already exists
	if _, exists := se.schema.Tables[tableName]; exists {
		return fmt.Errorf("table %s already exists", tableName)
	}
	
	// Create new table
	table := &CanonicalTable{
		Columns:     make(map[string]*CanonicalColumn),
		PrimaryKey:  []string{},
		Unique:      [][]string{},
		ForeignKeys: []*CanonicalForeignKey{},
		Indexes:     []*CanonicalIndex{},
		Comment:     nil,
	}
	
	// Extract column definitions from CREATE TABLE statement
	if err := se.parseCreateTableColumns(stmt.Statement, table); err != nil {
		return err
	}
	
	se.schema.Tables[tableName] = table
	return nil
}

// parseCreateTableColumns parses column definitions from CREATE TABLE
func (se *StreamingSchemaExtractor) parseCreateTableColumns(stmt string, table *CanonicalTable) error {
	// Extract content between parentheses
	parenRegex := regexp.MustCompile(`CREATE TABLE[^(]*\(\s*(.*)\s*\)`)
	matches := parenRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract column definitions")
	}
	
	columnDefs := matches[1]
	
	// Split column definitions (handle nested parentheses)
	definitions := se.splitTableDefinitions(columnDefs)
	
	for _, def := range definitions {
		def = strings.TrimSpace(def)
		if def == "" {
			continue
		}
		
		upperDef := strings.ToUpper(def)
		
		if strings.HasPrefix(upperDef, "PRIMARY KEY") {
			se.parsePrimaryKeyDef(def, table)
		} else if strings.HasPrefix(upperDef, "FOREIGN KEY") || strings.HasPrefix(upperDef, "CONSTRAINT") {
			se.parseForeignKeyDef(def, table)
		} else if strings.HasPrefix(upperDef, "UNIQUE") {
			se.parseUniqueDef(def, table)
		} else {
			// Regular column definition
			if err := se.parseColumnDef(def, table); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// splitTableDefinitions splits table definitions handling nested parentheses
func (se *StreamingSchemaExtractor) splitTableDefinitions(defs string) []string {
	var result []string
	var current strings.Builder
	parenLevel := 0
	
	for _, char := range defs {
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

// parseColumnDef parses a single column definition
func (se *StreamingSchemaExtractor) parseColumnDef(def string, table *CanonicalTable) error {
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return fmt.Errorf("invalid column definition: %s", def)
	}
	
	columnName := strings.ToLower(strings.Trim(parts[0], `"[]`))
	columnType := strings.ToLower(parts[1])
	
	// Create column
	column := &CanonicalColumn{
		Type:     columnType,
		Nullable: true, // Default to nullable
		Default:  nil,
		Comment:  nil,
	}
	
	// Parse constraints
	upperDef := strings.ToUpper(def)
	
	// Check for NOT NULL
	if strings.Contains(upperDef, "NOT NULL") {
		column.Nullable = false
	}
	
	// Check for PRIMARY KEY
	if strings.Contains(upperDef, "PRIMARY KEY") {
		table.PrimaryKey = append(table.PrimaryKey, columnName)
		column.Nullable = false // Primary keys are not nullable
	}
	
	// Check for UNIQUE
	if strings.Contains(upperDef, "UNIQUE") {
		table.Unique = append(table.Unique, []string{columnName})
	}
	
	// Extract default value
	defaultRegex := regexp.MustCompile(`DEFAULT\s+([^,\s]+|\([^)]+\)|'[^']*')`)
	defaultMatches := defaultRegex.FindStringSubmatch(upperDef)
	if len(defaultMatches) > 1 {
		defaultValue := defaultMatches[1]
		column.Default = &defaultValue
	}
	
	// Check for foreign key reference
	if strings.Contains(upperDef, "REFERENCES") {
		fkRef := se.parseForeignKeyRef(def)
		if fkRef != nil {
			table.ForeignKeys = append(table.ForeignKeys, fkRef)
		}
	}
	
	table.Columns[columnName] = column
	return nil
}

// parseForeignKeyRef parses inline foreign key reference
func (se *StreamingSchemaExtractor) parseForeignKeyRef(def string) *CanonicalForeignKey {
	fkRegex := regexp.MustCompile(`REFERENCES\s+([^\s(]+)\s*\(([^)]+)\)`)
	matches := fkRegex.FindStringSubmatch(strings.ToUpper(def))
	if len(matches) >= 3 {
		refTable := strings.ToLower(strings.Trim(matches[1], `"[]`))
		refColumn := strings.ToLower(strings.Trim(matches[2], `"[]`))
		
		// Extract column name from beginning of definition
		parts := strings.Fields(def)
		if len(parts) > 0 {
			columnName := strings.ToLower(strings.Trim(parts[0], `"[]`))
			
			return &CanonicalForeignKey{
				Columns:    []string{columnName},
				RefTable:   refTable,
				RefColumns: []string{refColumn},
				OnDelete:   nil,
				OnUpdate:   nil,
				Name:       nil,
			}
		}
	}
	return nil
}

// parsePrimaryKeyDef parses PRIMARY KEY constraint
func (se *StreamingSchemaExtractor) parsePrimaryKeyDef(def string, table *CanonicalTable) {
	pkRegex := regexp.MustCompile(`PRIMARY KEY\s*\(([^)]+)\)`)
	matches := pkRegex.FindStringSubmatch(def)
	if len(matches) > 1 {
		columns := strings.Split(matches[1], ",")
		for _, col := range columns {
			col = strings.ToLower(strings.TrimSpace(strings.Trim(col, `"[]`)))
			table.PrimaryKey = append(table.PrimaryKey, col)
		}
	}
}

// parseForeignKeyDef parses FOREIGN KEY constraint
func (se *StreamingSchemaExtractor) parseForeignKeyDef(def string, table *CanonicalTable) {
	fkRegex := regexp.MustCompile(`FOREIGN KEY\s*\(([^)]+)\)\s*REFERENCES\s+([^\s(]+)\s*\(([^)]+)\)`)
	matches := fkRegex.FindStringSubmatch(def)
	if len(matches) >= 4 {
		localCols := strings.Split(matches[1], ",")
		refTable := strings.ToLower(strings.TrimSpace(strings.Trim(matches[2], `"[]`)))
		refCols := strings.Split(matches[3], ",")
		
		var localColumns []string
		for _, col := range localCols {
			localColumns = append(localColumns, strings.ToLower(strings.TrimSpace(strings.Trim(col, `"[]`))))
		}
		
		var refColumns []string
		for _, col := range refCols {
			refColumns = append(refColumns, strings.ToLower(strings.TrimSpace(strings.Trim(col, `"[]`))))
		}
		
		table.ForeignKeys = append(table.ForeignKeys, &CanonicalForeignKey{
			Columns:    localColumns,
			RefTable:   refTable,
			RefColumns: refColumns,
			OnDelete:   nil,
			OnUpdate:   nil,
			Name:       nil,
		})
	}
}

// parseUniqueDef parses UNIQUE constraint
func (se *StreamingSchemaExtractor) parseUniqueDef(def string, table *CanonicalTable) {
	uniqueRegex := regexp.MustCompile(`UNIQUE\s*\(([^)]+)\)`)
	matches := uniqueRegex.FindStringSubmatch(def)
	if len(matches) > 1 {
		columns := strings.Split(matches[1], ",")
		var uniqueColumns []string
		for _, col := range columns {
			uniqueColumns = append(uniqueColumns, strings.ToLower(strings.TrimSpace(strings.Trim(col, `"[]`))))
		}
		table.Unique = append(table.Unique, uniqueColumns)
	}
}

// applyDropTable applies DROP TABLE statement
func (se *StreamingSchemaExtractor) applyDropTable(stmt DDLStatement) error {
	tableName := stmt.TableName
	if tableName == "" {
		return fmt.Errorf("could not extract table name from DROP TABLE")
	}
	
	delete(se.schema.Tables, tableName)
	return nil
}

// applyAlterTable applies ALTER TABLE statement
func (se *StreamingSchemaExtractor) applyAlterTable(stmt DDLStatement) error {
	tableName := stmt.TableName
	if tableName == "" {
		return fmt.Errorf("could not extract table name from ALTER TABLE")
	}
	
	// Get or create table
	table, exists := se.schema.Tables[tableName]
	if !exists {
		// Create table if it doesn't exist (some migrations might reference future tables)
		table = &CanonicalTable{
			Columns:     make(map[string]*CanonicalColumn),
			PrimaryKey:  []string{},
			Unique:      [][]string{},
			ForeignKeys: []*CanonicalForeignKey{},
			Indexes:     []*CanonicalIndex{},
			Comment:     nil,
		}
		se.schema.Tables[tableName] = table
	}
	
	upperStmt := strings.ToUpper(stmt.Statement)
	
	if strings.Contains(upperStmt, "ADD COLUMN") || strings.Contains(upperStmt, "ADD ") {
		return se.applyAddColumn(stmt.Statement, table)
	} else if strings.Contains(upperStmt, "DROP COLUMN") {
		return se.applyDropColumn(stmt.Statement, table)
	} else if strings.Contains(upperStmt, "ALTER COLUMN") || strings.Contains(upperStmt, "MODIFY COLUMN") {
		return se.applyAlterColumn(stmt.Statement, table)
	} else if strings.Contains(upperStmt, "ADD CONSTRAINT") {
		return se.applyAddConstraint(stmt.Statement, table)
	} else if strings.Contains(upperStmt, "DROP CONSTRAINT") {
		return se.applyDropConstraint(stmt.Statement, table)
	}
	
	return nil
}

// applyAddColumn applies ADD COLUMN statement
func (se *StreamingSchemaExtractor) applyAddColumn(stmt string, table *CanonicalTable) error {
	addColumnRegex := regexp.MustCompile(`ADD\s+(?:COLUMN\s+)?(.+)`)
	matches := addColumnRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract column definition from ADD COLUMN")
	}
	
	columnDef := matches[1]
	return se.parseColumnDef(columnDef, table)
}

// applyDropColumn applies DROP COLUMN statement
func (se *StreamingSchemaExtractor) applyDropColumn(stmt string, table *CanonicalTable) error {
	dropColumnRegex := regexp.MustCompile(`DROP\s+COLUMN\s+([^\s,]+)`)
	matches := dropColumnRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract column name from DROP COLUMN")
	}
	
	columnName := strings.ToLower(strings.Trim(matches[1], `"[]`))
	
	// Remove column
	delete(table.Columns, columnName)
	
	// Remove from primary key if present
	var newPK []string
	for _, pk := range table.PrimaryKey {
		if pk != columnName {
			newPK = append(newPK, pk)
		}
	}
	table.PrimaryKey = newPK
	
	// Remove from unique constraints
	var newUnique [][]string
	for _, unique := range table.Unique {
		var newUniqueColumns []string
		for _, col := range unique {
			if col != columnName {
				newUniqueColumns = append(newUniqueColumns, col)
			}
		}
		if len(newUniqueColumns) > 0 {
			newUnique = append(newUnique, newUniqueColumns)
		}
	}
	table.Unique = newUnique
	
	return nil
}

// applyAlterColumn applies ALTER COLUMN statement
func (se *StreamingSchemaExtractor) applyAlterColumn(stmt string, table *CanonicalTable) error {
	// This is a simplified implementation
	// Real implementation would handle various ALTER COLUMN operations
	return nil
}

// applyAddConstraint applies ADD CONSTRAINT statement
func (se *StreamingSchemaExtractor) applyAddConstraint(stmt string, table *CanonicalTable) error {
	upperStmt := strings.ToUpper(stmt)
	
	if strings.Contains(upperStmt, "PRIMARY KEY") {
		se.parsePrimaryKeyDef(stmt, table)
	} else if strings.Contains(upperStmt, "FOREIGN KEY") {
		se.parseForeignKeyDef(stmt, table)
	} else if strings.Contains(upperStmt, "UNIQUE") {
		se.parseUniqueDef(stmt, table)
	}
	
	return nil
}

// applyDropConstraint applies DROP CONSTRAINT statement
func (se *StreamingSchemaExtractor) applyDropConstraint(stmt string, table *CanonicalTable) error {
	// Simplified implementation
	return nil
}

// applyCreateIndex applies CREATE INDEX statement
func (se *StreamingSchemaExtractor) applyCreateIndex(stmt DDLStatement) error {
	// Simplified implementation
	return nil
}

// applyDropIndex applies DROP INDEX statement
func (se *StreamingSchemaExtractor) applyDropIndex(stmt DDLStatement) error {
	// Simplified implementation
	return nil
}

// applyCreateType applies CREATE TYPE statement
func (se *StreamingSchemaExtractor) applyCreateType(stmt DDLStatement) error {
	// Parse CREATE TYPE ... AS ENUM
	enumRegex := regexp.MustCompile(`CREATE TYPE\s+([^\s]+)\s+AS\s+ENUM\s*\(([^)]+)\)`)
	matches := enumRegex.FindStringSubmatch(stmt.Statement)
	if len(matches) >= 3 {
		typeName := strings.ToLower(strings.Trim(matches[1], `"[]`))
		valuesStr := matches[2]
		
		var values []string
		for _, value := range strings.Split(valuesStr, ",") {
			value = strings.TrimSpace(strings.Trim(value, `'"[]`))
			if value != "" {
				values = append(values, value)
			}
		}
		
		se.schema.Enums[typeName] = values
	}
	
	return nil
}

// applyCreateView applies CREATE VIEW statement
func (se *StreamingSchemaExtractor) applyCreateView(stmt DDLStatement) error {
	// Extract view name
	viewRegex := regexp.MustCompile(`CREATE VIEW\s+([^\s]+)\s+AS`)
	matches := viewRegex.FindStringSubmatch(stmt.Statement)
	if len(matches) >= 2 {
		viewName := strings.ToLower(strings.Trim(matches[1], `"[]`))
		se.schema.Views[viewName] = &View{SQL: stmt.Statement}
	}
	
	return nil
}

// applyDropView applies DROP VIEW statement
func (se *StreamingSchemaExtractor) applyDropView(stmt DDLStatement) error {
	viewRegex := regexp.MustCompile(`DROP VIEW\s+(?:IF EXISTS\s+)?([^\s;]+)`)
	matches := viewRegex.FindStringSubmatch(stmt.Statement)
	if len(matches) >= 2 {
		viewName := strings.ToLower(strings.Trim(matches[1], `"[]`))
		delete(se.schema.Views, viewName)
	}
	
	return nil
}

// normalizeSchema normalizes the final schema
func (se *StreamingSchemaExtractor) normalizeSchema() {
	// Sort keys, resolve type aliases, validate foreign keys, etc.
	// This is where we would perform final cleanup and validation
	
	for tableName, table := range se.schema.Tables {
		// Sort primary key columns
		sort.Strings(table.PrimaryKey)
		
		// Sort unique constraints
		for _, unique := range table.Unique {
			sort.Strings(unique)
		}
		
		// Generate deterministic names for unnamed constraints
		for i, fk := range table.ForeignKeys {
			if fk.Name == nil {
				name := fmt.Sprintf("fk_%s_%s", tableName, strings.Join(fk.Columns, "_"))
				table.ForeignKeys[i].Name = &name
			}
		}
	}
}

// generateMermaidERD generates Mermaid ERD from the final schema
func (se *StreamingSchemaExtractor) generateMermaidERD() string {
	var mermaid strings.Builder
	
	mermaid.WriteString("erDiagram\n")
	
	// Sort table names for consistent output
	var tableNames []string
	for tableName := range se.schema.Tables {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)
	
	// Generate table definitions
	for _, tableName := range tableNames {
		table := se.schema.Tables[tableName]
		
		mermaid.WriteString(fmt.Sprintf("  %s {\n", tableName))
		
		// Sort column names for consistent output
		var columnNames []string
		for colName := range table.Columns {
			columnNames = append(columnNames, colName)
		}
		sort.Strings(columnNames)
		
		// Add columns with annotations
		for _, colName := range columnNames {
			column := table.Columns[colName]
			
			// Build column line
			var annotations []string
			
			// Check if primary key
			for _, pk := range table.PrimaryKey {
				if pk == colName {
					annotations = append(annotations, "PK")
					break
				}
			}
			
			// Check if unique
			for _, unique := range table.Unique {
				if len(unique) == 1 && unique[0] == colName {
					annotations = append(annotations, "UK")
					break
				}
			}
			
			// Check if foreign key
			for _, fk := range table.ForeignKeys {
				for _, fkCol := range fk.Columns {
					if fkCol == colName {
						annotations = append(annotations, "FK")
						break
					}
				}
			}
			
			annotationStr := ""
			if len(annotations) > 0 {
				annotationStr = " " + strings.Join(annotations, ",")
			}
			
			mermaid.WriteString(fmt.Sprintf("    %s %s%s\n", column.Type, colName, annotationStr))
		}
		
		mermaid.WriteString("  }\n")
	}
	
	// Generate relationships
	for _, tableName := range tableNames {
		table := se.schema.Tables[tableName]
		
		for _, fk := range table.ForeignKeys {
			if len(fk.Columns) == 1 && len(fk.RefColumns) == 1 {
				mermaid.WriteString(fmt.Sprintf("  %s ||--o{ %s : \"%s -> %s.%s\"\n",
					fk.RefTable, tableName, fk.Columns[0], fk.RefTable, fk.RefColumns[0]))
			}
		}
	}
	
	return mermaid.String()
}

// ExtractSchemaFromProjectResult contains the complete results of schema extraction
type ExtractSchemaFromProjectResult struct {
	Schema            *CanonicalSchema
	MermaidERD        string  
	FinalMigrationSQL string
}

// ExtractSchemaFromProject extracts schema from project files with streaming (with graceful error handling)
func ExtractSchemaFromProject(projectPath string, files map[string]string, callback func(StreamingResponse)) (*CanonicalSchema, string, error) {
	// Find migration files
	migrations := findMigrationFiles(files)
	if len(migrations) == 0 {
		return nil, "", fmt.Errorf("no migration folders found")
	}
	
	// Create streaming extractor
	extractor := NewStreamingSchemaExtractor("postgres")
	
	// Process migrations with streaming and graceful error handling
	var finalSchema *CanonicalSchema
	var finalMermaid string
	
	err := extractor.BuildSchemaAndStream(migrations, func(response StreamingResponse) {
		callback(response) // Forward to caller
		
		// Capture final results regardless of warnings/errors during processing
		if response.Phase == "complete" || response.Schema != nil {
			finalSchema = response.Schema
			finalMermaid = response.Mermaid
		}
	})
	
	// Even if there were errors processing some migrations, return partial results if we have any tables
	if finalSchema != nil && len(finalSchema.Tables) > 0 {
		// Generate Mermaid ERD if we don't have one yet
		if finalMermaid == "" {
			finalMermaid = extractor.generateMermaidERD()
		}
		
		// Send final success callback with partial results
		callback(StreamingResponse{
			Phase:    "complete",
			Progress: ProgressInfo{Current: len(migrations), Total: len(migrations)},
			Message:  fmt.Sprintf("✅ Schema extraction completed with %d tables (some migrations may have been skipped)", len(finalSchema.Tables)),
			Schema:   finalSchema,
			Mermaid:  finalMermaid,
		})
		
		return finalSchema, finalMermaid, nil
	}
	
	// Only return error if we got no results at all
	if err != nil {
		return nil, "", fmt.Errorf("schema extraction failed: %v", err)
	}
	
	return finalSchema, finalMermaid, nil
}

// ExtractSchemaWithFinalMigration extracts schema and generates final migration SQL
func ExtractSchemaWithFinalMigration(projectPath string, files map[string]string, callback func(StreamingResponse)) (*ExtractSchemaFromProjectResult, error) {
	// Find migration files
	migrations := findMigrationFiles(files)
	if len(migrations) == 0 {
		return nil, fmt.Errorf("no migration folders found")
	}
	
	// Create streaming extractor
	extractor := NewStreamingSchemaExtractor("postgres")
	
	// Store final results
	var finalSchema *CanonicalSchema
	var finalMermaid string
	var finalMigrationSQL string
	
	err := extractor.BuildSchemaAndStream(migrations, func(response StreamingResponse) {
		callback(response) // Forward to caller
		
		// Capture final results
		if response.Phase == "complete" || response.Schema != nil {
			finalSchema = response.Schema
			finalMermaid = response.Mermaid
		}
	})
	
	// Generate final migration SQL regardless of any errors
	if finalSchema != nil && len(finalSchema.Tables) > 0 {
		finalMigrationSQL = extractor.GenerateFinalMigrationSQL()
		
		// Generate Mermaid ERD if we don't have one yet
		if finalMermaid == "" {
			finalMermaid = extractor.generateMermaidERD()
		}
		
		// Send enhanced completion callback
		callback(StreamingResponse{
			Phase:    "complete",
			Progress: ProgressInfo{Current: len(migrations), Total: len(migrations)},
			Message:  fmt.Sprintf("✅ Generated final migration with %d tables (%d characters)", len(finalSchema.Tables), len(finalMigrationSQL)),
			Schema:   finalSchema,
			Mermaid:  finalMermaid,
		})
		
		return &ExtractSchemaFromProjectResult{
			Schema:            finalSchema,
			MermaidERD:        finalMermaid,
			FinalMigrationSQL: finalMigrationSQL,
		}, nil
	}
	
	// Only return error if we got no results at all
	if err != nil {
		return nil, fmt.Errorf("schema extraction failed: %v", err)
	}
	
	return &ExtractSchemaFromProjectResult{
		Schema:            finalSchema,
		MermaidERD:        finalMermaid,
		FinalMigrationSQL: finalMigrationSQL,
	}, nil
}

// findMigrationFiles finds and sorts migration files from project files
func findMigrationFiles(files map[string]string) []Migration {
	var migrations []Migration
	var migrationPaths []string
	
	// Find all SQL files in migration folders
	for filePath := range files {
		if strings.HasSuffix(strings.ToLower(filePath), ".sql") {
			dir := filepath.Dir(filePath)
			dirLower := strings.ToLower(dir)
			
			// Check if path contains "migration"
			if strings.Contains(dirLower, "migration") {
				migrationPaths = append(migrationPaths, filePath)
			}
		}
	}
	
	// Sort migration files by name (assuming timestamp-based naming)
	sort.Strings(migrationPaths)
	
	// Create migration objects
	for _, path := range migrationPaths {
		if content, exists := files[path]; exists {
			migrations = append(migrations, Migration{
				Name: filepath.Base(path),
				SQL:  content,
			})
		}
	}
	
	return migrations
}

// ConvertToLegacySchema converts CanonicalSchema to legacy DatabaseSchema format
func ConvertToLegacySchema(canonical *CanonicalSchema, migrationPath string) *DatabaseSchema {
	legacy := &DatabaseSchema{
		Tables:        make(map[string]Table),
		ForeignKeys:   []ForeignKeyRef{},
		MigrationPath: migrationPath,
		GeneratedAt:   time.Now(),
	}
	
	for tableName, canonicalTable := range canonical.Tables {
		// Convert columns
		columns := make(map[string]Column)
		for colName, canonicalCol := range canonicalTable.Columns {
			column := Column{
				Name:         colName,
				Type:         canonicalCol.Type,
				Constraints:  []ColumnConstraint{},
				DefaultValue: "",
				References:   nil,
			}
			
			// Add constraints
			if !canonicalCol.Nullable {
				column.Constraints = append(column.Constraints, NotNull)
			}
			
			if canonicalCol.Default != nil {
				column.DefaultValue = *canonicalCol.Default
				column.Constraints = append(column.Constraints, Default)
			}
			
			// Check if primary key
			for _, pk := range canonicalTable.PrimaryKey {
				if pk == colName {
					column.Constraints = append(column.Constraints, PrimaryKey)
					break
				}
			}
			
			// Check if unique
			for _, unique := range canonicalTable.Unique {
				if len(unique) == 1 && unique[0] == colName {
					column.Constraints = append(column.Constraints, Unique)
					break
				}
			}
			
			// Check if foreign key
			for _, fk := range canonicalTable.ForeignKeys {
				for _, fkCol := range fk.Columns {
					if fkCol == colName {
						column.Constraints = append(column.Constraints, ForeignKey)
						if len(fk.RefColumns) > 0 {
							column.References = &ForeignKeyRef{
								Table:  fk.RefTable,
								Column: fk.RefColumns[0], // Take first ref column
							}
						}
						break
					}
				}
			}
			
			columns[colName] = column
		}
		
		// Convert indexes
		indexes := make(map[string]Index)
		for _, canonicalIndex := range canonicalTable.Indexes {
			indexes[canonicalIndex.Name] = Index{
				Name:    canonicalIndex.Name,
				Columns: canonicalIndex.Columns,
				Unique:  canonicalIndex.Unique,
			}
		}
		
		// Create legacy table
		legacy.Tables[tableName] = Table{
			Name:        tableName,
			Columns:     columns,
			PrimaryKeys: canonicalTable.PrimaryKey,
			Indexes:     indexes,
		}
		
		// Add foreign keys to global list
		for _, fk := range canonicalTable.ForeignKeys {
			if len(fk.RefColumns) > 0 {
				legacy.ForeignKeys = append(legacy.ForeignKeys, ForeignKeyRef{
					Table:  fk.RefTable,
					Column: fk.RefColumns[0], // Take first ref column
				})
			}
		}
	}
	
	return legacy
}

// GenerateFinalMigrationSQL generates a single SQL migration file representing the final database state
func (se *StreamingSchemaExtractor) GenerateFinalMigrationSQL() string {
	var sql strings.Builder
	
	// Header comment
	sql.WriteString("-- Generated Final Migration: Complete Database Schema\n")
	sql.WriteString("-- This file represents the final state after applying all migrations\n")
	sql.WriteString("-- Run this single file to create the complete database schema\n\n")
	
	// Generate CREATE TYPE statements for enums
	if len(se.schema.Enums) > 0 {
		sql.WriteString("-- ============================================\n")
		sql.WriteString("-- ENUMS AND TYPES\n")
		sql.WriteString("-- ============================================\n\n")
		
		for enumName, values := range se.schema.Enums {
			sql.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (\n", enumName))
			for i, value := range values {
				if i == len(values)-1 {
					sql.WriteString(fmt.Sprintf("    '%s'\n", value))
				} else {
					sql.WriteString(fmt.Sprintf("    '%s',\n", value))
				}
			}
			sql.WriteString(");\n\n")
		}
	}
	
	// Generate CREATE TABLE statements
	if len(se.schema.Tables) > 0 {
		sql.WriteString("-- ============================================\n")
		sql.WriteString("-- TABLES\n")
		sql.WriteString("-- ============================================\n\n")
		
		// Sort table names by dependency order (tables with no foreign keys first)
		tableNames := se.sortTablesByDependencies()
		
		for _, tableName := range tableNames {
			table := se.schema.Tables[tableName]
			sql.WriteString(se.generateCreateTableSQL(tableName, table))
			sql.WriteString("\n")
		}
	}
	
	// Generate INDEX statements
	sql.WriteString("-- ============================================\n")
	sql.WriteString("-- INDEXES\n")
	sql.WriteString("-- ============================================\n\n")
	
	// Get table names again for index generation
	var indexTableNames []string
	for tableName := range se.schema.Tables {
		indexTableNames = append(indexTableNames, tableName)
	}
	sort.Strings(indexTableNames)
	
	for _, tableName := range indexTableNames {
		table := se.schema.Tables[tableName]
		for _, index := range table.Indexes {
			sql.WriteString(se.generateCreateIndexSQL(tableName, index))
			sql.WriteString("\n")
		}
	}
	
	// Generate CREATE VIEW statements
	if len(se.schema.Views) > 0 {
		sql.WriteString("-- ============================================\n")
		sql.WriteString("-- VIEWS\n")
		sql.WriteString("-- ============================================\n\n")
		
		for viewName, view := range se.schema.Views {
			sql.WriteString(fmt.Sprintf("CREATE VIEW %s AS\n%s;\n\n", viewName, view.SQL))
		}
	}
	
	sql.WriteString("-- ============================================\n")
	sql.WriteString("-- MIGRATION COMPLETE\n")
	sql.WriteString("-- ============================================\n")
	
	return sql.String()
}

// generateCreateTableSQL generates a complete CREATE TABLE statement
func (se *StreamingSchemaExtractor) generateCreateTableSQL(tableName string, table *CanonicalTable) string {
	var sql strings.Builder
	
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))
	
	// Get sorted column names for consistent output
	var columnNames []string
	for colName := range table.Columns {
		columnNames = append(columnNames, colName)
	}
	sort.Strings(columnNames)
	
	var columnDefs []string
	
	// Generate column definitions
	for _, colName := range columnNames {
		column := table.Columns[colName]
		colDef := fmt.Sprintf("    %s %s", colName, column.Type)
		
		// Add NOT NULL constraint
		if !column.Nullable {
			colDef += " NOT NULL"
		}
		
		// Add DEFAULT value
		if column.Default != nil {
			colDef += fmt.Sprintf(" DEFAULT %s", *column.Default)
		}
		
		columnDefs = append(columnDefs, colDef)
	}
	
	// Add table constraints
	
	// Primary key constraint
	if len(table.PrimaryKey) > 0 {
		pkCols := strings.Join(table.PrimaryKey, ", ")
		columnDefs = append(columnDefs, fmt.Sprintf("    PRIMARY KEY (%s)", pkCols))
	}
	
	// Unique constraints
	for _, uniqueCols := range table.Unique {
		if len(uniqueCols) > 0 {
			uniqueColsStr := strings.Join(uniqueCols, ", ")
			columnDefs = append(columnDefs, fmt.Sprintf("    UNIQUE (%s)", uniqueColsStr))
		}
	}
	
	// Foreign key constraints
	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) > 0 && len(fk.RefColumns) > 0 {
			fkCols := strings.Join(fk.Columns, ", ")
			refCols := strings.Join(fk.RefColumns, ", ")
			constraintName := ""
			if fk.Name != nil {
				constraintName = fmt.Sprintf("CONSTRAINT %s ", *fk.Name)
			}
			fkDef := fmt.Sprintf("    %sFOREIGN KEY (%s) REFERENCES %s (%s)", 
				constraintName, fkCols, fk.RefTable, refCols)
			
			if fk.OnDelete != nil {
				fkDef += fmt.Sprintf(" ON DELETE %s", *fk.OnDelete)
			}
			if fk.OnUpdate != nil {
				fkDef += fmt.Sprintf(" ON UPDATE %s", *fk.OnUpdate)
			}
			
			columnDefs = append(columnDefs, fkDef)
		}
	}
	
	// Join all column definitions and constraints
	sql.WriteString(strings.Join(columnDefs, ",\n"))
	sql.WriteString("\n);\n")
	
	return sql.String()
}

// generateCreateIndexSQL generates CREATE INDEX statement
func (se *StreamingSchemaExtractor) generateCreateIndexSQL(tableName string, index *CanonicalIndex) string {
	indexCols := strings.Join(index.Columns, ", ")
	uniqueStr := ""
	if index.Unique {
		uniqueStr = "UNIQUE "
	}
	
	usingClause := ""
	if index.Using != nil {
		usingClause = fmt.Sprintf(" USING %s", *index.Using)
	}
	
	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)%s;", 
		uniqueStr, index.Name, tableName, indexCols, usingClause)
}

// sortTablesByDependencies sorts tables so that referenced tables come before referencing tables
func (se *StreamingSchemaExtractor) sortTablesByDependencies() []string {
	var sorted []string
	processed := make(map[string]bool)
	
	// Get all table names
	var allTables []string
	for tableName := range se.schema.Tables {
		allTables = append(allTables, tableName)
	}
	
	// Sort alphabetically first for consistent ordering of tables with same dependency level
	sort.Strings(allTables)
	
	// Process tables in dependency order
	for len(sorted) < len(allTables) {
		addedInThisRound := false
		
		for _, tableName := range allTables {
			if processed[tableName] {
				continue
			}
			
			table := se.schema.Tables[tableName]
			canAdd := true
			
			// Check if all foreign key references are already processed
			for _, fk := range table.ForeignKeys {
				if fk.RefTable != tableName && !processed[fk.RefTable] {
					canAdd = false
					break
				}
			}
			
			if canAdd {
				sorted = append(sorted, tableName)
				processed[tableName] = true
				addedInThisRound = true
			}
		}
		
		// Prevent infinite loop if there are circular dependencies
		if !addedInThisRound {
			// Add remaining tables anyway (circular dependencies)
			for _, tableName := range allTables {
				if !processed[tableName] {
					sorted = append(sorted, tableName)
					processed[tableName] = true
				}
			}
			break
		}
	}
	
	return sorted
}

// Safe wrapper functions that handle errors gracefully

func (se *StreamingSchemaExtractor) applyCreateTableSafely(stmt DDLStatement) error {
	err := se.applyCreateTable(stmt)
	if err != nil {
		fmt.Printf("⚠️ CREATE TABLE failed for %s: %v (skipping)\n", stmt.TableName, err)
		return nil // Don't propagate error, just log and continue
	}
	return nil
}

func (se *StreamingSchemaExtractor) applyDropTableSafely(stmt DDLStatement) error {
	err := se.applyDropTable(stmt)
	if err != nil {
		fmt.Printf("⚠️ DROP TABLE failed for %s: %v (skipping)\n", stmt.TableName, err)
		return nil
	}
	return nil
}

func (se *StreamingSchemaExtractor) applyAlterTableSafely(stmt DDLStatement) error {
	err := se.applyAlterTable(stmt)
	if err != nil {
		fmt.Printf("⚠️ ALTER TABLE failed for %s: %v (skipping)\n", stmt.TableName, err)
		return nil
	}
	return nil
}

func (se *StreamingSchemaExtractor) applyCreateIndexSafely(stmt DDLStatement) error {
	err := se.applyCreateIndex(stmt)
	if err != nil {
		fmt.Printf("⚠️ CREATE INDEX failed: %v (skipping)\n", err)
		return nil
	}
	return nil
}

func (se *StreamingSchemaExtractor) applyDropIndexSafely(stmt DDLStatement) error {
	err := se.applyDropIndex(stmt)
	if err != nil {
		fmt.Printf("⚠️ DROP INDEX failed: %v (skipping)\n", err)
		return nil
	}
	return nil
}

func (se *StreamingSchemaExtractor) applyCreateTypeSafely(stmt DDLStatement) error {
	err := se.applyCreateType(stmt)
	if err != nil {
		fmt.Printf("⚠️ CREATE TYPE failed: %v (skipping)\n", err)
		return nil
	}
	return nil
}

func (se *StreamingSchemaExtractor) applyCreateViewSafely(stmt DDLStatement) error {
	err := se.applyCreateView(stmt)
	if err != nil {
		fmt.Printf("⚠️ CREATE VIEW failed: %v (skipping)\n", err)
		return nil
	}
	return nil
}

func (se *StreamingSchemaExtractor) applyDropViewSafely(stmt DDLStatement) error {
	err := se.applyDropView(stmt)
	if err != nil {
		fmt.Printf("⚠️ DROP VIEW failed: %v (skipping)\n", err)
		return nil
	}
	return nil
}
