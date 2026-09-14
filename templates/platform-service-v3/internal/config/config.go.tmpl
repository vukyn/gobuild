package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	App struct {
		Name string `envconfig:"APP_NAME"`
		Port int    `envconfig:"APP_PORT"`
		Env  string `envconfig:"APP_ENV"`
		// ProxyHeader names the header carrying the real client IP when the
		// service runs behind a proxy ("Fly-Client-IP" on fly.io). Empty means
		// "no proxy" — trust the socket's remote address.
		//
		// ⚠️ In Fiber v3 this value alone does nothing: c.IP() consults the
		// header only when the connection comes from a TRUSTED proxy.
		// internal/server derives Config.TrustProxy from this field (see
		// proxyTrust); do not set ProxyHeader anywhere else without the same
		// wiring, or every caller collapses into the proxy's own address.
		ProxyHeader string `envconfig:"APP_PROXY_HEADER"`
	}
	Logger struct {
		Mode  string `envconfig:"LOGGER_MODE"`
		Level string `envconfig:"LOGGER_LEVEL"`
	}
	Graceful struct {
		Verbose               bool `envconfig:"GRACEFUL_VERBOSE"`
		StepDelay             int  `envconfig:"GRACEFUL_STEP_DELAY"`
		ServerShutdownTimeout int  `envconfig:"GRACEFUL_SERVER_SHUTDOWN_TIMEOUT"`
	}
	// Vite carries values injected into the embedded SPA's index.html at render
	// time (e.g. the API base URL the browser bundle should call).
	Vite struct {
		BaseURL string `envconfig:"VITE_API_BASE_URL"`
	}
	// CORS holds the browser origins allowed to call the API, as a
	// comma-separated list. Left empty the server falls back to the local
	// development origins rather than to a wildcard.
	CORS struct {
		AllowOrigins string `envconfig:"CORS_ALLOW_ORIGINS"`
	}
	DB struct {
		// Driver selects the backend. This preset is Postgres-only; the default
		// keeps the migration CLI and DI working without an explicit DB_DRIVER.
		Driver string `envconfig:"DB_DRIVER" default:"postgres"`
		// SQLitePath is unused in the Postgres-only setup but kept so the shared
		// kueryDb.Config wiring stays identical to the other services.
		SQLitePath string `envconfig:"DB_SQLITE_PATH" default:"db/app.db"`
		// Host/Port/User/Password/DBName are the Postgres connection fields, used
		// when DSN is empty.
		Host     string `envconfig:"DB_HOST"`
		Port     int    `envconfig:"DB_PORT"`
		User     string `envconfig:"DB_USER"`
		Password string `envconfig:"DB_PASSWORD"`
		DBName   string `envconfig:"DB_NAME"`
		// SSLMode is the Postgres sslmode. It has NO struct-tag default on purpose:
		// an empty value reaches kuery's DSN builder, which fails closed to
		// "require", so an operator who forgets the variable gets TLS rather than
		// cleartext. A `default:"disable"` here re-arms the old behaviour behind
		// kuery's back — never exploitable in production, where DB_DSN carries its
		// own sslmode and takes precedence over these discrete fields entirely,
		// but a trap for whoever clears DB_DSN to use them.
		// A non-TLS local Postgres therefore needs DB_SSLMODE=disable set
		// EXPLICITLY (it is in .env beside the docker-compose connection fields).
		SSLMode string `envconfig:"DB_SSLMODE"`
		// DSN is an optional full Postgres DSN override; when set it takes
		// precedence over the discrete Host/Port/... fields.
		DSN string `envconfig:"DB_DSN" default:""`
	}
}

func LoadConfig(envFiles ...string) (*Config, error) {
	// .env is optional — absent in deploy (fly.io etc.) where config is supplied
	// via real environment variables. A missing file is not fatal; envconfig reads
	// the OS environment below regardless.
	_ = godotenv.Load(envFiles...)

	cfg := new(Config)
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
