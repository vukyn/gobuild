package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	iapp "github.com/vukyn/testproj/internal/app"
	"github.com/vukyn/testproj/internal/config"
	itemHandlers "github.com/vukyn/testproj/internal/domains/item/handlers/http"
	"github.com/vukyn/testproj/internal/middlewares"
	"github.com/vukyn/testproj/internal/web"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v3"
	"github.com/vukyn/kuery/log"
	pkgRecover "github.com/vukyn/kuery/recoverv3"
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
	trustProxy, trustProxyConfig := proxyTrust(s.cfg)
	s.app = fiber.New(fiber.Config{
		AppName:     s.cfg.App.Name,
		Views:       engine,
		ProxyHeader: s.cfg.App.ProxyHeader,
		// EnableIPValidation makes Fiber parse and validate the address it reads
		// out of ProxyHeader instead of returning the raw header text. Required
		// whenever a proxy header is consulted at all, harmless otherwise.
		EnableIPValidation: true,
		TrustProxy:         trustProxy,
		TrustProxyConfig:   trustProxyConfig,
	})

	// Middlewares
	s.app.Use(s.corsMiddleware())

	// Request logging. Fiber v3 has no fiberzerolog equivalent (gofiber/contrib
	// never published a v2 module of it — its whole API is *fiber.Ctx), so the
	// built-in logger middleware is pointed at kuery's zerolog through a
	// LoggerFunc. Keeping the sink here, rather than in a third-party adapter,
	// also makes swapping it back out a one-function change if contrib ever ships
	// a v3 line.
	zerologLogger := log.New().Zerolog()
	s.app.Use(fiberLogger.New(fiberLogger.Config{
		LoggerFunc: func(c fiber.Ctx, data *fiberLogger.Data, cfg *fiberLogger.Config) error {
			zerologLogger.Info().
				Str("method", c.Method()).
				Str("path", c.Path()).
				// logger.Data carries no status — read it off the response.
				Int("status", c.Response().StatusCode()).
				Dur("latency", data.Stop.Sub(data.Start)).
				Msg("request")
			return nil
		},
	}))

	// Inject a request-scoped DI container into the Fiber ctx.
	s.app.Use(middlewares.DiContainerMiddleware(iapp.App))

	// Recover from panics. Mounted INSIDE the DI middleware on purpose, so the
	// container release above still runs while a panic unwinds.
	s.app.Use(pkgRecover.NewFiberRecover())

	// Static files - serve from root paths to match HTML references. Assets live
	// under the embedded FS's assets/ subtree. Fiber v3 replaced
	// middleware/filesystem with middleware/static, whose Config.FS takes an
	// fs.FS directly (no http.FS wrapper).
	assetsFS, err := fs.Sub(uiFS, "assets")
	if err != nil {
		log.New().Errorf("Failed to open embedded assets: %v", err)
		os.Exit(1)
	}
	s.app.Use("/assets", static.New("", static.Config{
		FS: assetsFS,
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

// defaultCORSAllowOrigins is the local development fallback: the Vite dev
// server plus the API's own port.
//
// TODO: set CORS_ALLOW_ORIGINS in the environment to this service's real
// browser origin(s) before deploying.
var defaultCORSAllowOrigins = []string{"http://localhost:5173", "http://localhost:8080"}

// corsMiddleware builds the CORS handler mounted by Start. It is a method
// rather than an inline call so the shipped test exercises the same
// construction the server actually mounts.
func (s *Server) corsMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: corsAllowOrigins(s.cfg),
	})
}

// corsAllowOrigins resolves the browser origins allowed to call the API.
//
// ⚠️ Fiber v3 computes `allowAllOrigins := len(cfg.AllowOrigins) == 0 &&
// cfg.AllowOriginsFunc == nil`, so an EMPTY SLICE reopens the API to every
// origin WITHOUT the slice ever containing "*". This function must never return
// an empty slice, and a test asserting only "the result is not \"*\"" would not
// catch it — assert the length. (Fiber v2's AllowOrigins was a string whose
// empty value was replaced by the literal "*"; same hazard, different trigger.)
//
// ⚠️ Second v3-only behaviour: a malformed origin makes cors' configDefault
// PANIC at boot ("[CORS] Invalid origin format in configuration") rather than
// silently failing to match. That is loud by design — fix the value.
func corsAllowOrigins(cfg *config.Config) []string {
	origins := make([]string, 0, 2)
	for _, origin := range strings.Split(cfg.CORS.AllowOrigins, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return defaultCORSAllowOrigins
	}
	return origins
}

// proxyTrust wires Fiber v3's trusted-proxy gate from APP_PROXY_HEADER.
//
// ⚠️ In Fiber v3, ProxyHeader ALONE does nothing: c.IP() reads it only when
// IsProxyTrusted() is true, which requires TrustProxy plus a TrustProxyConfig
// rule matching the remote address. Without this, a service behind a proxy gets
// the PROXY's address for every caller — one bucket for every rate limiter, one
// value in every audit row — with no error anywhere. Fiber v2 needed no such
// wiring, so a config ported straight across from a v2 service is wrong here.
//
// Loopback + Private cover the usual deployment shape (fly.io and friends reach
// the app over a private address). Widen with TrustProxyConfig.Proxies if the
// proxy is elsewhere.
func proxyTrust(cfg *config.Config) (bool, fiber.TrustProxyConfig) {
	if strings.TrimSpace(cfg.App.ProxyHeader) == "" {
		return false, fiber.TrustProxyConfig{}
	}
	return true, fiber.TrustProxyConfig{Loopback: true, Private: true}
}

// webRoutes serves the embedded single-page app: a root-level favicon route (so
// the SPA catch-all doesn't shadow it) plus a catch-all that renders index.html.
func (s *Server) webRoutes(app *fiber.App, uiFS fs.FS) {
	renderHomePage := func(c fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"APIBaseURL": s.cfg.Vite.BaseURL,
		})
	}

	// Root-level static assets that index.html references directly (outside
	// /assets). Without this, the SPA catch-all below renders index.html for
	// /favicon.svg, so the browser gets HTML instead of the icon and falls back
	// to its generic globe. SendFile sets content-type from the .svg extension.
	//
	// ⚠️ This shape is CORRECT as generated — one root file, one route — but it
	// does not scale: add a second root-level asset (a .webmanifest, PWA icons,
	// sw.js) without adding its own route and the catch-all answers it with
	// index.html at status 200, so nothing 404s and the logs look healthy. That
	// is exactly how a downstream service shipped a dead service worker and an
	// Add-to-Home-Screen with no name or icon. The defect is reproduced here
	// deliberately, to keep this preset identical in shape to `platform-service`
	// — see gobuild's docs/pwa-root-file-audit.md. Do not change the shape in
	// only one of the two presets.
	app.Get("/favicon.svg", func(c fiber.Ctx) error {
		return c.SendFile("favicon.svg", fiber.SendFile{FS: uiFS})
	})

	app.Get("/*", renderHomePage)
}
