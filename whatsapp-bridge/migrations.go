package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// MigrationManager handles database migrations
type MigrationManager struct {
	db *sql.DB
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{db: db}
}

// InitMigrationTable creates the migrations table if it doesn't exist
func (mm *MigrationManager) InitMigrationTable() error {
	_, err := mm.db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// GetAppliedMigrations returns list of applied migration versions
func (mm *MigrationManager) GetAppliedMigrations() ([]int, error) {
	rows, err := mm.db.Query("SELECT version FROM migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

// LoadMigrations loads all migration files from the migrations directory
func (mm *MigrationManager) LoadMigrations() ([]Migration, error) {
	migrationsDir := "migrations"
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, file := range files {
		filename := filepath.Base(file)
		
		// Extract version from filename (e.g., "001_add_starred_column.sql" -> 1)
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) < 2 {
			continue // Skip files that don't match the pattern
		}
		
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue // Skip files with invalid version numbers
		}

		// Read migration file content
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Extract name from filename (remove version and extension)
		name := strings.TrimSuffix(parts[1], ".sql")

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// ApplyMigration applies a single migration
func (mm *MigrationManager) ApplyMigration(migration Migration) error {
	// Execute the migration SQL
	_, err := mm.db.Exec(migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	// Record the migration as applied
	_, err = mm.db.Exec(
		"INSERT INTO migrations (version, name) VALUES (?, ?)",
		migration.Version, migration.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
	}

	fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Name)
	return nil
}

// RunMigrations runs all pending migrations
func (mm *MigrationManager) RunMigrations() error {
	// Initialize migration table
	if err := mm.InitMigrationTable(); err != nil {
		return fmt.Errorf("failed to initialize migration table: %w", err)
	}

	// Get applied migrations
	appliedVersions, err := mm.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create a map for quick lookup
	appliedMap := make(map[int]bool)
	for _, version := range appliedVersions {
		appliedMap[version] = true
	}

	// Load all migrations
	migrations, err := mm.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Apply pending migrations
	var appliedCount int
	for _, migration := range migrations {
		if !appliedMap[migration.Version] {
			if err := mm.ApplyMigration(migration); err != nil {
				return err
			}
			appliedCount++
		}
	}

	if appliedCount == 0 {
		fmt.Println("No pending migrations to apply")
	} else {
		fmt.Printf("Applied %d migrations successfully\n", appliedCount)
	}

	return nil
}