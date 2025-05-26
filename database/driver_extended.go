package database

type ExtendedDriver interface {
	// Embeds core database interaction capabilities.
	Driver

	// GetAllAppliedMigrations retrieves all migration timestamps currently marked as applied.
	GetAllAppliedMigrations() ([]int, error)

	// IsMigrationApplied checks if a specific migration version has already been applied.
	IsMigrationApplied(uint) (bool, error)

	// IsDatabaseDirty checks if any migration is currently in a "dirty" (incomplete or failed) state.
	IsDatabaseDirty() (int, bool, error)

	// AddDirtyMigration marks a specific migration version as dirty upon initiation.
	AddDirtyMigration(uint) error

	// UpdateMigrationDirtyFlag sets or clears the "dirty" flag for a given migration version.
	UpdateMigrationDirtyFlag(uint, bool) error

	// RemoveMigration deletes a migration record from the applied list.
	RemoveMigration(uint) error
}
