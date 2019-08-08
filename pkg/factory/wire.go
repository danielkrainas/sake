// +build wireinject

package factory

import (
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/google/wire"
)

func Coordinator(ctx RootContext, config *service.Config) (service.CoordinatorService, error) {
	wire.Build(InitializeCoordinator, InitializeHub, InitializeLoggingContext)
	return &service.Coordinator{}, nil
}
