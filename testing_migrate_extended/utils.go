package testing_postgres_extended

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	migrate "github.com/abramad-labs/histomigrate"
	"github.com/abramad-labs/histomigrate/database/postgres"
	_ "github.com/abramad-labs/histomigrate/source/file"
	"github.com/dhui/dktest"
	_ "github.com/lib/pq"
)

const (
	postgresImage   = "postgres:17-alpine"
	defaultUser     = "postgres"
	defaultPassword = "password"
	defaultDB       = "postgres"
)

func pgConnectionString(password, host, port string, options ...string) string {
	options = append(options, "sslmode=disable")
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?%s",
		defaultUser,
		password,
		host,
		port,
		defaultDB,
		strings.Join(options, "&"),
	)
}

func defaultEnvVars() map[string]string {
	return map[string]string{
		"POSTGRES_USER":     defaultUser,
		"POSTGRES_PASSWORD": defaultPassword,
		"POSTGRES_DB":       defaultDB,
	}
}

type ReadyFunc func(ctx context.Context, c dktest.ContainerInfo) bool

func containerReady(password string) ReadyFunc {
	return func(ctx context.Context, c dktest.ContainerInfo) bool {
		ip, port, err := c.FirstPort()
		if err != nil {
			return false
		}
		db, err := sql.Open("postgres", pgConnectionString(password, ip, port))
		if err != nil {
			return false
		}
		defer db.Close()
		return db.Ping() == nil
	}
}

func runPostgresContainer(t *testing.T, testFunc func(t *testing.T, c dktest.ContainerInfo, envVars map[string]string)) {
	envVars := defaultEnvVars()
	opts := dktest.Options{
		Env:          envVars,
		ReadyTimeout: time.Minute,
		PortRequired: true,
		ReadyFunc:    containerReady(envVars["POSTGRES_PASSWORD"]),
	}
	dktest.Run(t, postgresImage, opts, func(t *testing.T, c dktest.ContainerInfo) {
		testFunc(t, c, envVars)
	})
}

func setupContainerWithMigrator(t *testing.T, migrationsDir string, testFunc func(*testing.T, *sql.DB, *migrate.Migrate)) {
	t.Helper()
	runPostgresContainer(t, func(t *testing.T, c dktest.ContainerInfo, envVars map[string]string) {
		ip, port, err := c.FirstPort()
		if err != nil {
			t.Fatalf("failed to get container port: %v", err)
		}

		dataSource := pgConnectionString(envVars["POSTGRES_PASSWORD"], ip, port)
		db, migrator, err := newMigrator(envVars["POSTGRES_DB"], dataSource, migrationsDir)
		if err != nil {
			t.Fatalf("failed to create migrator: %v", err)
		}

		testFunc(t, db, migrator)
	})
}

func newMigrator(dbName, dataSourceName, migrationsDir string) (*sql.DB, *migrate.Migrate, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}
	migrationsPath := fmt.Sprintf("file://%s", filepath.ToSlash(filepath.Clean(migrationsDir)))
	migrator, err := migrate.NewWithDatabaseInstance(migrationsPath, dbName, driver)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	return db, migrator, nil
}

func makeOrAppendMigrationFile(filePath, correctedContent string) error {
	return os.WriteFile(filePath, []byte(correctedContent), 0644)
}

func getMigrationFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	return string(content), err
}

func deleteFile(filePath string) error {
	return os.Remove(filePath)
}
