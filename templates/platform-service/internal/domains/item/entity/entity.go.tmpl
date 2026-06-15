package entity

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type Item struct {
	bun.BaseModel `bun:"table:items,alias:itm"`
	ID            string    `bun:"id,pk,notnull"`
	Name          string    `bun:"name,notnull"`
	Description   string    `bun:"description,notnull"`
	Status        int32     `bun:"status,notnull"`
	CreatedAt     time.Time `bun:"created_at,default:current_timestamp"`
	CreatedBy     string    `bun:"created_by,nullzero"`
	UpdatedAt     time.Time `bun:"updated_at,default:current_timestamp"`
	UpdatedBy     string    `bun:"updated_by,nullzero"`
	DeletedAt     time.Time `bun:"deleted_at,soft_delete,nullzero"`
	DeletedBy     string    `bun:"deleted_by,nullzero"`
}

type CreateRequest struct {
	Name        string
	Description string
	Status      int32
}

type UpdateRequest struct {
	ID          string
	Name        *string
	Description *string
	Status      *int32
}

// === Hooks ===

func (i *Item) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch q := query.(type) {
	case *bun.InsertQuery:
		i.CreatedAt = time.Now().UTC()
	case *bun.UpdateQuery:
		q.Column("updated_at")
		i.UpdatedAt = time.Now().UTC()
	}
	return nil
}
