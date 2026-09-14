package app

import (
	"github.com/vukyn/testproj/internal/config"
	idi "github.com/vukyn/testproj/internal/di"

	"github.com/sarulabs/di/v2"
	"github.com/vukyn/kuery/log"
)

var (
	App    di.Container
	Config *config.Config
)

func Init() {
	app, err := idi.NewBuilder().Build() // build all dependencies
	if err != nil {
		log.New().Fatal("Failed to build app", err)
	}
	App = app
	Config = idi.GetConfig(app)

	if err := log.Init(log.Config{
		Mode:  Config.Logger.Mode,
		Level: Config.Logger.Level,
	}); err != nil {
		log.New().Fatal("Failed to initialize logger", err)
	}
	log.New().Info("Logger initialized")

	// Force database initialization by accessing it.
	_ = idi.GetDB(app)
}
