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
}

func LoadConfig(envFiles ...string) (*Config, error) {
	if err := godotenv.Load(envFiles...); err != nil {
		return nil, err
	}

	cfg := new(Config)
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
