package handlers

import (
	idi "github.com/vukyn/testproj/internal/di"
	"github.com/vukyn/testproj/internal/domains/item/models"

	pkgCtx "github.com/vukyn/kuery/ctx"
	pkgHttp "github.com/vukyn/kuery/http/fiber"

	"github.com/gofiber/fiber/v2"
)

func CreateItem(c *fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)
	defer ctn.Delete()

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	createRequest := models.CreateRequest{}
	if err := c.BodyParser(&createRequest); err != nil {
		return pkgHttp.Err(c, err)
	}

	itemResponse, err := usecase.Create(pkgCtx.NewContextFromFiberCtx(c), createRequest)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, itemResponse)
}

func ListItems(c *fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)
	defer ctn.Delete()

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	listRequest := models.ListRequest{}
	if err := c.QueryParser(&listRequest); err != nil {
		return pkgHttp.Err(c, err)
	}

	listResponse, err := usecase.List(pkgCtx.NewContextFromFiberCtx(c), listRequest)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, listResponse)
}

func GetItem(c *fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)
	defer ctn.Delete()

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	itemResponse, err := usecase.Get(pkgCtx.NewContextFromFiberCtx(c), c.Params("itemID"))
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, itemResponse)
}

func UpdateItem(c *fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)
	defer ctn.Delete()

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	updateRequest := models.UpdateRequest{}
	if err := c.BodyParser(&updateRequest); err != nil {
		return pkgHttp.Err(c, err)
	}

	itemResponse, err := usecase.Update(pkgCtx.NewContextFromFiberCtx(c), c.Params("itemID"), updateRequest)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, itemResponse)
}

func DeleteItem(c *fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)
	defer ctn.Delete()

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	if err := usecase.Delete(pkgCtx.NewContextFromFiberCtx(c), c.Params("itemID")); err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, nil)
}
