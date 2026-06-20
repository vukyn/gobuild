package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"

	iapp "github.com/vukyn/testproj/internal/app"
	"github.com/vukyn/testproj/internal/config"
	itemHandlers "github.com/vukyn/testproj/internal/domains/item/handlers/http"
	"github.com/vukyn/testproj/internal/middlewares"
	"github.com/vukyn/testproj/internal/web"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
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

	// The built UI is embedded in the binary (internal/web) — served from that FS
	// rather than a working-directory-relative path, so the binary is self-contained.
	uiFS := web.FS()
	engine := html.NewFileSystem(http.FS(uiFS), ".html")
	s.app = fiber.New(fiber.Config{
		AppName: s.cfg.App.Name,
		Views:   engine,
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

	// Static files - serve from root paths to match HTML references. Assets live
	// under the embedded FS's assets/ subtree.
	assetsFS, err := fs.Sub(uiFS, "assets")
	if err != nil {
		log.New().Errorf("Failed to open embedded assets: %v", err)
		os.Exit(1)
	}
	s.app.Use("/assets", filesystem.New(filesystem.Config{
		Root: http.FS(assetsFS),
	}))

	// api/v1
	apiV1 := s.app.Group("/api/v1")
	itemHandlers.SetupItemRoutes(apiV1)

	// web routes
	s.webRoutes(s.app, uiFS)

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

// webRoutes serves the embedded single-page app: a root-level favicon route (so
// the SPA catch-all doesn't shadow it) plus a catch-all that renders index.html.
func (s *Server) webRoutes(app *fiber.App, uiFS fs.FS) {
	renderHomePage := func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"APIBaseURL": s.cfg.Vite.BaseURL,
		})
	}

	// Root-level static assets that index.html references directly (outside
	// /assets). Without this, the SPA catch-all below renders index.html for
	// /favicon.svg, so the browser gets HTML instead of the icon and falls back
	// to its generic globe. SendFile sets content-type from the .svg extension.
	app.Get("/favicon.svg", func(c *fiber.Ctx) error {
		return filesystem.SendFile(c, http.FS(uiFS), "favicon.svg")
	})

	app.Get("/*", renderHomePage)
}
