package repository

import (
	"context"

	"github.com/vukyn/testproj/internal/domains/item/entity"
	"github.com/vukyn/testproj/internal/domains/item/models"
)

type IRepository interface {
	Create(ctx context.Context, req entity.CreateRequest) (string, error)
	GetByID(ctx context.Context, id string) (entity.Item, error)
	List(ctx context.Context, req models.ListRequest) ([]entity.Item, int64, error)
	Update(ctx context.Context, req entity.UpdateRequest) error
	Delete(ctx context.Context, id string) error
}
