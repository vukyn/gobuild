package handlers

import (
	idi "github.com/vukyn/testproj/internal/di"
	"github.com/vukyn/testproj/internal/domains/item/models"

	pkgCtx "github.com/vukyn/kuery/ctxv3"
	pkgHttp "github.com/vukyn/kuery/http/fiberv3"

	"github.com/gofiber/fiber/v3"
)

// Handlers only BORROW the request-scoped container out of the Fiber locals —
// there is deliberately no `defer ctn.Delete()` anywhere in this file.
// DiContainerMiddleware created it and releases it on every path; see the doc
// comment on internal/middlewares/middleware.go.
//
// Fiber v3 notes: the ctx is the `fiber.Ctx` INTERFACE (v2 passed a pointer),
// and body/query decoding moved to `c.Bind().Body()` / `c.Bind().Query()`.
// Bind defaults to WithoutAutoHandling, so those return the raw parse error
// exactly as v2's BodyParser/QueryParser did and pkgHttp.Err still owns the
// response envelope. Do NOT switch to .WithAutoHandling() — it sets a status
// behind the handler's back and breaks that contract.

func CreateItem(c fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	createRequest := models.CreateRequest{}
	if err := c.Bind().Body(&createRequest); err != nil {
		return pkgHttp.Err(c, err)
	}

	itemResponse, err := usecase.Create(pkgCtx.NewContextFromFiberCtx(c), createRequest)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, itemResponse)
}

func ListItems(c fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	listRequest := models.ListRequest{}
	if err := c.Bind().Query(&listRequest); err != nil {
		return pkgHttp.Err(c, err)
	}

	listResponse, err := usecase.List(pkgCtx.NewContextFromFiberCtx(c), listRequest)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, listResponse)
}

func GetItem(c fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)

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

func UpdateItem(c fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	updateRequest := models.UpdateRequest{}
	if err := c.Bind().Body(&updateRequest); err != nil {
		return pkgHttp.Err(c, err)
	}

	itemResponse, err := usecase.Update(pkgCtx.NewContextFromFiberCtx(c), c.Params("itemID"), updateRequest)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, itemResponse)
}

func DeleteItem(c fiber.Ctx) error {
	ctn := pkgCtx.GetDiContainerRequestFromFiberCtx(c)

	usecase, err := idi.GetItemUsecase(ctn)
	if err != nil {
		return pkgHttp.Err(c, err)
	}

	if err := usecase.Delete(pkgCtx.NewContextFromFiberCtx(c), c.Params("itemID")); err != nil {
		return pkgHttp.Err(c, err)
	}

	return pkgHttp.OK(c, nil)
}
