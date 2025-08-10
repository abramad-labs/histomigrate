package testing_postgres_extended

import (
	"database/sql"
	"fmt"
)

const (
	schemaMigrationsTable = "schema_migrations"
)

func existsQuery(db *sql.DB, query string, args ...any) (bool, error) {
	var exists bool
	err := db.QueryRow(query, args...).Scan(&exists)
	return exists, err
}

func tableExists(db *sql.DB, tableName string) (bool, error) {
	q := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = current_schema() AND table_name = $1
		)`
	return existsQuery(db, q, tableName)
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	q := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2
		)`
	return existsQuery(db, q, table, column)
}

func isMigratedVersionDirty(db *sql.DB, migrTimestamp int) (bool, error) {
	var dirty bool
	q := fmt.Sprintf(`SELECT dirty FROM %s WHERE migration_timestamp = $1`, schemaMigrationsTable)
	err := db.QueryRow(q, migrTimestamp).Scan(&dirty)
	return dirty, err
}

func isDirty(db *sql.DB) (bool, error) {
	q := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s 
			WHERE dirty = true
		)`, schemaMigrationsTable)
	return existsQuery(db, q)
}

func getAppliedVersions(db *sql.DB) ([]int, error) {
	q := fmt.Sprintf(`SELECT migration_timestamp FROM %s`, schemaMigrationsTable)
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}
