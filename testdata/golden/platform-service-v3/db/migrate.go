package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/vukyn/testproj/db/history"
	"github.com/vukyn/testproj/internal/config"

	kueryDb "github.com/vukyn/kuery/bun/db"
	"github.com/vukyn/kuery/bun/migrate"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run db/migrate.go <db_type> <command>")
		fmt.Println("Database types:")
		fmt.Println("  postgres - PostgreSQL database (default; connection from DB_* env vars)")
		fmt.Println("  sqlite   - SQLite database")
		fmt.Println("Commands:")
		fmt.Println("  up       - Run all pending migrations")
		fmt.Println("  down     - Rollback last migration")
		fmt.Println("  reset    - Rollback all migrations")
		os.Exit(1)
	}

	dbType := os.Args[1]
	command := os.Args[2]

	// The CLI db_type arg selects the driver and overrides DB_DRIVER from .env.
	// This preset is Postgres-only in practice (db/history holds Postgres DDL),
	// but the sqlite branch stays valid so the runner shape matches the other
	// platform services.
	switch dbType {
	case "sqlite", "postgres":
		// supported
	default:
		fmt.Printf("Unsupported database type: %s\n", dbType)
		os.Exit(1)
	}

	// Load connection settings from .env via the shared config loader, then build
	// the dialect-aware connection.
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := kueryDb.Open(kueryDb.Config{
		Driver:      kueryDb.Driver(dbType),
		SQLitePath:  cfg.DB.SQLitePath,
		PostgresDSN: cfg.DB.DSN,
		Host:        cfg.DB.Host,
		Port:        cfg.DB.Port,
		User:        cfg.DB.User,
		Password:    cfg.DB.Password,
		DBName:      cfg.DB.DBName,
		SSLMode:     cfg.DB.SSLMode,
	})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	switch command {
	case "up":
		stats, err := migrate.Run(db, history.Migrations)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		printMigrateReport(stats.TotalSuccess, stats.TotalSkipped, "All migrations completed successfully")
	case "down":
		rolledBack, err := migrate.RollbackLast(db, history.Migrations)
		if err != nil {
			if errors.Is(err, migrate.ErrNoMigrationsToRollback) {
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
		n, err := migrate.Reset(db, history.Migrations)
		if err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		printMigrateReport(n, 0, "All migrations rolled back successfully")
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
