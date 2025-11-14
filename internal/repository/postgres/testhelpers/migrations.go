package testhelpers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ApplyMigrations applies all .up.sql migration files from the specified directory
func ApplyMigrations(db *sql.DB, migrationsPath string) error {
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Filter only .up.sql files and sort them
	var upFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	// Apply migrations
	for _, file := range upFiles {
		content, err := os.ReadFile(filepath.Join(migrationsPath, file))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
		fmt.Printf("Applied migration: %s\n", file)
	}

	return nil
}
