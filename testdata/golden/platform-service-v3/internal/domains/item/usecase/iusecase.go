package usecase

import (
	"context"

	"github.com/vukyn/testproj/internal/domains/item/models"
)

type IUseCase interface {
	Create(ctx context.Context, req models.CreateRequest) (models.ItemResponse, error)
	Get(ctx context.Context, id string) (models.ItemResponse, error)
	List(ctx context.Context, req models.ListRequest) (models.ListResponse, error)
	Update(ctx context.Context, id string, req models.UpdateRequest) (models.ItemResponse, error)
	Delete(ctx context.Context, id string) error
}
