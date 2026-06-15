package entity

import (
	"time"

	"github.com/uptrace/bun"
)

type MigrationHistory struct {
	bun.BaseModel `bun:"table:migrations,alias:mig"`
	ID            int64     `bun:"id,pk,notnull"`
	Name          string    `bun:"name,notnull"`
	ExecutedAt    time.Time `bun:"executed_at,notnull"`
}
