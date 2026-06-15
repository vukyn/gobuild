package server

import (
	"fmt"
	"os"

	iapp "github.com/vukyn/testproj/internal/app"
	"github.com/vukyn/testproj/internal/config"
	itemHandlers "github.com/vukyn/testproj/internal/domains/item/handlers/http"
	"github.com/vukyn/testproj/internal/middlewares"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/vukyn/kuery/log"
	pkgRecover "github.com/vukyn/kuery/recover"
)

type Server struct {
	app *fiber.App
	cfg *config.Config
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) Start() {
	log.New().Info("Starting server")

	s.app = fiber.New(fiber.Config{
		AppName: s.cfg.App.Name,
	})

	// Middlewares
	s.app.Use(cors.New())
	zerologLogger := log.New().Zerolog()
	s.app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &zerologLogger,
	}))

	// Inject a request-scoped DI container into the Fiber ctx.
	s.app.Use(middlewares.DiContainerMiddleware(iapp.App))

	// Recover from panics.
	s.app.Use(pkgRecover.NewFiberRecover())

	// api/v1
	apiV1 := s.app.Group("/api/v1")
	itemHandlers.SetupItemRoutes(apiV1)

	// Start the server.
	go func() {
		if err := s.app.Listen(fmt.Sprintf(":%d", s.cfg.App.Port)); err != nil {
			log.New().Errorf("Failed to start server: %v", err)
			os.Exit(1)
		}
	}()
}

func (s *Server) Stop() error {
	return s.app.Shutdown()
}
