package di

import (
	"github.com/vukyn/testproj/internal/constants"
	itemUsecase "github.com/vukyn/testproj/internal/domains/item/usecase"

	"github.com/sarulabs/di/v2"
	"github.com/vukyn/kuery/log"
)

func defineUsecase() []*di.Def {
	return []*di.Def{
		defineItemUsecase(),
	}
}

func defineItemUsecase() *di.Def {
	return &di.Def{
		Name:  constants.CONTAINER_NAME_ITEM_USECASE,
		Scope: di.Request,
		Build: func(ctn di.Container) (any, error) {
			itemRepository, err := GetItemRepository(ctn)
			if err != nil {
				return nil, err
			}
			log.New().Debug("Item usecase initialized")
			return itemUsecase.NewUsecase(itemRepository), nil
		},
		Close: func(obj any) error {
			log.New().Debug("Item usecase destroyed")
			return nil
		},
	}
}

func GetItemUsecase(ctn di.Container) (itemUsecase.IUseCase, error) {
	usecase, err := ctn.SafeGet(constants.CONTAINER_NAME_ITEM_USECASE)
	if err != nil {
		return nil, err
	}
	return usecase.(itemUsecase.IUseCase), nil
}
