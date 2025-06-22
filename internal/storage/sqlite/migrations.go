package sqlite

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

func (db *DB) Migrate() error {
	ctx := context.Background()
	
	if err := db.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	migrations, err := db.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	currentVersion, err := db.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if err := db.runMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

func (db *DB) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL
		)
	`
	_, err := db.conn.ExecContext(ctx, query)
	return err
}

func (db *DB) getCurrentVersion(ctx context.Context) (int, error) {
	var version int
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`
	err := db.conn.QueryRowContext(ctx, query).Scan(&version)
	return version, err
}

func (db *DB) runMigration(ctx context.Context, migration Migration) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
		return fmt.Errorf("failed to execute up migration: %w", err)
	}

	query := `INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query, migration.Version, migration.Description, time.Now()); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

func (db *DB) loadMigrations() ([]Migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}

		migration, err := parseMigration(entry.Name(), string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func parseMigration(filename, content string) (Migration, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return Migration{}, fmt.Errorf("invalid migration filename: %s", filename)
	}

	var version int
	if _, err := fmt.Sscanf(parts[0], "%03d", &version); err != nil {
		return Migration{}, fmt.Errorf("invalid version in filename: %s", filename)
	}

	description := strings.TrimSuffix(parts[1], ".sql")
	description = strings.ReplaceAll(description, "_", " ")

	sections := strings.Split(content, "-- DOWN")
	if len(sections) != 2 {
		return Migration{}, fmt.Errorf("migration must contain -- DOWN separator")
	}

	up := strings.TrimSpace(strings.TrimPrefix(sections[0], "-- UP"))
	down := strings.TrimSpace(sections[1])

	return Migration{
		Version:     version,
		Description: description,
		Up:          up,
		Down:        down,
	}, nil
}