package testing_postgres_extended

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	migrate "github.com/abramad-labs/histomigrate"
	"github.com/dhui/dktest"
	"github.com/stretchr/testify/assert"
)

const (
	healthyMigrationSamplesDir       = "./mock_migrations/healthy_samples/"
	corruptedMigrationSamplesDir     = "./mock_migrations/corrupted_samples/"
	outOfOrderMigrationSamplesDir    = "./mock_migrations/out_of_order_samples/"
	expectedCorruptedMigrationTS     = 20250101000135
	expectedMigrationFileContentUp   = "ALTER TABLE orders ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();"
	expectedMigrationFileContentDown = "ALTER TABLE orders DROP COLUMN created_at;"
)

// -------------------------------------------------------------------------------------------------------------------
// TestHealthyMigrations is a test suite for validating standard migration operations.
func TestHealthyMigrations(t *testing.T) {
	t.Parallel()

	t.Run("TestUpMigration", func(t *testing.T) {
		// Test Scenario:
		// 1. Run the `Up` command to apply all migrations.
		// 2. Verify that no error is returned.
		// 3. Check if all expected tables ('orders', 'users') have been created.

		setupContainerWithMigrator(
			t,
			healthyMigrationSamplesDir,
			func(t *testing.T, db *sql.DB, migrator *migrate.Migrate) {
				assert.NoError(t, migrator.Up(), "Up() should not return an error")

				expectedTables := []string{"orders", "users"}
				for _, tableName := range expectedTables {
					exists, err := tableExists(db, tableName)
					assert.NoError(t, err, "tableExists() should not return an error")
					assert.True(t, exists, "Expected table '%s' to exist after up migration", tableName)
				}

				appliedVersions, err := getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions := []int{20250101000130, 20250101000135, 20250101000140}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")
			})
	})

	t.Run("TestStepOneMigration", func(t *testing.T) {
		// Test Scenario:
		// 1. Run the `Steps` command with a count of 1.
		// 2. Verify that the first migration ('orders' table) is applied.
		// 3. Verify that the second migration ('users' table) is NOT applied.

		setupContainerWithMigrator(
			t,
			healthyMigrationSamplesDir,
			func(t *testing.T, db *sql.DB, migrator *migrate.Migrate) {
				err := migrator.Steps(1)
				assert.NoError(t, err, "Steps(1) should not return an error")

				appliedVersions, err := getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions := []int{20250101000130}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err := tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist after 1 step")

				usersTableExists, err := tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table not to exist after 1 step")
			})
	})

	t.Run("TestStepOneByOneMigration", func(t *testing.T) {
		setupContainerWithMigrator(
			t,
			healthyMigrationSamplesDir,
			func(t *testing.T, db *sql.DB, migrator *migrate.Migrate) {
				appliedVersions, err := getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions := []int{}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err := tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, ordersTableExists, "Expected 'orders' table to exist after 1 step")

				usersTableExists, err := tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table to exist after 1 step")

				err = migrator.Steps(1)
				assert.NoError(t, err, "Steps(1) should not return an error")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{20250101000130}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err = tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist now")

				accountRefColumnExists, err := columnExists(db, "orders", "account_ref")
				assert.NoError(t, err, "accountRefColumnExists() should not return an error")
				assert.True(t, accountRefColumnExists, "Expected 'orders' table to have 'account_ref' column now")

				usersTableExists, err = tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table to not exist now")

				err = migrator.Steps(1)
				assert.NoError(t, err, "Steps(1) should not return an error")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{20250101000130, 20250101000135}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				accountRefColumnExists, err = columnExists(db, "orders", "account_ref")
				assert.NoError(t, err, "accountRefColumnExists() should not return an error")
				assert.False(t, accountRefColumnExists, "Expected 'orders' table to not have 'account_ref' column now")

				usersTableExists, err = tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table not to exist after 1 step")

				err = migrator.Steps(1)
				assert.NoError(t, err, "Steps(1) should not return an error")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{20250101000130, 20250101000135, 20250101000140}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				accountRefColumnExists, err = columnExists(db, "orders", "account_ref")
				assert.NoError(t, err, "accountRefColumnExists() should not return an error")
				assert.False(t, accountRefColumnExists, "Expected 'orders' table to not have 'account_ref' now")

				usersTableExists, err = tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, usersTableExists, "Expected 'users' table to exist now")
			})
	})

	t.Run("TestDownMigration", func(t *testing.T) {
		// Test Scenario:
		// 1. First, run all `Up` migrations.
		// 2. Then, run the `Down` command to roll back all migrations.
		// 3. Verify that all expected tables are dropped.

		setupContainerWithMigrator(
			t,
			healthyMigrationSamplesDir,
			func(t *testing.T, db *sql.DB, migrator *migrate.Migrate) {
				appliedVersions, err := getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions := []int{}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err := tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, ordersTableExists, "Expected 'orders' table to not exist now")

				usersTableExists, err := tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table to not exist now")

				assert.NoError(t, migrator.Up(), "Up() should not return an error")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{20250101000130, 20250101000135, 20250101000140}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err = tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist now")

				accountRefColExists, err := columnExists(db, "orders", "account_ref")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, accountRefColExists, "Expected 'orders' table 'account_ref' column to not exist now")

				usersTableExists, err = tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, usersTableExists, "Expected 'users' table to exist now")

				assert.NoError(t, migrator.Down(), "Down() should not return an error")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err = tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, ordersTableExists, "Expected 'orders' table to not exist now")

				usersTableExists, err = tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table to not exist now")
			})
	})

	t.Run("TestDownOneByOneMigration", func(t *testing.T) {
		// Test Scenario:
		// 1. Run all `Up` migrations.
		// 2. Run the `Steps` command with a count of -1.
		// 3. Verify that the last applied migration ('users' table) is rolled back.
		// 4. Verify that the first applied migration ('orders' table) remains.

		setupContainerWithMigrator(
			t,
			healthyMigrationSamplesDir,
			func(t *testing.T, db *sql.DB, migrator *migrate.Migrate) {
				assert.NoError(t, migrator.Up(), "Up() should not return an error")

				appliedVersions, err := getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions := []int{20250101000130, 20250101000135, 20250101000140}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err := tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist now")

				accountRefColExists, err := columnExists(db, "orders", "account_ref")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, accountRefColExists, "Expected 'orders' table 'account_ref' column to not exist now")

				usersTableExists, err := tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, usersTableExists, "Expected 'users' table to exist now")

				assert.NoError(t, migrator.Steps(-1), "Steps(-1) should not return an error")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{20250101000130, 20250101000135}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err = tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist now")

				accountRefColExists, err = columnExists(db, "orders", "account_ref")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, accountRefColExists, "Expected 'orders' table 'account_ref' column to not exist now")

				usersTableExists, err = tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table not to exist now")

				assert.NoError(t, migrator.Steps(-1), "Steps(-1) should not return an error")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{20250101000130}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err = tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist now")

				accountRefColExists, err = columnExists(db, "orders", "account_ref")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, accountRefColExists, "Expected 'orders' table 'account_ref' column to exist now")

				usersTableExists, err = tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.False(t, usersTableExists, "Expected 'users' table not to exist now")
			})
	})

	t.Run("TestReRunMigrationsIdempotent", func(t *testing.T) {
		// Test Scenario:
		// 1. Run the `Up` command to apply all migrations.
		// 2. Run the `Up` command again.
		// 3. Verify that the second run returns a specific `ErrNoChange` error or no error, indicating no new migrations were applied.

		setupContainerWithMigrator(
			t,
			healthyMigrationSamplesDir,
			func(t *testing.T, db *sql.DB, migrator *migrate.Migrate) {
				assert.NoError(t, migrator.Up(), "First Up() run should not return an error")

				err := migrator.Up()
				assert.True(
					t,
					err != nil && errors.Is(err, migrate.ErrNoChange),
					"Second Up() run should be idempotent",
				)
			})
	})
}

// -------------------------------------------------------------------------------------------------------------------
// TestCorruptedMigrations is a test suite for handling corrupted migration states.
func TestCorruptedMigrations(t *testing.T) {
	// Test Scenario:
	// 1. Attempt to run `Up` migrations with a corrupted file.
	// 2. Assert that an error is returned and the database is marked as dirty.
	// 3. Verify that the partially applied migration's table exists, but a subsequent column doesn't.
	// 4. Confirm that further migration commands fail with `ErrDirty`.
	t.Run("TestCorruptedMigration", func(t *testing.T) {
		setupContainerWithMigrator(t, corruptedMigrationSamplesDir, func(t *testing.T, db *sql.DB, migrator *migrate.Migrate) {
			err := migrator.Up()
			assert.Error(t, err, "Up() should fail with a corrupted migration")

			isDatabaseDirty, err := isDirty(db)
			assert.NoError(t, err, "isDirty() should not return an error")
			assert.True(t, isDatabaseDirty, "Database should be dirty after a failed migration")

			isMigrVersionDirty, err := isMigratedVersionDirty(db, 20250101000135)
			assert.NoError(t, err, "isDirty() should not return an error")
			assert.True(t, isMigrVersionDirty, "Database should be dirty after a failed migration")

			ordersTableExists, err := tableExists(db, "orders")
			assert.NoError(t, err, "tableExists() should not return an error")
			assert.True(t, ordersTableExists, "Expected 'orders' table to exist from the partially applied migration")

			createdAtColumnExists, err := columnExists(db, "orders", "created_at")
			assert.NoError(t, err, "columnExists() should not return an error")
			assert.False(t, createdAtColumnExists, "Expected 'created_at' column not to exist due to the corrupted migration")

			usersTableExists, err := tableExists(db, "users")
			assert.NoError(t, err, "tableExists() should not return an error")
			assert.False(t, usersTableExists, "Expected 'users' table to not exist from the partially applied migration")

			assert.IsType(t, migrate.ErrDirty{}, migrator.Up(), "Subsequent Up() should fail with ErrDirty")
			assert.IsType(t, migrate.ErrDirty{}, migrator.Down(), "Down() should fail with ErrDirty")
			assert.IsType(t, migrate.ErrDirty{}, migrator.Steps(1), "Steps(1) should fail with ErrDirty")
			assert.IsType(t, migrate.ErrDirty{}, migrator.Steps(-1), "Steps(-1) should fail with ErrDirty")
		})
	})

	t.Run("ForceAndFixCorruptedThenUpMigrations", func(t *testing.T) {
		// Test Scenario:
		// 1. Run `Up` migrations with a corrupted file, causing a failure and dirty state.
		// 2. Use the `Force` command to manually mark the failed migration as applied.
		// 3. Fix the corrupted migration file on the filesystem.
		// 4. Re-run `Up` and verify all remaining migrations are successfully applied.
		runPostgresContainer(
			t,
			func(t *testing.T, c dktest.ContainerInfo, envVars map[string]string) {
				ip, port, err := c.FirstPort()
				assert.NoError(t, err, "FirstPort() should not return an error")
				dataSourceName := pgConnectionString(envVars["POSTGRES_PASSWORD"], ip, port)

				db, migrator, err := newMigrator(envVars["POSTGRES_DB"], dataSourceName, corruptedMigrationSamplesDir)
				assert.NoError(t, err, "newMigrator() should not return an error")

				assert.Error(t, migrator.Up(), "Up() should fail with a corrupted migration")

				isDatabaseDirty, err := isDirty(db)
				assert.NoError(t, err, "isDirty() should not return an error")
				assert.True(t, isDatabaseDirty, "Database should be dirty after failed migration")

				assert.IsType(t, migrate.ErrDirty{}, migrator.Up(), "Subsequent Up() should fail with ErrDirty")
				assert.NoError(t, migrator.Force(expectedCorruptedMigrationTS), "Force() should not return an error")

				filePath := filepath.Join(corruptedMigrationSamplesDir, "20250101000135_alter_orders.up.sql")
				originalCorruptedContent, err := getMigrationFileContent(filePath)
				assert.NoError(t, err, "getMigrationFileContent() should not return an error")

				t.Cleanup(func() {
					err := makeOrAppendMigrationFile(filePath, originalCorruptedContent)
					assert.NoError(t, err, "cleanup: failed to restore original corrupted file content")
				})

				err = makeOrAppendMigrationFile(filePath, expectedMigrationFileContentUp)
				assert.NoError(t, err, "corrupted migration file fixed correctly")

				isDatabaseDirty, err = isDirty(db)
				assert.NoError(t, err, "isDirty() should not return an error")
				assert.False(t, isDatabaseDirty, "Database should no longer be dirty after Force")

				db, migrator, err = newMigrator(envVars["POSTGRES_DB"], dataSourceName, corruptedMigrationSamplesDir)
				assert.NoError(t, err, "newMigrator() should not return an error")

				assert.NoError(t, migrator.Up(), "Up() should succeed after fixing the file and forcing the migration")

				appliedVersions, err := getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions := []int{20250101000130, 20250101000135, 20250101000140}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				ordersTableExists, err := tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist now")

				createdAtColExists, err := columnExists(db, "orders", "created_at")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, createdAtColExists, "Expected 'orders' table 'created_at' column to exist now")

				usersTableExists, err := tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, usersTableExists, "Expected 'users' table to exist now")
			})
	})
}

// -------------------------------------------------------------------------------------------------------------------
// TestOutOfOrderMigrations is a test suite for handling migrations that are not in chronological order.
func TestOutOfOrderMigrations(t *testing.T) {
	// Test Scenario:
	// Migration files on disk are out of chronological order.
	// The migrator should still apply them correctly, out of order.

	t.Parallel()

	t.Run("OutOfOrderMigrations", func(t *testing.T) {
		runPostgresContainer(
			t,
			func(t *testing.T, c dktest.ContainerInfo, envVars map[string]string) {
				ip, port, err := c.FirstPort()
				assert.NoError(t, err, "FirstPort() should not return an error")
				dataSourceName := pgConnectionString(envVars["POSTGRES_PASSWORD"], ip, port)

				db, migrator, err := newMigrator(envVars["POSTGRES_DB"], dataSourceName, outOfOrderMigrationSamplesDir)
				assert.NoError(t, err, "newMigrator() should not return an error")

				assert.NoError(t, migrator.Up(), "Up() should not return an error with out-of-order migrations")

				appliedVersions, err := getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions := []int{20250101000130, 20250101000140}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in order")

				upfilePath := filepath.Join(outOfOrderMigrationSamplesDir, "20250101000135_alter_orders.up.sql")
				assert.NoError(t, err, "getMigrationFileContent() should not return an error")
				assert.NoError(t, makeOrAppendMigrationFile(upfilePath, expectedMigrationFileContentUp), "migration file created correctly")

				t.Cleanup(func() {
					assert.NoError(t, deleteFile(upfilePath), "cleanup: failed to restore original corrupted file content")
				})

				downfilePath := filepath.Join(outOfOrderMigrationSamplesDir, "20250101000135_alter_orders.down.sql")
				assert.NoError(t, err, "getMigrationFileContent() should not return an error")
				assert.NoError(t, makeOrAppendMigrationFile(downfilePath, expectedMigrationFileContentDown), "migration file created correctly")

				t.Cleanup(func() {
					assert.NoError(t, deleteFile(downfilePath), "cleanup: failed to restore original corrupted file content")
				})

				db, migrator, err = newMigrator(envVars["POSTGRES_DB"], dataSourceName, outOfOrderMigrationSamplesDir)
				assert.NoError(t, err, "newMigrator() should not return an error")

				assert.NoError(t, migrator.Up(), "Up() should not return an error with out-of-order migrations")

				appliedVersions, err = getAppliedVersions(db)
				assert.NoError(t, err, "getAppliedVersions() should not return an error")

				expectedVersions = []int{20250101000130, 20250101000135, 20250101000140}
				assert.ElementsMatch(t, expectedVersions, appliedVersions, "Migrations should be applied in chronological order")

				ordersTableExists, err := tableExists(db, "orders")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, ordersTableExists, "Expected 'orders' table to exist now")

				createdAtColExists, err := columnExists(db, "orders", "created_at")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, createdAtColExists, "Expected 'orders' table 'created_at' column to exist now")

				usersTableExists, err := tableExists(db, "users")
				assert.NoError(t, err, "tableExists() should not return an error")
				assert.True(t, usersTableExists, "Expected 'users' table to exist now")
			})
	})
}
