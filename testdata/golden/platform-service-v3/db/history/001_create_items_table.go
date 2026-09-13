package history

import (
	"context"

	"github.com/uptrace/bun"
	pkgMigrate "github.com/vukyn/kuery/bun/migrate"
)

// m001CreateItemsTable creates the example `item` domain's table. ID is a ULID
// text pk; standard audit + soft-delete columns. Postgres DDL (TIMESTAMPTZ).
//
// The three audit-actor columns are nullable with no DEFAULT: the entity tags
// them `nullzero`, so bun writes NULL rather than an empty string when no user
// is on the context.
//
// Signature note: Up/Down take a `bun.IDB`, not a `*bun.DB` — kuery's runner
// threads its per-migration transaction handle in, so the whole migration
// commits or rolls back as one.
var m001CreateItemsTable = pkgMigrate.Migration{
	Name: "001_create_items_table",
	Up: func(db bun.IDB) error {
		_, err := db.ExecContext(context.Background(), `
			CREATE TABLE IF NOT EXISTS items (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				description TEXT NOT NULL DEFAULT '',
				status INTEGER NOT NULL DEFAULT 1,
				created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
				created_by TEXT,
				updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
				updated_by TEXT,
				deleted_at TIMESTAMPTZ,
				deleted_by TEXT
			)
		`)
		return err
	},
	Down: func(db bun.IDB) error {
		_, err := db.ExecContext(context.Background(), `DROP TABLE IF EXISTS items`)
		return err
	},
}
