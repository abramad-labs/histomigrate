package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/abramad-labs/histomigrate/database"
	"github.com/hashicorp/go-multierror"
	"github.com/lib/pq"
)

func init() {
	db := PostgresExtras{
		Postgres: &Postgres{},
	}

	database.Register("postgres", &db)
	database.Register("postgresql", &db)
}

type PostgresExtras struct {
	*Postgres
}

// WithConnection initializes a new PostgresExtras instance using an existing, active sql.Conn and a Config struct.
// It ensures the connection is valid and, if not explicitly provided in the config, it automatically fetches the current database name and schema name from the connection.
// It also sets default values for the migrations table if none are specified and correctly parses quoted table names.
// Finally, it verifies the existence and readiness of the migrations version table in the database before returning the initialized PostgresExtras object.
func WithConnection(ctx context.Context, conn *sql.Conn, config *Config) (*PostgresExtras, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if err := conn.PingContext(ctx); err != nil {
		return nil, err
	}

	if config.DatabaseName == "" {
		query := `SELECT CURRENT_DATABASE()`
		var databaseName string
		if err := conn.QueryRowContext(ctx, query).Scan(&databaseName); err != nil {
			return nil, &database.Error{OrigErr: err, Query: []byte(query)}
		}

		if len(databaseName) == 0 {
			return nil, ErrNoDatabaseName
		}

		config.DatabaseName = databaseName
	}

	if config.SchemaName == "" {
		query := `SELECT CURRENT_SCHEMA()`
		var schemaName sql.NullString
		if err := conn.QueryRowContext(ctx, query).Scan(&schemaName); err != nil {
			return nil, &database.Error{OrigErr: err, Query: []byte(query)}
		}

		if !schemaName.Valid {
			return nil, ErrNoSchema
		}

		config.SchemaName = schemaName.String
	}

	if len(config.MigrationsTable) == 0 {
		config.MigrationsTable = DefaultMigrationsTable
	}

	config.migrationsSchemaName = config.SchemaName
	config.migrationsTableName = config.MigrationsTable
	if config.MigrationsTableQuoted {
		re := regexp.MustCompile(`"(.*?)"`)
		result := re.FindAllStringSubmatch(config.MigrationsTable, -1)
		config.migrationsTableName = result[len(result)-1][1]
		if len(result) == 2 {
			config.migrationsSchemaName = result[0][1]
		} else if len(result) > 2 {
			return nil, fmt.Errorf("\"%s\" MigrationsTable contains too many dot characters", config.MigrationsTable)
		}
	}

	px := &Postgres{
		conn:   conn,
		config: config,
	}

	if err := px.ensureVersionTable(); err != nil {
		return nil, err
	}

	return &PostgresExtras{
		Postgres: px,
	}, nil
}

// GetAllAppliedMigrations retrieves a list of all applied migration timestamps from the migrations table in the database.
// It constructs a SQL query to select migration_timestamp values, orders them in descending order, and executes the query against the database.
// The function then scans the results into a slice of integers and handles any potential database errors, including proper closing of the result rows.
func (p *PostgresExtras) GetAllAppliedMigrations() ([]int, error) {
	schema := pq.QuoteIdentifier(p.config.migrationsSchemaName)
	table := pq.QuoteIdentifier(p.config.migrationsTableName)

	query := fmt.Sprintf(
		`SELECT migration_timestamp FROM %s.%s ORDER BY migration_timestamp DESC`,
		schema,
		table,
	)

	rows, err := p.conn.QueryContext(context.Background(), query)
	if err != nil {
		return nil, &database.Error{
			OrigErr: err,
			Query:   []byte(query),
		}
	}

	defer func() {
		if errClose := rows.Close(); errClose != nil {
			err = multierror.Append(err, errClose)
		}
	}()

	var appliedMigrations []int
	for rows.Next() {
		var migrTs int
		if err := rows.Scan(&migrTs); err != nil {
			return nil, &database.Error{
				OrigErr: err,
				Query:   []byte(query),
			}
		}
		appliedMigrations = append(appliedMigrations, migrTs)
	}

	if err := rows.Err(); err != nil {
		return nil, &database.Error{
			OrigErr: err,
			Query:   []byte(query),
		}
	}

	return appliedMigrations, nil
}

// AddDirtyMigration marks a specific migration version as "dirty" or "in-progress" in the database's migrations table.
// It does this by inserting a new record with the provided version into the table within a database transaction.
// If the insertion fails or the transaction cannot be committed, it attempts to roll back the transaction and returns a detailed error.
func (p *PostgresExtras) AddDirtyMigration(version uint) error {
	tx, err := p.conn.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return &database.Error{
			OrigErr: err,
			Err:     "failed to start transaction",
		}
	}

	schema := pq.QuoteIdentifier(p.config.migrationsSchemaName)
	table := pq.QuoteIdentifier(p.config.migrationsTableName)
	query := fmt.Sprintf(
		`INSERT INTO %s.%s (migration_timestamp) VALUES ($1)`,
		schema,
		table,
	)

	if _, execErr := tx.Exec(query, version); execErr != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			execErr = multierror.Append(execErr, rbErr)
		}
		return &database.Error{
			OrigErr: execErr,
			Query:   []byte(query),
		}
	}

	if err := tx.Commit(); err != nil {
		return &database.Error{
			OrigErr: err,
			Err:     "failed to commit transaction",
		}
	}

	return nil
}

// UpdateMigrationDirtyFlag updates the dirty status and applied_at timestamp for a specific migration version in the database's migrations table.
// It sets the dirty flag to true or false based on the provided boolean value and records the current timestamp.
// The operation is performed within a database transaction, with robust error handling for transaction start, execution, and commit/rollback.
func (p *PostgresExtras) UpdateMigrationDirtyFlag(version uint, dirty bool) error {
	tx, err := p.conn.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return &database.Error{
			OrigErr: err,
			Err:     "failed to start transaction",
		}
	}

	schema := pq.QuoteIdentifier(p.config.migrationsSchemaName)
	table := pq.QuoteIdentifier(p.config.migrationsTableName)
	query := fmt.Sprintf(
		`UPDATE %s.%s SET dirty = $1, applied_at = NOW() WHERE migration_timestamp = $2`,
		schema,
		table,
	)

	if _, execErr := tx.Exec(query, dirty, version); execErr != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			execErr = multierror.Append(execErr, rbErr)
		}

		return &database.Error{
			OrigErr: execErr,
			Query:   []byte(query),
		}
	}

	if err := tx.Commit(); err != nil {
		return &database.Error{
			OrigErr: err,
			Err:     "failed to commit transaction",
		}
	}

	return nil
}

// IsMigrationApplied checks if a specific migration version has already been applied
// by querying the migrations table. It returns true if the migration is found,
// false if not found or if the table doesn't exist. It wraps other unexpected
// errors in a custom database.Error.
func (p *PostgresExtras) IsMigrationApplied(version uint) (bool, error) {
	schema := pq.QuoteIdentifier(p.config.migrationsSchemaName)
	table := pq.QuoteIdentifier(p.config.migrationsTableName)

	query := fmt.Sprintf(
		`SELECT COUNT(*) > 0 FROM %s.%s WHERE migration_timestamp = $1;`,
		schema,
		table,
	)

	var isApplied bool
	err := p.conn.QueryRowContext(context.Background(), query, version).Scan(&isApplied)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "undefined_table" {
			return false, nil
		}

		return false, &database.Error{
			OrigErr: err,
			Query:   []byte(query),
		}
	}

	return isApplied, nil
}

// RemoveMigration deletes a specific migration version from the database's migrations table.
// It constructs and executes a SQL DELETE statement based on the provided version within a database transaction.
// The function includes comprehensive error handling for transaction management, ensuring that any failures during the deletion process are properly rolled back and reported.
func (p *PostgresExtras) RemoveMigration(version uint) error {
	tx, err := p.conn.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return &database.Error{
			OrigErr: err,
			Err:     "transaction start failed",
		}
	}

	schema := pq.QuoteIdentifier(p.config.migrationsSchemaName)
	table := pq.QuoteIdentifier(p.config.migrationsTableName)
	query := fmt.Sprintf(
		`DELETE FROM %s.%s WHERE migration_timestamp = $1;`,
		schema,
		table,
	)

	if _, execErr := tx.Exec(query, version); execErr != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			execErr = multierror.Append(execErr, rbErr)
		}

		return &database.Error{
			OrigErr: execErr,
			Query:   []byte(query),
		}
	}

	if err := tx.Commit(); err != nil {
		return &database.Error{
			OrigErr: err,
			Err:     "transaction commit failed",
		}
	}

	return nil
}

// IsDatabaseDirty checks if the database's migrations table contains any "dirty" (in-progress or failed) migrations.
// It queries the migrations table for any entry where the dirty flag is set to true.
// If a dirty migration is found, it returns its migration_timestamp and true. If no dirty migrations are found,
// or if the migrations table itself doesn't exist (e.g., first run), it returns 0 and false. Any other database errors are wrapped and returned.
func (p *Postgres) IsDatabaseDirty() (int, bool, error) {
	schema := pq.QuoteIdentifier(p.config.migrationsSchemaName)
	table := pq.QuoteIdentifier(p.config.migrationsTableName)

	query := fmt.Sprintf(`SELECT migration_timestamp FROM %s.%s WHERE dirty = true LIMIT 1`, schema, table)

	var migr int

	err := p.conn.QueryRowContext(context.Background(), query).Scan(&migr)
	if err != nil {
		if e, ok := err.(*pq.Error); ok && e.Code.Name() == "undefined_table" {
			return 0, false, nil
		}

		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}

		return 0, false, &database.Error{
			OrigErr: err,
			Query:   []byte(query),
		}
	}

	return migr, true, nil
}
