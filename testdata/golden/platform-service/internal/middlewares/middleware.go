package middlewares

import (
	"github.com/vukyn/testproj/internal/config"

	pkgCtx "github.com/vukyn/kuery/ctx"

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

// DiContainerMiddleware creates a request-scoped sub-container off the given
// app container and stores it in the Fiber locals so handlers can resolve
// request-scoped dependencies. Handlers must call defer ctn.Delete().
func DiContainerMiddleware(app di.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		request, err := app.SubContainer()
		if err != nil {
			return err
		}
		pkgCtx.SetDiContainerRequestToFiberCtx(c, request)
		return c.Next()
	}
}
