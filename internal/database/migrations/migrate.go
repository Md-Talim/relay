package migrations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, db *pgxpool.Pool, migrationsDir string) error {
	migrationFiles, err := listSQLMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}

	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// Session-level advisory lock
	lockName := "relay:migrations"

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(hashtextextended($1, 0))", lockName); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		if _, err := conn.Exec(context.Background(), "SELECT pg_advisory_unlock(hashtextextended($1, 0))", lockName); err != nil {
			// log error here
		}
	}()

	// Ensure migrations tracking table exists
	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Read already-applied versions from DB
	applied, err := loadAppliedMigrations(ctx, conn)
	if err != nil {
		return err
	}

	// Strict validation: DB versions must all exists on disk
	if err = validateAppliedVersionsExistsOnDisk(applied, migrationFiles); err != nil {
		return err
	}

	// Execute only pending migrations, each in its own transaction
	for _, filename := range migrationFiles {
		if _, ok := applied[filename]; ok {
			continue
		}

		fullPath := filepath.Join(migrationsDir, filename)
		sqlBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", filename, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, filename); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}
	}

	return nil
}

func listSQLMigrations(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]string) // normalized -> original
	var out []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}

		normalized := strings.ToLower(name)
		if prev, ok := seen[normalized]; ok {
			return nil, fmt.Errorf("duplicate migration filename detected (case-insensitive): %q and %q in %s", prev, name, dir)
		}

		seen[normalized] = name
		out = append(out, name)
	}

	sort.Strings(out)
	return out, nil
}

func loadAppliedMigrations(ctx context.Context, conn *pgxpool.Conn) (map[string]struct{}, error) {
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read applied versions: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct{})

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied version: %w", err)
		}
		applied[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied versions: %w", err)
	}
	return applied, nil
}

func validateAppliedVersionsExistsOnDisk(appliedMigrations map[string]struct{}, migrationFiles []string) error {
	onDisk := make(map[string]struct{}, len(migrationFiles))
	for _, f := range migrationFiles {
		onDisk[f] = struct{}{}
	}

	var missing []string
	for migration := range appliedMigrations {
		if _, ok := onDisk[migration]; !ok {
			missing = append(missing, migration)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf(
			`migration history mismatch: applied version(s) missing on disk: %s.
			Migration filenames are immutable version IDs; do not rename or delete applied migration files.`,
			strings.Join(missing, ", "),
		)
	}

	return nil
}
