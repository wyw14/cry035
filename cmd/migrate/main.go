package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(2)
	}
	pattern := filepath.Join("migrations", "*.up.sql")
	if len(os.Args) > 1 && os.Args[1] == "down" {
		pattern = filepath.Join("migrations", "*.down.sql")
	}
	paths, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	sort.Strings(paths)
	if strings.HasSuffix(pattern, ".down.sql") {
		for left, right := 0, len(paths)-1; left < right; left, right = left+1, right-1 {
			paths[left], paths[right] = paths[right], paths[left]
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			panic(fmt.Errorf("apply %s: %w", path, err))
		}
		fmt.Println("applied", path)
	}
}
