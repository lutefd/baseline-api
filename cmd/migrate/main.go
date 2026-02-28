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

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			log.Fatalf("apply migration %s: %v", file, err)
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
