package middlewares

import (
	"github.com/vukyn/testproj/internal/config"

	pkgCtx "github.com/vukyn/kuery/ctx"
	"github.com/vukyn/kuery/log"

	"github.com/gofiber/fiber/v2"
	"github.com/sarulabs/di/v2"
)

type Middleware struct {
	cfg *config.Config
}

func NewMiddleware(cfg *config.Config) *Middleware {
	return &Middleware{
		cfg: cfg,
	}
}

// DiContainerMiddleware creates a request-scoped sub-container off the given app
// container, stores it in the Fiber locals so handlers can resolve request-scoped
// dependencies, and — the load-bearing part — RELEASES it when the request is done.
//
// ⚠️ THE MIDDLEWARE THAT CREATES THE CONTAINER OWNS ITS LIFETIME. Handlers must NOT
// call `ctn.Delete()`; they only read the container out of the locals. A
// handler-owned lifetime cannot cover a request that never reaches a handler: this
// middleware is mounted globally, ahead of the routes, so it has already built a
// sub-container by the time anything decides the request is not going to reach a
// handler — an auth 401, an RBAC 403, a rate-limit 429, `/assets`, `/favicon.svg`,
// a 404, every SPA catch-all render. `sarulabs/di` keeps every sub-container in its
// parent's `children` map until it is deleted, so each of those requests would
// retain its container for the life of the PROCESS, and those leaking paths are the
// cheap, unauthenticated ones an attacker picks. Only the creator can release it.
//
// The release is a `defer`, so it also runs while a panic unwinds (mount the recover
// middleware INSIDE this one) and on any error return.
//
// DeleteWithSubContainers, not Delete: Delete is conditional — with any child
// present it merely sets `deleteIfNoChild` and returns nil, leaving the container in
// the parent's children map, which is precisely the leak. The single owner of a
// lifetime needs an unconditional release. Safe as long as nothing uses the
// container after its handler returns — no streaming response
// (SendStream/SetBodyStreamWriter) and no goroutine that resolves from it. That is a
// precondition of this design: if you hand a request-scoped dependency to a
// goroutine that outlives the request, this release becomes a use-after-free and the
// ownership has to be rethought, not worked around.
func DiContainerMiddleware(app di.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		request, err := app.SubContainer()
		if err != nil {
			return err
		}
		defer func() {
			if err := request.DeleteWithSubContainers(); err != nil {
				log.New().Errorf("release request di container: %v", err)
			}
		}()
		pkgCtx.SetDiContainerRequestToFiberCtx(c, request)
		return c.Next()
	}
}
