package migrate

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/abramad-labs/histomigrate/database"
)

// DoMigration executes a single database migration.
// It acquires a lock, checks if the migration is already applied, queues it for processing (if not applied), runs it, and then releases the lock.
// It requires an ExtendedDriver.
func (m *Migrate) DoMigration(version uint) error {
	if err := m.lock(); err != nil {
		return err
	}

	ret := make(chan interface{}, m.PrefetchMigrations)

	ed, isExtended := m.databaseDrv.(database.ExtendedDriver)
	if isExtended {
		isApplied, err := ed.IsMigrationApplied(version)
		if err != nil {
			return m.unlockErr(err)
		}

		if isApplied {
			return m.unlockErr(ErrNoChange)
		}

		go m.queueUpSingleMigration(version, ret)
	} else {
		return m.unlockErr(errors.New("driver type is not right"))
	}

	return m.unlockErr(m.runMigrations(ret))
}

// UndoMigration rolls back a specific database migration.
// It acquires a lock, confirms the migration is currently applied (returning ErrNoChange if not), then queues and runs the "down" migration.
// It requires an ExtendedDriver.
func (m *Migrate) UndoMigration(version uint) error {
	if err := m.lock(); err != nil {
		return err
	}

	ret := make(chan interface{}, m.PrefetchMigrations)

	ed, isExtended := m.databaseDrv.(database.ExtendedDriver)
	if isExtended {
		isApplied, err := ed.IsMigrationApplied(version)
		if err != nil {
			return m.unlockErr(err)
		}

		if !isApplied {
			return m.unlockErr(ErrNoChange)
		}

		go m.queueDownSingleMigration(version, ret)
	} else {
		return m.unlockErr(errors.New("driver type is not right"))
	}

	return m.unlockErr(m.runMigrations(ret))
}

// queueUpMigrations function is responsible for identifying and preparing "up" (forward) migrations that need to be applied.
// It starts by determining the first available migration from a sourceDrv (source driver, likely a file system or similar).
// It then iterates through subsequent migrations, skipping any that have already been applied (as indicated by the appliedMigrs list).
// For each unapplied migration, it creates a Migration object, marks it as an "up" migration, and sends it to the ret channel for further processing.
// The function also asynchronously buffers the migration's content in a separate goroutine.
// It respects a limit on the number of new migrations to queue and can be stopped gracefully.
// If no new migrations are found or queued (and no background errors occur), it signals ErrNoChange.
func (m *Migrate) queueUpMigrations(appliedMigrs []int, limit int, ret chan<- interface{}) {
	defer close(ret)

	appliedSet := make(map[int]struct{}, len(appliedMigrs))
	for _, v := range appliedMigrs {
		appliedSet[v] = struct{}{}
	}

	targetVersion, err := m.sourceDrv.First()
	if errors.Is(err, os.ErrNotExist) {
		ret <- ErrNoChange
		return
	}
	if err != nil {
		ret <- err
		return
	}

	appliedCount := 0

	for limit == -1 || appliedCount < limit {
		if m.stop() {
			break
		}

		if _, ok := appliedSet[int(targetVersion)]; ok {
			targetVersion, err = m.sourceDrv.Next(targetVersion)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					break
				}

				ret <- err
				return
			}

			continue
		}

		appliedCount++
		migr, err := m.newMigration(targetVersion, int(targetVersion))
		if err != nil {
			ret <- err
			return
		}

		migr.UpKindMigration = true

		ret <- migr

		go func(migr *Migration) {
			if err := migr.Buffer(); err != nil {
				m.logErr(err)
			}
		}(migr)

		targetVersion, err = m.sourceDrv.Next(targetVersion)
		if errors.Is(err, os.ErrNotExist) {
			break
		}

		if err != nil {
			ret <- err
			return
		}
	}

	if appliedCount == 0 {
		ret <- ErrNoChange
	}
}

// queueUpSingleMigration finds and buffers the specified "up" migration.
// It assumes that applied migrations are already filtered out before calling.
// It sends either the prepared migration or an error to the provided channel.
func (m *Migrate) queueUpSingleMigration(version uint, ret chan<- interface{}) {
	defer close(ret)

	if err := m.versionExists(version); err != nil {
		ret <- err
		return
	}

	if m.stop() {
		return
	}

	migr, err := m.newMigration(version, int(version))
	if err != nil {
		ret <- err
		return
	}

	migr.UpKindMigration = true

	ret <- migr

	go func(migr *Migration) {
		if err := migr.Buffer(); err != nil {
			m.logErr(err)
		}
	}(migr)
}

// queueDownMigrations iterates through a provided list of applied migrations (assumed to be in descending order of version),
// preparing them for "down" (rollback) operations.
// For each migration, it determines the target version (which would be the previous version in the sequence, or -1 if it's the oldest migration), creates a Migration object, sends it to a channel for processing,
// and asynchronously buffers its content. The function respects a limit on the number of migrations to process and can be stopped gracefully.
// If no migrations are found or processed (and no background errors occur), it signals ErrNoChange.
func (m *Migrate) queueDownMigrations(appliedMigrs []int, limit int, ret chan<- interface{}) {
	defer close(ret)

	if len(appliedMigrs) == 0 || limit == 0 {
		ret <- ErrNoChange
		return
	}

	appliedCount := 0

	for i := 0; i < len(appliedMigrs); i++ {
		if m.stop() {
			break
		}

		if limit != -1 && appliedCount >= limit {
			break
		}

		version := uint(appliedMigrs[i])

		var targetVersion int
		if i == len(appliedMigrs)-1 {
			targetVersion = -1
		} else {
			targetVersion = appliedMigrs[i+1]
		}

		appliedCount++
		migr, err := m.newMigration(version, targetVersion)
		if err != nil {
			ret <- err
			return
		}

		ret <- migr

		go func(migr *Migration) {
			if err := migr.Buffer(); err != nil {
				m.logErr(err)
			}
		}(migr)
	}

	if appliedCount == 0 {
		ret <- ErrNoChange
	}
}

// queueDownSingleMigration finds and buffers the specified "down" migration.
// It calculates the target version (previous version) for the rollback.
// It sends either the prepared migration or an error to the provided channel.
func (m *Migrate) queueDownSingleMigration(version uint, ret chan<- interface{}) {
	defer close(ret)

	if err := m.versionExists(version); err != nil {
		ret <- err
		return
	}

	if m.stop() {
		return
	}

	targetVersion := -1
	prev, err := m.sourceDrv.Prev(version)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			ret <- err
			return
		}
	} else {
		targetVersion = int(prev)
	}

	migr, err := m.newMigration(version, targetVersion)
	if err != nil {
		ret <- err
		return
	}

	ret <- migr

	go func(migr *Migration) {
		if err := migr.Buffer(); err != nil {
			m.logErr(err)
		}
	}(migr)
}

// handleSingleMigration function is a private helper responsible for executing a single database migration. It takes a `*Migration` object and performs the following steps:
// 1.  Pre-Migration State Management: It first interacts with the database driver to set the migration's state to "dirty" (or "in-progress") before running its script. If the driver implements `database.ExtendedDriver`, it uses specific methods like `AddDirtyMigration` for "up" migrations or `UpdateMigrationDirtyFlag(..., true)` for "down" migrations. Otherwise, it defaults to `SetVersion(..., true)`.
// 2.  Execute Migration Body: If the migration contains a script (`migr.Body` is not nil), it logs the execution and then runs the migration's `BufferedBody` (the actual SQL or code) against the database using the driver's `Run` method.
// 3.  Post-Migration State Management: After successful execution of the body, it updates the migration's status to "clean" or "applied." If using an `ExtendedDriver`, it calls `UpdateMigrationDirtyFlag(..., false)` for "up" migrations or `RemoveMigration` for "down" migrations. For basic drivers, it calls `SetVersion(..., false)`.
// 4.  Logging Timings: Finally, it calculates and logs the time taken for buffering and running the migration, providing insights into performance.
// The function handles errors at each step, wrapping them with contextual information to indicate exactly where the failure occurred. It relies on the `m.databaseDrv` (which can be `database.ExtendedDriver` or a simpler `BasicDriver`) to interact with the underlying database.
func (m *Migrate) handleSingleMigration(migr *Migration) error {
	ed, isExtended := m.databaseDrv.(database.ExtendedDriver)

	if isExtended {
		if migr.UpKindMigration {
			if err := ed.AddDirtyMigration(migr.Version); err != nil {
				return fmt.Errorf("failed to add dirty migration for version %d: %w", migr.Version, err)
			}
		} else {
			if err := ed.UpdateMigrationDirtyFlag(migr.Version, true); err != nil {
				return fmt.Errorf("failed to set dirty flag for version %d: %w", migr.Version, err)
			}
		}
	} else {
		if err := m.databaseDrv.SetVersion(migr.TargetVersion, true); err != nil {
			return fmt.Errorf("failed to set dirty version %d: %w", migr.TargetVersion, err)
		}
	}

	if migr.Body != nil {
		m.logVerbosePrintf("Read and execute %v\n", migr.LogString())
		if err := m.databaseDrv.Run(migr.BufferedBody); err != nil {
			return fmt.Errorf("failed to run migration %d body: %w", migr.Version, err)
		}
	}

	if isExtended {
		if migr.UpKindMigration {
			if err := ed.UpdateMigrationDirtyFlag(migr.Version, false); err != nil {
				return fmt.Errorf("failed to clear dirty flag for version %d: %w", migr.Version, err)
			}
		} else {
			if err := ed.RemoveMigration(migr.Version); err != nil {
				return fmt.Errorf("failed to remove migration for version %d: %w", migr.Version, err)
			}
		}
	} else {
		if err := m.databaseDrv.SetVersion(migr.TargetVersion, false); err != nil {
			return fmt.Errorf("failed to set clean version %d: %w", migr.TargetVersion, err)
		}
	}

	endTime := time.Now()
	readTime := migr.FinishedReading.Sub(migr.StartedBuffering)
	runTime := endTime.Sub(migr.FinishedReading)

	if m.Log != nil {
		if m.Log.Verbose() {
			m.logPrintf("Finished %v (read %v, ran %v)\n", migr.LogString(), readTime, runTime)
		} else {
			m.logPrintf("%v (%v)\n", migr.LogString(), readTime+runTime)
		}
	}

	return nil
}
