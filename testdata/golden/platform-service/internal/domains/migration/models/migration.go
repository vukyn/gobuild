package models

import (
	"github.com/uptrace/bun"
)

// Migration represents a database migration.
type Migration struct {
	Name string
	Up   func(*bun.DB) error
	Down func(*bun.DB) error
}

// MigrationStats holds statistics about migration execution.
type MigrationStats struct {
	TotalSkipped int
	TotalSuccess int
}
