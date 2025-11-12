package learning

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// Store manages the SQLite database for adaptive learning
type Store struct {
	db     *sql.DB
	dbPath string
}

// NewStore creates a new Store instance and initializes the database
func NewStore(dbPath string) (*Store, error) {
	// Handle in-memory database
	if dbPath == ":memory:" {
		return openAndInitStore(dbPath)
	}

	// Ensure parent directory exists for file-based databases
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	return openAndInitStore(dbPath)
}

// openAndInitStore opens the database connection and initializes schema
func openAndInitStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	store := &Store{
		db:     db,
		dbPath: dbPath,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// initSchema executes the embedded SQL schema
func (s *Store) initSchema() error {
	_, err := s.db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}
	return nil
}

// getSchemaVersion retrieves the current schema version
func (s *Store) getSchemaVersion() (int, error) {
	var version int
	err := s.db.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("query schema version: %w", err)
	}
	return version, nil
}

// tableExists checks if a table exists in the database
func (s *Store) tableExists(tableName string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	err := s.db.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check table existence: %w", err)
	}
	return count > 0, nil
}

// indexExists checks if an index exists in the database
func (s *Store) indexExists(indexName string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`
	err := s.db.QueryRow(query, indexName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check index existence: %w", err)
	}
	return count > 0, nil
}
