package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers PostgreSQL instance.
type PostgresContainer struct {
	Container *postgres.PostgresContainer
	DSN       string
	Pool      *pgxpool.Pool
}

// NewPostgresContainer starts a PostgreSQL container for testing.
// The caller should defer container.Cleanup(t).
func NewPostgresContainer(ctx context.Context, t *testing.T) *PostgresContainer {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to create pgxpool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping postgres: %v", err)
	}

	return &PostgresContainer{
		Container: pgContainer,
		DSN:       dsn,
		Pool:      pool,
	}
}

// Cleanup terminates the container.
func (pc *PostgresContainer) Cleanup(t *testing.T) {
	t.Helper()

	if pc.Pool != nil {
		pc.Pool.Close()
	}

	if pc.Container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := pc.Container.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate postgres container: %v", err)
		}
	}
}

// RunMigrations runs migrations from the given directory against the test database.
// Migration files are expected to be .sql files and are executed in lexicographic order.
func (pc *PostgresContainer) RunMigrations(t *testing.T, migrationsDir string) {
	t.Helper()

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations directory %s: %v", migrationsDir, err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, file := range sqlFiles {
		path := filepath.Join(migrationsDir, file)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read migration file %s: %v", path, err)
		}

		if _, err := pc.Pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("failed to execute migration %s: %v", file, err)
		}

		fmt.Printf("applied migration: %s\n", file)
	}
}
