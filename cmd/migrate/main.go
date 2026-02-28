package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://baseline:baseline@localhost:5432/baseline?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	files, err := listUpMigrations("migrations")
	if err != nil {
		log.Fatalf("list migrations: %v", err)
	}
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		log.Fatalf("ensure migrations table: %v", err)
	}

	for _, file := range files {
		applied, err := isApplied(ctx, pool, file)
		if err != nil {
			log.Fatalf("check migration %s: %v", file, err)
		}
		if applied {
			fmt.Printf("skipped %s (already applied)\n", file)
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("read migration %s: %v", file, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Fatalf("begin tx for %s: %v", file, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("apply migration %s: %v", file, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO schema_migrations (filename)
			VALUES ($1)
		`, file); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("record migration %s: %v", file, err)
		}
		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("commit migration %s: %v", file, err)
		}
		fmt.Printf("applied %s\n", file)
	}
}

func listUpMigrations(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".up.sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`)
	return err
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, filename string) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM schema_migrations WHERE filename = $1
		)
	`, filename).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
