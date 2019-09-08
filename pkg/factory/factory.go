package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/danielkrainas/sake/pkg/api"
	"github.com/danielkrainas/sake/pkg/service"
	"github.com/danielkrainas/sake/pkg/util/log"
	"go.uber.org/zap/zapcore"
)

func InitializeComponentManager(ctx context.Context, coordinator service.CoordinatorService, server *api.Server) (*service.ComponentManager, error) {
	cm := service.NewComponentManager()
	cm.MustUse(server)
	if coordinator != nil {
		cm.MustUse(service.NewTaskComponent("expiration_trigger", 1*time.Second, zapcore.DebugLevel, &service.ExpirationTrigger{
			Coordinator: coordinator,
		}))

		cm.MustUse(coordinator)
	}

	return cm, nil
}

func InitializeCoordinator(ctx context.Context, hub service.HubConnector, storage service.StorageService, cache service.CacheService) (service.CoordinatorService, error) {
	coordinator, err := service.NewCoordinator(ctx, hub, cache, storage)
	if err != nil {
		return nil, err
	}

	return coordinator, nil
}

func InitializeHub(ctx context.Context, config *service.Config) (service.HubConnector, error) {
	var hub service.HubConnector
	var err error
	switch config.HubProvider {
	case "in-memory":
		hub = service.NewDebugHub()
	case "stan":
		hub, err = service.NewStanHub(config.Nats.ClusterID, config.Nats.Server, config.Nats.ClientID, config.Nats.DurableName)
	default:
		return nil, fmt.Errorf("invalid hub provider %q", config.HubProvider)
	}

	if err != nil {
		return nil, fmt.Errorf("hub init failed: %v", err)
	}

	log.InfoS("%s hub ready", config.HubProvider)
	return hub, nil
}

func InitializeStorage(ctx context.Context, config *service.Config) (service.StorageService, error) {
	storage, err := service.NewDebugStorage([]*service.Workflow{service.Workflows[1]}, nil)
	if err != nil {
		return nil, fmt.Errorf("storage init failed: %v", err)
	}

	return storage, nil
}

func InitializeCache(ctx context.Context, config *service.Config, storage service.StorageService) (service.CacheService, error) {
	var cache service.CacheService
	var err error
	cache, err = service.NewInMemoryCache()
	if err != nil {
		return nil, fmt.Errorf("cache init failed: %v", err)
	}

	cache = &service.WriteThruCache{
		CacheService: cache,
		Storage:      storage,
	}

	return cache, nil
}

func InitializeServer(ctx context.Context, config *service.Config, mux *api.Mux, cache service.CacheService, coordinator service.CoordinatorService) (*api.Server, error) {
	return api.NewServer(ctx, mux, cache, coordinator, api.ServerConfig{
		Addr: config.HTTP.Addr,
	})
}

func InitializeAPI() (*api.Mux, error) {
	return api.NewMux()
}
