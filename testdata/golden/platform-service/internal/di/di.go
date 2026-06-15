package di

import (
	"github.com/sarulabs/di/v2"
	"github.com/vukyn/kuery/log"
)

func NewBuilder() *di.EnhancedBuilder {
	builder, err := di.NewEnhancedBuilder()
	if err != nil {
		log.New().Fatal("Failed to create builder", err)
	}

	// Register in dependency order: config -> db -> middleware -> repos -> usecases.
	builder.Add(defineConfig())
	builder.Add(defineDB())
	builder.Add(defineMiddleware())
	for _, def := range defineRepository() {
		builder.Add(def)
	}
	for _, def := range defineUsecase() {
		builder.Add(def)
	}
	return builder
}
