// +build wireinject

package factory

import (
	"context"

	"github.com/danielkrainas/sake/pkg/service"
	"github.com/google/wire"
)

func Coordinator(ctx context.Context, config *service.Config) (service.CoordinatorService, error) {
	wire.Build(InitializeCoordinator, InitializeCache, InitializeStorage, InitializeHub)
	return &service.Coordinator{}, nil
}

func ComponentManagerWithCoordinator(ctx context.Context, config *service.Config) (*service.ComponentManager, error) {
	wire.Build(InitializeComponentManager, InitializeServer, InitializeAPI, InitializeCoordinator, InitializeCache, InitializeStorage, InitializeHub)
	return &service.ComponentManager{}, nil
}
