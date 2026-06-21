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
