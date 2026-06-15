package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	sqliteHistory "github.com/vukyn/testproj/db/history/sqlite"
	migrationEntity "github.com/vukyn/testproj/internal/domains/migration/entity"
	migrationModels "github.com/vukyn/testproj/internal/domains/migration/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// RunMigrations executes all pending migrations.
func RunMigrations(dbType, dbPath string) (migrationModels.MigrationStats, error) {
	stats := migrationModels.MigrationStats{}
	// Open the database connection (creates the file if it does not exist).
	sqldb, err := sql.Open(sqliteshim.ShimName, dbPath)
	if err != nil {
		return stats, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqldb.Close()

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create the migrations bookkeeping table if it does not exist.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return stats, fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read the already-executed migrations.
	var executedMigrations []migrationEntity.MigrationHistory
	ctx := context.Background()
	err = db.NewSelect().Table("migrations").Scan(ctx, &executedMigrations)
	if err != nil {
		return stats, fmt.Errorf("failed to get executed migrations: %w", err)
	}

	// Select the migration set for the database type.
	var migrations []migrationModels.Migration
	switch dbType {
	case "sqlite":
		migrations = sqliteHistory.Migrations
	default:
		return stats, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Execute pending migrations.
	lastMigratedID := int64(0)
	if len(executedMigrations) > 0 {
		lastMigratedID = executedMigrations[len(executedMigrations)-1].ID
	}
	for _, migration := range migrations {
		executed := slices.ContainsFunc(executedMigrations, func(m migrationEntity.MigrationHistory) bool {
			return m.Name == migration.Name
		})

		if !executed {
			log.Printf("Running migration: %s", migration.Name)

			tx, err := db.Begin()
			if err != nil {
				return stats, fmt.Errorf("failed to begin transaction for migration %s: %w", migration.Name, err)
			}

			if err := migration.Up(db); err != nil {
				tx.Rollback()
				return stats, fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
			}

			migrationRecord := &migrationEntity.MigrationHistory{
				ID:         lastMigratedID + 1,
				Name:       migration.Name,
				ExecutedAt: time.Now(),
			}
			if _, err = db.NewInsert().Model(migrationRecord).Exec(ctx); err != nil {
				tx.Rollback()
				return stats, fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
			}
			lastMigratedID++

			if err := tx.Commit(); err != nil {
				return stats, fmt.Errorf("failed to commit migration %s: %w", migration.Name, err)
			}

			stats.TotalSuccess++
			log.Printf("Migration %s completed successfully", migration.Name)
		} else {
			stats.TotalSkipped++
			log.Printf("Migration %s already executed, skipping", migration.Name)
		}
	}

	return stats, nil
}

// RollbackLastMigration rolls back the last executed migration. It returns true
// when a migration was rolled back, false when there were none to roll back.
func RollbackLastMigration(dbType, dbPath string) (bool, error) {
	sqldb, err := sql.Open(sqliteshim.ShimName, dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqldb.Close()

	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	lastMigration := &migrationEntity.MigrationHistory{}
	err = db.NewSelect().Model(lastMigration).Order("id DESC").Limit(1).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("no migrations to rollback")
		}
		return false, fmt.Errorf("failed to get last migration: %w", err)
	}

	var migrations []migrationModels.Migration
	switch dbType {
	case "sqlite":
		migrations = sqliteHistory.Migrations
	default:
		return false, fmt.Errorf("unsupported database type: %s", dbType)
	}

	var migration *migrationModels.Migration
	for _, m := range migrations {
		if m.Name == lastMigration.Name {
			migration = &m
			break
		}
	}

	if migration == nil {
		return false, fmt.Errorf("migration %s not found", lastMigration.Name)
	}

	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := migration.Down(db); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to rollback migration %s: %w", migration.Name, err)
	}

	_, err = db.NewDelete().Model(&migrationEntity.MigrationHistory{}).Where("name = ?", migration.Name).Exec(ctx)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit rollback: %w", err)
	}

	log.Printf("Migration %s rolled back successfully", migration.Name)
	return true, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run db/migrate.go <db_type> <command>")
		fmt.Println("Database types:")
		fmt.Println("  sqlite - SQLite database")
		fmt.Println("Commands:")
		fmt.Println("  up     - Run all pending migrations")
		fmt.Println("  down   - Rollback last migration")
		fmt.Println("  reset  - Rollback all migrations")
		os.Exit(1)
	}

	dbType := os.Args[1]
	command := os.Args[2]

	var dbPath string
	switch dbType {
	case "sqlite":
		dbPath = "db/app.db"
	default:
		fmt.Printf("Unsupported database type: %s\n", dbType)
		os.Exit(1)
	}

	switch command {
	case "up":
		stats, err := RunMigrations(dbType, dbPath)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		printMigrateReport(stats.TotalSuccess, stats.TotalSkipped, "All migrations completed successfully")
	case "down":
		rolledBack, err := RollbackLastMigration(dbType, dbPath)
		if err != nil {
			if err.Error() == "no migrations to rollback" {
				printMigrateReport(0, 0, "No migrations to rollback")
			} else {
				log.Fatalf("Rollback failed: %v", err)
			}
		} else {
			totalSuccess := 0
			if rolledBack {
				totalSuccess = 1
			}
			printMigrateReport(totalSuccess, 0, "Last migration rolled back successfully")
		}
	case "reset":
		totalRolledBack := 0
		for {
			rolledBack, err := RollbackLastMigration(dbType, dbPath)
			if err != nil {
				if err.Error() == "no migrations to rollback" {
					break
				}
				log.Fatalf("Rollback failed: %v", err)
			}
			if rolledBack {
				totalRolledBack++
			}
		}
		printMigrateReport(totalRolledBack, 0, "All migrations rolled back successfully")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

// printMigrateReport prints the migration report with statistics.
func printMigrateReport(totalSuccess, totalSkipped int, msg string) {
	fmt.Printf("\n=== Migration Report ===\n")
	fmt.Printf("Total Success: %d\n", totalSuccess)
	fmt.Printf("Total Skipped: %d\n", totalSkipped)
	fmt.Println(msg)
}
