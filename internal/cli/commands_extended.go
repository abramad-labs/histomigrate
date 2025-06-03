package cli

import migrate "github.com/abramad-labs/histomigrate"

func doMigrationCmd(m *migrate.Migrate, v uint) error {
	return m.DoMigration(v)
}

func undoMigrationCmd(m *migrate.Migrate, v uint) error {
	return m.UndoMigration(v)
}
