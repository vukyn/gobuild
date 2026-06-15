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
	defs := []*di.Def{defineConfig(), defineDB(), defineMiddleware()}
	defs = append(defs, defineRepository()...)
	defs = append(defs, defineUsecase()...)
	for _, def := range defs {
		if err := builder.Add(def); err != nil {
			log.New().Fatal("Failed to add definition to builder", err)
		}
	}
	return builder
}
